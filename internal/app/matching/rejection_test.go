package matching_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	matchingapp "github.com/xcreativs/caliber/internal/app/matching"
	"github.com/xcreativs/caliber/internal/domain/audit"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/mocks"
)

func fixedClock() time.Time { return time.Unix(1700000000, 0) }

func recorderDeps(t *testing.T) (*matchingapp.RejectionRecorder, *mocks.MockAuditRepository) {
	t.Helper()
	ctrl := gomock.NewController(t)
	auditRepo := mocks.NewMockAuditRepository(ctrl)
	return matchingapp.NewRejectionRecorder(auditRepo, fixedClock), auditRepo
}

func TestRecord_LogsHumanApprovedRejection(t *testing.T) {
	rec, auditRepo := recorderDeps(t)
	actor, roleID, candidateID := kernel.NewID(), kernel.NewID(), kernel.NewID()

	auditRepo.EXPECT().Append(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, e *audit.AuditEntry) error {
			assert.Equal(t, audit.ActionApproveRejection, e.Action)
			assert.Equal(t, actor, e.ActorUserID, "the approving human is the actor")
			assert.Equal(t, "match", e.Entity)
			assert.Equal(t, candidateID, e.EntityID)
			assert.Contains(t, e.AfterJSON, "below the seniority bar", "the reason is captured for explainability")
			return nil
		})

	id, err := rec.Record(context.Background(), actor, roleID, candidateID, "below the seniority bar", true)
	require.NoError(t, err)
	assert.False(t, id.IsZero(), "returns the id of the logged approval")
}

func TestRecord_RefusesWithoutHumanApproval(t *testing.T) {
	rec, auditRepo := recorderDeps(t)
	// No Append is expected: an unapproved rejection must never be logged.
	auditRepo.EXPECT().Append(gomock.Any(), gomock.Any()).Times(0)

	_, err := rec.Record(context.Background(), kernel.NewID(), kernel.NewID(), kernel.NewID(), "reason", false)
	require.Error(t, err)
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestRecord_RefusesEmptyReason(t *testing.T) {
	rec, auditRepo := recorderDeps(t)
	auditRepo.EXPECT().Append(gomock.Any(), gomock.Any()).Times(0)

	_, err := rec.Record(context.Background(), kernel.NewID(), kernel.NewID(), kernel.NewID(), "   ", true)
	require.Error(t, err)
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestRecord_FailsWhenAuditLogUnavailable(t *testing.T) {
	rec, auditRepo := recorderDeps(t)
	// The log IS the approval: if it cannot be written, the rejection must not
	// stand and the call fails (no silent, unlogged rejection).
	auditRepo.EXPECT().Append(gomock.Any(), gomock.Any()).Return(errors.New("audit store down"))

	_, err := rec.Record(context.Background(), kernel.NewID(), kernel.NewID(), kernel.NewID(), "reason", true)
	require.Error(t, err)
}
