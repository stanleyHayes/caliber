// Command migrate applies database migrations for local and deployed bootstraps.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/xcreativs/caliber/internal/platform/config"
	"github.com/xcreativs/caliber/internal/platform/logging"
	"github.com/xcreativs/caliber/internal/platform/migrate"
)

const defaultMigrationsDir = "db/migrations"

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
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return errors.New("CALIBER_DATABASE_URL is required")
	}
	log := logging.New(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return migrateUp(ctx, cfg.DatabaseURL, migrationsDir(), log)
}

func migrationsDir() string {
	if dir := strings.TrimSpace(os.Getenv("CALIBER_MIGRATIONS_DIR")); dir != "" {
		return dir
	}
	return defaultMigrationsDir
}

func migrateUp(ctx context.Context, databaseURL, dir string, log *slog.Logger) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	pingCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		return err
	}

	log.Info("applying database migrations", "dir", dir)
	if err := migrate.Up(db, dir); err != nil {
		return err
	}
	log.Info("database migrations applied")
	return nil
}
