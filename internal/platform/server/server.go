// Package server runs the API process: the gRPC server and its REST gateway
// with graceful shutdown. It is the composition glue, kept out of main.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	grpcadapter "github.com/xcreativs/caliber/internal/adapters/inbound/grpc"
	"github.com/xcreativs/caliber/internal/adapters/inbound/httpserver"
	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/platform/config"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

const shutdownTimeout = 10 * time.Second

// Option configures the server run.
type Option func(*runConfig)

type runConfig struct {
	asynqmonPath string
	asynqmon     http.Handler
	verifier     app.TokenService
	metrics      http.Handler
}

// WithAsynqmon mounts the Asynqmon monitoring UI at the given path, protected by
// the supplied token verifier. Only employer and recruiter principals are
// permitted (CAL-028).
func WithAsynqmon(path string, handler http.Handler, verifier app.TokenService) Option {
	return func(c *runConfig) {
		c.asynqmonPath = path
		c.asynqmon = handler
		c.verifier = verifier
	}
}

// WithMetrics mounts the AI quality metrics handler at /metrics (CAL-137). The
// handler is expected to serve PII-free JSON.
func WithMetrics(handler http.Handler) Option {
	return func(c *runConfig) { c.metrics = handler }
}

// Run starts the gRPC server and REST gateway, blocks until ctx is cancelled,
// then shuts both down gracefully.
func Run(
	ctx context.Context,
	cfg config.Config,
	log *slog.Logger,
	svc grpcadapter.Services,
	readiness ...httpserver.ReadinessChecker,
) error {
	return RunWithOptions(ctx, cfg, log, svc, readiness, nil)
}

//nolint:ireturn // Returns the standard chi.Router interface for mounting.
func buildRouter(
	mux *runtime.ServeMux,
	cfg config.Config,
	log *slog.Logger,
	readiness []httpserver.ReadinessChecker,
	runCfg runConfig,
) chi.Router {
	r := httpserver.NewRouter(mux, cfg.IsProd(), cfg.AllowedOrigins, log, readiness...)
	if runCfg.metrics != nil {
		r.Get("/metrics", runCfg.metrics.ServeHTTP)
	}
	if runCfg.asynqmon != nil && runCfg.verifier != nil {
		httpserver.MountAsynqmon(r, runCfg.asynqmonPath, runCfg.asynqmon, runCfg.verifier)
	}
	return r
}

// RunWithOptions starts the server with the supplied optional configuration.
// It is exported so tests and wiring can mount extra HTTP surfaces such as the
// Asynqmon dashboard (CAL-028).
func RunWithOptions(
	ctx context.Context,
	cfg config.Config,
	log *slog.Logger,
	svc grpcadapter.Services,
	readiness []httpserver.ReadinessChecker,
	opts []Option,
) error {
	//nolint:contextcheck // stream auth derives ctx from the live ServerStream at call time (grpc wrapper pattern)
	grpcSrv := grpcadapter.NewGRPCServer(svc)

	var lc net.ListenConfig
	lis, err := lc.Listen(ctx, "tcp", cfg.GRPCAddr)
	if err != nil {
		return err
	}
	go func() {
		log.Info("grpc server listening", "addr", cfg.GRPCAddr)
		if serveErr := grpcSrv.Serve(lis); serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			log.Error("grpc serve failed", "err", serveErr)
		}
	}()

	mux := runtime.NewServeMux()
	if err = grpcadapter.RegisterGateway(ctx, mux, grpcadapter.DialTarget(cfg.GRPCAddr)); err != nil {
		grpcSrv.GracefulStop()
		return err
	}

	runCfg := runConfig{asynqmonPath: "/asynqmon"}
	for _, opt := range opts {
		opt(&runCfg)
	}

	r := buildRouter(mux, cfg, log, readiness, runCfg)

	httpSrv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: shutdownTimeout,
	}
	go func() {
		log.Info("http gateway listening", "addr", cfg.HTTPAddr)
		if serveErr := httpSrv.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			log.Error("http serve failed", "err", serveErr)
		}
	}()

	<-ctx.Done()
	log.Info("shutdown signal received")
	shutCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	shutErr := httpSrv.Shutdown(shutCtx)
	grpcSrv.GracefulStop()
	return shutErr
}
