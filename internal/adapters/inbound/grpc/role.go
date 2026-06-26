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

	gen    *roles.SpecGenerator
	editor *roles.SpecEditor
}

// NewRoleServer builds the role gRPC service from its use-cases.
func NewRoleServer(gen *roles.SpecGenerator, editor *roles.SpecEditor) *RoleServer {
	return &RoleServer{gen: gen, editor: editor}
}

// GenerateRoleSpec turns a free-text hiring need into a structured, persisted Role.
func (s *RoleServer) GenerateRoleSpec(
	ctx context.Context,
	req *caliberv1.GenerateRoleSpecRequest,
) (*caliberv1.GenerateRoleSpecResponse, error) {
	r, err := s.gen.Generate(ctx, kernel.ID(req.GetEmployerId()), req.GetFreeText())
	if err != nil {
		return nil, errToStatus(err)
	}
	return &caliberv1.GenerateRoleSpecResponse{Role: roleToProto(r)}, nil
}

// GetRole returns a persisted role by id.
func (s *RoleServer) GetRole(ctx context.Context, req *caliberv1.GetRoleRequest) (*caliberv1.GetRoleResponse, error) {
	r, err := s.editor.Get(ctx, kernel.ID(req.GetRoleId()))
	if err != nil {
		return nil, errToStatus(err)
	}
	return &caliberv1.GetRoleResponse{Role: roleToProto(r)}, nil
}

// UpdateRoleSpec applies an edited spec and rubric (re-weighting) to a role.
func (s *RoleServer) UpdateRoleSpec(
	ctx context.Context,
	req *caliberv1.UpdateRoleSpecRequest,
) (*caliberv1.UpdateRoleSpecResponse, error) {
	r, err := s.editor.Update(ctx, kernel.ID(req.GetRoleId()), specFromProto(req.GetSpec()), rubricFromProto(req.GetRubric()))
	if err != nil {
		return nil, errToStatus(err)
	}
	return &caliberv1.UpdateRoleSpecResponse{Role: roleToProto(r)}, nil
}

// ListRoles returns a page of an employer's roles.
func (s *RoleServer) ListRoles(ctx context.Context, req *caliberv1.ListRolesRequest) (*caliberv1.ListRolesResponse, error) {
	page := pageFromProto(req.GetPage())
	roles, total, err := s.editor.List(ctx, kernel.ID(req.GetEmployerId()), page)
	if err != nil {
		return nil, errToStatus(err)
	}
	out := make([]*caliberv1.Role, 0, len(roles))
	for _, r := range roles {
		out = append(out, roleToProto(r))
	}
	return &caliberv1.ListRolesResponse{Roles: out, Page: pageResponseToProto(page, total)}, nil
}
