package main

import (
	"context"
	"log/slog"
	"testing"

	"github.com/xcreativs/caliber/internal/platform/config"
)

// TestBuildServicesWiresEveryServiceInMemory is a DI smoke test for the dev path:
// with no database configured, buildServices must assemble a fully-wired set of
// gRPC services (plus the auth verifier and rate limiter) over the in-memory
// stack, without error. The flow acceptance tests construct servers by hand, so
// this is the only check that the real composition root has no missing or nil
// dependency — a wiring regression (a forgotten service, a dropped repo) fails here.
func TestBuildServicesWiresEveryServiceInMemory(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	// Env "dev" + empty DatabaseURL selects the in-memory path (no Postgres, no
	// Redis); the token service falls back to an ephemeral secret and the LLM to
	// the deterministic dev provider.
	cfg := config.Config{Env: "dev"}

	svc, cleanup, ready, err := buildServices(context.Background(), cfg, log)
	if err != nil {
		t.Fatalf("buildServices returned error: %v", err)
	}
	if cleanup == nil {
		t.Fatal("expected a non-nil cleanup function")
	}
	defer cleanup()
	if ready == nil {
		t.Fatal("expected a non-nil readiness checker")
	}

	checks := map[string]any{
		"Identity":       svc.Identity,
		"Role":           svc.Role,
		"Match":          svc.Match,
		"Interview":      svc.Interview,
		"Agent":          svc.Agent,
		"Dashboard":      svc.Dashboard,
		"Talent":         svc.Talent,
		"Contest":        svc.Contest,
		"Audit":          svc.Audit,
		"Privacy":        svc.Privacy,
		"AccessVerifier": svc.AccessVerifier,
	}
	for name, dep := range checks {
		if dep == nil {
			t.Errorf("service %q is nil — composition root is missing a dependency", name)
		}
	}
	if svc.RateLimiter == nil {
		t.Error("RateLimiter is nil — the rate-limit interceptor would not be installed")
	}
}

// TestBuildServicesRequiresDatabaseInProd locks the production guard: prod without
// a database URL must fail fast rather than silently boot on in-memory storage.
func TestBuildServicesRequiresDatabaseInProd(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	svc, cleanup, ready, err := buildServices(context.Background(), config.Config{Env: "prod"}, log)
	_ = svc
	_ = cleanup
	_ = ready
	if err == nil {
		t.Fatal("expected buildServices to fail in prod without CALIBER_DATABASE_URL")
	}
}
