package config

import "testing"

func TestLoadAppliesDefaults(t *testing.T) {
	t.Setenv("CALIBER_HTTP_ADDR", "")
	t.Setenv("CALIBER_GRPC_ADDR", "")
	t.Setenv("CALIBER_ENV", "")
	t.Setenv("CALIBER_LOG_LEVEL", "")
	t.Setenv("CALIBER_WORKER_CONCURRENCY", "")

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
	if cfg.WorkerConcurrency != 4 {
		t.Errorf("WorkerConcurrency = %d, want 4", cfg.WorkerConcurrency)
	}
}

func TestLoadParsesWorkerConcurrency(t *testing.T) {
	t.Setenv("CALIBER_WORKER_CONCURRENCY", "7")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.WorkerConcurrency != 7 {
		t.Errorf("WorkerConcurrency = %d, want 7", cfg.WorkerConcurrency)
	}
}

func TestValidateReportsMissingSecrets(t *testing.T) {
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

func TestIsProd(t *testing.T) {
	if !(Config{Env: "prod"}).IsProd() {
		t.Error("IsProd() = false for env=prod, want true")
	}
	if (Config{Env: "dev"}).IsProd() {
		t.Error("IsProd() = true for env=dev, want false")
	}
}
