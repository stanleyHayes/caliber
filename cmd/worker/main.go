// Command worker runs background jobs (candidate-agent runs, interview scoring,
// batch re-matching) via Asynq (EPIC-03).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"

	"github.com/xcreativs/caliber/internal/adapters/inbound/jobs"
	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	queueadapter "github.com/xcreativs/caliber/internal/adapters/outbound/queue"
	candidateagentapp "github.com/xcreativs/caliber/internal/app/candidateagent"
	interviewapp "github.com/xcreativs/caliber/internal/app/interview"
	"github.com/xcreativs/caliber/internal/platform/config"
	"github.com/xcreativs/caliber/internal/platform/logging"
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
	log := logging.New(cfg.LogLevel)
	if missing := cfg.Validate(); len(missing) > 0 {
		if cfg.IsProd() {
			return fmt.Errorf("missing required configuration: %v", missing)
		}
		log.Warn("missing configuration", "missing", missing, "env", cfg.Env)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return runWorker(ctx, cfg, log)
}

func runWorker(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	if cfg.RedisURL == "" {
		return errors.New("CALIBER_REDIS_URL is required to run the worker")
	}

	model := wiring.BuildLLM(cfg, log)
	auditRepo := memory.NewAuditRepo()
	repos, cleanup, _, err := wiring.OpenRepositories(ctx, cfg, log)
	if err != nil {
		return err
	}
	defer cleanup()

	agentRunner := candidateagentapp.NewAgentRunner(
		repos.Candidates, repos.Profiles, repos.Roles, repos.Apps, model,
		candidateagentapp.WithAuditTrail(auditRepo, time.Now),
		candidateagentapp.WithWakeUpInsights(repos.Interviews, repos.Matches),
	)
	interviewer := interviewapp.NewInterviewer(
		repos.Roles, repos.Interviews, model, 0,
		interviewapp.WithPassportUpdater(repos.Profiles),
	)

	redisOpt, err := queueadapter.RedisOpt(cfg.RedisURL)
	if err != nil {
		return err
	}

	mux := jobs.NewMux(log)
	jobs.RegisterHandlers(mux, jobs.HandlerDeps{
		AgentRunner: agentRunner,
		Interviewer: interviewer,
	}, log)

	server := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency:     cfg.WorkerConcurrency,
		Queues:          queueadapter.Priorities(),
		ShutdownTimeout: 10 * time.Second,
		Logger:          &asynqLogger{log: log},
	})

	log.Info("worker started", "redis_url", redactedURL(cfg.RedisURL), "concurrency", cfg.WorkerConcurrency)
	if err := server.Start(mux); err != nil {
		return fmt.Errorf("worker: start server: %w", err)
	}

	<-ctx.Done()
	log.Info("worker shutting down")
	server.Stop()
	server.Shutdown()
	return nil
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
