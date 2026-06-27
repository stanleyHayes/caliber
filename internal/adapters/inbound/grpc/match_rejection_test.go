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
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

func rejectionServer(auditRepo audit.AuditRepository) *MatchServer {
	return NewMatchServer(nil, nil, matchingapp.NewRejectionRecorder(auditRepo, time.Now))
}

func TestRecordRejection_LogsApprovalAndIsAuditable(t *testing.T) {
	auditRepo := memory.NewAuditRepo()
	srv := rejectionServer(auditRepo)
	employer, candidateID, roleID := kernel.NewID(), kernel.NewID(), kernel.NewID()
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

func TestRecordRejection_RejectsNonReviewer(t *testing.T) {
	srv := rejectionServer(memory.NewAuditRepo())
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
	srv := rejectionServer(auditRepo)
	candidateID := kernel.NewID()
	ctx := asRole(context.Background(), identity.RoleEmployer)

	// human_approved=false is the AI-driven path the platform forbids.
	_, err := srv.RecordRejection(ctx, &caliberv1.RecordRejectionRequest{
		RoleId: kernel.NewID().String(), CandidateId: candidateID.String(), Reason: "r", HumanApproved: false,
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	// A missing reason is also refused — a decline must be explainable.
	_, err = srv.RecordRejection(ctx, &caliberv1.RecordRejectionRequest{
		RoleId: kernel.NewID().String(), CandidateId: candidateID.String(), Reason: "  ", HumanApproved: true,
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	// Nothing was logged: no rejection stands without a valid human approval.
	_, total, err := auditRepo.List(ctx, "match", candidateID, kernel.Page{Number: 1, Size: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
}
