package grpcadapter

import (
	"context"

	"github.com/xcreativs/caliber/internal/app/roles"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

// RoleServer implements caliberv1.RoleServiceServer (currently the Flow A.1
// spec generation; remaining methods fall back to Unimplemented).
type RoleServer struct {
	caliberv1.UnimplementedRoleServiceServer
	gen *roles.SpecGenerator
}

// NewRoleServer builds the role gRPC service from its use-case.
func NewRoleServer(gen *roles.SpecGenerator) *RoleServer { return &RoleServer{gen: gen} }

// GenerateRoleSpec turns a free-text hiring need into a structured, persisted Role.
func (s *RoleServer) GenerateRoleSpec(ctx context.Context, req *caliberv1.GenerateRoleSpecRequest) (*caliberv1.GenerateRoleSpecResponse, error) {
	r, err := s.gen.Generate(ctx, kernel.ID(req.GetEmployerId()), req.GetFreeText())
	if err != nil {
		return nil, errToStatus(err)
	}
	return &caliberv1.GenerateRoleSpecResponse{Role: roleToProto(r)}, nil
}
