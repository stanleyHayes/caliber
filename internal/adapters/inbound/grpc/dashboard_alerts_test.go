package grpcadapter

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
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
	"github.com/xcreativs/caliber/internal/mocks"
)

func TestDashboardGetAlertsHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	candidates := mocks.NewMockCandidateRepository(ctrl)
	profiles := mocks.NewMockTalentProfileRepository(ctrl)
	users := mocks.NewMockUserRepository(ctrl)
	roles := mocks.NewMockRoleRepository(ctrl)

	email, err := identity.NewEmail("ama@example.com")
	require.NoError(t, err)
	user, err := identity.NewUser(email, identity.RoleCandidate, "Ama Mensah", "hash", time.Unix(1, 0))
	require.NoError(t, err)
	cand, err := talent.NewCandidate(user.ID, "Accra", talent.CandidateIntake{})
	require.NoError(t, err)
	profile, err := talent.NewTalentProfile(cand.ID, "backend", []talent.ProfileCompetency{
		{Name: "Go", Level: 5, EvidenceQuote: "built services"},
		{Name: "SQL", Level: 4, EvidenceQuote: "schemas"},
	})
	require.NoError(t, err)
	fitRole, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{Title: "Backend Engineer", Location: "Accra", Seniority: role.SeniorityMid},
		role.Rubric{Competencies: []role.Competency{
			{Name: "Go", Weight: 0.6, MustHave: true}, {Name: "SQL", Weight: 0.4},
		}}, time.Unix(1, 0))
	require.NoError(t, err)

	roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).Return([]*role.Role{fitRole}, int64(1), nil)
	candidates.EXPECT().List(gomock.Any(), gomock.Any()).Return([]*talent.Candidate{cand}, int64(1), nil)
	profiles.EXPECT().ByCandidateID(gomock.Any(), cand.ID).Return(profile, nil)
	users.EXPECT().ByID(gomock.Any(), cand.UserID).Return(user, nil)

	srv := NewDashboardServer(dashboardapp.NewAggregator(candidates, profiles, users, roles))
	resp, err := srv.GetAlerts(asRole(context.Background(), identity.RoleEmployer), &caliberv1.GetAlertsRequest{
		Page: &caliberv1.PageRequest{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	require.Len(t, resp.GetAlerts(), 2)
	assert.Equal(t, int64(2), resp.GetPage().GetTotalItems())

	types := map[caliberv1.AlertType]bool{}
	for _, al := range resp.GetAlerts() {
		types[al.GetType()] = true
		assert.Equal(t, fitRole.ID.String(), al.GetRoleId())
		assert.NotEmpty(t, al.GetMessage())
	}
	assert.True(t, types[caliberv1.AlertType_ALERT_TYPE_CANDIDATE_FOR_ROLE])
	assert.True(t, types[caliberv1.AlertType_ALERT_TYPE_ROLE_FOR_CANDIDATE])
}

func TestAlertTypeToProto(t *testing.T) {
	assert.Equal(t, caliberv1.AlertType_ALERT_TYPE_CANDIDATE_FOR_ROLE,
		alertTypeToProto(dashboardapp.AlertCandidateForRole))
	assert.Equal(t, caliberv1.AlertType_ALERT_TYPE_ROLE_FOR_CANDIDATE,
		alertTypeToProto(dashboardapp.AlertRoleForCandidate))
	assert.Equal(t, caliberv1.AlertType_ALERT_TYPE_UNSPECIFIED, alertTypeToProto("nonsense"))
}
