package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/platform/config"
	"github.com/xcreativs/caliber/internal/platform/telemetry"
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

func TestRunWorkerExposesMetricsServer(t *testing.T) {
	redis := miniredis.RunT(t)

	lnCfg := &net.ListenConfig{}
	ln, err := lnCfg.Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	metricsAddr := ln.Addr().String()
	require.NoError(t, ln.Close())

	tele, err := telemetry.New(config.Config{Env: "test", OTelExporter: "noop", ServiceName: "caliber-worker-test"})
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = tele.Shutdown(ctx)
	}()

	cfg := config.Config{
		RedisURL:          "redis://" + redis.Addr() + "/0",
		WorkerConcurrency: 1,
		SeedDemo:          false,
		MetricsAddr:       metricsAddr,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- runWorker(ctx, cfg, slog.New(slog.DiscardHandler), tele) }()

	client := &http.Client{Timeout: 2 * time.Second}
	var body string
	require.Eventually(t, func() bool {
		req, reqErr := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+metricsAddr+"/metrics", nil)
		if reqErr != nil {
			return false
		}
		resp, err := client.Do(req)
		if err != nil {
			return false
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return false
		}
		buf := make([]byte, 4096)
		n, _ := resp.Body.Read(buf)
		body = string(buf[:n])
		return true
	}, 3*time.Second, 50*time.Millisecond)

	assert.Contains(t, body, "target_info")

	cancel()
	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("runWorker did not stop after context cancellation")
	}
}
