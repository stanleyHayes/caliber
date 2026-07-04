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
	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	matchingdom "github.com/xcreativs/caliber/internal/domain/matching"
	roledom "github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/mocks"
)

func clock() time.Time { return time.Unix(1700000000, 0) }

// cmocks bundles the contest service's collaborators so Resolve tests can set
// up the ownership-resolution path (match/interview -> role -> employer).
type cmocks struct {
	contests   *mocks.MockContestRepository
	audit      *mocks.MockAuditRepository
	matches    *mocks.MockMatchRepository
	interviews *mocks.MockInterviewRepository
	roles      *mocks.MockRoleRepository
}

func build(t *testing.T) (*contestapp.Service, cmocks) {
	t.Helper()
	ctrl := gomock.NewController(t)
	m := cmocks{
		contests:   mocks.NewMockContestRepository(ctrl),
		audit:      mocks.NewMockAuditRepository(ctrl),
		matches:    mocks.NewMockMatchRepository(ctrl),
		interviews: mocks.NewMockInterviewRepository(ctrl),
		roles:      mocks.NewMockRoleRepository(ctrl),
	}
	svc := contestapp.NewService(m.contests, m.audit, m.matches, m.interviews, m.roles, clock)
	return svc, m
}

// deps is the slim helper for tests that never reach the ownership path
// (List and contest-not-found), returning just the contest mock.
func deps(t *testing.T) (*contestapp.Service, *mocks.MockContestRepository) {
	t.Helper()
	svc, m := build(t)
	return svc, m.contests
}

// expectOwner wires the match->role->employer resolution so ownerOf returns
// owner as the contested match's owning employer.
func expectOwner(m cmocks, owner kernel.ID) {
	roleID := kernel.NewID()
	m.matches.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(&matchingdom.Match{RoleID: roleID}, nil)
	m.roles.EXPECT().ByID(gomock.Any(), roleID).Return(&roledom.Role{EmployerID: owner}, nil)
}

// expectSubjectOwnedBy wires a Raise's subject resolution: the contested match
// belongs to candidateID (passing the candidate-side ownership check) and its
// role to owner (the audit-scope employer).
func expectSubjectOwnedBy(m cmocks, candidateID, owner kernel.ID) {
	roleID := kernel.NewID()
	m.matches.EXPECT().ByID(gomock.Any(), gomock.Any()).
		Return(&matchingdom.Match{RoleID: roleID, CandidateID: candidateID}, nil)
	m.roles.EXPECT().ByID(gomock.Any(), roleID).Return(&roledom.Role{EmployerID: owner}, nil)
}

func TestRaise_CreatesAndAudits(t *testing.T) {
	svc, m := build(t)
	cid, sid := kernel.NewID(), kernel.NewID()
	owner := kernel.NewID()

	m.contests.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	// The contested match belongs to the raiser (cid) and its role to owner.
	expectSubjectOwnedBy(m, cid, owner)
	m.audit.EXPECT().Append(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, e *audit.AuditEntry) error {
			assert.Equal(t, audit.ActionContestRaised, e.Action)
			assert.Equal(t, cid, e.ActorUserID)
			assert.Equal(t, "contest", e.Entity)
			// CAL-153: the raise is owned by the employer, not the candidate, so
			// it joins the owner's scoped audit trail.
			assert.Equal(t, owner, e.OwnerID)
			return nil
		})

	c, err := svc.Raise(context.Background(), cid, sid, contestdom.SubjectMatch, "  the breakdown missed my Go work  ")
	require.NoError(t, err)
	assert.Equal(t, contestdom.StatusOpen, c.Status)
	assert.Equal(t, "the breakdown missed my Go work", c.Reason)
}

func TestRaise_RejectsInvalidBeforePersisting(t *testing.T) {
	svc, _ := deps(t) // no Create/Append expectations: nothing must be persisted
	_, err := svc.Raise(context.Background(), kernel.NewID(), kernel.NewID(), contestdom.SubjectMatch, "   ")
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestListForCandidate(t *testing.T) {
	svc, contests := deps(t)
	cid := kernel.NewID()
	want := []*contestdom.Contest{{ID: kernel.NewID(), CandidateID: cid}}
	contests.EXPECT().ByCandidate(gomock.Any(), cid, gomock.Any()).Return(want, int64(1), nil)

	got, total, err := svc.ListForCandidate(context.Background(), cid, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, want, got)
}

func TestListForSubject(t *testing.T) {
	svc, contests := deps(t)
	sid := kernel.NewID()
	want := []*contestdom.Contest{{ID: kernel.NewID(), SubjectID: sid}}
	contests.EXPECT().BySubject(gomock.Any(), sid, gomock.Any()).Return(want, int64(1), nil)

	got, total, err := svc.ListForSubject(context.Background(), sid, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, want, got)
}

func TestRaise_PropagatesCreateFailure(t *testing.T) {
	svc, m := build(t) // no Append/roles: a failed create logs no "raised" audit
	cand := kernel.NewID()
	// The subject belongs to the raiser, so the ownership check passes and Create runs.
	m.matches.EXPECT().ByID(gomock.Any(), gomock.Any()).
		Return(&matchingdom.Match{RoleID: kernel.NewID(), CandidateID: cand}, nil)
	m.contests.EXPECT().Create(gomock.Any(), gomock.Any()).Return(kernel.Conflict("dup"))

	_, err := svc.Raise(context.Background(), cand, kernel.NewID(), contestdom.SubjectMatch, "a real reason")
	assert.Equal(t, kernel.KindConflict, kernel.KindOf(err))
}

// TestRaise_RejectsForeignSubject is the CAL-153 candidate-side IDOR guard: a
// candidate cannot raise a contest over an assessment that belongs to someone
// else — the raise is rejected with Forbidden before any Create or audit write.
func TestRaise_RejectsForeignSubject(t *testing.T) {
	svc, m := build(t) // no Create/Append: nothing is persisted for a foreign subject
	m.matches.EXPECT().ByID(gomock.Any(), gomock.Any()).
		Return(&matchingdom.Match{RoleID: kernel.NewID(), CandidateID: kernel.NewID()}, nil)

	_, err := svc.Raise(context.Background(), kernel.NewID(), kernel.NewID(), contestdom.SubjectMatch, "a real reason")
	assert.Equal(t, kernel.KindForbidden, kernel.KindOf(err))
}

// TestRaise_SubjectNotFound: a contest against an assessment that does not exist
// surfaces NotFound and persists nothing (validates subject_id is a real assessment).
func TestRaise_SubjectNotFound(t *testing.T) {
	svc, m := build(t)
	m.matches.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(nil, kernel.NotFound("gone"))

	_, err := svc.Raise(context.Background(), kernel.NewID(), kernel.NewID(), contestdom.SubjectMatch, "a real reason")
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
}

func TestResolve_PropagatesUpdateFailure(t *testing.T) {
	svc, m := build(t) // no Append: a failed update logs no "resolved" audit
	reviewer := kernel.NewID()
	open, err := contestdom.NewContest(kernel.NewID(), kernel.NewID(), contestdom.SubjectMatch, "missed evidence", clock())
	require.NoError(t, err)
	m.contests.EXPECT().ByID(gomock.Any(), open.ID).Return(open, nil)
	expectOwner(m, reviewer)
	m.contests.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errors.New("db down"))

	_, err = svc.Resolve(context.Background(), reviewer, open.ID, true, "agreed")
	require.Error(t, err)
}

func TestResolve_UpholdLoadsUpdatesAudits(t *testing.T) {
	svc, m := build(t)
	reviewer := kernel.NewID()
	open, err := contestdom.NewContest(kernel.NewID(), kernel.NewID(), contestdom.SubjectMatch, "missed evidence", clock())
	require.NoError(t, err)

	m.contests.EXPECT().ByID(gomock.Any(), open.ID).Return(open, nil)
	expectOwner(m, reviewer)
	m.contests.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	m.audit.EXPECT().Append(gomock.Any(), gomock.Any()).DoAndReturn(
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
	svc, m := build(t)
	reviewer := kernel.NewID()
	done, _ := contestdom.NewContest(kernel.NewID(), kernel.NewID(), contestdom.SubjectMatch, "reason", clock())
	require.NoError(t, done.Resolve(false, "dismissed", clock()))
	// Owner matches, but the contest is already resolved: Update/Append must NOT run.
	m.contests.EXPECT().ByID(gomock.Any(), done.ID).Return(done, nil)
	expectOwner(m, reviewer)

	_, err := svc.Resolve(context.Background(), reviewer, done.ID, true, "again")
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestResolve_NotFound(t *testing.T) {
	svc, contests := deps(t)
	// Contest lookup fails before ownership resolution — nothing else is consulted.
	contests.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(nil, kernel.NotFound("nope"))
	_, err := svc.Resolve(context.Background(), kernel.NewID(), kernel.NewID(), true, "x")
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
}

// TestResolve_NonOwnerForbidden is the CAL-153 cross-tenant IDOR guard: a
// reviewer from a different employer cannot resolve a contest over an assessment
// they do not own. The contest is loaded and its owner resolved, then the resolve
// is rejected with Forbidden before any state change (no Update, no audit).
func TestResolve_NonOwnerForbidden(t *testing.T) {
	svc, m := build(t)
	open, err := contestdom.NewContest(kernel.NewID(), kernel.NewID(), contestdom.SubjectMatch, "missed evidence", clock())
	require.NoError(t, err)
	m.contests.EXPECT().ByID(gomock.Any(), open.ID).Return(open, nil)
	expectOwner(m, kernel.NewID()) // owned by some other employer, not the caller

	_, err = svc.Resolve(context.Background(), kernel.NewID(), open.ID, true, "agreed")
	assert.Equal(t, kernel.KindForbidden, kernel.KindOf(err))
}

// TestResolve_ReportCardResolvesViaInterview covers the report-card subject:
// ownership resolves through the interview -> role -> employer chain.
func TestResolve_ReportCardResolvesViaInterview(t *testing.T) {
	svc, m := build(t)
	reviewer, roleID := kernel.NewID(), kernel.NewID()
	open, err := contestdom.NewContest(kernel.NewID(), kernel.NewID(), contestdom.SubjectReportCard, "evidence misquoted", clock())
	require.NoError(t, err)
	m.contests.EXPECT().ByID(gomock.Any(), open.ID).Return(open, nil)
	m.interviews.EXPECT().ByID(gomock.Any(), open.SubjectID).Return(&interviewdom.Interview{RoleID: roleID}, nil)
	m.roles.EXPECT().ByID(gomock.Any(), roleID).Return(&roledom.Role{EmployerID: reviewer}, nil)
	m.contests.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	m.audit.EXPECT().Append(gomock.Any(), gomock.Any()).Return(nil)

	resolved, err := svc.Resolve(context.Background(), reviewer, open.ID, false, "dismissed after review")
	require.NoError(t, err)
	assert.Equal(t, contestdom.StatusDismissed, resolved.Status)
}

// TestResolve_SubjectNotFoundPropagates: a contest whose assessment no longer
// exists cannot be resolved — the missing subject surfaces NotFound and nothing
// is mutated (this also validates subject_id references a real assessment).
func TestResolve_SubjectNotFoundPropagates(t *testing.T) {
	svc, m := build(t)
	open, err := contestdom.NewContest(kernel.NewID(), kernel.NewID(), contestdom.SubjectMatch, "missed evidence", clock())
	require.NoError(t, err)
	m.contests.EXPECT().ByID(gomock.Any(), open.ID).Return(open, nil)
	m.matches.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(nil, kernel.NotFound("gone"))

	_, err = svc.Resolve(context.Background(), kernel.NewID(), open.ID, true, "x")
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
}
