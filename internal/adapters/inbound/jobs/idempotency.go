package jobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// HeaderIdempotencyKey lets an enqueueing adapter provide a business-level
// idempotency key. When absent, live Asynq deliveries use the stable Asynq task
// id, and direct tests fall back to a deterministic task-type + payload digest.
const HeaderIdempotencyKey = "Idempotency-Key"

const jobsTracerName = "github.com/xcreativs/caliber/internal/adapters/inbound/jobs"

// IdempotencyStore claims and completes task-processing keys. Implementations
// must be safe for concurrent worker goroutines.
type IdempotencyStore interface {
	Claim(ctx context.Context, key string) (claimed bool, err error)
	Complete(ctx context.Context, key string) error
	Release(ctx context.Context, key string) error
}

// MemoryIdempotencyStore is a process-local idempotency store. It is deliberately
// small and test-friendly; production can inject a durable Redis/Postgres store
// without changing handler code.
type MemoryIdempotencyStore struct {
	mu     sync.Mutex
	states map[string]idempotencyState
}

type idempotencyState uint8

const (
	idempotencyInProgress idempotencyState = iota + 1
	idempotencyCompleted
)

// NewMemoryIdempotencyStore builds an in-memory idempotency store.
func NewMemoryIdempotencyStore() *MemoryIdempotencyStore {
	return &MemoryIdempotencyStore{states: make(map[string]idempotencyState)}
}

// Claim records that a task key is being processed. It returns false when the
// key is already in progress or already completed.
func (m *MemoryIdempotencyStore) Claim(_ context.Context, key string) (bool, error) {
	if strings.TrimSpace(key) == "" {
		return false, errors.New("jobs: empty idempotency key")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.states[key]; exists {
		return false, nil
	}
	m.states[key] = idempotencyInProgress
	return true, nil
}

// Complete marks a claimed key as successfully processed.
func (m *MemoryIdempotencyStore) Complete(_ context.Context, key string) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("jobs: empty idempotency key")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[key] = idempotencyCompleted
	return nil
}

// Release drops an in-progress key so a failed task can be retried. Completed
// keys are retained and continue suppressing duplicate delivery.
func (m *MemoryIdempotencyStore) Release(_ context.Context, key string) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("jobs: empty idempotency key")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.states[key] == idempotencyCompleted {
		return nil
	}
	delete(m.states, key)
	return nil
}

type jobFunc func(context.Context, *asynq.Task) error

type jobFramework struct {
	log   *slog.Logger
	store IdempotencyStore
}

type jobMeta struct {
	name   string
	typ    string
	key    string
	taskID string
	queue  string
	retry  int
}

func newJobFramework(log *slog.Logger, store IdempotencyStore) jobFramework {
	if store == nil {
		store = NewMemoryIdempotencyStore()
	}
	return jobFramework{log: log, store: store}
}

func (j jobFramework) wrap(name string, fn jobFunc) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, task *asynq.Task) error {
		if task == nil {
			return errors.New("jobs: nil task")
		}

		tracer := otel.Tracer(jobsTracerName)
		ctx, span := tracer.Start(ctx, "jobs."+task.Type())
		defer span.End()

		meta := newJobMeta(ctx, name, task)
		setJobSpanAttributes(span, meta)

		claimed, err := j.store.Claim(ctx, meta.key)
		if err != nil {
			recordJobError(span, err)
			return err
		}
		if !claimed {
			span.SetAttributes(attribute.Bool("job.duplicate", true))
			j.logJob("job skipped duplicate", meta)
			return nil
		}
		return j.runClaimed(ctx, span, meta, fn, task)
	}
}

func (j jobFramework) runClaimed(
	ctx context.Context,
	span trace.Span,
	meta jobMeta,
	fn jobFunc,
	task *asynq.Task,
) error {
	start := time.Now()
	completed := false
	defer func() {
		if !completed {
			_ = j.store.Release(context.WithoutCancel(ctx), meta.key)
		}
	}()

	j.logJob("job started", meta)
	if err := fn(ctx, task); err != nil {
		recordJobError(span, err)
		completed = true
		_ = j.store.Release(context.WithoutCancel(ctx), meta.key)
		j.errorJob("job failed", err, meta, time.Since(start))
		return err
	}
	if err := j.store.Complete(ctx, meta.key); err != nil {
		recordJobError(span, err)
		return fmt.Errorf("jobs: complete idempotency key: %w", err)
	}
	completed = true
	j.logJob("job completed", meta, "duration_ms", time.Since(start).Milliseconds())
	return nil
}

func newJobMeta(ctx context.Context, name string, task *asynq.Task) jobMeta {
	taskID, _ := asynq.GetTaskID(ctx)
	queueName, _ := asynq.GetQueueName(ctx)
	retryCount, _ := asynq.GetRetryCount(ctx)
	return jobMeta{
		name:   name,
		typ:    task.Type(),
		key:    idempotencyKey(ctx, task),
		taskID: taskID,
		queue:  queueName,
		retry:  retryCount,
	}
}

func setJobSpanAttributes(span trace.Span, meta jobMeta) {
	span.SetAttributes(
		attribute.String("job.name", meta.name),
		attribute.String("job.type", meta.typ),
		attribute.String("job.idempotency_key", meta.key),
		attribute.String("messaging.system", "asynq"),
		attribute.String("messaging.destination.name", meta.queue),
		attribute.String("messaging.message.id", meta.taskID),
		attribute.Int("messaging.message.retry_count", meta.retry),
	)
}

func (j jobFramework) logJob(msg string, meta jobMeta, args ...any) {
	if j.log != nil {
		j.log.Info(msg, jobLogArgs(meta, args...)...)
	}
}

func (j jobFramework) errorJob(msg string, err error, meta jobMeta, elapsed time.Duration) {
	if j.log != nil {
		args := jobLogArgs(meta, "duration_ms", elapsed.Milliseconds(), "err", err)
		j.log.Error(msg, args...)
	}
}

func jobLogArgs(meta jobMeta, extra ...any) []any {
	args := make([]any, 0, 12+len(extra))
	args = append(args,
		"job", meta.name,
		"task_type", meta.typ,
		"idempotency_key", meta.key,
		"task_id", meta.taskID,
		"queue", meta.queue,
		"retry_count", meta.retry,
	)
	return append(args, extra...)
}

func idempotencyKey(ctx context.Context, task *asynq.Task) string {
	if key := strings.TrimSpace(task.Headers()[HeaderIdempotencyKey]); key != "" {
		return "header:" + key
	}
	if taskID, ok := asynq.GetTaskID(ctx); ok && strings.TrimSpace(taskID) != "" {
		return "asynq:" + taskID
	}
	sum := sha256.New()
	_, _ = sum.Write([]byte(task.Type()))
	_, _ = sum.Write([]byte{0})
	_, _ = sum.Write(task.Payload())
	return "payload:" + hex.EncodeToString(sum.Sum(nil))
}

func recordJobError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
