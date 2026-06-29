// Package queue defines the application port for dispatching background tasks.
// Domain code never imports this package; use-cases and inbound/outbound adapters may.
package queue

import (
	"context"
	"time"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// Queue names used by workers.
const (
	// QueueCritical is for urgent, user-visible background work.
	QueueCritical = "critical"

	// QueueDefault is for normal application work.
	QueueDefault = "default"

	// QueueLow is for best-effort or maintenance work.
	QueueLow = "low"
)

// TaskType identifies the kind of background work to enqueue.
type TaskType string

const (
	// TypeCandidateAgentRun enqueues a candidate-agent scan-and-apply pass.
	TypeCandidateAgentRun TaskType = "candidate_agent:run"

	// TypeInterviewScoring enqueues final report-card generation.
	TypeInterviewScoring TaskType = "interview:score"

	// TypeBatchRematch enqueues a role re-match/re-rank pass.
	TypeBatchRematch TaskType = "matching:rematch"
)

// CandidateAgentRunPayload travels as JSON inside TypeCandidateAgentRun tasks.
type CandidateAgentRunPayload struct {
	CandidateID string `json:"candidate_id"`
}

// InterviewScoringPayload travels as JSON inside TypeInterviewScoring tasks.
type InterviewScoringPayload struct {
	InterviewID string `json:"interview_id"`
}

// BatchRematchPayload travels as JSON inside TypeBatchRematch tasks.
type BatchRematchPayload struct {
	RoleID string `json:"role_id"`
}

// DispatchOption customizes a single task enqueue.
type DispatchOption func(*Opts)

// Opts is the resolved set of dispatch options. It is returned by ApplyOpts so
// implementations can read the configured values without leaking provider types.
type Opts struct {
	ProcessIn time.Duration
	UniqueTTL time.Duration
	MaxRetry  int
	Queue     string
}

// ProcessIn schedules the task to run after the given delay.
func ProcessIn(d time.Duration) DispatchOption {
	return func(o *Opts) { o.ProcessIn = d }
}

// Unique prevents duplicate task enqueue within the given TTL (per task type + payload).
func Unique(ttl time.Duration) DispatchOption {
	return func(o *Opts) { o.UniqueTTL = ttl }
}

// MaxRetry overrides the default retry count for this task.
func MaxRetry(n int) DispatchOption {
	return func(o *Opts) { o.MaxRetry = n }
}

// Queue routes the task to a named queue (e.g. "critical", "default").
func Queue(name string) DispatchOption {
	return func(o *Opts) { o.Queue = name }
}

// NewDefaultOpts returns the default dispatch options applied when none are supplied.
func NewDefaultOpts() *Opts {
	return &Opts{MaxRetry: -1, Queue: QueueDefault}
}

// ApplyOpts applies the supplied options onto the default option set.
func ApplyOpts(opts ...DispatchOption) *Opts {
	o := NewDefaultOpts()
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// TaskDispatcher is the outbound port used by the API to enqueue background work.
// Implementations are provider-specific (Asynq, in-memory, etc.).
type TaskDispatcher interface {
	// DispatchCandidateAgentRun enqueues a candidate-agent scan-and-apply run.
	DispatchCandidateAgentRun(ctx context.Context, candidateID kernel.ID, opts ...DispatchOption) (taskID string, err error)

	// DispatchInterviewScoring enqueues final report-card generation for an interview.
	DispatchInterviewScoring(ctx context.Context, interviewID kernel.ID, opts ...DispatchOption) (taskID string, err error)

	// DispatchBatchRematch enqueues a re-match/re-rank pass for a role.
	DispatchBatchRematch(ctx context.Context, roleID kernel.ID, opts ...DispatchOption) (taskID string, err error)

	// Close releases any resources held by the dispatcher.
	Close() error
}
