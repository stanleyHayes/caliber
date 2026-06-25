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
	identityapp "github.com/xcreativs/caliber/internal/app/identity"
	matchingapp "github.com/xcreativs/caliber/internal/app/matching"
	"github.com/xcreativs/caliber/internal/app/roles"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/role"
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

func buildServices(ctx context.Context, cfg config.Config, log *slog.Logger) (grpcadapter.Services, func(), error) {
	model := buildLLM(cfg, log)
	embedder := buildEmbedder(cfg, log)
	cleanup := func() {}
	svc := grpcadapter.Services{}

	var roleRepo role.RoleRepository = memory.NewRoleRepo()
	var userRepo identity.UserRepository = memory.NewUserRepo()
	if cfg.DatabaseURL != "" {
		pool, perr := pgxpool.New(ctx, cfg.DatabaseURL)
		if perr != nil {
			return svc, cleanup, perr
		}
		if perr = pool.Ping(ctx); perr != nil {
			pool.Close()
			return svc, cleanup, perr
		}
		cleanup = pool.Close
		roleRepo = postgres.NewRoleRepo(pool)
		userRepo = postgres.NewUserRepo(pool)
		shortlister := matchingapp.NewShortlister(
			roleRepo, postgres.NewCandidateRepo(pool), postgres.NewTalentProfileRepo(pool),
			postgres.NewRecaller(pool), embedder, model, postgres.NewMatchRepo(pool),
		)
		svc.Match = grpcadapter.NewMatchServer(shortlister)
		log.Info("persistence selected", "provider", "postgres")
	} else {
		log.Warn("CALIBER_DATABASE_URL not set; using in-memory repositories (matching disabled)")
	}

	tokens, terr := buildTokenService(cfg, log)
	if terr != nil {
		return svc, cleanup, terr
	}
	identitySvc := identityapp.NewService(userRepo, authadapter.NewArgon2idHasher(), tokens, memory.NewRefreshStore(), time.Now)
	svc.Identity = grpcadapter.NewIdentityServer(identitySvc)
	svc.AccessVerifier = tokens

	svc.Role = grpcadapter.NewRoleServer(roles.NewSpecGenerator(model, roleRepo, time.Now))
	return svc, cleanup, nil
}

// buildTokenService constructs the JWT service. In production a strong
// CALIBER_JWT_SECRET is mandatory (boot fails without it); in dev an ephemeral
// random secret is generated so the server runs, with a warning.
//
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
