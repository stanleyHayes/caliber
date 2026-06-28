package logging

import (
	"context"
	"log/slog"
	"testing"
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
