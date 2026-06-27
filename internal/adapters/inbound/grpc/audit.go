package grpcadapter

import (
	"context"

	"github.com/xcreativs/caliber/internal/domain/audit"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// AuditServer implements caliberv1.AuditServiceServer (CAL-084): it surfaces the
// append-only audit trail (human approvals, score overrides, agent actions,
// contest resolutions) for a given entity. Reading the trail is restricted to
// reviewers (employer/recruiter) since it exposes actor ids and actions.
type AuditServer struct {
	caliberv1.UnimplementedAuditServiceServer

	audit audit.AuditRepository
}

// NewAuditServer builds the audit gRPC service over the audit repository.
func NewAuditServer(repo audit.AuditRepository) *AuditServer { return &AuditServer{audit: repo} }

// ListAuditLog returns the audit entries for an entity, newest first, paginated.
func (s *AuditServer) ListAuditLog(
	ctx context.Context, req *caliberv1.ListAuditLogRequest,
) (*caliberv1.ListAuditLogResponse, error) {
	if _, err := RequireRole(ctx, identity.RoleEmployer, identity.RoleRecruiter); err != nil {
		return nil, errToStatus(err)
	}
	entity := req.GetEntity()
	entityID := kernel.ID(req.GetEntityId())
	if entity == "" || entityID.IsZero() {
		return nil, errToStatus(kernel.Invalid("audit: entity and entity_id are required"))
	}
	page := pageFromProto(req.GetPage())
	entries, total, err := s.audit.List(ctx, entity, entityID, page)
	if err != nil {
		return nil, errToStatus(err)
	}
	out := make([]*caliberv1.AuditEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, auditEntryToProto(e))
	}
	return &caliberv1.ListAuditLogResponse{Entries: out, Page: pageResponseToProto(page, total)}, nil
}

func auditEntryToProto(e *audit.AuditEntry) *caliberv1.AuditEntry {
	return &caliberv1.AuditEntry{
		Id:          e.ID.String(),
		ActorUserId: e.ActorUserID.String(),
		Action:      e.Action,
		Entity:      e.Entity,
		EntityId:    e.EntityID.String(),
		BeforeJson:  e.BeforeJSON,
		AfterJson:   e.AfterJSON,
		Timestamp:   timestamppb.New(e.Timestamp),
	}
}
