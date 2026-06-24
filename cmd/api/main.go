// Command api runs the Caliber backend: a gRPC server fronted by a
// grpc-gateway REST/JSON layer (mounted on chi alongside health checks).
// Service handlers are stubbed (Unimplemented) here; real implementations land
// in their respective epics. This proves the contract pipeline end-to-end.
package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
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
		log.Warn("missing configuration", "missing", missing, "env", cfg.Env)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// gRPC server with all services registered (stub implementations for now).
	grpcSrv := grpc.NewServer()
	registerServices(grpcSrv)
	reflection.Register(grpcSrv)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Error("grpc listen failed", "addr", cfg.GRPCAddr, "err", err)
		panic(err)
	}
	go func() {
		log.Info("grpc server listening", "addr", cfg.GRPCAddr)
		if err := grpcSrv.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Error("grpc serve failed", "err", err)
		}
	}()

	// REST gateway mounted on chi, plus health/readiness.
	gw, err := newGateway(ctx, dialTarget(cfg.GRPCAddr))
	if err != nil {
		log.Error("gateway init failed", "err", err)
		panic(err)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Get("/healthz", health("ok"))
	r.Get("/readyz", health("ready"))
	r.Handle("/v1/*", gw)

	httpSrv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		log.Info("http gateway listening", "addr", cfg.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http serve failed", "err", err)
		}
	}()

	<-ctx.Done()
	log.Info("shutdown signal received")
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutCtx); err != nil {
		log.Warn("http shutdown", "err", err)
	}
	grpcSrv.GracefulStop()
	log.Info("stopped cleanly")
}

func registerServices(s *grpc.Server) {
	caliberv1.RegisterIdentityServiceServer(s, caliberv1.UnimplementedIdentityServiceServer{})
	caliberv1.RegisterRoleServiceServer(s, caliberv1.UnimplementedRoleServiceServer{})
	caliberv1.RegisterTalentServiceServer(s, caliberv1.UnimplementedTalentServiceServer{})
	caliberv1.RegisterMatchingServiceServer(s, caliberv1.UnimplementedMatchingServiceServer{})
	caliberv1.RegisterInterviewServiceServer(s, caliberv1.UnimplementedInterviewServiceServer{})
	caliberv1.RegisterCandidateAgentServiceServer(s, caliberv1.UnimplementedCandidateAgentServiceServer{})
	caliberv1.RegisterDashboardServiceServer(s, caliberv1.UnimplementedDashboardServiceServer{})
	caliberv1.RegisterAuditServiceServer(s, caliberv1.UnimplementedAuditServiceServer{})
}

type gatewayRegistrar func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error

func newGateway(ctx context.Context, target string) (*runtime.ServeMux, error) {
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	for _, reg := range []gatewayRegistrar{
		caliberv1.RegisterIdentityServiceHandlerFromEndpoint,
		caliberv1.RegisterRoleServiceHandlerFromEndpoint,
		caliberv1.RegisterTalentServiceHandlerFromEndpoint,
		caliberv1.RegisterMatchingServiceHandlerFromEndpoint,
		caliberv1.RegisterInterviewServiceHandlerFromEndpoint,
		caliberv1.RegisterCandidateAgentServiceHandlerFromEndpoint,
		caliberv1.RegisterDashboardServiceHandlerFromEndpoint,
		caliberv1.RegisterAuditServiceHandlerFromEndpoint,
	} {
		if err := reg(ctx, mux, target, opts); err != nil {
			return nil, err
		}
	}
	return mux, nil
}

func health(status string) http.HandlerFunc {
	body := []byte(`{"status":"` + status + `"}`)
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

func dialTarget(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "localhost" + addr
	}
	return addr
}
