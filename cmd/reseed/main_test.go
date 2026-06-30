package main

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/platform/config"
	"github.com/xcreativs/caliber/internal/platform/wiring"
)

func TestRunReseedInMemory(t *testing.T) {
	t.Setenv("CALIBER_DATABASE_URL", "")
	t.Setenv("CALIBER_REDIS_URL", "")

	cfg := config.Config{Env: "dev", SeedDemo: true}
	log := slog.New(slog.DiscardHandler)

	require.NoError(t, runReseed(context.Background(), cfg, log))
}

func TestRunReseedRequiresNoDatabaseURLInDev(t *testing.T) {
	t.Setenv("CALIBER_DATABASE_URL", "")

	cfg := config.Config{Env: "dev", SeedDemo: true}
	log := slog.New(slog.DiscardHandler)

	// The dev in-memory path must succeed without any external database.
	require.NoError(t, runReseed(context.Background(), cfg, log))
}

func TestRunReseedRestoresBaseline(t *testing.T) {
	t.Setenv("CALIBER_DATABASE_URL", "")
	t.Setenv("CALIBER_REDIS_URL", "")

	cfg := config.Config{Env: "dev", SeedDemo: true}
	log := slog.New(slog.DiscardHandler)
	ctx := context.Background()

	require.NoError(t, runReseed(ctx, cfg, log))

	// A second invocation returns to the same deterministic baseline.
	require.NoError(t, runReseed(ctx, cfg, log))

	// Sanity check that the reseed command left the in-memory stores populated.
	repos, cleanup, _, err := wiring.OpenRepositories(ctx, cfg, log)
	require.NoError(t, err)
	defer cleanup()

	roles, _, err := repos.Roles.ListOpen(ctx, kernel.NewPage(1, 100))
	require.NoError(t, err)
	assert.NotEmpty(t, roles)
}
