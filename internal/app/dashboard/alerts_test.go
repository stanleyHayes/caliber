package dashboard_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	dashboardapp "github.com/xcreativs/caliber/internal/app/dashboard"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

func alertRole(t *testing.T, title, location string, comps []role.Competency) *role.Role {
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

func alertCandidate(t *testing.T) (*talent.Candidate, *identity.User) {
	t.Helper()
	email, err := identity.NewEmail("ama@example.com")
	require.NoError(t, err)
	user, err := identity.NewUser(email, identity.RoleCandidate, "Ama Mensah", "hash", time.Unix(1, 0))
	require.NoError(t, err)
	cand, err := talent.NewCandidate(user.ID, "Accra", talent.CandidateIntake{})
	require.NoError(t, err)
	return cand, user
}

func goSQLProfileFor(t *testing.T, cid kernel.ID) *talent.TalentProfile {
	t.Helper()
	p, err := talent.NewTalentProfile(cid, "backend engineer", []talent.ProfileCompetency{
		{Name: "Go", Level: 5, EvidenceQuote: "built services"},
		{Name: "SQL", Level: 4, EvidenceQuote: "designed schemas"},
	})
	require.NoError(t, err)
	return p
}

func TestAlerts_GeneratesTwoWayAlerts(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)

	cand, user := alertCandidate(t)
	fitRole := alertRole(t, "Backend Engineer", "Accra",
		[]role.Competency{{Name: "Go", Weight: 0.6, MustHave: true}, {Name: "SQL", Weight: 0.4}})
	farRole := alertRole(t, "SRE", "Lagos",
		[]role.Competency{{Name: "Go", Weight: 1, MustHave: true}}) // logistically incompatible

	d.roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).
		Return([]*role.Role{fitRole, farRole}, int64(2), nil)
	d.candidates.EXPECT().List(gomock.Any(), gomock.Any()).
		Return([]*talent.Candidate{cand}, int64(1), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cand.ID).Return(goSQLProfileFor(t, cand.ID), nil)
	d.users.EXPECT().ByID(gomock.Any(), cand.UserID).Return(user, nil)

	alerts, total, err := d.agg().Alerts(context.Background(), kernel.NewPage(1, 10))
	require.NoError(t, err)

	require.Equal(t, int64(2), total, "one candidate_for_role + one best-fit role_for_candidate")
	require.Len(t, alerts, 2)

	byType := map[string]dashboardapp.MatchAlert{}
	for _, al := range alerts {
		byType[al.Type] = al
		assert.Equal(t, fitRole.ID, al.RoleID, "only the compatible role produces alerts")
		assert.Equal(t, cand.ID, al.CandidateID)
		assert.Contains(t, al.Message, "Ama Mensah")
	}
	assert.Contains(t, byType, dashboardapp.AlertCandidateForRole)
	assert.Contains(t, byType, dashboardapp.AlertRoleForCandidate)
	assert.Contains(t, byType[dashboardapp.AlertCandidateForRole].Message, "Backend Engineer")
	// Deterministic, idempotent alert id.
	assert.True(t, strings.HasPrefix(byType[dashboardapp.AlertCandidateForRole].ID.String(),
		dashboardapp.AlertCandidateForRole+":"))
}

func TestAlerts_WeakFitProducesNoAlert(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)

	cand, _ := alertCandidate(t)
	// Candidate lacks the must-have "Rust": no strong fit, no alert.
	role1 := alertRole(t, "Systems Engineer", "Accra",
		[]role.Competency{{Name: "Rust", Weight: 1, MustHave: true}})

	d.roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).Return([]*role.Role{role1}, int64(1), nil)
	d.candidates.EXPECT().List(gomock.Any(), gomock.Any()).
		Return([]*talent.Candidate{cand}, int64(1), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cand.ID).Return(goSQLProfileFor(t, cand.ID), nil)
	d.users.EXPECT().ByID(gomock.Any(), cand.UserID).Return(nil, kernel.NotFound("no user")).AnyTimes()

	alerts, total, err := d.agg().Alerts(context.Background(), kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, alerts)
}

func TestAlerts_Paginates(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)

	cand, user := alertCandidate(t)
	fitRole := alertRole(t, "Backend Engineer", "Accra",
		[]role.Competency{{Name: "Go", Weight: 0.6, MustHave: true}, {Name: "SQL", Weight: 0.4}})

	d.roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).Return([]*role.Role{fitRole}, int64(1), nil)
	d.candidates.EXPECT().List(gomock.Any(), gomock.Any()).
		Return([]*talent.Candidate{cand}, int64(1), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cand.ID).Return(goSQLProfileFor(t, cand.ID), nil)
	d.users.EXPECT().ByID(gomock.Any(), cand.UserID).Return(user, nil)

	// Two alerts total; page size 1 returns the first, total still reports 2.
	alerts, total, err := d.agg().Alerts(context.Background(), kernel.NewPage(1, 1))
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, alerts, 1)
}
