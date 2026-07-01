// Package logging provides the structured (slog) logger used across the app.
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/xcreativs/caliber/internal/platform/config"
	"github.com/xcreativs/caliber/internal/platform/logging/loki"
)

// New returns a JSON structured logger at the given level and installs it as
// the slog default. Every request/job attaches a correlation id downstream. The
// JSON handler is wrapped in a redacting handler so personal data (emails,
// bearer tokens, JWTs, and secret-named fields) is scrubbed from every line as a
// defense-in-depth backstop to PII-free logging (CAL-117).
//
// New is the stdout-only sink for tests and one-shot tools; long-running
// binaries should use NewWithConfig so Loki shipping is wired when configured.
func New(level string) *slog.Logger {
	return newLogger(level, os.Stdout)
}

// NewWithConfig creates a JSON logger with optional Loki shipping. When
// CALIBER_LOKI_URL is set, the same redacted JSON stream is written to stdout
// and batched to Loki's /loki/api/v1/push endpoint. The returned cleanup func
// flushes any pending Loki batch and should be deferred from the process main.
func NewWithConfig(cfg config.Config) (*slog.Logger, func(context.Context) error, error) {
	var sink io.Writer = os.Stdout
	var lokiWriter *loki.Writer
	if cfg.LokiURL != "" {
		w, err := loki.New(loki.Config{
			URL:           cfg.LokiURL,
			BatchSize:     cfg.LokiBatchSize,
			FlushInterval: cfg.LokiFlushInterval,
			Timeout:       cfg.LokiTimeout,
			TenantID:      cfg.LokiTenantID,
			ServiceName:   cfg.ServiceName,
			Env:           cfg.Env,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("logging: loki writer: %w", err)
		}
		lokiWriter = w
		sink = io.MultiWriter(os.Stdout, lokiWriter)
	}

	cleanup := func(ctx context.Context) error {
		if lokiWriter != nil {
			return lokiWriter.Close(ctx)
		}
		return nil
	}
	return newLogger(cfg.LogLevel, sink), cleanup, nil
}

func newLogger(level string, w io.Writer) *slog.Logger {
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: parseLevel(level),
	})
	l := slog.New(newRedactingHandler(h))
	slog.SetDefault(l)
	return l
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
