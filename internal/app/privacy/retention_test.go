package privacy_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	privacyapp "github.com/xcreativs/caliber/internal/app/privacy"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

type fakeLister struct {
	ids    []kernel.ID
	err    error
	cutoff time.Time // captured from the call
	called bool
}

func (f *fakeLister) ListEligibleForRetention(_ context.Context, createdOnOrBefore time.Time) ([]kernel.ID, error) {
	f.called = true
	f.cutoff = createdOnOrBefore
	return f.ids, f.err
}

type fakeEraser struct {
	erased []kernel.ID
	failOn map[kernel.ID]bool
}

func (f *fakeEraser) EraseCandidate(_ context.Context, id kernel.ID) error {
	if f.failOn[id] {
		return errors.New("boom")
	}
	f.erased = append(f.erased, id)
	return nil
}

func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }

func TestRetentionSweep_ErasesEligibleAndComputesCutoff(t *testing.T) {
	now := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	a, b := kernel.NewID(), kernel.NewID()
	lister := &fakeLister{ids: []kernel.ID{a, b}}
	eraser := &fakeEraser{}
	sweeper := privacyapp.NewRetentionSweeper(lister, eraser, 30*24*time.Hour, fixedClock(now))

	res, err := sweeper.Sweep(context.Background())
	require.NoError(t, err)
	assert.Equal(t, privacyapp.RetentionResult{Eligible: 2, Erased: 2, Failed: 0}, res)
	assert.ElementsMatch(t, []kernel.ID{a, b}, eraser.erased)
	// The lister is asked for candidates created on or before now-window.
	assert.Equal(t, now.Add(-30*24*time.Hour), lister.cutoff)
}

func TestRetentionSweep_DisabledWindowIsNoOp(t *testing.T) {
	lister := &fakeLister{ids: []kernel.ID{kernel.NewID()}}
	eraser := &fakeEraser{}
	sweeper := privacyapp.NewRetentionSweeper(lister, eraser, 0, fixedClock(time.Now()))

	res, err := sweeper.Sweep(context.Background())
	require.NoError(t, err)
	assert.Zero(t, res.Eligible)
	assert.False(t, lister.called, "a disabled window never queries the lister")
	assert.Empty(t, eraser.erased)
}

func TestRetentionSweep_ContinuesPastFailuresButReportsError(t *testing.T) {
	good, bad := kernel.NewID(), kernel.NewID()
	lister := &fakeLister{ids: []kernel.ID{good, bad}}
	eraser := &fakeEraser{failOn: map[kernel.ID]bool{bad: true}}
	sweeper := privacyapp.NewRetentionSweeper(lister, eraser, time.Hour, fixedClock(time.Now()))

	res, err := sweeper.Sweep(context.Background())
	require.Error(t, err, "a failed erasure surfaces so the job retries")
	assert.Equal(t, 2, res.Eligible)
	assert.Equal(t, 1, res.Erased)
	assert.Equal(t, 1, res.Failed)
	assert.Equal(t, []kernel.ID{good}, eraser.erased, "the healthy candidate is still erased")
}

func TestRetentionSweep_ListerErrorPropagates(t *testing.T) {
	lister := &fakeLister{err: errors.New("db down")}
	eraser := &fakeEraser{}
	sweeper := privacyapp.NewRetentionSweeper(lister, eraser, time.Hour, fixedClock(time.Now()))

	_, err := sweeper.Sweep(context.Background())
	require.Error(t, err)
	assert.Empty(t, eraser.erased)
}
