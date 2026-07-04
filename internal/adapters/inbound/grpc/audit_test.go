package grpcadapter

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/audit"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAuditServer_ListAuditLog(t *testing.T) {
	repo := memory.NewAuditRepo()
	owner, candidate, entityID := kernel.NewID(), kernel.NewID(), kernel.NewID()
	// The contest trail on the owner's assessment: a candidate's raise and the
	// owner's resolve — BOTH owned by the employer (CAL-153) so the owner sees
	// the full trail in their scoped read.
	raise, _ := audit.NewAuditEntry(candidate, audit.ActionContestRaised, "contest", entityID, "", "", time.Unix(1, 0))
	raise.OwnerID = owner
	resolve, _ := audit.NewAuditEntry(owner, audit.ActionContestResolved, "contest", entityID, "", "", time.Unix(2, 0))
	resolve.OwnerID = owner
	require.NoError(t, repo.Append(context.Background(), raise))
	require.NoError(t, repo.Append(context.Background(), resolve))

	srv := NewAuditServer(repo)
	req := &caliberv1.ListAuditLogRequest{
		Entity: "contest", EntityId: entityID.String(),
		Page: &caliberv1.PageRequest{Page: 1, PageSize: 10},
	}

	// The owning employer sees the full trail (raise + resolve), newest first.
	ownerCtx := context.WithValue(context.Background(), principalKey{},
		app.Principal{UserID: owner, Role: identity.RoleEmployer.String()})
	resp, err := srv.ListAuditLog(ownerCtx, req)
	require.NoError(t, err)
	require.Len(t, resp.GetEntries(), 2)
	assert.Equal(t, audit.ActionContestResolved, resp.GetEntries()[0].GetAction(), "newest first")
	assert.Equal(t, int64(2), resp.GetPage().GetTotalItems())

	// CAL-153: a DIFFERENT reviewer sharing the subject sees none of it — the
	// cross-tenant read is closed.
	otherCtx := context.WithValue(context.Background(), principalKey{},
		app.Principal{UserID: kernel.NewID(), Role: identity.RoleRecruiter.String()})
	other, err := srv.ListAuditLog(otherCtx, req)
	require.NoError(t, err)
	assert.Empty(t, other.GetEntries(), "a non-owning reviewer cannot read another employer's trail")
	assert.Equal(t, int64(0), other.GetPage().GetTotalItems())

	// A platform admin reads the whole trail (unscoped).
	adminCtx := context.WithValue(context.Background(), principalKey{},
		app.Principal{UserID: kernel.NewID(), Role: identity.RoleAdmin.String()})
	adm, err := srv.ListAuditLog(adminCtx, req)
	require.NoError(t, err)
	assert.Len(t, adm.GetEntries(), 2, "an admin reads the full trail unscoped")
}

func TestAuditServer_AuthzAndValidation(t *testing.T) {
	srv := NewAuditServer(memory.NewAuditRepo())
	req := &caliberv1.ListAuditLogRequest{Entity: "contest", EntityId: kernel.NewID().String()}

	// candidate may not read the audit trail
	candCtx := context.WithValue(context.Background(), principalKey{},
		app.Principal{UserID: kernel.NewID(), Role: identity.RoleCandidate.String()})
	_, err := srv.ListAuditLog(candCtx, req)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	// unauthenticated -> unauthenticated
	_, err = srv.ListAuditLog(context.Background(), req)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))

	// employer with missing filters -> invalid argument
	empCtx := context.WithValue(context.Background(), principalKey{},
		app.Principal{UserID: kernel.NewID(), Role: identity.RoleEmployer.String()})
	_, err = srv.ListAuditLog(empCtx, &caliberv1.ListAuditLogRequest{Entity: "contest"})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestAuditServer_ExportAuditReport_JSON(t *testing.T) {
	repo := memory.NewAuditRepo()
	actor := kernel.NewID()
	owner := kernel.NewID()
	entityID := kernel.NewID()
	e1, _ := audit.NewAuditEntry(actor, audit.ActionApproveRejection, "match", entityID, `{"old":1}`, `{"new":2}`, time.Unix(1, 0))
	e1.OwnerID = owner
	e2, _ := audit.NewAuditEntry(actor, audit.ActionAgentSubmit, "application", kernel.NewID(), "", "", time.Unix(2, 0))
	e2.OwnerID = owner
	require.NoError(t, repo.Append(context.Background(), e1))
	require.NoError(t, repo.Append(context.Background(), e2))

	srv := NewAuditServer(repo)
	ctx := context.WithValue(context.Background(), principalKey{},
		app.Principal{UserID: owner, Role: identity.RoleEmployer.String()})

	resp, err := srv.ExportAuditReport(ctx, &caliberv1.ExportAuditReportRequest{
		StartTime: timestamppb.New(time.Unix(0, 0)),
		EndTime:   timestamppb.New(time.Unix(3, 0)),
		Format:    caliberv1.ExportFormat_EXPORT_FORMAT_JSON,
	})
	require.NoError(t, err)
	assert.Equal(t, "application/json", resp.GetContentType())
	assert.True(t, strings.HasSuffix(resp.GetFilename(), ".json"))

	var rows []map[string]any
	require.NoError(t, json.Unmarshal(resp.GetPayload(), &rows))
	require.Len(t, rows, 2)
	assert.Equal(t, audit.ActionAgentSubmit, rows[0]["action"])
	assert.Equal(t, audit.ActionApproveRejection, rows[1]["action"])
	assert.Equal(t, `{"new":2}`, rows[1]["after_json"])
}

func TestAuditServer_ExportAuditReport_CSV(t *testing.T) {
	repo := memory.NewAuditRepo()
	actor := kernel.NewID()
	owner := kernel.NewID()
	e1, _ := audit.NewAuditEntry(actor, audit.ActionOverrideScore, "match", kernel.NewID(), "", "", time.Unix(5, 0))
	e1.OwnerID = owner
	require.NoError(t, repo.Append(context.Background(), e1))

	srv := NewAuditServer(repo)
	ctx := context.WithValue(context.Background(), principalKey{},
		app.Principal{UserID: owner, Role: identity.RoleRecruiter.String()})

	resp, err := srv.ExportAuditReport(ctx, &caliberv1.ExportAuditReportRequest{
		StartTime: timestamppb.New(time.Unix(0, 0)),
		EndTime:   timestamppb.New(time.Unix(10, 0)),
		Actions:   []string{audit.ActionOverrideScore},
		Entities:  []string{"match"},
		Format:    caliberv1.ExportFormat_EXPORT_FORMAT_CSV,
	})
	require.NoError(t, err)
	assert.Equal(t, "text/csv", resp.GetContentType())

	records, err := csv.NewReader(strings.NewReader(string(resp.GetPayload()))).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 2)
	assert.Equal(t, "id", records[0][0])
	assert.Equal(t, audit.ActionOverrideScore, records[1][2])
}

// TestAuditServer_ExportAuditReport_CrossTenantIsolation is the CAL-153 export
// counterpart of the ListAuditLog scoping test: two employers act on a shared
// subject, and each may export only their own decisions — never the other's.
// A platform admin exports the whole trail unscoped.
func TestAuditServer_ExportAuditReport_CrossTenantIsolation(t *testing.T) {
	repo := memory.NewAuditRepo()
	actor := kernel.NewID()
	ownerA, ownerB, subj := kernel.NewID(), kernel.NewID(), kernel.NewID()

	ea, _ := audit.NewAuditEntry(actor, audit.ActionApproveRejection, "match", subj, "", "", time.Unix(1, 0))
	ea.OwnerID = ownerA
	eb, _ := audit.NewAuditEntry(actor, audit.ActionOverrideScore, "match", subj, "", "", time.Unix(2, 0))
	eb.OwnerID = ownerB
	require.NoError(t, repo.Append(context.Background(), ea))
	require.NoError(t, repo.Append(context.Background(), eb))

	srv := NewAuditServer(repo)
	req := &caliberv1.ExportAuditReportRequest{
		StartTime: timestamppb.New(time.Unix(0, 0)),
		EndTime:   timestamppb.New(time.Unix(3, 0)),
		Format:    caliberv1.ExportFormat_EXPORT_FORMAT_JSON,
	}
	exportFor := func(t *testing.T, uid kernel.ID, role string) []map[string]any {
		t.Helper()
		ctx := context.WithValue(context.Background(), principalKey{},
			app.Principal{UserID: uid, Role: role})
		resp, err := srv.ExportAuditReport(ctx, req)
		require.NoError(t, err)
		var rows []map[string]any
		require.NoError(t, json.Unmarshal(resp.GetPayload(), &rows))
		return rows
	}

	// Employer A exports only A's entry — not B's decision on the shared subject.
	rowsA := exportFor(t, ownerA, identity.RoleEmployer.String())
	require.Len(t, rowsA, 1)
	assert.Equal(t, audit.ActionApproveRejection, rowsA[0]["action"])

	// Employer B exports only B's entry.
	rowsB := exportFor(t, ownerB, identity.RoleRecruiter.String())
	require.Len(t, rowsB, 1)
	assert.Equal(t, audit.ActionOverrideScore, rowsB[0]["action"])

	// A platform admin exports the whole trail unscoped.
	rowsAdmin := exportFor(t, kernel.NewID(), identity.RoleAdmin.String())
	assert.Len(t, rowsAdmin, 2, "an admin exports the full trail unscoped")
}

func TestAuditServer_ExportAuditReport_ValidationAndAuthz(t *testing.T) {
	srv := NewAuditServer(memory.NewAuditRepo())
	base := &caliberv1.ExportAuditReportRequest{
		StartTime: timestamppb.New(time.Unix(2, 0)),
		EndTime:   timestamppb.New(time.Unix(1, 0)),
		Format:    caliberv1.ExportFormat_EXPORT_FORMAT_JSON,
	}

	// candidate is forbidden.
	candCtx := context.WithValue(context.Background(), principalKey{},
		app.Principal{UserID: kernel.NewID(), Role: identity.RoleCandidate.String()})
	_, err := srv.ExportAuditReport(candCtx, base)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	// unauthenticated.
	_, err = srv.ExportAuditReport(context.Background(), base)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))

	// employer: invalid date range.
	empCtx := context.WithValue(context.Background(), principalKey{},
		app.Principal{UserID: kernel.NewID(), Role: identity.RoleEmployer.String()})
	_, err = srv.ExportAuditReport(empCtx, base)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	// missing format.
	_, err = srv.ExportAuditReport(empCtx, &caliberv1.ExportAuditReportRequest{
		StartTime: timestamppb.New(time.Unix(0, 0)),
		EndTime:   timestamppb.New(time.Unix(1, 0)),
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	// missing timestamps.
	_, err = srv.ExportAuditReport(empCtx, &caliberv1.ExportAuditReportRequest{
		Format: caliberv1.ExportFormat_EXPORT_FORMAT_JSON,
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}
