package dashboard_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	dashboardapp "github.com/xcreativs/caliber/internal/app/dashboard"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

// TestAlerts_PaginationOffsets exercises pageAlerts beyond offset 0: a one-
// candidate/one-role run yields two alerts (candidate_for_role + role_for_
// candidate), so page (1,1) and (2,1) return disjoint single alerts and page
// (3,1) returns an empty slice — with total always 2 and no panic.
func TestAlerts_PaginationOffsets(t *testing.T) {
	run := func(page kernel.Page) ([]dashboardapp.MatchAlert, int64) {
		ctrl := gomock.NewController(t)
		d := newDeps(ctrl)
		cand, user := alertCandidate(t)
		fitRole := alertRole(t, "Backend Engineer", "Accra",
			[]role.Competency{{Name: "Go", Weight: 0.6, MustHave: true}, {Name: "SQL", Weight: 0.4}})
		d.roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).Return([]*role.Role{fitRole}, int64(1), nil)
		d.candidates.EXPECT().List(gomock.Any(), gomock.Any()).Return([]*talent.Candidate{cand}, int64(1), nil)
		d.profiles.EXPECT().ByCandidateID(gomock.Any(), cand.ID).Return(goSQLProfileFor(t, cand.ID), nil)
		d.users.EXPECT().ByID(gomock.Any(), cand.UserID).Return(user, nil)
		alerts, total, err := d.agg().Alerts(context.Background(), page)
		require.NoError(t, err)
		return alerts, total
	}

	p1, total1 := run(kernel.NewPage(1, 1))
	p2, total2 := run(kernel.NewPage(2, 1))
	p3, total3 := run(kernel.NewPage(3, 1))

	require.Len(t, p1, 1)
	require.Len(t, p2, 1)
	assert.Empty(t, p3, "offset past the end returns an empty slice, no panic")
	assert.Equal(t, int64(2), total1)
	assert.Equal(t, int64(2), total2)
	assert.Equal(t, int64(2), total3, "total reflects the full set regardless of page")

	assert.NotEqual(t, p1[0].ID, p2[0].ID, "page 1 and page 2 are disjoint")
	assert.Equal(t, dashboardapp.AlertCandidateForRole, p1[0].Type)
	assert.Equal(t, dashboardapp.AlertRoleForCandidate, p2[0].Type)
}
