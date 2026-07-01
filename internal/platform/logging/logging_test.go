package logging

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/platform/config"
)

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"debug":    slog.LevelDebug,
		"DEBUG":    slog.LevelDebug,
		"  warn  ": slog.LevelWarn,
		"warning":  slog.LevelWarn,
		"error":    slog.LevelError,
		"info":     slog.LevelInfo,
		"":         slog.LevelInfo, // unknown/empty defaults to info
		"verbose":  slog.LevelInfo,
	}
	for in, want := range cases {
		if got := parseLevel(in); got != want {
			t.Errorf("parseLevel(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestNewReturnsLeveledDefaultLogger(t *testing.T) {
	l := New("warn")
	if l == nil {
		t.Fatal("New returned nil")
	}
	if slog.Default() != l {
		t.Error("New did not install the logger as the slog default")
	}
	ctx := context.Background()
	if l.Enabled(ctx, slog.LevelInfo) {
		t.Error("info should be below the warn threshold (disabled)")
	}
	if !l.Enabled(ctx, slog.LevelWarn) {
		t.Error("warn should be enabled at the warn threshold")
	}
}

func TestNewWithConfigLokiDisabled(t *testing.T) {
	cfg := config.Config{LogLevel: "info"}
	log, cleanup, err := NewWithConfig(cfg)
	require.NoError(t, err)
	require.NotNil(t, log)
	require.NoError(t, cleanup(context.Background()))
}

func TestNewWithConfigShipsRedactedLogsToLoki(t *testing.T) {
	type lokiPayload struct {
		Streams []struct {
			Stream map[string]string `json:"stream"`
			Values [][2]string       `json:"values"`
		} `json:"streams"`
	}

	var got lokiPayload
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/loki/api/v1/push", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		defer mu.Unlock()
		if err := json.Unmarshal(body, &got); err != nil {
			t.Errorf("unmarshal payload: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	cfg := config.Config{
		LogLevel:      "info",
		LokiURL:       srv.URL,
		LokiBatchSize: 1,
		LokiTimeout:   5 * time.Second,
		ServiceName:   "caliber-log-test",
		Env:           "test",
	}
	log, cleanup, err := NewWithConfig(cfg)
	require.NoError(t, err)

	log.Info("hello loki",
		slog.String("request_id", "req-123"),
		slog.String("trace_id", "trace-abc"),
		slog.String("email", "leak@example.com"),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, cleanup(ctx))

	mu.Lock()
	streams := got.Streams
	mu.Unlock()
	require.Len(t, streams, 1)
	assert.Equal(t, map[string]string{"service": "caliber-log-test", "env": "test"}, streams[0].Stream)
	require.Len(t, streams[0].Values, 1)

	line := streams[0].Values[0][1]
	assert.Contains(t, line, `"msg":"hello loki"`)
	assert.Contains(t, line, `"request_id":"req-123"`)
	assert.Contains(t, line, `"trace_id":"trace-abc"`)
	assert.Contains(t, line, redactedPlaceholder)
	assert.NotContains(t, line, "leak@example.com")
}
