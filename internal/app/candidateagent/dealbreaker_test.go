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

// TestRunSkipsDealBreakerRole proves CAL-046: a candidate's stated deal-breaker
// (here "on-call") excludes a role whose text requires it — the agent neither
// surfaces nor applies to it, even when the candidate is otherwise eligible.
func TestRunSkipsDealBreakerRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	cid := kernel.NewID()
	profile := profileWith(t, cid, talent.ProfileCompetency{Name: "Go", Level: 4, EvidenceQuote: "x"})
	rl, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{
			Title: "Backend Engineer", Location: "Accra", Seniority: role.SeniorityMid,
			Responsibilities: []string{"Participate in the on-call rotation."},
		},
		role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 1, MustHave: true}}},
		time.Unix(1, 0))
	require.NoError(t, err)

	cand, err := talent.NewCandidate(kernel.NewID(), "Accra", talent.CandidateIntake{DealBreakers: []string{"on-call"}})
	require.NoError(t, err)

	d.candidates.EXPECT().ByID(gomock.Any(), cid).Return(cand, nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(profile, nil)
	d.roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).Return([]*role.Role{rl}, int64(1), nil)
	// no llm.Complete and no apps.Create: gated out by the deal-breaker before assessment

	view, err := d.runner().Run(context.Background(), cid, 10)
	require.NoError(t, err)
	assert.Zero(t, view.NewMatches, "a role with a declared deal-breaker is not surfaced or applied to")
}
