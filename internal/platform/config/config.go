// Package config loads typed application configuration from the environment.
// Secrets never live in code or VCS (see .env.example); only env vars are read.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	corsOriginsEnv       = "CALIBER_CORS_ORIGINS"
	legacyCORSOriginsEnv = "CALIBER_CORS_ALLOWED_ORIGINS"
)

// Config holds all runtime configuration for the API and worker processes.
type Config struct {
	Env      string // "dev" | "staging" | "prod"
	LogLevel string // "debug" | "info" | "warn" | "error"

	HTTPAddr string // REST gateway + health, e.g. ":8080"
	GRPCAddr string // gRPC services, e.g. ":9090"

	// AllowedOrigins is the strict CORS allowlist (CAL-114): exact SPA origins
	// permitted to call the API cross-origin. Empty means same-origin only.
	AllowedOrigins []string

	DatabaseURL string // Postgres + pgvector DSN
	RedisURL    string // Redis (Asynq) URL

	AnthropicAPIKey      string // Claude
	AnthropicModel       string // Claude model id (default claude-opus-4-8)
	OpenAIAPIKey         string // embeddings
	OpenAIEmbeddingModel string // embedding model (default text-embedding-3-small)
	JWTSecret            string // access/refresh token signing
	JWTIssuer            string // token "iss" claim
	JWTAudience          string // token "aud" claim
	AccessTokenTTL       time.Duration
	RefreshTokenTTL      time.Duration

	SeedDemo      bool // load the hand-curated demo dataset into the in-memory dev stack
	SeedGenerated bool // generate a larger demo dataset through the real parsers

	RateLimitRPS   float64 // per-principal sustained request rate (token-bucket refill/sec)
	RateLimitBurst float64 // per-principal burst ceiling (max tokens)

	DashboardCacheTTL time.Duration // Talent Radar snapshot TTL (CAL-080)

	InterviewMaxQuestions int           // Flow B hard cap on question count (CAL-104)
	InterviewMaxDuration  time.Duration // Flow B hard cap on total elapsed time (CAL-104)

	WorkerConcurrency int           // Asynq worker concurrency
	TaskMaxRetry      int           // Asynq max retries per task
	TaskRetention     time.Duration // how long completed tasks remain inspectable

	// Observability configuration (CAL-130/131).
	OTelExporter   string // "noop" | "stdout"
	ServiceName    string // service name for traces and metrics
	ServiceVersion string // service version for traces and metrics

	// Loki centralized logging configuration (CAL-132).
	LokiURL           string        // Loki push URL, e.g. http://localhost:3100
	LokiBatchSize     int           // max entries per push payload
	LokiFlushInterval time.Duration // max time before flushing a partial batch
	LokiTimeout       time.Duration // HTTP push timeout
	LokiTenantID      string        // optional X-Scope-OrgID tenant header
}

// Load reads configuration from the environment, applying sane defaults.
// It returns an error only for values that are malformed; missing secrets are
// reported by Validate so a bare server can still boot in local/dev.
func Load() (Config, error) {
	env := getenv("CALIBER_ENV", "dev")
	allowedOrigins, err := getorigins(env)
	if err != nil {
		return Config{}, err
	}
	c := Config{
		Env:                  env,
		LogLevel:             getenv("CALIBER_LOG_LEVEL", "info"),
		HTTPAddr:             getenv("CALIBER_HTTP_ADDR", ":8080"),
		GRPCAddr:             getenv("CALIBER_GRPC_ADDR", ":9090"),
		AllowedOrigins:       allowedOrigins,
		DatabaseURL:          os.Getenv("CALIBER_DATABASE_URL"),
		RedisURL:             os.Getenv("CALIBER_REDIS_URL"),
		AnthropicAPIKey:      os.Getenv("ANTHROPIC_API_KEY"),
		AnthropicModel:       getenv("CALIBER_ANTHROPIC_MODEL", "claude-opus-4-8"),
		OpenAIAPIKey:         os.Getenv("OPENAI_API_KEY"),
		OpenAIEmbeddingModel: getenv("CALIBER_OPENAI_EMBEDDING_MODEL", "text-embedding-3-small"),
		JWTSecret:            os.Getenv("CALIBER_JWT_SECRET"),
		JWTIssuer:            getenv("CALIBER_JWT_ISSUER", "caliber"),
		JWTAudience:          getenv("CALIBER_JWT_AUDIENCE", "caliber-api"),
		AccessTokenTTL:       getdur("CALIBER_ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:      getdur("CALIBER_REFRESH_TOKEN_TTL", 7*24*time.Hour),
		SeedDemo:             getbool("CALIBER_SEED_DEMO", true),
		SeedGenerated:        getbool("CALIBER_SEED_GENERATED", false),
		// Generous defaults: no human-driven session approaches these, but they
		// cap floods and runaway clients on the expensive AI endpoints (CAL-112).
		RateLimitRPS:      getfloat("CALIBER_RATE_LIMIT_RPS", 30),
		RateLimitBurst:    getfloat("CALIBER_RATE_LIMIT_BURST", 60),
		DashboardCacheTTL: getdur("CALIBER_DASHBOARD_CACHE_TTL", 30*time.Second),
		InterviewMaxQuestions: getint("CALIBER_INTERVIEW_MAX_QUESTIONS", 4),
		InterviewMaxDuration:  getdur("CALIBER_INTERVIEW_MAX_DURATION", 10*time.Minute),
		WorkerConcurrency: getint("CALIBER_WORKER_CONCURRENCY", 4),
		TaskMaxRetry:      getint("CALIBER_TASK_MAX_RETRY", 3),
		TaskRetention:     getdur("CALIBER_TASK_RETENTION", 24*time.Hour),

		OTelExporter:   getenv("CALIBER_OTEL_EXPORTER", "noop"),
		ServiceName:    getenv("CALIBER_SERVICE_NAME", "caliber-api"),
		ServiceVersion: getenv("CALIBER_SERVICE_VERSION", "dev"),

		LokiURL:           os.Getenv("CALIBER_LOKI_URL"),
		LokiBatchSize:     getint("CALIBER_LOKI_BATCH_SIZE", 100),
		LokiFlushInterval: getdur("CALIBER_LOKI_FLUSH_INTERVAL", 5*time.Second),
		LokiTimeout:       getdur("CALIBER_LOKI_TIMEOUT", 10*time.Second),
		LokiTenantID:      os.Getenv("CALIBER_LOKI_TENANT_ID"),
	}
	if c.HTTPAddr == "" || c.GRPCAddr == "" {
		return Config{}, errors.New("config: HTTP and gRPC addresses must be set")
	}
	if c.LokiURL != "" {
		parsed, err := url.Parse(c.LokiURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return Config{}, fmt.Errorf("config: invalid CALIBER_LOKI_URL %q", c.LokiURL)
		}
	}
	return c, nil
}

// IsProd reports whether the process runs in a production-like environment.
func (c Config) IsProd() bool { return strings.EqualFold(c.Env, "prod") }

// Validate returns the names of required-but-missing settings for the given
// environment. Callers decide whether to fail hard (prod) or warn (dev).
func (c Config) Validate() []string {
	var missing []string
	required := map[string]string{
		"CALIBER_DATABASE_URL": c.DatabaseURL,
		"CALIBER_REDIS_URL":    c.RedisURL,
		"ANTHROPIC_API_KEY":    c.AnthropicAPIKey,
		"OPENAI_API_KEY":       c.OpenAIAPIKey,
		"CALIBER_JWT_SECRET":   c.JWTSecret,
	}
	for name, val := range required {
		if strings.TrimSpace(val) == "" {
			missing = append(missing, name)
		}
	}
	if c.IsProd() && len(c.AllowedOrigins) == 0 {
		missing = append(missing, corsOriginsEnv)
	}
	return missing
}

func getbool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func getint(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// getfloat parses a float from the environment, falling back to def when unset
// or malformed.
func getfloat(key string, def float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getorigins(env string) ([]string, error) {
	raw, source := corsOriginsValue()
	if strings.TrimSpace(raw) == "" {
		return append([]string(nil), defaultCORSAllowedOrigins(env)...), nil
	}

	parts := strings.Split(raw, ",")
	seen := make(map[string]struct{}, len(parts))
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		origin := strings.TrimRight(strings.TrimSpace(part), "/")
		if origin == "" {
			continue
		}
		normalized, err := normalizeOrigin(source, origin)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		origins = append(origins, normalized)
	}
	return origins, nil
}

func corsOriginsValue() (string, string) {
	if raw := os.Getenv(corsOriginsEnv); strings.TrimSpace(raw) != "" {
		return raw, corsOriginsEnv
	}
	if raw := os.Getenv(legacyCORSOriginsEnv); strings.TrimSpace(raw) != "" {
		return raw, legacyCORSOriginsEnv
	}
	return "", corsOriginsEnv
}

func defaultCORSAllowedOrigins(env string) []string {
	if strings.EqualFold(env, "prod") {
		return nil
	}
	return []string{
		"http://localhost:5173",
		"http://127.0.0.1:5173",
		"http://localhost:4173",
		"http://127.0.0.1:4173",
	}
}

func normalizeOrigin(source, origin string) (string, error) {
	if strings.Contains(origin, "*") {
		return "", errors.New("config: " + source + " must not contain wildcard origins")
	}
	u, err := url.Parse(origin)
	if err != nil {
		return "", errors.New("config: invalid CORS origin " + origin)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("config: CORS origins must use http or https")
	}
	if u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("config: CORS origins must be exact scheme://host[:port] values")
	}
	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host), nil
}

// getdur parses a Go duration (e.g. "15m", "168h") from the environment,
// falling back to def when unset or malformed.
func getdur(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
