package grpcadapter

import (
	"context"

	"github.com/xcreativs/caliber/internal/app/roles"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

// AvailabilityCounter reports how many candidates are an immediate, honest fit
// for a role — logistically compatible with a verified profile already covering
// the must-haves — without LLM scoring. It backs the instant pool-depth signal.
type AvailabilityCounter interface {
	CountAvailable(ctx context.Context, roleID kernel.ID) (int, error)
}

// RoleServer implements caliberv1.RoleServiceServer (currently the Flow A.1
// spec generation; remaining methods fall back to Unimplemented).
type RoleServer struct {
	caliberv1.UnimplementedRoleServiceServer

	gen     *roles.SpecGenerator
	editor  *roles.SpecEditor
	counter AvailabilityCounter
}

// NewRoleServer builds the role gRPC service from its use-cases. counter is
// optional: when nil, generated roles report 0 available matches.
func NewRoleServer(gen *roles.SpecGenerator, editor *roles.SpecEditor, counter AvailabilityCounter) *RoleServer {
	return &RoleServer{gen: gen, editor: editor, counter: counter}
}

// GenerateRoleSpec turns a free-text hiring need into a structured, persisted Role
// and returns the instant "N strong matches already in your pool" signal
// (CAL-055/037) alongside it.
func (s *RoleServer) GenerateRoleSpec(
	ctx context.Context,
	req *caliberv1.GenerateRoleSpecRequest,
) (*caliberv1.GenerateRoleSpecResponse, error) {
	// An employer creates roles only under their own id (CAL-116 IDOR guard).
	if err := requireSelfEmployer(ctx, req.GetEmployerId()); err != nil {
		return nil, errToStatus(err)
	}
	r, err := s.gen.Generate(ctx, kernel.ID(req.GetEmployerId()), req.GetFreeText())
	if err != nil {
		return nil, errToStatus(err)
	}
	resp := &caliberv1.GenerateRoleSpecResponse{Role: roleToProto(r)}
	// Best-effort teaser: a counting hiccup must not fail role creation.
	if s.counter != nil {
		if n, cerr := s.counter.CountAvailable(ctx, r.ID); cerr == nil {
			resp.AvailableMatches = int32(n) //nolint:gosec // pool count is small, bounded by recallWindow
		}
	}
	return resp, nil
}

// GetRole returns a persisted role by id. Any authenticated user may view a role
// (candidates see postings to apply); writes are reviewer-only (CAL-116).
func (s *RoleServer) GetRole(ctx context.Context, req *caliberv1.GetRoleRequest) (*caliberv1.GetRoleResponse, error) {
	if _, err := RequireAuth(ctx); err != nil {
		return nil, errToStatus(err)
	}
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
	principal, err := RequireRole(ctx, identity.RoleEmployer, identity.RoleRecruiter)
	if err != nil {
		return nil, errToStatus(err)
	}
	// Ownership (CAL-116 IDOR guard): an employer may only edit their own role.
	existing, err := s.editor.Get(ctx, kernel.ID(req.GetRoleId()))
	if err != nil {
		return nil, errToStatus(err)
	}
	if existing.EmployerID.String() != principal.UserID.String() {
		return nil, errToStatus(kernel.Forbidden("auth: may only edit your own roles"))
	}
	r, err := s.editor.Update(ctx, kernel.ID(req.GetRoleId()), specFromProto(req.GetSpec()), rubricFromProto(req.GetRubric()))
	if err != nil {
		return nil, errToStatus(err)
	}
	return &caliberv1.UpdateRoleSpecResponse{Role: roleToProto(r)}, nil
}

// ListRoles returns a page of an employer's roles.
func (s *RoleServer) ListRoles(ctx context.Context, req *caliberv1.ListRolesRequest) (*caliberv1.ListRolesResponse, error) {
	// An employer lists only their own roles (CAL-116 IDOR guard).
	if err := requireSelfEmployer(ctx, req.GetEmployerId()); err != nil {
		return nil, errToStatus(err)
	}
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
