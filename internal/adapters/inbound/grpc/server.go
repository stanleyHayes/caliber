package grpcadapter

import (
	"context"
	"strings"

	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

// Services holds the concrete gRPC service implementations to register; any
// unset service falls back to its generated Unimplemented stub.
type Services struct {
	Role caliberv1.RoleServiceServer
}

// NewGRPCServer builds a gRPC server with every Caliber service registered.
func NewGRPCServer(svc Services) *grpc.Server {
	s := grpc.NewServer()
	role := svc.Role
	if role == nil {
		role = caliberv1.UnimplementedRoleServiceServer{}
	}
	caliberv1.RegisterIdentityServiceServer(s, caliberv1.UnimplementedIdentityServiceServer{})
	caliberv1.RegisterRoleServiceServer(s, role)
	caliberv1.RegisterTalentServiceServer(s, caliberv1.UnimplementedTalentServiceServer{})
	caliberv1.RegisterMatchingServiceServer(s, caliberv1.UnimplementedMatchingServiceServer{})
	caliberv1.RegisterInterviewServiceServer(s, caliberv1.UnimplementedInterviewServiceServer{})
	caliberv1.RegisterCandidateAgentServiceServer(s, caliberv1.UnimplementedCandidateAgentServiceServer{})
	caliberv1.RegisterDashboardServiceServer(s, caliberv1.UnimplementedDashboardServiceServer{})
	caliberv1.RegisterAuditServiceServer(s, caliberv1.UnimplementedAuditServiceServer{})
	reflection.Register(s)
	return s
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
