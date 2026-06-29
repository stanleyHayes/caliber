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
	contestapp "github.com/xcreativs/caliber/internal/app/contest"
	dashboardapp "github.com/xcreativs/caliber/internal/app/dashboard"
	identityapp "github.com/xcreativs/caliber/internal/app/identity"
	interviewapp "github.com/xcreativs/caliber/internal/app/interview"
	matchingapp "github.com/xcreativs/caliber/internal/app/matching"
	profilesapp "github.com/xcreativs/caliber/internal/app/profiles"
	"github.com/xcreativs/caliber/internal/app/provisioning"
	"github.com/xcreativs/caliber/internal/app/roles"
	"github.com/xcreativs/caliber/internal/domain/audit"
	candidateagentdom "github.com/xcreativs/caliber/internal/domain/candidateagent"
	"github.com/xcreativs/caliber/internal/domain/identity"
	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
	matchingdom "github.com/xcreativs/caliber/internal/domain/matching"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
	"github.com/xcreativs/caliber/internal/platform/config"
	"github.com/xcreativs/caliber/internal/platform/logging"
	"github.com/xcreativs/caliber/internal/platform/readiness"
	"github.com/xcreativs/caliber/internal/platform/seed"
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

	svc, cleanup, ready, err := buildServices(ctx, cfg, log)
	if err != nil {
		return err
	}
	defer cleanup()

	return server.Run(ctx, cfg, log, svc, ready)
}

// repositories bundles the persistence ports the services share.
type repositories struct {
	roles      role.RoleRepository
	users      identity.UserRepository
	refresh    app.RefreshTokenStore
	candidates talent.CandidateRepository
	profiles   talent.TalentProfileRepository
	apps       candidateagentdom.ApplicationRepository
	interviews interviewdom.InterviewRepository
	matchRepo  matchingdom.MatchRepository
}

// openRepositories selects in-memory (dev) or Postgres repositories. With a
// database it also builds the pgvector-backed shortlist service.
func openRepositories(
	ctx context.Context, cfg config.Config, model app.LLMClient, embedder app.Embedder,
	auditRepo audit.AuditRepository, log *slog.Logger,
) (repositories, *grpcadapter.MatchServer, func(), []readiness.NamedCheck, error) {
	repos := repositories{
		roles: memory.NewRoleRepo(), users: memory.NewUserRepo(), refresh: memory.NewRefreshStore(),
		candidates: memory.NewCandidateRepo(), profiles: memory.NewTalentProfileRepo(), apps: memory.NewApplicationRepo(),
		// Interviews are in-memory regardless of provider until CAL-066 lands a
		// Postgres adapter; the match repo is shared so Flow C's wake-up view can
		// read the shortlist matches Flow A produced.
		interviews: memory.NewInterviewRepo(), matchRepo: memory.NewMatchRepo(),
	}
	cleanup := func() {}
	if cfg.DatabaseURL == "" {
		log.Warn("CALIBER_DATABASE_URL not set; using in-memory repositories (in-memory shortlist recall)")
		seedDemo(ctx, cfg, repos, log)
		shortlister := matchingapp.NewShortlister(
			repos.roles, repos.candidates, repos.profiles,
			memory.NewRecaller(repos.candidates), embedder, model, repos.matchRepo)
		rejections := matchingapp.NewRejectionRecorder(repos.roles, auditRepo, time.Now)
		match := grpcadapter.NewMatchServer(shortlister, matchingapp.NewRefiner(repos.roles, shortlister), rejections)
		return repos, match, cleanup, nil, nil
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return repos, nil, cleanup, nil, err
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return repos, nil, cleanup, nil, err
	}
	repos.roles = postgres.NewRoleRepo(pool)
	repos.users = postgres.NewUserRepo(pool)
	repos.refresh = postgres.NewRefreshStore(pool)
	repos.candidates = postgres.NewCandidateRepo(pool)
	repos.profiles = postgres.NewTalentProfileRepo(pool)
	repos.apps = postgres.NewApplicationRepo(pool)
	repos.matchRepo = postgres.NewMatchRepo(pool)
	shortlister := matchingapp.NewShortlister(
		repos.roles, repos.candidates, repos.profiles, postgres.NewRecaller(pool), embedder, model, repos.matchRepo)
	rejections := matchingapp.NewRejectionRecorder(repos.roles, auditRepo, time.Now)
	log.Info("persistence selected", "provider", "postgres")
	checks := []readiness.NamedCheck{
		{Name: "postgres", Check: readiness.Func(pool.Ping)},
	}
	match := grpcadapter.NewMatchServer(
		shortlister,
		matchingapp.NewRefiner(repos.roles, shortlister),
		rejections,
	)
	return repos, match, pool.Close, checks, nil
}

func buildServices(ctx context.Context, cfg config.Config, log *slog.Logger) (grpcadapter.Services, func(), *readiness.Aggregate, error) {
	model := buildLLM(cfg, log)
	embedder := buildEmbedder(cfg, log)
	cleanup := func() {}
	svc := grpcadapter.Services{}
	ready := readiness.New()
	if cfg.IsProd() && cfg.DatabaseURL == "" {
		return svc, cleanup, ready, errors.New("CALIBER_DATABASE_URL is required in production")
	}

	auditRepo := memory.NewAuditRepo()
	repos, match, cleanup, checks, err := openRepositories(ctx, cfg, model, embedder, auditRepo, log)
	if err != nil {
		return svc, cleanup, ready, err
	}
	if cfg.RedisURL != "" {
		checks = append(checks, readiness.Redis(cfg.RedisURL))
	}
	ready = readiness.New(checks...)
	if match != nil {
		svc.Match = match
	}

	tokens, terr := buildTokenService(cfg, log)
	if terr != nil {
		return svc, cleanup, ready, terr
	}
	idOpts := []identityapp.Option{
		identityapp.WithProvisioner(provisioning.NewCandidateProvisioner(repos.candidates)),
		identityapp.WithThrottle(memory.NewLoginThrottle(time.Now, 0, 0, 0)),
	}
	identitySvc := identityapp.NewService(repos.users, authadapter.NewArgon2idHasher(), tokens, repos.refresh, time.Now, idOpts...)
	svc.Identity = grpcadapter.NewIdentityServer(identitySvc)
	svc.AccessVerifier = tokens
	svc.RateLimiter = grpcadapter.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst, time.Now)

	// These read/act on candidate + role data only (no pgvector), so they run in
	// both the in-memory dev path and the Postgres path.
	var availability grpcadapter.AvailabilityCounter
	if match != nil {
		availability = match.AvailabilityCounter() // the shortlister backs the instant pool-depth signal
	}
	svc.Role = grpcadapter.NewRoleServer(
		roles.NewSpecGenerator(model, repos.roles, time.Now), roles.NewSpecEditor(repos.roles), availability)
	svc.Interview = grpcadapter.NewInterviewServer(
		interviewapp.NewInterviewer(repos.roles, repos.interviews, model, 0, interviewapp.WithPassportUpdater(repos.profiles)))
	svc.Talent = grpcadapter.NewTalentServer(profilesapp.NewProfileBuilder(repos.candidates, repos.profiles, model))
	svc.Agent = grpcadapter.NewAgentServer(
		candidateagentapp.NewAgentRunner(repos.candidates, repos.profiles, repos.roles, repos.apps, model,
			candidateagentapp.WithAuditTrail(auditRepo, time.Now),
			candidateagentapp.WithWakeUpInsights(repos.interviews, repos.matchRepo)), repos.apps)
	svc.Dashboard = grpcadapter.NewDashboardServer(dashboardapp.NewAggregator(repos.candidates, repos.profiles, repos.users, repos.roles))
	svc.Contest = grpcadapter.NewContestServer(contestapp.NewService(memory.NewContestRepo(), auditRepo, time.Now))
	svc.Audit = grpcadapter.NewAuditServer(auditRepo)
	return svc, cleanup, ready, nil
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

// LLM guardrail defaults (CAL-035): bound per-call tokens, simultaneous in-flight
// calls, and the request budget so a misbehaving client or adversarial prompt
// cannot run up provider cost.
const (
	llmMaxTokensCap   = 2048
	llmMaxConcurrency = 8
	llmRatePerSecond  = 20
	llmRateBurst      = 40
)

//nolint:ireturn // returns the audited+guarded LLM facade as the app.LLMClient port; interface return is intentional.
func buildLLM(cfg config.Config, log *slog.Logger) app.LLMClient {
	guarded := llm.NewGuarded(newLLMProvider(cfg, log),
		llm.WithMaxTokens(llmMaxTokensCap),
		llm.WithConcurrency(llmMaxConcurrency),
		llm.WithRateLimiter(llm.NewTokenBucket(llmRatePerSecond, llmRateBurst, nil)),
		llm.WithInjectionHook(func(categories []string) {
			// Category labels only — never prompt content — so logs stay PII-safe.
			log.Warn("llm prompt-injection signal detected", "categories", categories)
		}),
	)
	// Outermost: trace every call (redacted: sizes + latency, no content) for
	// cost and explainability observability (CAL-036).
	return llm.NewAudited(guarded, llm.NewSlogRecorder(log), modelLabel(cfg), nil)
}

func modelLabel(cfg config.Config) string {
	if cfg.AnthropicAPIKey != "" {
		return cfg.AnthropicModel
	}
	return "dev"
}

//nolint:ireturn // selects a concrete LLM implementation from config; interface return is intentional.
func newLLMProvider(cfg config.Config, log *slog.Logger) app.LLMClient {
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

// seedDemo loads the deterministic demo dataset into the in-memory dev stack so
// the Radar, alerts, and pool are populated out of the box (CAL-016). It is a
// no-op when seeding is disabled or any step fails (best-effort, never blocks boot).
func seedDemo(ctx context.Context, cfg config.Config, repos repositories, log *slog.Logger) {
	if !cfg.SeedDemo {
		return
	}
	res, err := seed.Load(ctx, seed.Repositories{
		Users: repos.users, Candidates: repos.candidates, Profiles: repos.profiles, Roles: repos.roles,
	}, authadapter.NewArgon2idHasher(), time.Now())
	if err != nil {
		log.Warn("demo seed skipped", "err", err)
		return
	}
	log.Info("loaded demo dataset",
		"employers", res.Employers, "roles", res.Roles, "candidates", res.Candidates,
		"demo_login_password", seed.DefaultPassword)
}
