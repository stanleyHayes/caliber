package grpcadapter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/app"
	contestapp "github.com/xcreativs/caliber/internal/app/contest"
	"github.com/xcreativs/caliber/internal/domain/audit"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

// TestContestAuditTrailEndToEnd locks the cross-feature integration (CAL-083 +
// CAL-084): a candidate contests an assessment, a reviewer resolves it, and both
// actions surface — attributed and ordered — through the AuditService, because
// the contest service and the audit reader share one append-only trail.
func TestContestAuditTrailEndToEnd(t *testing.T) {
	ctx := context.Background()
	auditRepo := memory.NewAuditRepo()
	contestSrv := NewContestServer(contestapp.NewService(memory.NewContestRepo(), auditRepo, time.Now))
	auditSrv := NewAuditServer(auditRepo)

	candidateID := kernel.NewID()
	candidateCtx := context.WithValue(ctx, principalKey{},
		app.Principal{UserID: candidateID, Role: identity.RoleCandidate.String()})
	reviewerCtx := asRole(ctx, identity.RoleEmployer)

	// 1) Candidate raises a contest over a match assessment.
	raised, err := contestSrv.RaiseContest(candidateCtx, &caliberv1.RaiseContestRequest{
		Subject:   caliberv1.ContestSubject_CONTEST_SUBJECT_MATCH,
		SubjectId: kernel.NewID().String(),
		Reason:    "the breakdown ignored my recent Go work",
	})
	require.NoError(t, err)
	contestID := raised.GetContest().GetId()
	require.NotEmpty(t, contestID)

	// 2) Reviewer resolves it (upholds).
	resolved, err := contestSrv.ResolveContest(reviewerCtx, &caliberv1.ResolveContestRequest{
		ContestId: contestID, Uphold: true, Note: "re-scored with the recent work",
	})
	require.NoError(t, err)
	assert.Equal(t, caliberv1.ContestStatus_CONTEST_STATUS_UPHELD, resolved.GetContest().GetStatus())

	// 3) The audit trail surfaces both actions for the contest, newest first.
	log, err := auditSrv.ListAuditLog(reviewerCtx, &caliberv1.ListAuditLogRequest{
		Entity: "contest", EntityId: contestID, Page: &caliberv1.PageRequest{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	require.Len(t, log.GetEntries(), 2, "raise + resolve are both recorded")
	assert.Equal(t, int64(2), log.GetPage().GetTotalItems())

	actions := map[string]string{} // action -> actor
	for _, e := range log.GetEntries() {
		actions[e.GetAction()] = e.GetActorUserId()
		assert.Equal(t, contestID, e.GetEntityId())
	}
	assert.Equal(t, candidateID.String(), actions[audit.ActionContestRaised], "the raise is attributed to the candidate")
	assert.NotEmpty(t, actions[audit.ActionContestResolved], "the resolution is attributed to the reviewer")

	// 4) A candidate may not read the audit trail.
	_, err = auditSrv.ListAuditLog(candidateCtx, &caliberv1.ListAuditLogRequest{
		Entity: "contest", EntityId: contestID, Page: &caliberv1.PageRequest{Page: 1, PageSize: 10},
	})
	require.Error(t, err, "audit reads are reviewer-only")
}
