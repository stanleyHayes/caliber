package grpcadapter

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/xcreativs/caliber/internal/domain/audit"
	"github.com/xcreativs/caliber/internal/domain/authz"
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
	principal, err := RequirePermission(ctx, authz.PermReadAuditLog)
	if err != nil {
		return nil, errToStatus(err)
	}
	entity := req.GetEntity()
	entityID := kernel.ID(req.GetEntityId())
	if entity == "" || entityID.IsZero() {
		return nil, errToStatus(kernel.Invalid("audit: entity and entity_id are required"))
	}
	// Tenant-scope the trail (CAL-153): an employer/recruiter sees only entries
	// they own — their hiring decisions plus contests raised on their assessments
	// — not another employer's decisions on a shared subject. A platform admin
	// sees the whole trail (unscoped).
	owner := principal.UserID
	if principal.Role == identity.RoleAdmin.String() {
		owner = ""
	}
	page := pageFromProto(req.GetPage())
	entries, total, err := s.audit.ListForOwner(ctx, entity, entityID, owner, page)
	if err != nil {
		return nil, errToStatus(err)
	}
	out := make([]*caliberv1.AuditEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, auditEntryToProto(e))
	}
	return &caliberv1.ListAuditLogResponse{Entries: out, Page: pageResponseToProto(page, total)}, nil
}

// ExportAuditReport returns a filtered audit-log report as a downloadable JSON
// or CSV payload. The endpoint is reviewer-only and never includes raw payloads
// beyond the redacted snapshots already stored in the audit trail.
func (s *AuditServer) ExportAuditReport(
	ctx context.Context, req *caliberv1.ExportAuditReportRequest,
) (*caliberv1.ExportAuditReportResponse, error) {
	principal, err := RequirePermission(ctx, authz.PermReadAuditLog)
	if err != nil {
		return nil, errToStatus(err)
	}
	filter, err := exportFilterFromProto(req)
	if err != nil {
		return nil, errToStatus(err)
	}
	// Tenant-scope the export exactly like ListAuditLog (CAL-153): a reviewer can
	// only export their own hiring-decision trail, never another employer's
	// decisions on a shared subject. A platform admin exports the whole trail.
	owner := principal.UserID
	if principal.Role == identity.RoleAdmin.String() {
		owner = ""
	}
	entries, err := collectReportEntries(ctx, s.audit, filter, owner)
	if err != nil {
		return nil, errToStatus(err)
	}
	resp, err := renderReport(entries, req.GetFormat())
	if err != nil {
		return nil, errToStatus(err)
	}
	return resp, nil
}

func exportFilterFromProto(req *caliberv1.ExportAuditReportRequest) (audit.ReportFilter, error) {
	startPb, endPb := req.GetStartTime(), req.GetEndTime()
	if startPb == nil || endPb == nil {
		return audit.ReportFilter{}, kernel.Invalid("audit: start_time and end_time are required")
	}
	start, end := startPb.AsTime(), endPb.AsTime()
	if end.Before(start) {
		return audit.ReportFilter{}, kernel.Invalid("audit: end_time must be after start_time")
	}
	return audit.ReportFilter{
		Start:    start,
		End:      end,
		Actions:  req.GetActions(),
		Entities: req.GetEntities(),
	}, nil
}

func renderReport(entries []*audit.AuditEntry, format caliberv1.ExportFormat) (*caliberv1.ExportAuditReportResponse, error) {
	switch format {
	case caliberv1.ExportFormat_EXPORT_FORMAT_CSV:
		payload, err := marshalAuditCSV(entries)
		if err != nil {
			return nil, err
		}
		return &caliberv1.ExportAuditReportResponse{
			Payload:     payload,
			Filename:    reportFilename("audit-report", "csv"),
			ContentType: "text/csv",
		}, nil
	case caliberv1.ExportFormat_EXPORT_FORMAT_JSON:
		payload, err := marshalAuditJSON(entries)
		if err != nil {
			return nil, err
		}
		return &caliberv1.ExportAuditReportResponse{
			Payload:     payload,
			Filename:    reportFilename("audit-report", "json"),
			ContentType: "application/json",
		}, nil
	default:
		return nil, kernel.Invalid("audit: format must be JSON or CSV")
	}
}

const (
	maxReportEntries = 10000
	reportPageSize   = 100
)

func collectReportEntries(
	ctx context.Context, repo audit.AuditRepository, filter audit.ReportFilter, ownerID kernel.ID,
) ([]*audit.AuditEntry, error) {
	var all []*audit.AuditEntry
	for pageNum := 1; ; pageNum++ {
		page := kernel.NewPage(pageNum, reportPageSize)
		batch, total, err := repo.SearchForOwner(ctx, filter, ownerID, page)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if int64(len(all)) > maxReportEntries {
			return nil, kernel.Invalidf("audit: report exceeds %d rows; narrow the filter", maxReportEntries)
		}
		if len(batch) == 0 || int64(len(all)) >= total {
			return all, nil
		}
	}
}

func marshalAuditJSON(entries []*audit.AuditEntry) ([]byte, error) {
	rows := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, map[string]any{
			"id":            e.ID.String(),
			"actor_user_id": e.ActorUserID.String(),
			"action":        e.Action,
			"entity":        e.Entity,
			"entity_id":     e.EntityID.String(),
			"before_json":   e.BeforeJSON,
			"after_json":    e.AfterJSON,
			"timestamp":     e.Timestamp.Format(time.RFC3339),
		})
	}
	return json.Marshal(rows)
}

func marshalAuditCSV(entries []*audit.AuditEntry) ([]byte, error) {
	var b strings.Builder
	w := csv.NewWriter(&b)
	if err := w.Write([]string{"id", "actor_user_id", "action", "entity", "entity_id", "before_json", "after_json", "timestamp"}); err != nil {
		return nil, err
	}
	for _, e := range entries {
		if err := w.Write([]string{
			e.ID.String(),
			e.ActorUserID.String(),
			e.Action,
			e.Entity,
			e.EntityID.String(),
			e.BeforeJSON,
			e.AfterJSON,
			e.Timestamp.Format(time.RFC3339),
		}); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return []byte(b.String()), w.Error()
}

func reportFilename(prefix, ext string) string {
	return fmt.Sprintf("%s-%s.%s", prefix, time.Now().UTC().Format("20060102T150405Z"), ext)
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
