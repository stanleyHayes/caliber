// Command worker runs background jobs (candidate-agent runs, interview scoring,
// batch re-matching, time-advance) via Asynq. Asynq wiring lands in CAL-024;
// this skeleton boots config + logging and blocks until signalled.
package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/xcreativs/caliber/internal/platform/config"
	"github.com/xcreativs/caliber/internal/platform/logging"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	log := logging.New(cfg.LogLevel)

	if missing := cfg.Validate(); len(missing) > 0 {
		log.Warn("missing configuration (worker)", "missing", missing, "env", cfg.Env)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Info("worker started", "env", cfg.Env)
	<-ctx.Done()
	log.Info("worker shutting down")
}
