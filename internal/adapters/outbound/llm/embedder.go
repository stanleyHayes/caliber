package llm

import (
	"context"
	"time"

	"github.com/xcreativs/caliber/internal/app"
)

// CostRecordingEmbedder wraps an app.Embedder and records an AICallRecord for
// every embedding call, so embedding spend counts toward the same cost/FinOps
// budget and alerting as LLM calls (CAL-159). Previously embeddings flowed
// through a port that carried no recorder, so their spend was invisible. The
// record carries sizes only — never the embedded text — so it stays PII-safe,
// and recording never blocks or fails the embed.
type CostRecordingEmbedder struct {
	inner    app.Embedder
	recorder app.AICallRecorder
	model    string
	now      func() time.Time
}

// NewCostRecordingEmbedder wraps inner so each Embed emits an AICallRecord
// (Operation "embed", the given model id, and the input character count) to the
// recorder. A nil now defaults to time.Now.
func NewCostRecordingEmbedder(
	inner app.Embedder, recorder app.AICallRecorder, model string, now func() time.Time,
) *CostRecordingEmbedder {
	if now == nil {
		now = time.Now
	}
	return &CostRecordingEmbedder{inner: inner, recorder: recorder, model: model, now: now}
}

// Embed delegates to the wrapped embedder and records the call's size + latency.
func (e *CostRecordingEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	start := e.now()
	vec, err := e.inner.Embed(ctx, text)
	if e.recorder != nil {
		e.recorder.Record(app.AICallRecord{
			Operation:   "embed",
			Model:       e.model,
			Latency:     e.now().Sub(start),
			PromptChars: len(text),
			Failed:      err != nil,
			At:          start,
		})
	}
	return vec, err
}

var _ app.Embedder = (*CostRecordingEmbedder)(nil)
