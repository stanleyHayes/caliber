// Command api runs the Caliber backend (gRPC + REST gateway). Wiring only;
// lifecycle lives in internal/platform/server and routing in httpserver.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	grpcadapter "github.com/xcreativs/caliber/internal/adapters/inbound/grpc"
	authadapter "github.com/xcreativs/caliber/internal/adapters/outbound/auth"
	"github.com/xcreativs/caliber/internal/adapters/outbound/embeddings"
	"github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/adapters/outbound/postgres"
	"github.com/xcreativs/caliber/internal/app"
	candidateagentapp "github.com/xcreativs/caliber/internal/app/candidateagent"
	dashboardapp "github.com/xcreativs/caliber/internal/app/dashboard"
	identityapp "github.com/xcreativs/caliber/internal/app/identity"
	interviewapp "github.com/xcreativs/caliber/internal/app/interview"
	matchingapp "github.com/xcreativs/caliber/internal/app/matching"
	profilesapp "github.com/xcreativs/caliber/internal/app/profiles"
	"github.com/xcreativs/caliber/internal/app/provisioning"
	"github.com/xcreativs/caliber/internal/app/roles"
	candidateagentdom "github.com/xcreativs/caliber/internal/domain/candidateagent"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
	"github.com/xcreativs/caliber/internal/platform/config"
	"github.com/xcreativs/caliber/internal/platform/logging"
	"github.com/xcreativs/caliber/internal/platform/server"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logging.New(cfg.LogLevel)
	if missing := cfg.Validate(); len(missing) > 0 {
		log.Warn("missing configuration", "missing", missing, "env", cfg.Env)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	svc, cleanup, err := buildServices(ctx, cfg, log)
	if err != nil {
		return err
	}
	defer cleanup()

	return server.Run(ctx, cfg, log, svc)
}

// repositories bundles the persistence ports the services share.
type repositories struct {
	roles      role.RoleRepository
	users      identity.UserRepository
	refresh    app.RefreshTokenStore
	candidates talent.CandidateRepository
	profiles   talent.TalentProfileRepository
	apps       candidateagentdom.ApplicationRepository
}

// openRepositories selects in-memory (dev) or Postgres repositories. With a
// database it also builds the pgvector-backed shortlist service.
func openRepositories(
	ctx context.Context, cfg config.Config, model app.LLMClient, embedder app.Embedder, log *slog.Logger,
) (repositories, *grpcadapter.MatchServer, func(), error) {
	repos := repositories{
		roles: memory.NewRoleRepo(), users: memory.NewUserRepo(), refresh: memory.NewRefreshStore(),
		candidates: memory.NewCandidateRepo(), profiles: memory.NewTalentProfileRepo(), apps: memory.NewApplicationRepo(),
	}
	cleanup := func() {}
	if cfg.DatabaseURL == "" {
		log.Warn("CALIBER_DATABASE_URL not set; using in-memory repositories (shortlist recall disabled)")
		return repos, nil, cleanup, nil
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return repos, nil, cleanup, err
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return repos, nil, cleanup, err
	}
	repos.roles = postgres.NewRoleRepo(pool)
	repos.users = postgres.NewUserRepo(pool)
	repos.refresh = postgres.NewRefreshStore(pool)
	repos.candidates = postgres.NewCandidateRepo(pool)
	repos.profiles = postgres.NewTalentProfileRepo(pool)
	repos.apps = postgres.NewApplicationRepo(pool)
	shortlister := matchingapp.NewShortlister(
		repos.roles, repos.candidates, repos.profiles, postgres.NewRecaller(pool), embedder, model, postgres.NewMatchRepo(pool))
	log.Info("persistence selected", "provider", "postgres")
	return repos, grpcadapter.NewMatchServer(shortlister, matchingapp.NewRefiner(repos.roles, shortlister)), pool.Close, nil
}

func buildServices(ctx context.Context, cfg config.Config, log *slog.Logger) (grpcadapter.Services, func(), error) {
	model := buildLLM(cfg, log)
	embedder := buildEmbedder(cfg, log)
	cleanup := func() {}
	svc := grpcadapter.Services{}
	if cfg.IsProd() && cfg.DatabaseURL == "" {
		return svc, cleanup, errors.New("CALIBER_DATABASE_URL is required in production")
	}

	repos, match, cleanup, err := openRepositories(ctx, cfg, model, embedder, log)
	if err != nil {
		return svc, cleanup, err
	}
	if match != nil {
		svc.Match = match
	}

	tokens, terr := buildTokenService(cfg, log)
	if terr != nil {
		return svc, cleanup, terr
	}
	idOpts := []identityapp.Option{
		identityapp.WithProvisioner(provisioning.NewCandidateProvisioner(repos.candidates)),
		identityapp.WithThrottle(memory.NewLoginThrottle(time.Now, 0, 0, 0)),
	}
	identitySvc := identityapp.NewService(repos.users, authadapter.NewArgon2idHasher(), tokens, repos.refresh, time.Now, idOpts...)
	svc.Identity = grpcadapter.NewIdentityServer(identitySvc)
	svc.AccessVerifier = tokens

	// These read/act on candidate + role data only (no pgvector), so they run in
	// both the in-memory dev path and the Postgres path.
	svc.Role = grpcadapter.NewRoleServer(roles.NewSpecGenerator(model, repos.roles, time.Now), roles.NewSpecEditor(repos.roles))
	svc.Interview = grpcadapter.NewInterviewServer(
		interviewapp.NewInterviewer(repos.roles, memory.NewInterviewRepo(), model, 0, interviewapp.WithPassportUpdater(repos.profiles)))
	svc.Talent = grpcadapter.NewTalentServer(profilesapp.NewProfileBuilder(repos.candidates, repos.profiles, model))
	svc.Agent = grpcadapter.NewAgentServer(
		candidateagentapp.NewAgentRunner(repos.candidates, repos.profiles, repos.roles, repos.apps, model), repos.apps)
	svc.Dashboard = grpcadapter.NewDashboardServer(dashboardapp.NewAggregator(repos.candidates, repos.profiles, repos.users, repos.roles))
	return svc, cleanup, nil
}

//nolint:ireturn // selects/constructs the concrete TokenService; interface return is intentional.
func buildTokenService(cfg config.Config, log *slog.Logger) (app.TokenService, error) {
	secret := cfg.JWTSecret
	if len(secret) < 32 {
		if cfg.IsProd() {
			return nil, errors.New("CALIBER_JWT_SECRET must be at least 32 bytes in production")
		}
		ephemeral := make([]byte, 32)
		if _, err := rand.Read(ephemeral); err != nil {
			return nil, err
		}
		secret = hex.EncodeToString(ephemeral)
		log.Warn("CALIBER_JWT_SECRET unset/weak; using an ephemeral dev secret (tokens reset on restart)")
	}
	return authadapter.NewJWTService(authadapter.JWTConfig{
		Secret: secret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience,
		AccessTTL: cfg.AccessTokenTTL, RefreshTTL: cfg.RefreshTokenTTL,
	})
}

//nolint:ireturn // selects a concrete LLM implementation from config; interface return is intentional.
func buildLLM(cfg config.Config, log *slog.Logger) app.LLMClient {
	if cfg.AnthropicAPIKey != "" {
		log.Info("llm provider selected", "provider", "claude", "model", cfg.AnthropicModel)
		return llm.NewClaude(llm.WithAPIKey(cfg.AnthropicAPIKey), llm.WithModel(cfg.AnthropicModel))
	}
	log.Warn("ANTHROPIC_API_KEY not set; using deterministic dev LLM")
	return llm.NewDev()
}

//nolint:ireturn // selects a concrete embedder implementation from config; interface return is intentional.
func buildEmbedder(cfg config.Config, log *slog.Logger) app.Embedder {
	if cfg.OpenAIAPIKey != "" {
		log.Info("embedder selected", "provider", "openai", "model", cfg.OpenAIEmbeddingModel)
		return embeddings.NewOpenAI(embeddings.WithOpenAIKey(cfg.OpenAIAPIKey), embeddings.WithOpenAIModel(cfg.OpenAIEmbeddingModel))
	}
	log.Warn("OPENAI_API_KEY not set; using deterministic dev embedder")
	return embeddings.NewDev()
}
