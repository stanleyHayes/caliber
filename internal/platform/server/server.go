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

	grpcadapter "github.com/xcreativs/caliber/internal/adapters/inbound/grpc"
	"github.com/xcreativs/caliber/internal/adapters/inbound/httpserver"
	"github.com/xcreativs/caliber/internal/platform/config"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

const shutdownTimeout = 10 * time.Second

// Run starts the gRPC server and REST gateway, blocks until ctx is cancelled,
// then shuts both down gracefully.
func Run(
	ctx context.Context,
	cfg config.Config,
	log *slog.Logger,
	svc grpcadapter.Services,
	readiness ...httpserver.ReadinessChecker,
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

	httpSrv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpserver.NewRouter(mux, cfg.IsProd(), log, readiness...),
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
