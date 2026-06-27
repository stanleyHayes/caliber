package llm_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/app"
)

type stubLLM struct {
	resp app.LLMResponse
	err  error
}

func (s stubLLM) Complete(_ context.Context, _ app.LLMRequest) (app.LLMResponse, error) {
	return s.resp, s.err
}

func seqClock(times ...time.Time) func() time.Time {
	i := 0
	return func() time.Time {
		t := times[min(i, len(times)-1)]
		i++
		return t
	}
}

func TestAudited_RecordsRedactedTraceOnSuccess(t *testing.T) {
	rec := llm.NewMemoryRecorder(8)
	start := time.Unix(1700000000, 0)
	clock := seqClock(start, start.Add(7*time.Millisecond))
	a := llm.NewAudited(stubLLM{resp: app.LLMResponse{Text: "0123456789"}}, rec, "claude-opus-4-8", clock)

	_, err := a.Complete(context.Background(), app.LLMRequest{
		Source: app.PromptRef{ID: "interview_question", Version: "v1"},
		Prompt: "hello",
	})
	require.NoError(t, err)

	snap := rec.Snapshot()
	require.Len(t, snap, 1)
	assert.Equal(t, "interview_question", snap[0].Operation)
	assert.Equal(t, "interview_question", snap[0].PromptID)
	assert.Equal(t, "v1", snap[0].PromptVersion)
	assert.Equal(t, "claude-opus-4-8", snap[0].Model)
	assert.Equal(t, 5, snap[0].PromptChars)
	assert.Equal(t, 10, snap[0].ResponseChars)
	assert.Equal(t, 7*time.Millisecond, snap[0].Latency)
	assert.False(t, snap[0].Failed)
	assert.Equal(t, start, snap[0].At)
}

func TestAudited_RecordsFailureAndPropagates(t *testing.T) {
	rec := llm.NewMemoryRecorder(8)
	boom := errors.New("provider down")
	a := llm.NewAudited(stubLLM{err: boom}, rec, "dev", nil)

	_, err := a.Complete(context.Background(), app.LLMRequest{Source: app.PromptRef{ID: "interview_report", Version: "v1"}, Prompt: "x"})
	require.ErrorIs(t, err, boom, "the inner error is propagated unchanged")

	snap := rec.Snapshot()
	require.Len(t, snap, 1)
	assert.True(t, snap[0].Failed)
	assert.Equal(t, "interview_report", snap[0].Operation)
	assert.Zero(t, snap[0].ResponseChars)
}

func TestAudited_NilRecorderIsSafe(t *testing.T) {
	a := llm.NewAudited(stubLLM{resp: app.LLMResponse{Text: "ok"}}, nil, "dev", nil)
	resp, err := a.Complete(context.Background(), app.LLMRequest{System: "x", Prompt: "y"})
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Text)
}

func TestAudited_UnknownSourceIsRecordedAsUnknown(t *testing.T) {
	// A request with no Source (e.g. an ad-hoc call not built through the
	// registry) records Operation "unknown" and empty prompt id/version, rather
	// than guessing from the system text.
	rec := llm.NewMemoryRecorder(1)
	a := llm.NewAudited(stubLLM{}, rec, "dev", nil)
	_, _ = a.Complete(context.Background(), app.LLMRequest{System: "anything"})
	snap := rec.Snapshot()
	require.Len(t, snap, 1)
	assert.Equal(t, "unknown", snap[0].Operation)
	assert.Empty(t, snap[0].PromptID)
	assert.Empty(t, snap[0].PromptVersion)
}

func TestMemoryRecorder_RingBufferEvictsOldest(t *testing.T) {
	rec := llm.NewMemoryRecorder(2)
	rec.Record(app.AICallRecord{Operation: "a"})
	rec.Record(app.AICallRecord{Operation: "b"})
	rec.Record(app.AICallRecord{Operation: "c"})

	snap := rec.Snapshot()
	require.Len(t, snap, 2)
	assert.Equal(t, "b", snap[0].Operation, "oldest ('a') evicted")
	assert.Equal(t, "c", snap[1].Operation)
}

func TestMemoryRecorder_DefaultsCapacity(t *testing.T) {
	rec := llm.NewMemoryRecorder(0)
	for range 300 {
		rec.Record(app.AICallRecord{Operation: "x"})
	}
	assert.Len(t, rec.Snapshot(), 256, "non-positive capacity defaults to 256")
}

func TestSlogRecorder_DoesNotPanic(t *testing.T) {
	r := llm.NewSlogRecorder(slog.New(slog.DiscardHandler))
	assert.NotPanics(t, func() {
		r.Record(app.AICallRecord{Operation: "cv_extract", Model: "dev", PromptChars: 10})
	})
}

func TestMemoryRecorderStats(t *testing.T) {
	r := llm.NewMemoryRecorder(10)
	r.Record(app.AICallRecord{Operation: "score", Latency: 100 * time.Millisecond, Failed: false})
	r.Record(app.AICallRecord{Operation: "score", Latency: 300 * time.Millisecond, Failed: true})

	stats := r.Stats()
	assert.Equal(t, 2, stats.TotalCalls)
	assert.Equal(t, 1, stats.FailedCalls)
	assert.InDelta(t, 0.5, stats.FailureRate, 1e-9)
	assert.Equal(t, 2, stats.ByOperation["score"].Calls)
}
