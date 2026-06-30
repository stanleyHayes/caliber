// Package queue defines the application port for dispatching background tasks.
// Domain code never imports this package; use-cases and inbound/outbound adapters may.
package queue

import (
	"context"
	"math/rand"
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

// RetryPolicy configures how retries are scheduled for a task type.
// MaxRetry is the hard ceiling; once exhausted the task is archived (dead-lettered).
// InitialDelay and MaxDelay bound the exponential backoff, and Jitter adds
// randomized spread to avoid thundering-herd retries.
type RetryPolicy struct {
	MaxRetry     int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Jitter       float64 // 0.0-1.0 fraction of the calculated delay to randomize
}

// DefaultRetryPolicy returns the recommended retry policy for a task type.
func DefaultRetryPolicy(taskType TaskType) RetryPolicy {
	switch taskType {
	case TypeCandidateAgentRun, TypeInterviewScoring:
		return RetryPolicy{
			MaxRetry:     3,
			InitialDelay: 5 * time.Second,
			MaxDelay:     5 * time.Minute,
			Jitter:       0.2,
		}
	case TypeBatchRematch:
		return RetryPolicy{
			MaxRetry:     2,
			InitialDelay: 10 * time.Second,
			MaxDelay:     2 * time.Minute,
			Jitter:       0.2,
		}
	default:
		return RetryPolicy{
			MaxRetry:     3,
			InitialDelay: 10 * time.Second,
			MaxDelay:     1 * time.Minute,
			Jitter:       0.2,
		}
	}
}

// ComputeBackoff calculates the delay before the nth retry using exponential
// backoff capped at MaxDelay plus bounded jitter. n is zero-based: n=0 is the
// first retry after the initial failure.
func ComputeBackoff(policy RetryPolicy, n int) time.Duration {
	if n < 0 {
		n = 0
	}
	if policy.InitialDelay <= 0 {
		policy.InitialDelay = time.Second
	}
	if policy.MaxDelay <= 0 {
		policy.MaxDelay = policy.InitialDelay
	}
	if policy.MaxDelay < policy.InitialDelay {
		policy.MaxDelay = policy.InitialDelay
	}

	delay := policy.InitialDelay << n
	if delay <= 0 || delay > policy.MaxDelay {
		delay = policy.MaxDelay
	}

	if policy.Jitter > 0 {
		spread := time.Duration(float64(delay) * policy.Jitter)
		if spread > 0 {
			offset := time.Duration(rand.Int63n(int64(spread) + 1))
			delay = delay - spread/2 + offset
			if delay < 0 {
				delay = 0
			}
		}
	}
	return delay
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
