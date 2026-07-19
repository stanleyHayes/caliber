// Package wiring holds shared construction logic used by both the API and worker
// entrypoints. It keeps the cmd/ binaries thin and avoids duplicating repository
// and provider setup between processes.
package wiring

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/hibiken/asynqmon"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	authadapter "github.com/xcreativs/caliber/internal/adapters/outbound/auth"
	"github.com/xcreativs/caliber/internal/adapters/outbound/embeddings"
	"github.com/xcreativs/caliber/internal/adapters/outbound/fieldcrypto"
	"github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/adapters/outbound/postgres"
	queueadapter "github.com/xcreativs/caliber/internal/adapters/outbound/queue"
	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/app/prompts"
	candidateagentdom "github.com/xcreativs/caliber/internal/domain/candidateagent"
	"github.com/xcreativs/caliber/internal/domain/identity"
	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
	matchingdom "github.com/xcreativs/caliber/internal/domain/matching"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
	"github.com/xcreativs/caliber/internal/platform/config"
	"github.com/xcreativs/caliber/internal/platform/readiness"
	"github.com/xcreativs/caliber/internal/platform/seed"
	"github.com/xcreativs/caliber/internal/platform/telemetry"
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

// OpenRepositories selects in-memory (dev) or Postgres repositories and, in the
// in-memory dev path, loads the deterministic demo dataset when cfg.SeedDemo is
// true. With a database it also returns pgvector-backed readiness checks.
func OpenRepositories(
	ctx context.Context, cfg config.Config, log *slog.Logger,
) (Repositories, func(), []readiness.NamedCheck, error) {
	repos, cleanup, checks, err := openRepositories(ctx, cfg, log)
	if err != nil {
		return repos, cleanup, checks, err
	}
	if cfg.DatabaseURL == "" && cfg.SeedDemo {
		// Boot is best-effort: a seed failure must never block the API starting.
		if serr := SeedDemo(ctx, cfg, repos, log); serr != nil {
			log.Warn("demo seed skipped", "err", serr)
		}
	}
	return repos, cleanup, checks, nil
}

// openRepositories creates repositories without seeding, so reseed can open a
// blank store and then reload the demo dataset itself.
func openRepositories(
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
		return repos, cleanup, nil, nil
	}
	pool, err := newPGPool(ctx, cfg)
	if err != nil {
		return repos, cleanup, nil, err
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return repos, cleanup, nil, err
	}
	fieldCipher, cerr := fieldcrypto.NewFieldCipher(cfg.FieldEncryptionKey, cfg.FieldEncryptionKeyPrevious...)
	if cerr != nil {
		pool.Close()
		return repos, cleanup, nil, cerr
	}
	repos.Roles = postgres.NewRoleRepo(pool)
	repos.Users = postgres.NewUserRepo(pool)
	repos.Refresh = postgres.NewRefreshStore(pool)
	repos.Candidates = postgres.NewCandidateRepo(pool, postgres.WithCandidateCipher(fieldCipher))
	repos.Profiles = postgres.NewTalentProfileRepo(pool, postgres.WithFieldCipher(fieldCipher))
	repos.Apps = postgres.NewApplicationRepo(pool)
	repos.Matches = postgres.NewMatchRepo(pool, postgres.WithMatchCipher(fieldCipher))
	repos.Interviews = postgres.NewInterviewRepo(pool, postgres.WithInterviewCipher(fieldCipher))
	repos.Pool = pool
	log.Info("persistence selected", "provider", "postgres")
	checks := []readiness.NamedCheck{{Name: "postgres", Check: readiness.Func(pool.Ping)}}
	return repos, pool.Close, checks, nil
}

// newPGPool builds the pgvector connection pool. When HNSWEfSearch is set
// (CAL-156), each pooled connection runs `SET hnsw.ef_search` once on connect so
// the recall query trades a little latency for better HNSW recall under load.
func newPGPool(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
	pcfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	if cfg.HNSWEfSearch > 0 {
		ef := cfg.HNSWEfSearch
		pcfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			// ef is a validated positive config int (never user input), and a GUC
			// name cannot be bound as a parameter, so formatting it is safe.
			_, execErr := conn.Exec(ctx, fmt.Sprintf("SET hnsw.ef_search = %d", ef))
			return execErr
		}
	}
	return pgxpool.NewWithConfig(ctx, pcfg)
}

// ResetRepositories wipes all data from the provided repositories. It supports
// both Postgres (via TRUNCATE against repos.Pool) and in-memory dev
// implementations. The audit trail is only present as an in-memory repository in
// the current architecture and is therefore reset only in the in-memory path.
func ResetRepositories(ctx context.Context, repos Repositories) error {
	if repos.Pool != nil {
		return postgres.TruncateAll(ctx, repos.Pool)
	}
	resetMemory(repos.Users)
	resetMemory(repos.Roles)
	resetMemory(repos.Candidates)
	resetMemory(repos.Profiles)
	resetMemory(repos.Apps)
	resetMemory(repos.Interviews)
	resetMemory(repos.Matches)
	resetMemory(repos.Refresh)
	return nil
}

type memoryResetter interface{ Reset() }

func resetMemory(r any) {
	if rr, ok := r.(memoryResetter); ok {
		rr.Reset()
	}
}

// Reseed opens repositories, wipes them, and reloads the deterministic demo
// dataset. It works for both Postgres and in-memory dev paths (CAL-103).
func Reseed(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	repos, cleanup, _, err := openRepositories(ctx, cfg, log)
	if err != nil {
		return err
	}
	defer cleanup()

	log.Info("resetting repositories")
	if err := ResetRepositories(ctx, repos); err != nil {
		return fmt.Errorf("reset: %w", err)
	}

	// The reseed command always seeds; preserve the user's choice of generated
	// vs hand-curated demo data. Unlike boot, reseed surfaces a seed failure as a
	// non-zero exit so CI (and operators) see an empty-database load rather than a
	// silent success followed by confusing downstream failures.
	seedCfg := cfg
	seedCfg.SeedDemo = true
	if err := SeedDemo(ctx, seedCfg, repos, log); err != nil {
		return fmt.Errorf("reseed: %w", err)
	}
	return nil
}

// SeedDemo loads the deterministic demo dataset into the in-memory dev stack so
// the Radar, alerts, and pool are populated out of the box (CAL-016). When
// CALIBER_SEED_GENERATED is true it uses the parser-driven generation pipeline
// instead (CAL-098). It is a no-op when seeding is disabled, and returns the
// seed error otherwise so callers can choose whether to surface it: boot treats
// it as best-effort (logs and continues), while reseed fails on it.
func SeedDemo(ctx context.Context, cfg config.Config, repos Repositories, log *slog.Logger) error {
	if !cfg.SeedDemo {
		return nil
	}
	seedRepos := seed.Repositories{
		Users: repos.Users, Candidates: repos.Candidates, Profiles: repos.Profiles, Roles: repos.Roles,
		Interviews: repos.Interviews, Applications: repos.Apps,
	}
	// Use the raw provider (dev/Claude) rather than the guarded/audited facade
	// for seeding: we are generating fixtures, not serving user traffic, and
	// the rate/concurrency guard would otherwise throttle a batch run.
	seedEmbedder := BuildEmbedder(cfg, log)
	if cfg.SeedGenerated {
		gen := seed.NewGenerator(authadapter.NewArgon2idHasher(), newLLMProvider(cfg, log), time.Now, seed.WithGeneratorEmbedder(seedEmbedder))
		res, err := gen.Generate(ctx, seedRepos)
		if err != nil {
			return fmt.Errorf("generated demo seed: %w", err)
		}
		log.Info("generated demo dataset",
			"employers", res.Employers, "roles", res.Roles, "candidates", res.Candidates,
			"demo_login_password", seed.DefaultPassword)
		return nil
	}

	// Pre-run interview/agent fixtures don't need a real model — use the fast,
	// deterministic dev provider so boot stays quick and doesn't spend real API
	// budget on demo data every restart. Runtime traffic and the vector embedder
	// still use the configured providers (matching needs one embedding space).
	fixtureLLM := llm.NewDev()
	res, err := seed.Load(ctx, seedRepos, authadapter.NewArgon2idHasher(), time.Now(),
		seed.WithEmbedder(seedEmbedder),
		seed.WithPreRunInterviews(fixtureLLM),
		seed.WithPreSeededAgentState(fixtureLLM, repos.Apps),
	)
	if err != nil {
		return fmt.Errorf("demo seed: %w", err)
	}
	log.Info("loaded demo dataset",
		"employers", res.Employers, "roles", res.Roles, "candidates", res.Candidates,
		"interviews", res.Interviews, "applications", res.Applications,
		"demo_login_password", seed.DefaultPassword)
	return nil
}

// LLM guardrail production defaults (CAL-035). The Config object carries the
// effective values loaded from env vars; these constants are the fallback when
// a zero value slips through (e.g., tests that build a bare Config{}).
const (
	llmMaxTokensCap   = 4096
	llmMaxConcurrency = 8
	llmRatePerSecond  = 20
	llmRateBurst      = 40
)

func effectiveLLMGuard(cfg config.Config) (int, int, int, float64) {
	maxTokens := cfg.LLMMaxTokens
	if maxTokens <= 0 {
		maxTokens = llmMaxTokensCap
	}
	maxConcurrency := cfg.LLMMaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = llmMaxConcurrency
	}
	rateBurst := cfg.LLMRateBurst
	if rateBurst <= 0 {
		rateBurst = llmRateBurst
	}
	ratePerSecond := cfg.LLMRatePerSecond
	if ratePerSecond <= 0 {
		ratePerSecond = llmRatePerSecond
	}
	return maxTokens, maxConcurrency, rateBurst, ratePerSecond
}

// BuildLLM constructs the audited+guarded LLM facade from config. It returns the
// client plus a queryable memory recorder that powers the AI-quality debug
// endpoint (CAL-137). The recorder retains the last 256 redacted call traces.
// When tele is non-nil, AI call records are also converted to Prometheus metrics
// (CAL-131).
//
//nolint:ireturn,nolintlint // port constructor intentionally returns the app.LLMClient interface.
func BuildLLM(
	cfg config.Config, log *slog.Logger, tele *telemetry.Provider,
) (app.LLMClient, *llm.MemoryRecorder, *app.CostTracker) {
	mem := llm.NewMemoryRecorder(0)
	recorders := []app.AICallRecorder{mem, llm.NewSlogRecorder(log)}
	if tele != nil {
		if aiMetrics, err := telemetry.NewAIMetricsRecorder(tele); err == nil {
			recorders = append(recorders, aiMetrics)
		} else {
			log.Warn("telemetry: ai metrics recorder unavailable", "err", err)
		}
	}
	// Cost/FinOps controls (CAL-159): a tracker observes every call's estimated
	// spend, alerts as budget thresholds are crossed, and — when a budget is set —
	// gates the guarded client so calls fail fast once the budget is exhausted.
	cost := app.NewCostTracker(cfg.LLMBudgetUSD, func(a app.CostAlert) {
		const msg = "llm spend budget threshold crossed"
		attrs := []any{
			"alert", "llm_budget",
			"spent_usd", a.SpentUSD, "budget_usd", a.BudgetUSD,
			"fraction", a.Fraction, "exceeded", a.Exceeded,
		}
		// Warn as thresholds approach; escalate to error once the budget is spent.
		if a.Exceeded {
			log.Error(msg, attrs...)
		} else {
			log.Warn(msg, attrs...)
		}
	})
	recorders = append(recorders, cost)
	rec := llm.NewMultiRecorder(recorders...)
	maxTokens, maxConcurrency, rateBurst, ratePerSecond := effectiveLLMGuard(cfg)
	guarded := llm.NewGuarded(newLLMProvider(cfg, log),
		llm.WithMaxTokens(maxTokens),
		llm.WithConcurrency(maxConcurrency),
		llm.WithRateLimiter(llm.NewTokenBucket(ratePerSecond, rateBurst, nil)),
		llm.WithBudgetGuard(cost),
		llm.WithInjectionHook(func(categories []string) {
			// Category labels only — never prompt content — so logs stay PII-safe.
			log.Warn("llm prompt-injection signal detected", "categories", categories)
		}),
		llm.WithRecorder(mem),
	)
	// Trace every call (redacted: sizes + latency, no content) for cost and
	// explainability observability (CAL-036).
	audited := llm.NewAudited(guarded, rec, modelLabel(cfg), nil)
	// Outermost: model-tier routing (CAL-159) — route mechanical ops (CV text
	// extraction) to a cheaper model when configured, before the audited recorder
	// attributes spend, so cost tracks the model actually used.
	return llm.NewTierRouter(audited, cfg.LLMCheapModel, string(prompts.IDCVExtract)), mem, cost
}

func modelLabel(cfg config.Config) string {
	if cfg.AnthropicAPIKey != "" || cfg.AnthropicAuthToken != "" {
		return cfg.AnthropicModel
	}
	return "dev"
}

//nolint:ireturn,nolintlint // provider selection intentionally returns the app.LLMClient port.
func newLLMProvider(cfg config.Config, log *slog.Logger) app.LLMClient {
	// A real provider is configured when either credential is present: the
	// first-party Anthropic x-api-key, or a bearer token for an
	// Anthropic-compatible gateway (e.g. Kimi/Moonshot). Base URL is empty for the
	// former and points at the gateway for the latter.
	if cfg.AnthropicAPIKey != "" || cfg.AnthropicAuthToken != "" {
		opts := []llm.ClaudeOption{llm.WithModel(cfg.AnthropicModel), llm.WithBaseURL(cfg.AnthropicBaseURL)}
		provider := "claude"
		if cfg.AnthropicAuthToken != "" {
			// Prefer the bearer token so a stray x-api-key isn't sent alongside it.
			opts = append(opts, llm.WithAuthToken(cfg.AnthropicAuthToken))
			provider = "anthropic-compatible"
		} else {
			opts = append(opts, llm.WithAPIKey(cfg.AnthropicAPIKey))
		}
		// base_url is not a secret; log it to make the active provider obvious.
		log.Info("llm provider selected", "provider", provider, "model", cfg.AnthropicModel, "base_url", cfg.AnthropicBaseURL)
		return llm.NewClaude(opts...)
	}
	log.Warn("no LLM credential set (ANTHROPIC_API_KEY / ANTHROPIC_AUTH_TOKEN); using deterministic dev LLM")
	return llm.NewDev()
}

// BuildEmbedder constructs the embedder adapter from config.
//
//nolint:ireturn,nolintlint // embedder selection intentionally returns the app.Embedder port.
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
