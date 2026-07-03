package config

import "testing"

func TestProdSafetyIssues(t *testing.T) {
	// Outside prod: never flagged, even pointing at local endpoints.
	dev := Config{Env: "dev", DatabaseURL: "postgres://localhost:5432/x", RedisURL: "redis://localhost:6379"}
	if got := dev.ProdSafetyIssues(); len(got) != 0 {
		t.Errorf("dev should have no prod-safety issues, got %v", got)
	}

	// Prod with local endpoints: both flagged (a dev config promoted to prod).
	prodLocal := Config{Env: "prod", DatabaseURL: "postgres://caliber@127.0.0.1:5432/db", RedisURL: "redis://localhost:6379"}
	if got := prodLocal.ProdSafetyIssues(); len(got) != 2 {
		t.Errorf("prod with local endpoints should have 2 issues, got %v", got)
	}

	// Prod with managed endpoints: clean.
	prodOK := Config{Env: "prod", DatabaseURL: "postgres://u:p@db.internal.example.com:5432/db", RedisURL: "rediss://cache.example.com:6379"}
	if got := prodOK.ProdSafetyIssues(); len(got) != 0 {
		t.Errorf("prod with managed endpoints should be clean, got %v", got)
	}

	// Empty endpoints are not "local" — Validate reports missing ones separately.
	if got := (Config{Env: "prod"}).ProdSafetyIssues(); len(got) != 0 {
		t.Errorf("empty endpoints must not be flagged as local, got %v", got)
	}
}
