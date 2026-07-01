package main

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/platform/config"
)

func TestRunRequiresRedisURL(t *testing.T) {
	cfg := config.Config{RedisURL: ""}
	err := runWorker(context.Background(), cfg, slog.New(slog.DiscardHandler), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CALIBER_REDIS_URL")
}

func TestRedactedURLHidesPassword(t *testing.T) {
	assert.Equal(t, "redis://user:redacted@host:6379/0", redactedURL("redis://user:secret@host:6379/0"))
	assert.Equal(t, "redis://localhost:6379/0", redactedURL("redis://localhost:6379/0"))
}

func TestRunWorkerStartsAndStops(t *testing.T) {
	redis := miniredis.RunT(t)
	cfg := config.Config{
		RedisURL:          "redis://" + redis.Addr() + "/0",
		WorkerConcurrency: 1,
		SeedDemo:          false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := runWorker(ctx, cfg, slog.New(slog.DiscardHandler), nil)
	require.NoError(t, err)
}
