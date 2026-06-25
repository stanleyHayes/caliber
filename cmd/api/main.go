// Command api runs the Caliber backend (gRPC + REST gateway). Wiring only;
// lifecycle lives in internal/platform/server and routing in httpserver.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	grpcadapter "github.com/xcreativs/caliber/internal/adapters/inbound/grpc"
	"github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/adapters/outbound/postgres"
	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/app/roles"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/platform/config"
	"github.com/xcreativs/caliber/internal/platform/logging"
	"github.com/xcreativs/caliber/internal/platform/server"
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
		log.Warn("missing configuration", "missing", missing, "env", cfg.Env)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var model app.LLMClient
	if cfg.AnthropicAPIKey != "" {
		model = llm.NewClaude(llm.WithAPIKey(cfg.AnthropicAPIKey), llm.WithModel(cfg.AnthropicModel))
		log.Info("llm provider selected", "provider", "claude", "model", cfg.AnthropicModel)
	} else {
		model = llm.NewDev()
		log.Warn("ANTHROPIC_API_KEY not set; using deterministic dev LLM")
	}

	var roleRepo role.RoleRepository = memory.NewRoleRepo()
	if cfg.DatabaseURL != "" {
		pool, perr := pgxpool.New(ctx, cfg.DatabaseURL)
		if perr != nil {
			return perr
		}
		defer pool.Close()
		if perr = pool.Ping(ctx); perr != nil {
			return perr
		}
		roleRepo = postgres.NewRoleRepo(pool)
		log.Info("persistence selected", "provider", "postgres")
	} else {
		log.Warn("CALIBER_DATABASE_URL not set; using in-memory repositories")
	}

	roleSrv := grpcadapter.NewRoleServer(roles.NewSpecGenerator(model, roleRepo, time.Now))
	return server.Run(ctx, cfg, log, grpcadapter.Services{Role: roleSrv})
}
