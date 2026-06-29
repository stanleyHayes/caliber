// Package jobs registers Asynq task handlers and runs worker servers.
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	"github.com/xcreativs/caliber/internal/adapters/outbound/queue"
	candidateagentapp "github.com/xcreativs/caliber/internal/app/candidateagent"
	interviewapp "github.com/xcreativs/caliber/internal/app/interview"
	appqueue "github.com/xcreativs/caliber/internal/app/queue"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// TypeHealthcheck is a harmless smoke-test task proving enqueue-to-process.
const TypeHealthcheck = "caliber.healthcheck"

// HealthcheckPayload is a harmless smoke-test task proving enqueue-to-process.
type HealthcheckPayload struct {
	Probe string `json:"probe"`
}

// EncodeHealthcheckPayload serializes a healthcheck payload for dispatch.
func EncodeHealthcheckPayload(payload HealthcheckPayload) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("jobs: marshal healthcheck payload: %w", err)
	}
	return b, nil
}

// HealthcheckCallback is invoked when the healthcheck task is processed.
type HealthcheckCallback func(context.Context, HealthcheckPayload) error

type options struct {
	healthcheck HealthcheckCallback
	idempotency IdempotencyStore
}

// Option customizes worker handlers.
type Option func(*options)

// WithHealthcheckCallback observes healthcheck task processing. Tests use this
// to prove a real enqueue-to-process round trip; production leaves it nil.
func WithHealthcheckCallback(fn HealthcheckCallback) Option {
	return func(opts *options) {
		opts.healthcheck = fn
	}
}

// WithIdempotencyStore injects the store used by the handler framework. Tests
// can share one store across muxes; production can provide a durable store later.
func WithIdempotencyStore(store IdempotencyStore) Option {
	return func(opts *options) {
		opts.idempotency = store
	}
}

// HandlerDeps bundles the use-cases the business task handlers need.
type HandlerDeps struct {
	AgentRunner *candidateagentapp.AgentRunner
	Interviewer *interviewapp.Interviewer
	Idempotency IdempotencyStore
}

// NewMux builds the task-handler mux.
func NewMux(log *slog.Logger, opts ...Option) *asynq.ServeMux {
	cfg := options{}
	for _, opt := range opts {
		opt(&cfg)
	}
	mux := asynq.NewServeMux()
	framework := newJobFramework(log, cfg.idempotency)
	mux.HandleFunc(TypeHealthcheck, framework.wrap("healthcheck", handleHealthcheck(log, cfg.healthcheck)))
	return mux
}

// RegisterHandlers adds business handlers to an existing Asynq mux.
func RegisterHandlers(mux *asynq.ServeMux, deps HandlerDeps, log *slog.Logger) {
	framework := newJobFramework(log, deps.Idempotency)
	mux.HandleFunc(
		string(appqueue.TypeCandidateAgentRun),
		framework.wrap("candidate_agent_run", handleCandidateAgentRun(deps.AgentRunner, log)),
	)
	mux.HandleFunc(
		string(appqueue.TypeInterviewScoring),
		framework.wrap("interview_scoring", handleInterviewScoring(deps.Interviewer, log)),
	)
	mux.HandleFunc(string(appqueue.TypeBatchRematch), framework.wrap("batch_rematch", handleBatchRematch(log)))
}

func handleHealthcheck(log *slog.Logger, callback HealthcheckCallback) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, task *asynq.Task) error {
		var payload HealthcheckPayload
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return fmt.Errorf("jobs: decode healthcheck payload: %w", err)
		}
		if log != nil {
			log.Info("processed healthcheck task", "probe", payload.Probe)
		}
		if callback != nil {
			return callback(ctx, payload)
		}
		return nil
	}
}

func handleCandidateAgentRun(runner *candidateagentapp.AgentRunner, log *slog.Logger) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, task *asynq.Task) error {
		var payload appqueue.CandidateAgentRunPayload
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return fmt.Errorf("jobs: decode candidate agent payload: %w", err)
		}
		candidateID, err := kernel.IDFromString(payload.CandidateID)
		if err != nil {
			return fmt.Errorf("jobs: invalid candidate_id: %w", err)
		}
		if log != nil {
			log.Info("candidate agent run started", "candidate_id", candidateID)
		}
		if runner == nil {
			return errors.New("jobs: candidate agent runner not wired")
		}
		if _, err := runner.Run(ctx, candidateID, 0); err != nil {
			return fmt.Errorf("jobs: candidate agent run: %w", err)
		}
		if log != nil {
			log.Info("candidate agent run completed", "candidate_id", candidateID)
		}
		return nil
	}
}

func handleInterviewScoring(interviewer *interviewapp.Interviewer, log *slog.Logger) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, task *asynq.Task) error {
		var payload appqueue.InterviewScoringPayload
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return fmt.Errorf("jobs: decode interview scoring payload: %w", err)
		}
		interviewID, err := kernel.IDFromString(payload.InterviewID)
		if err != nil {
			return fmt.Errorf("jobs: invalid interview_id: %w", err)
		}
		if interviewer == nil {
			return errors.New("jobs: interviewer not wired")
		}
		if log != nil {
			log.Info("interview scoring started", "interview_id", interviewID)
		}
		if _, err := interviewer.ScoreAsync(ctx, interviewID); err != nil {
			return fmt.Errorf("jobs: interview scoring: %w", err)
		}
		if log != nil {
			log.Info("interview scoring completed", "interview_id", interviewID)
		}
		return nil
	}
}

func handleBatchRematch(log *slog.Logger) func(context.Context, *asynq.Task) error {
	return func(_ context.Context, task *asynq.Task) error {
		var payload appqueue.BatchRematchPayload
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return fmt.Errorf("jobs: decode batch rematch payload: %w", err)
		}
		roleID, err := kernel.IDFromString(payload.RoleID)
		if err != nil {
			return fmt.Errorf("jobs: invalid role_id: %w", err)
		}
		if log != nil {
			log.Info("batch rematch not implemented", "role_id", roleID)
		}
		return nil
	}
}

// Worker wraps an Asynq server with Caliber's lifecycle conventions.
type Worker struct {
	server  *asynq.Server
	handler asynq.Handler
}

// NewWorker builds a worker over Redis.
func NewWorker(redisURL string, log *slog.Logger, opts ...Option) (*Worker, error) {
	redisOpt, err := queue.RedisOpt(redisURL)
	if err != nil {
		return nil, err
	}
	return NewWorkerFromRedis(redisOpt, log, opts...), nil
}

// NewWorkerFromRedis builds a worker from an Asynq Redis connection option.
func NewWorkerFromRedis(redisOpt asynq.RedisConnOpt, log *slog.Logger, opts ...Option) *Worker {
	return &Worker{
		server: asynq.NewServer(redisOpt, asynq.Config{
			Concurrency:     4,
			Queues:          queue.Priorities(),
			ShutdownTimeout: 10 * time.Second,
			Logger:          slogAdapter{log: log},
		}),
		handler: NewMux(log, opts...),
	}
}

// Run starts the worker, blocks until ctx is cancelled, then drains work.
func (w *Worker) Run(ctx context.Context) error {
	if w == nil || w.server == nil || w.handler == nil {
		return errors.New("jobs: nil worker")
	}
	if err := w.server.Start(w.handler); err != nil {
		return fmt.Errorf("jobs: start worker: %w", err)
	}
	<-ctx.Done()
	w.server.Stop()
	w.server.Shutdown()
	return nil
}

type slogAdapter struct {
	log *slog.Logger
}

func (s slogAdapter) Debug(args ...any) { s.write(slog.LevelDebug, args...) }
func (s slogAdapter) Info(args ...any)  { s.write(slog.LevelInfo, args...) }
func (s slogAdapter) Warn(args ...any)  { s.write(slog.LevelWarn, args...) }
func (s slogAdapter) Error(args ...any) { s.write(slog.LevelError, args...) }
func (s slogAdapter) Fatal(args ...any) { s.write(slog.LevelError, args...) }

func (s slogAdapter) write(level slog.Level, args ...any) {
	if s.log == nil {
		return
	}
	s.log.Log(context.Background(), level, fmt.Sprint(args...))
}
