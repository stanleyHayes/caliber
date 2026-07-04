package llm_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/app"
)

type fakeEmbedder struct {
	vec  []float32
	err  error
	last string
}

func (f *fakeEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	f.last = text
	return f.vec, f.err
}

type capturingRecorder struct{ records []app.AICallRecord }

func (c *capturingRecorder) Record(rec app.AICallRecord) { c.records = append(c.records, rec) }

func TestCostRecordingEmbedder_RecordsSuccess(t *testing.T) {
	inner := &fakeEmbedder{vec: []float32{0.1, 0.2}}
	rec := &capturingRecorder{}
	tracker := app.NewCostTracker(0, nil) // observe-only
	// Fan out to both the capturing recorder and the cost tracker.
	multi := llm.NewMultiRecorder(rec, tracker)
	emb := llm.NewCostRecordingEmbedder(inner, multi, "text-embedding-3-small", func() time.Time { return time.Unix(0, 0) })

	got, err := emb.Embed(context.Background(), "some candidate profile text")
	require.NoError(t, err)
	assert.Equal(t, inner.vec, got)
	assert.Equal(t, "some candidate profile text", inner.last, "delegates the text unchanged")

	require.Len(t, rec.records, 1)
	r := rec.records[0]
	assert.Equal(t, "embed", r.Operation)
	assert.Equal(t, "text-embedding-3-small", r.Model)
	assert.Equal(t, len("some candidate profile text"), r.PromptChars)
	assert.Zero(t, r.ResponseChars, "embeddings have no output text")
	assert.False(t, r.Failed)
	assert.Positive(t, tracker.SpentUSD(), "embedding spend now counts toward the budget")
}

func TestCostRecordingEmbedder_RecordsFailureAndPropagates(t *testing.T) {
	inner := &fakeEmbedder{err: errors.New("provider down")}
	rec := &capturingRecorder{}
	emb := llm.NewCostRecordingEmbedder(inner, rec, "text-embedding-3-small", nil)

	_, err := emb.Embed(context.Background(), "text")
	require.Error(t, err, "the underlying error propagates")
	require.Len(t, rec.records, 1)
	assert.True(t, rec.records[0].Failed, "a failed embed is still recorded")
}

func TestCostRecordingEmbedder_NilRecorderIsSafe(t *testing.T) {
	inner := &fakeEmbedder{vec: []float32{1}}
	emb := llm.NewCostRecordingEmbedder(inner, nil, "m", nil)
	got, err := emb.Embed(context.Background(), "x")
	require.NoError(t, err)
	assert.Equal(t, inner.vec, got)
}
