package grpcadapter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dashboardapp "github.com/xcreativs/caliber/internal/app/dashboard"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
	"github.com/xcreativs/caliber/internal/mocks"
)

func TestDashboardTimeToShortlistHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	agg := dashboardapp.NewAggregator(
		mocks.NewMockCandidateRepository(ctrl), mocks.NewMockTalentProfileRepository(ctrl),
		mocks.NewMockUserRepository(ctrl), mocks.NewMockRoleRepository(ctrl))
	resp, err := NewDashboardServer(agg).GetTimeToShortlist(asRole(context.Background(), identity.RoleEmployer), &caliberv1.GetTimeToShortlistRequest{})
	require.NoError(t, err)
	assert.InDelta(t, 504.0, resp.GetMetric().GetBaselineHours(), 0.01)
	assert.Greater(t, resp.GetMetric().GetImprovementFactor(), 1.0)
}

func TestDashboardGetPoolHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	candidates := mocks.NewMockCandidateRepository(ctrl)
	profiles := mocks.NewMockTalentProfileRepository(ctrl)
	users := mocks.NewMockUserRepository(ctrl)
	roles := mocks.NewMockRoleRepository(ctrl)

	cand, err := talent.NewCandidate(kernel.NewID(), "Accra", talent.CandidateIntake{})
	require.NoError(t, err)
	profile, err := talent.NewTalentProfile(cand.ID, "s", []talent.ProfileCompetency{{Name: "Go", Level: 4, EvidenceQuote: "x"}})
	require.NoError(t, err)
	profile.MarkScreened()
	candidates.EXPECT().List(gomock.Any(), gomock.Any()).Return([]*talent.Candidate{cand}, int64(1), nil)
	users.EXPECT().ByID(gomock.Any(), cand.UserID).Return(nil, kernel.NotFound("none"))
	profiles.EXPECT().ByCandidateID(gomock.Any(), cand.ID).Return(profile, nil)

	srv := NewDashboardServer(dashboardapp.NewAggregator(candidates, profiles, users, roles))
	resp, err := srv.GetPool(asRole(context.Background(), identity.RoleEmployer), &caliberv1.GetPoolRequest{Page: &caliberv1.PageRequest{Page: 1, PageSize: 10}})
	require.NoError(t, err)
	require.Len(t, resp.GetCandidates(), 1)
	assert.Equal(t, caliberv1.PassportStatus_PASSPORT_STATUS_SCREENED, resp.GetCandidates()[0].GetPassportStatus())
	assert.Equal(t, int64(1), resp.GetPage().GetTotalItems())
}

func TestDashboardSupplyDemandHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	candidates := mocks.NewMockCandidateRepository(ctrl)
	roles := mocks.NewMockRoleRepository(ctrl)
	rl, err := role.NewRole(kernel.NewID(), role.RoleSpec{Title: "E", Seniority: role.SeniorityMid},
		role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 1}}}, time.Unix(1, 0))
	require.NoError(t, err)
	roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).Return([]*role.Role{rl}, int64(1), nil)
	candidates.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, int64(3), nil)

	srv := NewDashboardServer(dashboardapp.NewAggregator(
		candidates, mocks.NewMockTalentProfileRepository(ctrl), mocks.NewMockUserRepository(ctrl), roles))
	resp, err := srv.GetSupplyDemand(asRole(context.Background(), identity.RoleEmployer), &caliberv1.GetSupplyDemandRequest{})
	require.NoError(t, err)
	require.Len(t, resp.GetItems(), 1)
	assert.Equal(t, "mid", resp.GetItems()[0].GetRoleFamily())
	assert.Equal(t, int32(1), resp.GetItems()[0].GetOpenRoles())
	assert.Equal(t, int32(3), resp.GetItems()[0].GetAvailableCandidates())
}

func TestDashboardRequiresReviewer(t *testing.T) {
	ctrl := gomock.NewController(t)
	srv := NewDashboardServer(dashboardapp.NewAggregator(
		mocks.NewMockCandidateRepository(ctrl), mocks.NewMockTalentProfileRepository(ctrl),
		mocks.NewMockUserRepository(ctrl), mocks.NewMockRoleRepository(ctrl)))

	// A candidate cannot read the employer-facing Talent Radar.
	_, err := srv.GetPool(asRole(context.Background(), identity.RoleCandidate),
		&caliberv1.GetPoolRequest{Page: &caliberv1.PageRequest{Page: 1, PageSize: 10}})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	// An unauthenticated caller is rejected outright.
	_, err = srv.GetTimeToShortlist(context.Background(), &caliberv1.GetTimeToShortlistRequest{})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}
