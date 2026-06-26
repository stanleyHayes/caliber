package matching_test

import (
	"context"
	"testing"
	"time"

	matchingapp "github.com/xcreativs/caliber/internal/app/matching"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
	"github.com/xcreativs/caliber/internal/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func openRole(t *testing.T, title, location string, comps []role.Competency) *role.Role {
	t.Helper()
	rl, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{
			Title: title, Location: location, Seniority: role.SeniorityMid,
			Responsibilities: []string{"build services"},
		},
		role.Rubric{Competencies: comps},
		time.Unix(1700000000, 0))
	require.NoError(t, err)
	return rl
}

func goSQLProfile(t *testing.T, cid kernel.ID) *talent.TalentProfile {
	t.Helper()
	p, err := talent.NewTalentProfile(cid, "backend engineer", []talent.ProfileCompetency{
		{Name: "Go", Level: 5, EvidenceQuote: "built services"},
		{Name: "SQL", Level: 4, EvidenceQuote: "designed schemas"},
	})
	require.NoError(t, err)
	return p
}

type twoWayDeps struct {
	roles      *mocks.MockRoleRepository
	profiles   *mocks.MockTalentProfileRepository
	candidates *mocks.MockCandidateRepository
}

func newTwoWay(ctrl *gomock.Controller) (twoWayDeps, *matchingapp.PassiveMatcher) {
	d := twoWayDeps{
		roles:      mocks.NewMockRoleRepository(ctrl),
		profiles:   mocks.NewMockTalentProfileRepository(ctrl),
		candidates: mocks.NewMockCandidateRepository(ctrl),
	}
	return d, matchingapp.NewPassiveMatcher(d.roles, d.profiles, d.candidates)
}

func TestRolesForCandidate_RanksAndFilters(t *testing.T) {
	ctrl := gomock.NewController(t)
	d, matcher := newTwoWay(ctrl)

	cid := kernel.NewID()

	// A: fully covered, higher fit. D: fully covered, lower fit.
	// B: missing a must-have (Kubernetes). C: logistically incompatible (Lagos).
	roleA := openRole(t, "Backend Engineer", "Accra",
		[]role.Competency{{Name: "Go", Weight: 0.6, MustHave: true}, {Name: "SQL", Weight: 0.4}})
	roleD := openRole(t, "Data Engineer", "Accra",
		[]role.Competency{{Name: "Go", Weight: 0.3, MustHave: true}, {Name: "SQL", Weight: 0.7}})
	roleB := openRole(t, "Platform Engineer", "Accra",
		[]role.Competency{{Name: "Go", Weight: 0.5, MustHave: true}, {Name: "Kubernetes", Weight: 0.5, MustHave: true}})
	roleC := openRole(t, "SRE", "Lagos",
		[]role.Competency{{Name: "Go", Weight: 1, MustHave: true}})

	d.candidates.EXPECT().ByID(gomock.Any(), cid).Return(candidateAt(t, "Accra"), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(goSQLProfile(t, cid), nil)
	d.roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).
		Return([]*role.Role{roleB, roleA, roleD, roleC}, int64(4), nil)

	fits, err := matcher.RolesForCandidate(context.Background(), cid, 10)
	require.NoError(t, err)

	require.Len(t, fits, 2, "B (unmet must-have) and C (location) are filtered out")
	assert.Equal(t, roleA.ID, fits[0].Role.ID, "higher structural fit ranks first")
	assert.Equal(t, roleD.ID, fits[1].Role.ID)
	assert.Greater(t, fits[0].Fit.Score, fits[1].Fit.Score)
	assert.True(t, fits[0].Fit.MustHavesMet)
	assert.ElementsMatch(t, []string{"Go", "SQL"}, fits[0].Fit.Covered)
}

func TestRolesForCandidate_NoProfileYields(t *testing.T) {
	ctrl := gomock.NewController(t)
	d, matcher := newTwoWay(ctrl)

	cid := kernel.NewID()
	d.candidates.EXPECT().ByID(gomock.Any(), cid).Return(candidateAt(t, "Accra"), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(nil, kernel.NotFound("no profile"))

	fits, err := matcher.RolesForCandidate(context.Background(), cid, 10)
	require.NoError(t, err)
	assert.Empty(t, fits)
}

func TestRolesForCandidate_LimitCaps(t *testing.T) {
	ctrl := gomock.NewController(t)
	d, matcher := newTwoWay(ctrl)

	cid := kernel.NewID()
	roleA := openRole(t, "Backend Engineer", "Accra",
		[]role.Competency{{Name: "Go", Weight: 1, MustHave: true}})
	roleD := openRole(t, "Data Engineer", "Accra",
		[]role.Competency{{Name: "Go", Weight: 1, MustHave: true}})

	d.candidates.EXPECT().ByID(gomock.Any(), cid).Return(candidateAt(t, "Accra"), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(goSQLProfile(t, cid), nil)
	d.roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).
		Return([]*role.Role{roleA, roleD}, int64(2), nil)

	fits, err := matcher.RolesForCandidate(context.Background(), cid, 1)
	require.NoError(t, err)
	assert.Len(t, fits, 1, "limit caps the result set")
}
