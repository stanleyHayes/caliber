package contest_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	contestapp "github.com/xcreativs/caliber/internal/app/contest"
	"github.com/xcreativs/caliber/internal/domain/audit"
	contestdom "github.com/xcreativs/caliber/internal/domain/contest"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/mocks"
)

func clock() time.Time { return time.Unix(1700000000, 0) }

func deps(t *testing.T) (*contestapp.Service, *mocks.MockContestRepository, *mocks.MockAuditRepository) {
	t.Helper()
	ctrl := gomock.NewController(t)
	contests := mocks.NewMockContestRepository(ctrl)
	auditRepo := mocks.NewMockAuditRepository(ctrl)
	return contestapp.NewService(contests, auditRepo, clock), contests, auditRepo
}

func TestRaise_CreatesAndAudits(t *testing.T) {
	svc, contests, auditRepo := deps(t)
	cid, sid := kernel.NewID(), kernel.NewID()

	contests.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	auditRepo.EXPECT().Append(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, e *audit.AuditEntry) error {
			assert.Equal(t, audit.ActionContestRaised, e.Action)
			assert.Equal(t, cid, e.ActorUserID)
			assert.Equal(t, "contest", e.Entity)
			return nil
		})

	c, err := svc.Raise(context.Background(), cid, sid, contestdom.SubjectMatch, "  the breakdown missed my Go work  ")
	require.NoError(t, err)
	assert.Equal(t, contestdom.StatusOpen, c.Status)
	assert.Equal(t, "the breakdown missed my Go work", c.Reason)
}

func TestRaise_RejectsInvalidBeforePersisting(t *testing.T) {
	svc, _, _ := deps(t) // no Create/Append expectations: nothing must be persisted
	_, err := svc.Raise(context.Background(), kernel.NewID(), kernel.NewID(), contestdom.SubjectMatch, "   ")
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestListForCandidate(t *testing.T) {
	svc, contests, _ := deps(t)
	cid := kernel.NewID()
	want := []*contestdom.Contest{{ID: kernel.NewID(), CandidateID: cid}}
	contests.EXPECT().ByCandidate(gomock.Any(), cid, gomock.Any()).Return(want, int64(1), nil)

	got, total, err := svc.ListForCandidate(context.Background(), cid, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, want, got)
}

func TestListForSubject(t *testing.T) {
	svc, contests, _ := deps(t)
	sid := kernel.NewID()
	want := []*contestdom.Contest{{ID: kernel.NewID(), SubjectID: sid}}
	contests.EXPECT().BySubject(gomock.Any(), sid, gomock.Any()).Return(want, int64(1), nil)

	got, total, err := svc.ListForSubject(context.Background(), sid, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, want, got)
}

func TestRaise_PropagatesCreateFailure(t *testing.T) {
	svc, contests, _ := deps(t) // no Append: a failed create logs no "raised" audit
	contests.EXPECT().Create(gomock.Any(), gomock.Any()).Return(kernel.Conflict("dup"))

	_, err := svc.Raise(context.Background(), kernel.NewID(), kernel.NewID(), contestdom.SubjectMatch, "a real reason")
	assert.Equal(t, kernel.KindConflict, kernel.KindOf(err))
}

func TestResolve_PropagatesUpdateFailure(t *testing.T) {
	svc, contests, _ := deps(t) // no Append: a failed update logs no "resolved" audit
	open, err := contestdom.NewContest(kernel.NewID(), kernel.NewID(), contestdom.SubjectMatch, "missed evidence", clock())
	require.NoError(t, err)
	contests.EXPECT().ByID(gomock.Any(), open.ID).Return(open, nil)
	contests.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errors.New("db down"))

	_, err = svc.Resolve(context.Background(), kernel.NewID(), open.ID, true, "agreed")
	require.Error(t, err)
}

func TestResolve_UpholdLoadsUpdatesAudits(t *testing.T) {
	svc, contests, auditRepo := deps(t)
	reviewer := kernel.NewID()
	open, err := contestdom.NewContest(kernel.NewID(), kernel.NewID(), contestdom.SubjectMatch, "missed evidence", clock())
	require.NoError(t, err)

	contests.EXPECT().ByID(gomock.Any(), open.ID).Return(open, nil)
	contests.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	auditRepo.EXPECT().Append(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, e *audit.AuditEntry) error {
			assert.Equal(t, audit.ActionContestResolved, e.Action)
			assert.Equal(t, reviewer, e.ActorUserID)
			return nil
		})

	resolved, err := svc.Resolve(context.Background(), reviewer, open.ID, true, "agreed; rescoring")
	require.NoError(t, err)
	assert.Equal(t, contestdom.StatusUpheld, resolved.Status)
	assert.Equal(t, "agreed; rescoring", resolved.Resolution)
}

func TestResolve_AlreadyResolvedDoesNotUpdate(t *testing.T) {
	svc, contests, _ := deps(t)
	done, _ := contestdom.NewContest(kernel.NewID(), kernel.NewID(), contestdom.SubjectMatch, "reason", clock())
	require.NoError(t, done.Resolve(false, "dismissed", clock()))
	// ByID returns an already-resolved contest; Update/Append must NOT be called.
	contests.EXPECT().ByID(gomock.Any(), done.ID).Return(done, nil)

	_, err := svc.Resolve(context.Background(), kernel.NewID(), done.ID, true, "again")
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestResolve_NotFound(t *testing.T) {
	svc, contests, _ := deps(t)
	contests.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(nil, kernel.NotFound("nope"))
	_, err := svc.Resolve(context.Background(), kernel.NewID(), kernel.NewID(), true, "x")
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
}
