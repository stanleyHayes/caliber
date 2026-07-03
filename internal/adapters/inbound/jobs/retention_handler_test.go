package jobs_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/inbound/jobs"
	privacyapp "github.com/xcreativs/caliber/internal/app/privacy"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

type stubLister struct{ ids []kernel.ID }

func (s stubLister) ListEligibleForRetention(context.Context, time.Time) ([]kernel.ID, error) {
	return s.ids, nil
}

type stubEraser struct{ erased int }

func (s *stubEraser) EraseCandidate(context.Context, kernel.ID) error {
	s.erased++
	return nil
}

func TestDataRetentionHandler_RunsSweep(t *testing.T) {
	eraser := &stubEraser{}
	sweeper := privacyapp.NewRetentionSweeper(
		stubLister{ids: []kernel.ID{kernel.NewID(), kernel.NewID()}}, eraser, time.Hour, time.Now)

	mux := jobs.NewMux(slog.New(slog.DiscardHandler))
	jobs.RegisterHandlers(mux, jobs.HandlerDeps{RetentionSweeper: sweeper}, slog.New(slog.DiscardHandler))

	require.NoError(t, mux.ProcessTask(context.Background(), jobs.NewDataRetentionTask()))
	assert.Equal(t, 2, eraser.erased, "every eligible candidate is erased by the sweep")
}

func TestDataRetentionHandler_NotWired(t *testing.T) {
	mux := jobs.NewMux(slog.New(slog.DiscardHandler))
	jobs.RegisterHandlers(mux, jobs.HandlerDeps{}, slog.New(slog.DiscardHandler))

	// With no sweeper wired the task fails (so it is retried/visible) rather than
	// silently succeeding.
	require.Error(t, mux.ProcessTask(context.Background(), jobs.NewDataRetentionTask()))
}
