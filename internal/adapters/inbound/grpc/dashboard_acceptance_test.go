package grpcadapter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	dashboardapp "github.com/xcreativs/caliber/internal/app/dashboard"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

// TestTalentRadarEndToEnd is the closing demo-beat acceptance test: the Talent
// Radar dashboard rendered over a realistic pool — the live candidate pool, the
// supply/demand snapshot, the two-way match alerts, and the time-to-shortlist
// headline — all served through the gRPC handlers over the in-memory stack.
func TestTalentRadarEndToEnd(t *testing.T) {
	ctx := context.Background()
	users := memory.NewUserRepo()
	candRepo := memory.NewCandidateRepo()
	profRepo := memory.NewTalentProfileRepo()
	roleRepo := memory.NewRoleRepo()

	// A candidate (Ama) with a verified Go/SQL profile in Accra.
	email, err := identity.NewEmail("ama@example.com")
	require.NoError(t, err)
	user, err := identity.NewUser(email, identity.RoleCandidate, "Ama Mensah", "hash", time.Unix(1, 0))
	require.NoError(t, err)
	require.NoError(t, users.Create(ctx, user))
	cand, err := talent.NewCandidate(user.ID, "Accra", talent.CandidateIntake{})
	require.NoError(t, err)
	require.NoError(t, candRepo.Create(ctx, cand))
	profile, err := talent.NewTalentProfile(cand.ID, "backend engineer", []talent.ProfileCompetency{
		{Name: "Go", Level: 5, EvidenceQuote: "led a payments platform", SourceSpan: "CV"},
		{Name: "SQL", Level: 4, EvidenceQuote: "designed schemas", SourceSpan: "CV"},
	})
	require.NoError(t, err)
	require.NoError(t, profRepo.Create(ctx, profile))

	// An open role Ama is a strong fit for (drives a two-way alert).
	rl, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{Title: "Backend Engineer", Location: "Accra", Seniority: role.SeniorityMid},
		role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 0.6, MustHave: true}, {Name: "SQL", Weight: 0.4}}},
		time.Unix(1700000000, 0))
	require.NoError(t, err)
	require.NoError(t, rl.Open())
	require.NoError(t, roleRepo.Create(ctx, rl))

	srv := NewDashboardServer(dashboardapp.NewAggregator(candRepo, profRepo, users, roleRepo))
	page := &caliberv1.PageRequest{Page: 1, PageSize: 10}
	// The Talent Radar is an employer/recruiter view (CAL-116).
	reviewer := asRole(ctx, identity.RoleEmployer)

	// A candidate cannot read the dashboard.
	_, err = srv.GetPool(asRole(ctx, identity.RoleCandidate), &caliberv1.GetPoolRequest{Page: page})
	require.Error(t, err, "the dashboard is not candidate-facing")

	// Live pool: the candidate appears with their name + passport status.
	pool, err := srv.GetPool(reviewer, &caliberv1.GetPoolRequest{Page: page})
	require.NoError(t, err)
	require.Len(t, pool.GetCandidates(), 1)
	assert.Equal(t, "Ama Mensah", pool.GetCandidates()[0].GetName())
	assert.NotNil(t, pool.GetPage())

	// Supply/demand: the mid-seniority family shows the open role.
	sd, err := srv.GetSupplyDemand(reviewer, &caliberv1.GetSupplyDemandRequest{})
	require.NoError(t, err)
	require.NotEmpty(t, sd.GetItems())
	var foundMid bool
	for _, it := range sd.GetItems() {
		if it.GetRoleFamily() == "mid" {
			foundMid = true
			assert.GreaterOrEqual(t, it.GetOpenRoles(), int32(1))
		}
	}
	assert.True(t, foundMid, "the mid-seniority family is represented")

	// Two-way alerts: Ama <-> the Backend role is surfaced.
	alerts, err := srv.GetAlerts(reviewer, &caliberv1.GetAlertsRequest{Page: page})
	require.NoError(t, err)
	assert.NotEmpty(t, alerts.GetAlerts(), "a strong candidate<->role pair raises an alert")
	assert.NotNil(t, alerts.GetPage())

	// The headline metric: weeks collapse to hours.
	ttl, err := srv.GetTimeToShortlist(reviewer, &caliberv1.GetTimeToShortlistRequest{})
	require.NoError(t, err)
	m := ttl.GetMetric()
	require.NotNil(t, m)
	assert.Positive(t, m.GetBaselineHours())
	assert.Positive(t, m.GetCurrentHours())
	assert.Greater(t, m.GetBaselineHours(), m.GetCurrentHours(), "the platform is faster than the manual baseline")
	assert.Greater(t, m.GetImprovementFactor(), 1.0)
}
