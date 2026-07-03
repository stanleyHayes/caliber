package dashboard_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	dashboardapp "github.com/xcreativs/caliber/internal/app/dashboard"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	matchingdom "github.com/xcreativs/caliber/internal/domain/matching"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
	"github.com/xcreativs/caliber/internal/mocks"
)

type deps struct {
	candidates *mocks.MockCandidateRepository
	profiles   *mocks.MockTalentProfileRepository
	users      *mocks.MockUserRepository
	roles      *mocks.MockRoleRepository
	matches    *mocks.MockMatchRepository
}

func newDeps(ctrl *gomock.Controller) deps {
	return deps{
		candidates: mocks.NewMockCandidateRepository(ctrl),
		profiles:   mocks.NewMockTalentProfileRepository(ctrl),
		users:      mocks.NewMockUserRepository(ctrl),
		roles:      mocks.NewMockRoleRepository(ctrl),
		matches:    mocks.NewMockMatchRepository(ctrl),
	}
}

func (d deps) agg() *dashboardapp.Aggregator {
	return dashboardapp.NewAggregator(d.candidates, d.profiles, d.users, d.roles)
}

func (d deps) aggWithMatches() *dashboardapp.Aggregator {
	return dashboardapp.NewAggregator(d.candidates, d.profiles, d.users, d.roles, d.matches)
}

func TestPoolEnrichesCandidates(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	cand, err := talent.NewCandidate(kernel.NewID(), "Accra", talent.CandidateIntake{})
	require.NoError(t, err)
	email, err := identity.NewEmail("ama@example.com")
	require.NoError(t, err)
	user, err := identity.NewUser(email, identity.RoleCandidate, "Ama Mensah", "hash", time.Unix(1, 0))
	require.NoError(t, err)
	profile, err := talent.NewTalentProfile(cand.ID, "s",
		[]talent.ProfileCompetency{{Name: "Go", Level: 4, EvidenceQuote: "x"}, {Name: "SQL", Level: 5, EvidenceQuote: "y"}})
	require.NoError(t, err)
	profile.MarkScreened()

	d.candidates.EXPECT().List(gomock.Any(), gomock.Any()).Return([]*talent.Candidate{cand}, int64(1), nil)
	d.users.EXPECT().ByID(gomock.Any(), cand.UserID).Return(user, nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cand.ID).Return(profile, nil)

	pool, total, err := d.agg().Pool(context.Background(), kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, pool, 1)
	assert.Equal(t, "Ama Mensah", pool[0].Name)
	assert.Equal(t, talent.PassportScreened, pool[0].PassportStatus)
	assert.InDelta(t, 0.9, pool[0].HeadlineScore, 0.01, "mean level (4.5) / 5")
}

func TestPoolUsesRequestedPageAndReturnsTotal(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	page := kernel.NewPage(2, 2)
	cand, err := talent.NewCandidate(kernel.NewID(), "Kumasi", talent.CandidateIntake{})
	require.NoError(t, err)
	email, err := identity.NewEmail("kofi@example.com")
	require.NoError(t, err)
	user, err := identity.NewUser(email, identity.RoleCandidate, "Kofi Boateng", "hash", time.Unix(2, 0))
	require.NoError(t, err)
	profile, err := talent.NewTalentProfile(cand.ID, "platform engineer",
		[]talent.ProfileCompetency{{Name: "Kubernetes", Level: 5, EvidenceQuote: "ran clusters"}})
	require.NoError(t, err)

	d.candidates.EXPECT().List(gomock.Any(), page).Return([]*talent.Candidate{cand}, int64(3), nil)
	d.users.EXPECT().ByID(gomock.Any(), cand.UserID).Return(user, nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cand.ID).Return(profile, nil)

	pool, total, err := d.agg().Pool(context.Background(), page)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total, "total reflects the full pool, not just the current page")
	require.Len(t, pool, 1)
	assert.Equal(t, cand.ID, pool[0].CandidateID)
	assert.Equal(t, "Kofi Boateng", pool[0].Name)
}

func TestPoolToleratesMissingUserAndProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	cand, err := talent.NewCandidate(kernel.NewID(), "Accra", talent.CandidateIntake{})
	require.NoError(t, err)
	d.candidates.EXPECT().List(gomock.Any(), gomock.Any()).Return([]*talent.Candidate{cand}, int64(1), nil)
	d.users.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(nil, kernel.NotFound("no user"))
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), gomock.Any()).Return(nil, kernel.NotFound("no profile"))

	pool, _, err := d.agg().Pool(context.Background(), kernel.NewPage(1, 10))
	require.NoError(t, err)
	require.Len(t, pool, 1)
	assert.Empty(t, pool[0].Name)
	assert.Equal(t, talent.PassportUnset, pool[0].PassportStatus)
	assert.Zero(t, pool[0].HeadlineScore)
}

func TestSupplyDemandGroupsBySeniority(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	mid := openRole(t, role.SeniorityMid)
	senior := openRole(t, role.SenioritySenior)
	d.roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).Return([]*role.Role{mid, senior, openRole(t, role.SenioritySenior)}, int64(3), nil)
	d.candidates.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, int64(5), nil)

	items, err := d.agg().SupplyDemand(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 2)
	// sorted by family: "mid" then "senior"
	assert.Equal(t, "mid", items[0].RoleFamily)
	assert.Equal(t, 1, items[0].OpenRoles)
	assert.Equal(t, 5, items[0].AvailableCandidates)
	assert.Equal(t, "senior", items[1].RoleFamily)
	assert.Equal(t, 2, items[1].OpenRoles)
	assert.Equal(t, 2-5, items[1].Gap)
}

func TestSupplyDemandReconcilesEachFamilyAgainstPoolTotal(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	d.roles.EXPECT().ListOpen(gomock.Any(), kernel.NewPage(1, kernel.MaxPageSize)).Return([]*role.Role{
		openRole(t, role.SeniorityJunior),
		openRole(t, role.SeniorityMid),
		openRole(t, role.SeniorityMid),
	}, int64(3), nil)
	d.candidates.EXPECT().List(gomock.Any(), kernel.NewPage(1, 1)).Return(nil, int64(7), nil)

	items, err := d.agg().SupplyDemand(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, dashboardapp.SupplyDemandItem{
		RoleFamily:          "junior",
		OpenRoles:           1,
		AvailableCandidates: 7,
		Gap:                 -6,
	}, items[0])
	assert.Equal(t, dashboardapp.SupplyDemandItem{
		RoleFamily:          "mid",
		OpenRoles:           2,
		AvailableCandidates: 7,
		Gap:                 -5,
	}, items[1])
}

func TestTimeToShortlist(t *testing.T) {
	d := newDeps(gomock.NewController(t))
	m := d.agg().TimeToShortlist(context.Background())
	assert.InDelta(t, 504.0, m.BaselineHours, 0.01)
	assert.Positive(t, m.CurrentHours)
	assert.InDelta(t, m.BaselineHours/m.CurrentHours, m.ImprovementFactor, 0.01)
}

func TestTimeToShortlistComputesFromFirstRoleMatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	base := time.Unix(1700000000, 0)
	slow := openRole(t, role.SenioritySenior)
	slow.CreatedAt = base
	fast := openRole(t, role.SeniorityMid)
	fast.CreatedAt = base.Add(time.Hour)

	d.roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).Return([]*role.Role{slow, fast}, int64(2), nil)
	d.matches.EXPECT().ByRole(gomock.Any(), slow.ID, gomock.Any()).Return([]*matchingdom.Match{
		{RoleID: slow.ID, CandidateID: kernel.NewID(), CreatedAt: base.Add(4 * time.Hour)},
		{RoleID: slow.ID, CandidateID: kernel.NewID(), CreatedAt: base.Add(3 * time.Hour)},
	}, int64(2), nil)
	d.matches.EXPECT().ByRole(gomock.Any(), fast.ID, gomock.Any()).Return([]*matchingdom.Match{
		{RoleID: fast.ID, CandidateID: kernel.NewID(), CreatedAt: base.Add(2 * time.Hour)},
	}, int64(1), nil)

	m := d.aggWithMatches().TimeToShortlist(context.Background())
	assert.InDelta(t, 504.0, m.BaselineHours, 0.01)
	assert.InDelta(t, 2.0, m.CurrentHours, 0.01, "average of 3h and 1h role-to-first-match durations")
	assert.InDelta(t, 252.0, m.ImprovementFactor, 0.01)
}

func openRole(t *testing.T, seniority role.Seniority) *role.Role {
	t.Helper()
	r, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{Title: "Engineer", Seniority: seniority},
		role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 1}}}, time.Unix(1, 0))
	require.NoError(t, err)
	return r
}
