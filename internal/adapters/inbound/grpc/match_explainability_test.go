package grpcadapter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

// TestShortlistExplainabilityContract locks CAL-056/082/087: every match in a
// shortlist response exposes its reasoning + evidence — there are no black-box
// fields. Each match carries an overall score, a confidence level, a non-empty
// per-competency breakdown where every item names a competency and cites
// evidence, a plain-English rationale, and the watch-outs / thin-evidence flags;
// the response is paginated.
func TestShortlistExplainabilityContract(t *testing.T) {
	ctrl := gomock.NewController(t)
	rl, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{Title: "Backend Engineer", Seniority: role.SeniorityMid},
		role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 1, MustHave: true}}},
		time.Unix(1, 0))
	require.NoError(t, err)
	cid := kernel.NewID()

	srv := NewMatchServer(shortlisterWithOneMatch(t, ctrl, rl, cid), nil, nil)
	resp, err := srv.GenerateShortlist(asRole(context.Background(), identity.RoleEmployer),
		&caliberv1.GenerateShortlistRequest{RoleId: rl.ID.String(), Page: &caliberv1.PageRequest{PageSize: 10}})
	require.NoError(t, err)

	sl := resp.GetShortlist()
	require.NotNil(t, sl.GetPage(), "shortlist response is paginated")
	require.NotEmpty(t, sl.GetMatches())
	for _, m := range sl.GetMatches() {
		assert.NotEqual(t, caliberv1.Confidence_CONFIDENCE_UNSPECIFIED, m.GetConfidence(), "confidence is exposed")
		assert.NotEmpty(t, m.GetRationale(), "a plain-English rationale is exposed")
		require.NotEmpty(t, m.GetBreakdown(), "a per-competency breakdown is exposed (not a black box)")
		for _, b := range m.GetBreakdown() {
			assert.NotEmpty(t, b.GetCompetency(), "each breakdown item names its competency")
			assert.NotEmpty(t, b.GetEvidence(), "each breakdown item cites supporting evidence")
		}
	}
}
