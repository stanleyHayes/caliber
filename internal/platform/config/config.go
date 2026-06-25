// Package config loads typed application configuration from the environment.
// Secrets never live in code or VCS (see .env.example); only env vars are read.
package config

import (
	"errors"
	"os"
	"strings"
)

// Config holds all runtime configuration for the API and worker processes.
type Config struct {
	Env      string // "dev" | "staging" | "prod"
	LogLevel string // "debug" | "info" | "warn" | "error"

	HTTPAddr string // REST gateway + health, e.g. ":8080"
	GRPCAddr string // gRPC services, e.g. ":9090"

	DatabaseURL string // Postgres + pgvector DSN
	RedisURL    string // Redis (Asynq) URL

	AnthropicAPIKey string // Claude
	OpenAIAPIKey    string // embeddings
	JWTSecret       string // access/refresh token signing
}

// Load reads configuration from the environment, applying sane defaults.
// It returns an error only for values that are malformed; missing secrets are
// reported by Validate so a bare server can still boot in local/dev.
func Load() (Config, error) {
	c := Config{
		Env:             getenv("CALIBER_ENV", "dev"),
		LogLevel:        getenv("CALIBER_LOG_LEVEL", "info"),
		HTTPAddr:        getenv("CALIBER_HTTP_ADDR", ":8080"),
		GRPCAddr:        getenv("CALIBER_GRPC_ADDR", ":9090"),
		DatabaseURL:     os.Getenv("CALIBER_DATABASE_URL"),
		RedisURL:        os.Getenv("CALIBER_REDIS_URL"),
		AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
		OpenAIAPIKey:    os.Getenv("OPENAI_API_KEY"),
		JWTSecret:       os.Getenv("CALIBER_JWT_SECRET"),
	}
	if c.HTTPAddr == "" || c.GRPCAddr == "" {
		return Config{}, errors.New("config: HTTP and gRPC addresses must be set")
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
	return missing
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
