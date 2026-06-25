// Package logging provides the structured (slog) logger used across the app.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a JSON structured logger at the given level and installs it as
// the slog default. Every request/job attaches a correlation id downstream.
func New(level string) *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLevel(level),
	})
	l := slog.New(h)
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
