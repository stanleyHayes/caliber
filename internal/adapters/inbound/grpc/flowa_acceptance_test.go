package grpcadapter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/embeddings"
	"github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	matchingapp "github.com/xcreativs/caliber/internal/app/matching"
	"github.com/xcreativs/caliber/internal/app/roles"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

// TestFlowAEndToEnd is the Flow A demo-beat acceptance test (CAL-059): a messy
// hiring sentence becomes a structured role spec + rubric with an instant
// availability signal, and a ranked, explainable shortlist over a realistic pool —
// exclusions surfaced, never silent. It runs the real use-cases through the gRPC
// handlers over the in-memory stack + deterministic dev model.
func TestFlowAEndToEnd(t *testing.T) {
	ctx := context.Background()
	roleRepo := memory.NewRoleRepo()
	candRepo := memory.NewCandidateRepo()
	profRepo := memory.NewTalentProfileRepo()

	ev := func(name string, level float64) talent.ProfileCompetency {
		return talent.ProfileCompetency{Name: name, Level: level, EvidenceQuote: "demonstrated in the CV", SourceSpan: "CV"}
	}
	seedCandidate := func(t *testing.T, comps ...talent.ProfileCompetency) {
		t.Helper()
		c, err := talent.NewCandidate(kernel.NewID(), "Accra, Ghana", talent.CandidateIntake{Location: "Accra, Ghana"})
		require.NoError(t, err)
		require.NoError(t, candRepo.Create(ctx, c))
		p, perr := talent.NewTalentProfile(c.ID, "summary", comps)
		require.NoError(t, perr)
		require.NoError(t, profRepo.Create(ctx, p))
	}
	// The dev role-spec rubric is [Core skills (must-have), Communication, System design].
	seedCandidate(t, ev("Core skills", 5), ev("Communication", 4), ev("System design", 4)) // strong, covers all
	seedCandidate(t, ev("Core skills", 4), ev("Communication", 3))                         // covers the must-have
	seedCandidate(t, ev("Communication", 5))                                               // lacks the must-have -> excluded

	shortlister := matchingapp.NewShortlister(
		roleRepo, candRepo, profRepo, memory.NewRecaller(candRepo), embeddings.NewDev(), llm.NewDev(), memory.NewMatchRepo())
	matchSrv := NewMatchServer(shortlister, matchingapp.NewRefiner(roleRepo, shortlister), nil)
	roleSrv := NewRoleServer(
		roles.NewSpecGenerator(llm.NewDev(), roleRepo, time.Now), roles.NewSpecEditor(roleRepo), matchSrv.AvailabilityCounter())

	// 1) Messy sentence -> structured spec + rubric + instant availability.
	gen, err := roleSrv.GenerateRoleSpec(ctx, &caliberv1.GenerateRoleSpecRequest{
		EmployerId: kernel.NewID().String(),
		FreeText:   "Senior Go engineer in Accra to lead our payments platform",
	})
	require.NoError(t, err)
	role := gen.GetRole()
	assert.NotEmpty(t, role.GetTitle())
	require.NotEmpty(t, role.GetRubric().GetCompetencies(), "a weighted rubric is generated")
	assert.GreaterOrEqual(t, gen.GetAvailableMatches(), int32(2), "two candidates cover the must-have on paper")

	// 2) Ranked, explainable shortlist over the pool.
	slResp, err := matchSrv.GenerateShortlist(asRole(ctx, identity.RoleEmployer), &caliberv1.GenerateShortlistRequest{RoleId: role.GetId()})
	require.NoError(t, err)
	sl := slResp.GetShortlist()
	matches := sl.GetMatches()
	require.GreaterOrEqual(t, len(matches), 2, "both qualifying candidates make the shortlist")

	// Ranked by overall fit, descending.
	for i := 1; i < len(matches); i++ {
		assert.GreaterOrEqual(t, matches[i-1].GetOverallScore(), matches[i].GetOverallScore(), "shortlist is ranked")
	}

	// The top match is fully explainable.
	top := matches[0]
	assert.Positive(t, top.GetOverallScore())
	assert.NotEqual(t, caliberv1.Confidence_CONFIDENCE_UNSPECIFIED, top.GetConfidence(), "confidence is set")
	assert.NotEmpty(t, top.GetRationale(), "plain-English why-this-person")
	require.NotEmpty(t, top.GetBreakdown(), "per-competency breakdown")
	for _, b := range top.GetBreakdown() {
		assert.NotEmpty(t, b.GetCompetency(), "every breakdown item names its competency")
	}

	// Exclusions are surfaced (the candidate missing the must-have), never silent.
	assert.NotEmpty(t, sl.GetExclusions(), "the must-have miss is explained, not dropped silently")
	assert.GreaterOrEqual(t, sl.GetPoolDepth(), int32(2), "pool depth reflects the strong-match total")
	require.NotNil(t, sl.GetPage(), "the shortlist response is paginated")
}
