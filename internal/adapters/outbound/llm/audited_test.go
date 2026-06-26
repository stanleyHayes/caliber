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
		System: "You are an adaptive technical screening interviewer.",
		Prompt: "hello",
	})
	require.NoError(t, err)

	snap := rec.Snapshot()
	require.Len(t, snap, 1)
	assert.Equal(t, "interview_question", snap[0].Operation)
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

	_, err := a.Complete(context.Background(), app.LLMRequest{System: "score a screening interview", Prompt: "x"})
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

func TestAudited_OperationClassification(t *testing.T) {
	cases := map[string]string{
		"You are an adaptive technical screening interviewer.":         "interview_question",
		"You score a screening interview against the role rubric.":     "interview_report",
		"You are a candidate's honest job-application agent.":          "agent_assess",
		"You extract a structured talent profile from a CV.":           "cv_extract",
		"You score a candidate against a role rubric.":                 "shortlist_score",
		"You convert a hiring need into a structured role spec and...": "role_spec",
		"some unrecognized system prompt":                              "unknown",
	}
	for system, want := range cases {
		rec := llm.NewMemoryRecorder(1)
		a := llm.NewAudited(stubLLM{}, rec, "dev", nil)
		_, _ = a.Complete(context.Background(), app.LLMRequest{System: system})
		assert.Equal(t, want, rec.Snapshot()[0].Operation, "system=%q", system)
	}
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
