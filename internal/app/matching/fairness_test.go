package matching_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	matchingdom "github.com/xcreativs/caliber/internal/domain/matching"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// profileWithSummary builds a profile whose competencies are fixed but whose
// free-text summary varies — the lever the metamorphic test perturbs.
func profileWithSummary(t *testing.T, cid kernel.ID, summary string) *talent.TalentProfile {
	t.Helper()
	p, err := talent.NewTalentProfile(cid, summary,
		[]talent.ProfileCompetency{{Name: "Go", Level: 4, EvidenceQuote: "built services"}})
	require.NoError(t, err)
	return p
}

// TestScoringIsInvariantToProtectedAttributes is the core fairness guarantee
// (CAL-085): two candidates with identical competencies must yield byte-identical
// scoring prompts even when one carries protected attributes (age, gender,
// religion, nationality, marital status, disability) in non-competency fields.
// We capture the exact text sent to the scorer and the embedder and assert (a)
// the per-candidate scoring prompts are equal, and (b) no protected-attribute
// term ever reaches the model. This is a metamorphic test: perturbing only the
// protected dimensions leaves the model's view unchanged.
func TestScoringIsInvariantToProtectedAttributes(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)

	rl := validRole(t)
	cBaseline, cProtected := kernel.NewID(), kernel.NewID()

	// The perturbed candidate's summary is saturated with protected attributes.
	const protectedSummary = "48-year-old married Muslim Ghanaian woman with a physical disability"

	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)

	var embedInput string
	d.embedder.EXPECT().Embed(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, text string) ([]float32, error) {
			embedInput = text
			return []float32{0.1, 0.2}, nil
		})
	d.recaller.EXPECT().Recall(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]kernel.ID{cBaseline, cProtected}, nil)
	d.candidates.EXPECT().ByID(gomock.Any(), cBaseline).Return(candidateAt(t, "Accra"), nil)
	d.candidates.EXPECT().ByID(gomock.Any(), cProtected).Return(candidateAt(t, "Accra"), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cBaseline).
		Return(profileWithSummary(t, cBaseline, "experienced backend engineer"), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cProtected).
		Return(profileWithSummary(t, cProtected, protectedSummary), nil)

	var scoringPrompts []string
	d.scorer.EXPECT().Complete(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req app.LLMRequest) (app.LLMResponse, error) {
			scoringPrompts = append(scoringPrompts, req.Prompt)
			// The system prompt must instruct the model to ignore protected data.
			assert.Contains(t, strings.ToLower(req.System), "protected attributes")
			return app.LLMResponse{Text: score06}, nil
		}).Times(2)
	d.matchRepo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	_, err := d.shortlister().GenerateShortlist(context.Background(), rl.ID, 10)
	require.NoError(t, err)

	require.Len(t, scoringPrompts, 2)
	assert.Equal(t, scoringPrompts[0], scoringPrompts[1],
		"identical competencies must produce identical scoring prompts regardless of protected attributes")

	// No protected-attribute term leaked into the scorer's or embedder's input.
	leaks := []string{"48", "married", "muslim", "ghanaian", "woman", "disability"}
	for _, prompt := range append(scoringPrompts, embedInput) {
		lower := strings.ToLower(prompt)
		for _, term := range leaks {
			assert.NotContainsf(t, lower, term,
				"protected-attribute term %q must never reach the model", term)
		}
	}
}

// TestBiasedRubricIsRejectedBeforeScoring proves the pipeline guard: a rubric
// whose competency names include a protected attribute fails EnsureBiasSafe and
// the shortlist run aborts before any embedding, recall, or scoring happens.
func TestBiasedRubricIsRejectedBeforeScoring(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)

	biased, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{
			Title: "Backend Engineer", Location: "Accra", Seniority: role.SeniorityMid,
			Responsibilities: []string{"build services"}, MustHaves: []string{"Go"},
		},
		role.Rubric{Competencies: []role.Competency{
			{Name: "Go", Weight: 0.5, MustHave: true},
			{Name: "gender", Weight: 0.5}, // a protected attribute as a ranking signal
		}},
		time.Unix(1700000000, 0))
	require.NoError(t, err)

	// Only ByID is reached: the bias-safety gate fires before any model call.
	d.roles.EXPECT().ByID(gomock.Any(), biased.ID).Return(biased, nil)

	_, err = d.shortlister().GenerateShortlist(context.Background(), biased.ID, 10)
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err),
		"a protected attribute among ranking signals must abort the shortlist")
}

// TestHardFilterGatesAreBiasSafe documents that the hard-filter gate identifiers
// are logistical, not protected: "location" (work logistics) is deliberately
// distinct from the protected attribute "nationality".
func TestHardFilterGatesAreBiasSafe(t *testing.T) {
	gates := []string{
		matchingdom.GateLocation,
		matchingdom.GateSalaryFloor,
		matchingdom.GateMustHave,
	}
	assert.NoError(t, matchingdom.EnsureBiasSafe(gates),
		"hard-filter gates must never be protected attributes")
}
