package candidateagent_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

// TestRunGatesOnsiteRoleDespiteRemoteInAvailability is a regression guard: the
// free-text availability/start-date field must NOT be scanned for "remote". An
// on-site role whose availability incidentally mentions "remote" must still gate
// out a geographically-incompatible candidate, so the agent never auto-applies
// across an impassable location gate. (With the pre-fix behaviour the agent
// scanned availability, treated the role as remote, and would have applied.)
func TestRunGatesOnsiteRoleDespiteRemoteInAvailability(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	cid := kernel.NewID()
	profile := profileWith(t, cid, talent.ProfileCompetency{Name: "Go", Level: 4, EvidenceQuote: "x"})

	rl, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{
			Title:        "Backend Engineer",
			Location:     "Accra", // on-site
			Availability: "within 1 month; remote teams experience required",
			Seniority:    role.SeniorityMid,
		},
		role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 1, MustHave: true}}},
		time.Unix(1, 0))
	require.NoError(t, err)

	d.candidates.EXPECT().ByID(gomock.Any(), cid).Return(candidate(t, "London"), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(profile, nil)
	d.roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).Return([]*role.Role{rl}, int64(1), nil)
	// No llm.Complete and no apps.Create: the candidate is gated out pre-assessment.

	view, err := d.runner().Run(context.Background(), cid, 10)
	require.NoError(t, err)
	assert.Zero(t, view.NewMatches,
		"an on-site role must not match a London candidate despite 'remote' in availability")
	assert.Zero(t, view.ApplicationsSubmitted)
}
