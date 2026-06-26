package llm

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/xcreativs/caliber/internal/app"
)

// Audited wraps an app.LLMClient and records a redacted trace of every call to
// an app.AICallRecorder (CAL-036). It implements app.LLMClient, so it composes
// in the same decorator chain as Guarded. The clock is injectable for tests.
type Audited struct {
	inner    app.LLMClient
	recorder app.AICallRecorder
	model    string
	now      func() time.Time
}

// NewAudited wraps inner so each call is traced to recorder under the given model
// label. now defaults to time.Now when nil.
func NewAudited(inner app.LLMClient, recorder app.AICallRecorder, model string, now func() time.Time) *Audited {
	if now == nil {
		now = time.Now
	}
	return &Audited{inner: inner, recorder: recorder, model: model, now: now}
}

// Complete delegates to the inner client and records a redacted trace (sizes and
// latency only — never content) regardless of success or failure.
func (a *Audited) Complete(ctx context.Context, req app.LLMRequest) (app.LLMResponse, error) {
	start := a.now()
	resp, err := a.inner.Complete(ctx, req)
	if a.recorder != nil {
		a.recorder.Record(app.AICallRecord{
			Operation:     operationOf(req.System),
			Model:         a.model,
			Latency:       a.now().Sub(start),
			PromptChars:   len(req.Prompt),
			ResponseChars: len(resp.Text),
			Failed:        err != nil,
			At:            start,
		})
	}
	return resp, err
}

// operationOf classifies a call by a stable label derived from its system
// prompt, so traces are groupable without storing prompt content.
func operationOf(system string) string {
	switch {
	case strings.Contains(system, "screening interviewer"):
		return "interview_question"
	case strings.Contains(system, "score a screening interview"):
		return "interview_report"
	case strings.Contains(system, "honest job-application agent"):
		return "agent_assess"
	case strings.Contains(system, "structured talent profile from a CV"):
		return "cv_extract"
	case strings.Contains(system, "candidate against a role rubric"):
		return "shortlist_score"
	case strings.Contains(system, "structured role spec"):
		return "role_spec"
	default:
		return "unknown"
	}
}

// SlogRecorder logs each AI-call trace via slog at info level. It records only
// redacted metadata (operation, model, sizes, latency), never prompt content.
type SlogRecorder struct {
	log *slog.Logger
}

// NewSlogRecorder builds a structured-logging recorder.
func NewSlogRecorder(log *slog.Logger) *SlogRecorder { return &SlogRecorder{log: log} }

// Record emits a redacted structured log line for the call.
func (r *SlogRecorder) Record(rec app.AICallRecord) {
	r.log.Info("ai call",
		"operation", rec.Operation,
		"model", rec.Model,
		"latency_ms", rec.Latency.Milliseconds(),
		"prompt_chars", rec.PromptChars,
		"response_chars", rec.ResponseChars,
		"failed", rec.Failed,
	)
}

// MemoryRecorder keeps the most recent AI-call traces in a bounded ring buffer,
// queryable via Snapshot. It is safe for concurrent use and is handy for a debug
// view or tests.
type MemoryRecorder struct {
	mu      sync.Mutex
	cap     int
	records []app.AICallRecord
}

// NewMemoryRecorder builds a recorder retaining the last capacity traces
// (capacity <= 0 defaults to 256).
func NewMemoryRecorder(capacity int) *MemoryRecorder {
	if capacity <= 0 {
		capacity = 256
	}
	return &MemoryRecorder{cap: capacity}
}

// Record appends a trace, evicting the oldest once at capacity.
func (m *MemoryRecorder) Record(rec app.AICallRecord) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.records) == m.cap {
		m.records = m.records[1:]
	}
	m.records = append(m.records, rec)
}

// Snapshot returns a copy of the retained traces, oldest first.
func (m *MemoryRecorder) Snapshot() []app.AICallRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]app.AICallRecord, len(m.records))
	copy(out, m.records)
	return out
}
