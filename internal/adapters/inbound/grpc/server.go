package grpcadapter

import (
	"context"
	"strings"

	"github.com/xcreativs/caliber/internal/app"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

// Services holds the concrete gRPC service implementations to register; any
// unset service falls back to its generated Unimplemented stub.
type Services struct {
	Identity  caliberv1.IdentityServiceServer
	Role      caliberv1.RoleServiceServer
	Match     caliberv1.MatchingServiceServer
	Interview caliberv1.InterviewServiceServer
	Agent     caliberv1.CandidateAgentServiceServer
	Dashboard caliberv1.DashboardServiceServer
	Talent    caliberv1.TalentServiceServer
	Contest   caliberv1.ContestServiceServer
	Audit     caliberv1.AuditServiceServer

	// AccessVerifier, when set, installs the auth interceptor that authenticates
	// bearer access tokens and injects the principal into each request context.
	AccessVerifier app.TokenService

	// RateLimiter, when set, installs the token-bucket rate-limit interceptor
	// (CAL-112). It runs after auth so it can key by the authenticated principal.
	RateLimiter *RateLimiter
}

// NewGRPCServer builds a gRPC server with every Caliber service registered.
func NewGRPCServer(svc Services) *grpc.Server {
	var unary []grpc.UnaryServerInterceptor
	var stream []grpc.StreamServerInterceptor
	if svc.AccessVerifier != nil {
		unary = append(unary, NewAuthInterceptor(svc.AccessVerifier))
		// Streaming RPCs (StartInterview) need their own interceptor — unary ones
		// don't run for streams — so the principal reaches the stream handler.
		stream = append(stream, NewAuthStreamInterceptor(svc.AccessVerifier))
	}
	if svc.RateLimiter != nil {
		unary = append(unary, NewRateLimitInterceptor(svc.RateLimiter))
	}
	var opts []grpc.ServerOption
	if len(unary) > 0 {
		opts = append(opts, grpc.ChainUnaryInterceptor(unary...))
	}
	if len(stream) > 0 {
		opts = append(opts, grpc.ChainStreamInterceptor(stream...))
	}
	svc = withStubs(svc)
	s := grpc.NewServer(opts...)
	caliberv1.RegisterIdentityServiceServer(s, svc.Identity)
	caliberv1.RegisterRoleServiceServer(s, svc.Role)
	caliberv1.RegisterTalentServiceServer(s, svc.Talent)
	caliberv1.RegisterMatchingServiceServer(s, svc.Match)
	caliberv1.RegisterInterviewServiceServer(s, svc.Interview)
	caliberv1.RegisterCandidateAgentServiceServer(s, svc.Agent)
	caliberv1.RegisterDashboardServiceServer(s, svc.Dashboard)
	caliberv1.RegisterContestServiceServer(s, svc.Contest)
	caliberv1.RegisterAuditServiceServer(s, svc.Audit)
	reflection.Register(s)
	return s
}

// withStubs fills any unset service with its generated Unimplemented stub, so a
// partially-wired Services value still registers every service cleanly.
func withStubs(svc Services) Services {
	if svc.Identity == nil {
		svc.Identity = caliberv1.UnimplementedIdentityServiceServer{}
	}
	if svc.Role == nil {
		svc.Role = caliberv1.UnimplementedRoleServiceServer{}
	}
	if svc.Talent == nil {
		svc.Talent = caliberv1.UnimplementedTalentServiceServer{}
	}
	if svc.Match == nil {
		svc.Match = caliberv1.UnimplementedMatchingServiceServer{}
	}
	if svc.Interview == nil {
		svc.Interview = caliberv1.UnimplementedInterviewServiceServer{}
	}
	if svc.Agent == nil {
		svc.Agent = caliberv1.UnimplementedCandidateAgentServiceServer{}
	}
	if svc.Dashboard == nil {
		svc.Dashboard = caliberv1.UnimplementedDashboardServiceServer{}
	}
	if svc.Contest == nil {
		svc.Contest = caliberv1.UnimplementedContestServiceServer{}
	}
	if svc.Audit == nil {
		svc.Audit = caliberv1.UnimplementedAuditServiceServer{}
	}
	return svc
}

type gatewayRegistrar func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error

// RegisterGateway wires every REST/JSON gateway handler to the gRPC endpoint.
func RegisterGateway(ctx context.Context, mux *runtime.ServeMux, endpoint string) error {
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	for _, reg := range []gatewayRegistrar{
		caliberv1.RegisterIdentityServiceHandlerFromEndpoint,
		caliberv1.RegisterRoleServiceHandlerFromEndpoint,
		caliberv1.RegisterTalentServiceHandlerFromEndpoint,
		caliberv1.RegisterMatchingServiceHandlerFromEndpoint,
		caliberv1.RegisterInterviewServiceHandlerFromEndpoint,
		caliberv1.RegisterCandidateAgentServiceHandlerFromEndpoint,
		caliberv1.RegisterDashboardServiceHandlerFromEndpoint,
		caliberv1.RegisterContestServiceHandlerFromEndpoint,
		caliberv1.RegisterAuditServiceHandlerFromEndpoint,
	} {
		if err := reg(ctx, mux, endpoint, opts); err != nil {
			return err
		}
	}
	return nil
}

// DialTarget normalizes a listen address (e.g. ":9090") into a dial target.
func DialTarget(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "localhost" + addr
	}
	return addr
}
