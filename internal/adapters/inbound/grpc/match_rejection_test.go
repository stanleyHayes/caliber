package grpcadapter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/app"
	matchingapp "github.com/xcreativs/caliber/internal/app/matching"
	"github.com/xcreativs/caliber/internal/domain/audit"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

// rejectionServer builds a MatchServer whose recorder reads roles from roleRepo
// and writes to auditRepo. RBAC-only tests can pass an empty role repo because
// the role is loaded only after the reviewer check passes.
func rejectionServer(roleRepo role.RoleRepository, auditRepo audit.AuditRepository) *MatchServer {
	return NewMatchServer(nil, nil, matchingapp.NewRejectionRecorder(roleRepo, auditRepo, time.Now))
}

// seedOwnedRole stores a role owned by owner and returns its id, so a rejection
// against it passes the CAL-116 ownership guard.
func seedOwnedRole(t *testing.T, roleRepo *memory.RoleRepo, owner kernel.ID) kernel.ID {
	t.Helper()
	rl, err := role.NewRole(owner,
		role.RoleSpec{Title: "Backend Engineer", Seniority: role.SeniorityMid},
		role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 1, MustHave: true}}},
		time.Unix(1, 0))
	require.NoError(t, err)
	require.NoError(t, roleRepo.Create(context.Background(), rl))
	return rl.ID
}

func TestRecordRejection_LogsApprovalAndIsAuditable(t *testing.T) {
	auditRepo := memory.NewAuditRepo()
	roleRepo := memory.NewRoleRepo()
	srv := rejectionServer(roleRepo, auditRepo)
	employer, candidateID := kernel.NewID(), kernel.NewID()
	roleID := seedOwnedRole(t, roleRepo, employer)
	ctx := context.WithValue(context.Background(), principalKey{},
		app.Principal{UserID: employer, Role: identity.RoleEmployer.String()})

	resp, err := srv.RecordRejection(ctx, &caliberv1.RecordRejectionRequest{
		RoleId:        roleID.String(),
		CandidateId:   candidateID.String(),
		Reason:        "strong, but the role needs deeper distributed-systems depth",
		HumanApproved: true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.GetAuditEntryId())

	// The decline is now discoverable in the audit trail, attributed to the human.
	entries, total, err := auditRepo.List(ctx, "match", candidateID, kernel.Page{Number: 1, Size: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, entries, 1)
	assert.Equal(t, audit.ActionApproveRejection, entries[0].Action)
	assert.Equal(t, employer, entries[0].ActorUserID)
}

// TestRecordRejection_RejectsOtherEmployer is the Flow A IDOR guard (CAL-116): a
// reviewer who passes the RBAC check but does not own the role is denied, and no
// rejection is logged.
func TestRecordRejection_RejectsOtherEmployer(t *testing.T) {
	auditRepo := memory.NewAuditRepo()
	roleRepo := memory.NewRoleRepo()
	srv := rejectionServer(roleRepo, auditRepo)
	candidateID := kernel.NewID()
	// The role belongs to a different employer than the caller.
	roleID := seedOwnedRole(t, roleRepo, kernel.NewID())

	_, err := srv.RecordRejection(asEmployer(context.Background(), kernel.NewID()), &caliberv1.RecordRejectionRequest{
		RoleId: roleID.String(), CandidateId: candidateID.String(), Reason: "r", HumanApproved: true,
	})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	_, total, err := auditRepo.List(context.Background(), "match", candidateID, kernel.Page{Number: 1, Size: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total, "no rejection is logged for a role you do not own")
}

func TestRecordRejection_RejectsNonReviewer(t *testing.T) {
	srv := rejectionServer(memory.NewRoleRepo(), memory.NewAuditRepo())
	req := &caliberv1.RecordRejectionRequest{
		RoleId: kernel.NewID().String(), CandidateId: kernel.NewID().String(), Reason: "r", HumanApproved: true,
	}

	// A candidate cannot reject other candidates.
	_, err := srv.RecordRejection(asRole(context.Background(), identity.RoleCandidate), req)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	// Unauthenticated callers cannot reject anyone.
	_, err = srv.RecordRejection(context.Background(), req)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestRecordRejection_RequiresExplicitHumanApproval(t *testing.T) {
	auditRepo := memory.NewAuditRepo()
	roleRepo := memory.NewRoleRepo()
	srv := rejectionServer(roleRepo, auditRepo)
	employer, candidateID := kernel.NewID(), kernel.NewID()
	roleID := seedOwnedRole(t, roleRepo, employer)
	ctx := asEmployer(context.Background(), employer)

	// human_approved=false is the AI-driven path the platform forbids.
	_, err := srv.RecordRejection(ctx, &caliberv1.RecordRejectionRequest{
		RoleId: roleID.String(), CandidateId: candidateID.String(), Reason: "r", HumanApproved: false,
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	// A missing reason is also refused — a decline must be explainable.
	_, err = srv.RecordRejection(ctx, &caliberv1.RecordRejectionRequest{
		RoleId: roleID.String(), CandidateId: candidateID.String(), Reason: "  ", HumanApproved: true,
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	// Nothing was logged: no rejection stands without a valid human approval.
	_, total, err := auditRepo.List(ctx, "match", candidateID, kernel.Page{Number: 1, Size: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
}
