// Command worker runs background jobs (candidate-agent runs, interview scoring,
// batch re-matching) via Asynq (EPIC-03).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"github.com/xcreativs/caliber/internal/adapters/inbound/jobs"
	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/app"
	queueadapter "github.com/xcreativs/caliber/internal/adapters/outbound/queue"
	candidateagentapp "github.com/xcreativs/caliber/internal/app/candidateagent"
	interviewapp "github.com/xcreativs/caliber/internal/app/interview"
	privacyapp "github.com/xcreativs/caliber/internal/app/privacy"
	"github.com/xcreativs/caliber/internal/domain/audit"
	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/platform/config"
	"github.com/xcreativs/caliber/internal/platform/logging"
	"github.com/xcreativs/caliber/internal/platform/telemetry"
	"github.com/xcreativs/caliber/internal/platform/wiring"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log, logShutdown, err := logging.NewWithConfig(cfg)
	if err != nil {
		return err
	}
	if missing := cfg.Validate(); len(missing) > 0 {
		if cfg.IsProd() {
			return fmt.Errorf("missing required configuration: %v", missing)
		}
		log.Warn("missing configuration", "missing", missing, "env", cfg.Env)
	}
	if issues := cfg.ProdSafetyIssues(); len(issues) > 0 {
		return fmt.Errorf("unsafe production configuration: %v", issues)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = logShutdown(shutdownCtx)
	}()

	tele, err := telemetry.New(cfg)
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = tele.Shutdown(shutdownCtx)
	}()

	return runWorker(ctx, cfg, log, tele)
}

func runWorker(ctx context.Context, cfg config.Config, log *slog.Logger, tele *telemetry.Provider) error {
	if cfg.RedisURL == "" {
		return errors.New("CALIBER_REDIS_URL is required to run the worker")
	}

	model, _, _ := wiring.BuildLLM(cfg, log, tele)
	auditRepo := memory.NewAuditRepo()
	repos, cleanup, _, err := wiring.OpenRepositories(ctx, cfg, log)
	if err != nil {
		return err
	}
	defer cleanup()

	agentRunner, interviewer := buildWorkerUsecases(cfg, repos, model, auditRepo)

	redisOpt, idempotency, closeRedis, err := newWorkerRedis(cfg)
	if err != nil {
		return err
	}
	defer closeRedis()

	retentionSweeper := buildRetentionSweeper(cfg, repos, auditRepo, log)

	mux := jobs.NewMux(log)
	jobs.RegisterHandlers(mux, jobs.HandlerDeps{
		AgentRunner:      agentRunner,
		Interviewer:      interviewer,
		RetentionSweeper: retentionSweeper,
		Idempotency:      idempotency,
	}, log)

	server := newWorkerServer(cfg, redisOpt, log)

	shutdownMetrics := startMetricsServer(ctx, cfg, log, tele)
	defer shutdownMetrics()

	log.Info("worker started", "redis_url", redactedURL(cfg.RedisURL), "concurrency", cfg.WorkerConcurrency)
	if err := server.Start(mux); err != nil {
		return fmt.Errorf("worker: start server: %w", err)
	}

	stopScheduler, err := startRetentionScheduler(cfg, redisOpt, retentionSweeper != nil, log)
	if err != nil {
		server.Stop()
		server.Shutdown()
		return err
	}
	defer stopScheduler()

	<-ctx.Done()
	log.Info("worker shutting down")
	server.Stop()
	server.Shutdown()
	return nil
}

// buildWorkerUsecases constructs the background use-cases the worker's handlers
// run: the autonomous candidate agent and the interviewer/scorer.
func buildWorkerUsecases(
	cfg config.Config, repos wiring.Repositories, model app.LLMClient, auditRepo *memory.AuditRepo,
) (*candidateagentapp.AgentRunner, *interviewapp.Interviewer) {
	agentRunner := candidateagentapp.NewAgentRunner(
		repos.Candidates, repos.Profiles, repos.Roles, repos.Apps, model,
		candidateagentapp.WithAuditTrail(auditRepo, time.Now),
		candidateagentapp.WithWakeUpInsights(repos.Interviews, repos.Matches),
	)
	interviewer := interviewapp.NewInterviewer(
		repos.Roles, repos.Interviews, model,
		interviewdom.Config{MaxQuestions: cfg.InterviewMaxQuestions, MaxDuration: cfg.InterviewMaxDuration},
		interviewapp.WithPassportUpdater(repos.Profiles),
	)
	return agentRunner, interviewer
}

// newWorkerRedis builds the Asynq connection option and a Redis-backed
// idempotency store from the same Redis URL. The store gives exactly-once
// effects across scaled worker replicas (CAL-157) — a process-local store would
// let two replicas each run a re-delivered task. The returned closer releases
// the idempotency client on shutdown.
//
//nolint:ireturn // returns the asynq/idempotency port interfaces intentionally, matching the wiring convention.
func newWorkerRedis(cfg config.Config) (asynq.RedisConnOpt, jobs.IdempotencyStore, func(), error) {
	redisOpt, err := queueadapter.RedisOpt(cfg.RedisURL)
	if err != nil {
		return nil, nil, nil, err
	}
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, nil, nil, err
	}
	client := redis.NewClient(opt)
	return redisOpt, jobs.NewRedisIdempotencyStore(client), func() { _ = client.Close() }, nil
}

// newWorkerServer builds the Asynq server with Caliber's worker conventions.
func newWorkerServer(cfg config.Config, redisOpt asynq.RedisConnOpt, log *slog.Logger) *asynq.Server {
	return asynq.NewServer(redisOpt, asynq.Config{
		Concurrency:     cfg.WorkerConcurrency,
		Queues:          queueadapter.Priorities(),
		ShutdownTimeout: 10 * time.Second,
		Logger:          &asynqLogger{log: log},
		RetryDelayFunc:  queueadapter.RetryDelayFunc(),
		ErrorHandler:    jobs.NewArchiveAlertHandler(log),
	})
}

// buildRetentionSweeper wires the data-retention sweep (CAL-158) when a retention
// window is configured and the repositories support erasure. It returns nil
// otherwise (retention disabled), so the handler reports it is not wired.
func buildRetentionSweeper(
	cfg config.Config, repos wiring.Repositories, auditRepo audit.AuditRepository, log *slog.Logger,
) *privacyapp.RetentionSweeper {
	if cfg.RetentionWindow <= 0 {
		return nil
	}
	lister, ok := repos.Users.(privacyapp.RetentionLister)
	if !ok {
		log.Warn("data retention disabled: user repository cannot list retention-eligible candidates")
		return nil
	}
	eraser := wiring.BuildEraser(repos, auditRepo, memory.NewContestRepo())
	if eraser == nil {
		log.Warn("data retention disabled: repositories do not provide the erasure cascade")
		return nil
	}
	return privacyapp.NewRetentionSweeper(lister, eraser, cfg.RetentionWindow, time.Now)
}

// startRetentionScheduler runs an Asynq periodic scheduler that enqueues a
// data-retention task on cfg.RetentionCron (CAL-158). It is a no-op (returns a
// no-op stop func) when retention is not enabled.
func startRetentionScheduler(
	cfg config.Config, redisOpt asynq.RedisConnOpt, enabled bool, log *slog.Logger,
) (func(), error) {
	if !enabled {
		return func() {}, nil
	}
	scheduler := asynq.NewScheduler(redisOpt, nil)
	if _, err := scheduler.Register(cfg.RetentionCron, jobs.NewDataRetentionTask()); err != nil {
		return func() {}, fmt.Errorf("worker: register retention schedule %q: %w", cfg.RetentionCron, err)
	}
	if err := scheduler.Start(); err != nil {
		return func() {}, fmt.Errorf("worker: start retention scheduler: %w", err)
	}
	log.Info("data retention scheduler started", "cron", cfg.RetentionCron, "window", cfg.RetentionWindow)
	return scheduler.Shutdown, nil
}

// startMetricsServer starts a Prometheus exposition HTTP server for the worker
// when telemetry is configured. It returns a shutdown func that must be deferred
// by the caller.
func startMetricsServer(
	ctx context.Context,
	cfg config.Config,
	log *slog.Logger,
	tele *telemetry.Provider,
) func() {
	if tele == nil || cfg.MetricsAddr == "" {
		return func() {}
	}
	metricsSrv := &http.Server{
		Addr:              cfg.MetricsAddr,
		Handler:           tele.PrometheusHandler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		log.Info("worker metrics listening", "addr", cfg.MetricsAddr)
		if serveErr := metricsSrv.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			log.Error("worker metrics server failed", "err", serveErr)
		}
	}()
	return func() {
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = metricsSrv.Shutdown(shutdownCtx)
	}
}

func redactedURL(u string) string {
	parsed, err := url.Parse(u)
	if err != nil || parsed.User == nil {
		return u
	}
	if _, ok := parsed.User.Password(); !ok {
		return u
	}
	parsed.User = url.UserPassword(parsed.User.Username(), "redacted")
	return parsed.String()
}

// asynqLogger adapts slog to Asynq's logging interface.
type asynqLogger struct {
	log *slog.Logger
}

func (a *asynqLogger) Debug(args ...any) { a.write(slog.LevelDebug, args...) }
func (a *asynqLogger) Info(args ...any)  { a.write(slog.LevelInfo, args...) }
func (a *asynqLogger) Warn(args ...any)  { a.write(slog.LevelWarn, args...) }
func (a *asynqLogger) Error(args ...any) { a.write(slog.LevelError, args...) }
func (a *asynqLogger) Fatal(args ...any) { a.write(slog.LevelError, args...) }

func (a *asynqLogger) write(level slog.Level, args ...any) {
	if a.log == nil {
		return
	}
	a.log.Log(context.Background(), level, fmt.Sprint(args...))
}
