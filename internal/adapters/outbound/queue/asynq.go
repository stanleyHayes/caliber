// Package queue adapts the application task-dispatch port to Asynq.
package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	appqueue "github.com/xcreativs/caliber/internal/app/queue"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/platform/telemetry/queuemetrics"
)

// Priorities returns the weighted queue priorities used by all workers.
func Priorities() map[string]int {
	return map[string]int{
		appqueue.QueueCritical: 6,
		appqueue.QueueDefault:  3,
		appqueue.QueueLow:      1,
	}
}

// RedisOpt parses a Redis URL into Asynq's connection option.
//
//nolint:ireturn // Asynq exposes Redis connection choices through this provider interface.
func RedisOpt(redisURL string) (asynq.RedisConnOpt, error) {
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return nil, fmt.Errorf("queue: parse redis url: %w", err)
	}
	return opt, nil
}

// Dispatcher enqueues tasks through Asynq.
type Dispatcher struct {
	client *asynq.Client
}

// NewDispatcher builds an Asynq-backed dispatcher from a Redis URL.
func NewDispatcher(redisURL string) (*Dispatcher, error) {
	opt, err := RedisOpt(redisURL)
	if err != nil {
		return nil, err
	}
	return NewDispatcherFromClient(asynq.NewClient(opt)), nil
}

// NewDispatcherFromClient builds a dispatcher from an existing Asynq client.
func NewDispatcherFromClient(client *asynq.Client) *Dispatcher {
	return &Dispatcher{client: client}
}

// DispatchCandidateAgentRun enqueues a candidate-agent pass.
func (d *Dispatcher) DispatchCandidateAgentRun(
	ctx context.Context, candidateID kernel.ID, opts ...appqueue.DispatchOption,
) (string, error) {
	return d.dispatch(ctx, appqueue.TypeCandidateAgentRun, appqueue.CandidateAgentRunPayload{
		CandidateID: candidateID.String(),
	}, opts...)
}

// DispatchInterviewScoring enqueues final report-card scoring for an interview.
func (d *Dispatcher) DispatchInterviewScoring(
	ctx context.Context, interviewID kernel.ID, opts ...appqueue.DispatchOption,
) (string, error) {
	return d.dispatch(ctx, appqueue.TypeInterviewScoring, appqueue.InterviewScoringPayload{
		InterviewID: interviewID.String(),
	}, opts...)
}

// DispatchBatchRematch enqueues a role re-match pass.
func (d *Dispatcher) DispatchBatchRematch(
	ctx context.Context,
	roleID kernel.ID,
	opts ...appqueue.DispatchOption,
) (string, error) {
	return d.dispatch(ctx, appqueue.TypeBatchRematch, appqueue.BatchRematchPayload{
		RoleID: roleID.String(),
	}, opts...)
}

// Close releases the underlying Asynq client.
func (d *Dispatcher) Close() error {
	if d == nil || d.client == nil {
		return nil
	}
	return d.client.Close()
}

func (d *Dispatcher) dispatch(
	ctx context.Context,
	taskType appqueue.TaskType,
	payload any,
	opts ...appqueue.DispatchOption,
) (string, error) {
	if d == nil || d.client == nil {
		return "", errors.New("queue: nil dispatcher")
	}
	tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer("github.com/xcreativs/caliber/internal/adapters/outbound/queue")
	ctx, span := tracer.Start(ctx, "queue.enqueue")
	defer span.End()
	span.SetAttributes(attribute.String("messaging.system", "asynq"))

	body, err := json.Marshal(payload)
	if err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("queue: marshal %s: %w", taskType, err)
	}
	resolved := appqueue.ApplyOpts(opts...)
	queueName := queueFor(taskType, resolved)
	span.SetAttributes(attribute.String("messaging.destination.name", queueName))

	// Propagate the current trace context so workers can continue the span.
	headers := make(map[string]string)
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(headers))
	task := asynq.NewTaskWithHeaders(string(taskType), body, headers)

	info, err := d.client.EnqueueContext(ctx, task, taskOptions(taskType, resolved)...)
	if err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("queue: enqueue %s: %w", taskType, err)
	}
	queuemetrics.RecordEnqueue(ctx, string(taskType))
	span.SetAttributes(attribute.String("messaging.message.id", info.ID))
	return info.ID, nil
}

func queueFor(taskType appqueue.TaskType, opts *appqueue.Opts) string {
	queue := appqueue.QueueDefault
	if opts != nil && opts.Queue != "" {
		queue = opts.Queue
	}
	_ = taskType
	return queue
}

// RetryDelayFunc returns an Asynq RetryDelayFunc that applies Caliber's
// per-task-type exponential backoff with jitter.
func RetryDelayFunc() asynq.RetryDelayFunc {
	return func(n int, _ error, t *asynq.Task) time.Duration {
		policy := appqueue.DefaultRetryPolicy(appqueue.TaskType(t.Type()))
		return appqueue.ComputeBackoff(policy, n)
	}
}

func taskOptions(taskType appqueue.TaskType, opts *appqueue.Opts) []asynq.Option {
	queue := appqueue.QueueDefault
	if opts != nil && opts.Queue != "" {
		queue = opts.Queue
	}

	policy := appqueue.DefaultRetryPolicy(taskType)
	maxRetry := policy.MaxRetry
	if opts != nil && opts.MaxRetry >= 0 {
		maxRetry = opts.MaxRetry
	}

	asynqOpts := []asynq.Option{
		asynq.Queue(queue),
		asynq.MaxRetry(maxRetry),
	}
	if opts == nil {
		return asynqOpts
	}
	if opts.MaxRetry >= 0 {
		asynqOpts = append(asynqOpts, asynq.MaxRetry(opts.MaxRetry))
	}
	if !opts.ProcessAt.IsZero() {
		asynqOpts = append(asynqOpts, asynq.ProcessAt(opts.ProcessAt))
	} else if opts.ProcessIn > 0 {
		asynqOpts = append(asynqOpts, asynq.ProcessIn(opts.ProcessIn))
	}
	if opts.UniqueTTL > 0 {
		asynqOpts = append(asynqOpts, asynq.Unique(opts.UniqueTTL))
	}
	return asynqOpts
}
