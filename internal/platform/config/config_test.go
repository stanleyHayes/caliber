package config

import (
	"slices"
	"testing"
	"time"
)

func clearCORSOriginsEnv(t *testing.T) {
	t.Helper()
	t.Setenv("CALIBER_CORS_ORIGINS", "")
	t.Setenv("CALIBER_CORS_ALLOWED_ORIGINS", "")
}

func TestLoadAppliesDefaults(t *testing.T) {
	t.Setenv("CALIBER_HTTP_ADDR", "")
	t.Setenv("CALIBER_GRPC_ADDR", "")
	t.Setenv("CALIBER_ENV", "")
	t.Setenv("CALIBER_LOG_LEVEL", "")
	t.Setenv("CALIBER_WORKER_CONCURRENCY", "")
	clearCORSOriginsEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.GRPCAddr != ":9090" {
		t.Errorf("GRPCAddr = %q, want :9090", cfg.GRPCAddr)
	}
	if cfg.Env != "dev" {
		t.Errorf("Env = %q, want dev", cfg.Env)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if len(cfg.AllowedOrigins) != 4 {
		t.Errorf("AllowedOrigins len = %d, want 4", len(cfg.AllowedOrigins))
	}
	if cfg.WorkerConcurrency != 4 {
		t.Errorf("WorkerConcurrency = %d, want 4", cfg.WorkerConcurrency)
	}
	if cfg.DashboardCacheTTL != 30*time.Second {
		t.Errorf("DashboardCacheTTL = %v, want 30s", cfg.DashboardCacheTTL)
	}
	if cfg.InterviewMaxQuestions != 4 {
		t.Errorf("InterviewMaxQuestions = %d, want 4", cfg.InterviewMaxQuestions)
	}
	if cfg.InterviewMaxDuration != 10*time.Minute {
		t.Errorf("InterviewMaxDuration = %v, want 10m", cfg.InterviewMaxDuration)
	}
}

func TestLoadParsesStrictCORSOrigins(t *testing.T) {
	t.Setenv("CALIBER_ENV", "")
	clearCORSOriginsEnv(t)
	t.Setenv(
		"CALIBER_CORS_ORIGINS",
		"https://app.example.com, https://ADMIN.example.com/, https://app.example.com",
	)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := []string{"https://app.example.com", "https://admin.example.com"}
	if len(cfg.AllowedOrigins) != len(want) {
		t.Fatalf("AllowedOrigins = %v, want %v", cfg.AllowedOrigins, want)
	}
	for i := range want {
		if cfg.AllowedOrigins[i] != want[i] {
			t.Errorf("AllowedOrigins[%d] = %q, want %q", i, cfg.AllowedOrigins[i], want[i])
		}
	}
}

func TestLoadParsesLegacyCORSAllowedOriginsAlias(t *testing.T) {
	t.Setenv("CALIBER_ENV", "")
	clearCORSOriginsEnv(t)
	t.Setenv("CALIBER_CORS_ALLOWED_ORIGINS", "https://legacy.example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if want := []string{"https://legacy.example.com"}; !slices.Equal(cfg.AllowedOrigins, want) {
		t.Fatalf("AllowedOrigins = %v, want %v", cfg.AllowedOrigins, want)
	}
}

func TestLoadRejectsPermissiveOrMalformedCORSOrigins(t *testing.T) {
	for _, raw := range []string{"*", "example.com", "https://app.example.com/path", "ftp://app.example.com"} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv("CALIBER_ENV", "")
			clearCORSOriginsEnv(t)
			t.Setenv("CALIBER_CORS_ORIGINS", raw)
			if _, err := Load(); err == nil {
				t.Fatal("Load() error = nil, want invalid CORS origin error")
			}
		})
	}
}

func TestLoadParsesWorkerConcurrency(t *testing.T) {
	t.Setenv("CALIBER_ENV", "")
	t.Setenv("CALIBER_WORKER_CONCURRENCY", "7")
	clearCORSOriginsEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.WorkerConcurrency != 7 {
		t.Errorf("WorkerConcurrency = %d, want 7", cfg.WorkerConcurrency)
	}
}

func TestValidateReportsMissingSecrets(t *testing.T) {
	t.Setenv("CALIBER_ENV", "")
	clearCORSOriginsEnv(t)
	for _, k := range []string{
		"CALIBER_DATABASE_URL", "CALIBER_REDIS_URL",
		"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "CALIBER_JWT_SECRET",
	} {
		t.Setenv(k, "")
	}
	cfg, _ := Load()
	if got := cfg.Validate(); len(got) != 5 {
		t.Errorf("Validate() reported %d missing (%v), want 5", len(got), got)
	}
}

func TestValidateRequiresCORSOriginsInProd(t *testing.T) {
	t.Setenv("CALIBER_ENV", "prod")
	clearCORSOriginsEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	missing := cfg.Validate()
	if !slices.Contains(missing, "CALIBER_CORS_ORIGINS") {
		t.Fatalf("Validate() missing = %v, want CALIBER_CORS_ORIGINS", missing)
	}
}

func TestLoadAppliesLokiDefaults(t *testing.T) {
	t.Setenv("CALIBER_ENV", "")
	clearCORSOriginsEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.LokiURL != "" {
		t.Errorf("LokiURL = %q, want empty", cfg.LokiURL)
	}
	if cfg.LokiBatchSize != 100 {
		t.Errorf("LokiBatchSize = %d, want 100", cfg.LokiBatchSize)
	}
	if cfg.LokiFlushInterval != 5*time.Second {
		t.Errorf("LokiFlushInterval = %v, want 5s", cfg.LokiFlushInterval)
	}
	if cfg.LokiTimeout != 10*time.Second {
		t.Errorf("LokiTimeout = %v, want 10s", cfg.LokiTimeout)
	}
	if cfg.LokiTenantID != "" {
		t.Errorf("LokiTenantID = %q, want empty", cfg.LokiTenantID)
	}
	if cfg.MetricsAddr != ":8081" {
		t.Errorf("MetricsAddr = %q, want :8081", cfg.MetricsAddr)
	}
}

func TestLoadParsesLokiOverrides(t *testing.T) {
	t.Setenv("CALIBER_ENV", "")
	clearCORSOriginsEnv(t)
	t.Setenv("CALIBER_LOKI_URL", "http://loki.example.com:3100")
	t.Setenv("CALIBER_LOKI_BATCH_SIZE", "250")
	t.Setenv("CALIBER_LOKI_FLUSH_INTERVAL", "15s")
	t.Setenv("CALIBER_LOKI_TIMEOUT", "30s")
	t.Setenv("CALIBER_LOKI_TENANT_ID", "tenant-42")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.LokiURL != "http://loki.example.com:3100" {
		t.Errorf("LokiURL = %q, want http://loki.example.com:3100", cfg.LokiURL)
	}
	if cfg.LokiBatchSize != 250 {
		t.Errorf("LokiBatchSize = %d, want 250", cfg.LokiBatchSize)
	}
	if cfg.LokiFlushInterval != 15*time.Second {
		t.Errorf("LokiFlushInterval = %v, want 15s", cfg.LokiFlushInterval)
	}
	if cfg.LokiTimeout != 30*time.Second {
		t.Errorf("LokiTimeout = %v, want 30s", cfg.LokiTimeout)
	}
	if cfg.LokiTenantID != "tenant-42" {
		t.Errorf("LokiTenantID = %q, want tenant-42", cfg.LokiTenantID)
	}
}

func TestLoadRejectsInvalidLokiURL(t *testing.T) {
	for _, raw := range []string{"ftp://loki.example.com", "://bad", " "} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv("CALIBER_ENV", "")
			clearCORSOriginsEnv(t)
			t.Setenv("CALIBER_LOKI_URL", raw)
			if _, err := Load(); err == nil {
				t.Fatal("Load() error = nil, want invalid Loki URL error")
			}
		})
	}
}

func TestIsProd(t *testing.T) {
	if !(Config{Env: "prod"}).IsProd() {
		t.Error("IsProd() = false for env=prod, want true")
	}
	if (Config{Env: "dev"}).IsProd() {
		t.Error("IsProd() = true for env=dev, want false")
	}
}
