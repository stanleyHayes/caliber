// Package wiring holds shared construction logic used by both the API and worker
// entrypoints. It keeps the cmd/ binaries thin and avoids duplicating repository
// and provider setup between processes.
package wiring

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/hibiken/asynqmon"
	"github.com/jackc/pgx/v5/pgxpool"

	authadapter "github.com/xcreativs/caliber/internal/adapters/outbound/auth"
	"github.com/xcreativs/caliber/internal/adapters/outbound/embeddings"
	"github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/adapters/outbound/postgres"
	queueadapter "github.com/xcreativs/caliber/internal/adapters/outbound/queue"
	"github.com/xcreativs/caliber/internal/app"
	candidateagentdom "github.com/xcreativs/caliber/internal/domain/candidateagent"
	"github.com/xcreativs/caliber/internal/domain/identity"
	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
	matchingdom "github.com/xcreativs/caliber/internal/domain/matching"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
	"github.com/xcreativs/caliber/internal/platform/config"
	"github.com/xcreativs/caliber/internal/platform/readiness"
	"github.com/xcreativs/caliber/internal/platform/seed"
)

// Repositories bundles the persistence ports the services share.
type Repositories struct {
	Roles      role.RoleRepository
	Users      identity.UserRepository
	Refresh    app.RefreshTokenStore
	Candidates talent.CandidateRepository
	Profiles   talent.TalentProfileRepository
	Apps       candidateagentdom.ApplicationRepository
	Interviews interviewdom.InterviewRepository
	Matches    matchingdom.MatchRepository
	Pool       *pgxpool.Pool // non-nil when DatabaseURL is configured
}

// OpenRepositories selects in-memory (dev) or Postgres repositories. With a
// database it also returns pgvector-backed readiness checks for Postgres.
func OpenRepositories(
	ctx context.Context, cfg config.Config, log *slog.Logger,
) (Repositories, func(), []readiness.NamedCheck, error) {
	repos := Repositories{
		Roles: memory.NewRoleRepo(), Users: memory.NewUserRepo(), Refresh: memory.NewRefreshStore(),
		Candidates: memory.NewCandidateRepo(), Profiles: memory.NewTalentProfileRepo(), Apps: memory.NewApplicationRepo(),
		Interviews: memory.NewInterviewRepo(), Matches: memory.NewMatchRepo(),
	}
	cleanup := func() {}
	if cfg.DatabaseURL == "" {
		log.Warn("CALIBER_DATABASE_URL not set; using in-memory repositories")
		SeedDemo(ctx, cfg, repos, log)
		return repos, cleanup, nil, nil
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return repos, cleanup, nil, err
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return repos, cleanup, nil, err
	}
	repos.Roles = postgres.NewRoleRepo(pool)
	repos.Users = postgres.NewUserRepo(pool)
	repos.Refresh = postgres.NewRefreshStore(pool)
	repos.Candidates = postgres.NewCandidateRepo(pool)
	repos.Profiles = postgres.NewTalentProfileRepo(pool)
	repos.Apps = postgres.NewApplicationRepo(pool)
	repos.Matches = postgres.NewMatchRepo(pool)
	repos.Pool = pool
	// Interviews remain in-memory until CAL-066 lands a Postgres adapter.
	log.Info("persistence selected", "provider", "postgres")
	checks := []readiness.NamedCheck{{Name: "postgres", Check: readiness.Func(pool.Ping)}}
	return repos, pool.Close, checks, nil
}

// SeedDemo loads the deterministic demo dataset into the in-memory dev stack so
// the Radar, alerts, and pool are populated out of the box (CAL-016). When
// CALIBER_SEED_GENERATED is true it uses the parser-driven generation pipeline
// instead (CAL-098). It is a no-op when seeding is disabled or any step fails
// (best-effort, never blocks boot).
func SeedDemo(ctx context.Context, cfg config.Config, repos Repositories, log *slog.Logger) {
	if !cfg.SeedDemo {
		return
	}
	seedRepos := seed.Repositories{
		Users: repos.Users, Candidates: repos.Candidates, Profiles: repos.Profiles, Roles: repos.Roles,
		Interviews: repos.Interviews, Applications: repos.Apps,
	}
	// Use the raw provider (dev/Claude) rather than the guarded/audited facade
	// for seeding: we are generating fixtures, not serving user traffic, and
	// the rate/concurrency guard would otherwise throttle a batch run.
	seedLLM := newLLMProvider(cfg, log)
	if cfg.SeedGenerated {
		gen := seed.NewGenerator(authadapter.NewArgon2idHasher(), seedLLM, time.Now)
		res, err := gen.Generate(ctx, seedRepos)
		if err != nil {
			log.Warn("generated demo seed skipped", "err", err)
			return
		}
		log.Info("generated demo dataset",
			"employers", res.Employers, "roles", res.Roles, "candidates", res.Candidates,
			"demo_login_password", seed.DefaultPassword)
		return
	}

	res, err := seed.Load(ctx, seedRepos, authadapter.NewArgon2idHasher(), time.Now(),
		seed.WithPreRunInterviews(seedLLM),
		seed.WithPreSeededAgentState(seedLLM, repos.Apps),
	)
	if err != nil {
		log.Warn("demo seed skipped", "err", err)
		return
	}
	log.Info("loaded demo dataset",
		"employers", res.Employers, "roles", res.Roles, "candidates", res.Candidates,
		"interviews", res.Interviews, "applications", res.Applications,
		"demo_login_password", seed.DefaultPassword)
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

// BuildLLM constructs the audited+guarded LLM facade from config.
//
//nolint:ireturn // returns the audited+guarded LLM facade as the app.LLMClient port; interface return is intentional.
func BuildLLM(cfg config.Config, log *slog.Logger) app.LLMClient {
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

// BuildEmbedder constructs the embedder adapter from config.
//
//nolint:ireturn // selects a concrete embedder implementation from config; interface return is intentional.
func BuildEmbedder(cfg config.Config, log *slog.Logger) app.Embedder {
	if cfg.OpenAIAPIKey != "" {
		log.Info("embedder selected", "provider", "openai", "model", cfg.OpenAIEmbeddingModel)
		return embeddings.NewOpenAI(embeddings.WithOpenAIKey(cfg.OpenAIAPIKey), embeddings.WithOpenAIModel(cfg.OpenAIEmbeddingModel))
	}
	log.Warn("OPENAI_API_KEY not set; using deterministic dev embedder")
	return embeddings.NewDev()
}

// NewAsynqmonHandler builds the Asynqmon monitoring UI handler for the given
// Redis URL. It returns nil when the URL is empty so the dashboard can be left
// unmounted in local dev stacks without Redis (CAL-028). The caller should call
// the returned cleanup function on shutdown to close Redis connections.
func NewAsynqmonHandler(redisURL string) (http.Handler, func(), error) {
	if redisURL == "" {
		return nil, func() {}, nil
	}
	opt, err := queueadapter.RedisOpt(redisURL)
	if err != nil {
		return nil, func() {}, err
	}
	h := asynqmon.New(asynqmon.Options{RootPath: "/asynqmon", RedisConnOpt: opt})
	return h, func() { _ = h.Close() }, nil
}
