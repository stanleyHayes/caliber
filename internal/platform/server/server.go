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
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/platform/config"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

const shutdownTimeout = 10 * time.Second

// Option configures the server run.
type Option func(*runConfig)

type runConfig struct {
	asynqmonPath     string
	asynqmon         http.Handler
	verifier         app.TokenService
	metrics          http.Handler
	aiQualityMetrics http.Handler
	onListen         func(Addrs)
}

// Addrs reports the concrete network addresses the server bound. With an
// ephemeral port (":0") the actual port is only known after binding, so callers
// read it here instead of assuming the configured value.
type Addrs struct {
	HTTP string
	GRPC string
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

// WithMetrics mounts the supplied handler at /metrics. For CAL-131 this should
// be the Prometheus exposition handler.
func WithMetrics(handler http.Handler) Option {
	return func(c *runConfig) { c.metrics = handler }
}

// WithAIQualityMetrics mounts the PII-free AI quality JSON handler at
// /debug/ai-quality (CAL-137), preserving the human-readable surface while
// /metrics serves Prometheus exposition format.
func WithAIQualityMetrics(handler http.Handler) Option {
	return func(c *runConfig) { c.aiQualityMetrics = handler }
}

// WithListenCallback registers fn to be invoked once both listeners are bound
// (before serving begins), with their concrete addresses. It is the supported
// way to discover an OS-assigned port when HTTPAddr/GRPCAddr use ":0" — e.g. in
// tests, which bind ephemeral ports to avoid colliding with other processes.
func WithListenCallback(fn func(Addrs)) Option {
	return func(c *runConfig) { c.onListen = fn }
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

func buildRouter(
	mux *runtime.ServeMux,
	cfg config.Config,
	log *slog.Logger,
	readiness []httpserver.ReadinessChecker,
	runCfg runConfig,
) *chi.Mux {
	r := httpserver.NewRouter(mux, cfg.IsProd(), cfg.AllowedOrigins, log, readiness...)
	// /metrics is the Prometheus scrape target: scrapers authenticate at the
	// network layer (a private metrics network / not exposed to public ingress),
	// not with a user bearer token, so gating it behind operator roles would break
	// scraping. It stays unauthenticated at the app layer — protect it in the
	// deployment topology (CAL-120).
	if runCfg.metrics != nil {
		r.Get("/metrics", runCfg.metrics.ServeHTTP)
	}
	// /debug/ai-quality is a human debug surface with no scraper contract, so gate
	// it behind the same operator authorization as the Asynqmon UI when a verifier
	// is configured (CAL-120). Without a verifier (local dev) it stays open.
	if runCfg.aiQualityMetrics != nil {
		r.With(operatorGuard(runCfg.verifier)...).Get("/debug/ai-quality", runCfg.aiQualityMetrics.ServeHTTP)
	}
	if runCfg.asynqmon != nil && runCfg.verifier != nil {
		httpserver.MountAsynqmon(r, runCfg.asynqmonPath, runCfg.asynqmon, runCfg.verifier)
	}
	return r
}

// operatorGuard returns the middleware that restricts an operator-only HTTP
// surface to authenticated employer/recruiter/admin principals. With no verifier
// configured (local dev) it returns nothing, leaving the route open.
func operatorGuard(verifier app.TokenService) []func(http.Handler) http.Handler {
	if verifier == nil {
		return nil
	}
	return []func(http.Handler) http.Handler{
		httpserver.Authorize(verifier, identity.RoleEmployer, identity.RoleRecruiter, identity.RoleAdmin),
	}
}

// listeners binds the gRPC and HTTP listeners and derives the gateway dial
// target from the gRPC listener's ACTUAL port, so an ephemeral ":0" bind still
// resolves (loopback is correct — the gateway dials the same-process gRPC
// server; a fixed port yields the same "localhost:<port>" as before). It closes
// what it opened on failure.
func listeners(ctx context.Context, cfg config.Config) (net.Listener, net.Listener, string, error) {
	var lc net.ListenConfig
	grpcLis, err := lc.Listen(ctx, "tcp", cfg.GRPCAddr)
	if err != nil {
		return nil, nil, "", err
	}
	_, grpcPort, err := net.SplitHostPort(grpcLis.Addr().String())
	if err != nil {
		_ = grpcLis.Close()
		return nil, nil, "", err
	}
	httpLis, err := lc.Listen(ctx, "tcp", cfg.HTTPAddr)
	if err != nil {
		_ = grpcLis.Close()
		return nil, nil, "", err
	}
	return grpcLis, httpLis, grpcadapter.DialTarget(":" + grpcPort), nil
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

	grpcLis, httpLis, dialTarget, err := listeners(ctx, cfg)
	if err != nil {
		return err
	}
	go func() {
		log.Info("grpc server listening", "addr", grpcLis.Addr().String())
		if serveErr := grpcSrv.Serve(grpcLis); serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			log.Error("grpc serve failed", "err", serveErr)
		}
	}()

	mux := runtime.NewServeMux()
	if err = grpcadapter.RegisterGateway(ctx, mux, dialTarget); err != nil {
		grpcSrv.GracefulStop()
		_ = httpLis.Close()
		return err
	}

	runCfg := runConfig{asynqmonPath: "/asynqmon"}
	for _, opt := range opts {
		opt(&runCfg)
	}
	if runCfg.onListen != nil {
		runCfg.onListen(Addrs{HTTP: httpLis.Addr().String(), GRPC: grpcLis.Addr().String()})
	}

	httpSrv := &http.Server{
		Handler:           buildRouter(mux, cfg, log, readiness, runCfg),
		ReadHeaderTimeout: shutdownTimeout,
	}
	go func() {
		log.Info("http gateway listening", "addr", httpLis.Addr().String())
		if serveErr := httpSrv.Serve(httpLis); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
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
