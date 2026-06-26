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

// TestScoringDependsOnlyOnCompetenciesNotIdentity proves the structural
// bias-safety invariant (CAL-085): the text the scorer and embedder see is built
// ONLY from the role and the candidate's competencies — never from identity-
// bearing fields. Two candidates with identical competencies yield byte-identical
// scoring prompts even when one carries protected attributes in its summary and a
// different passport status, so nothing in those fields can influence the score.
// (Protected attributes that appear INSIDE a competency's evidence quote are a
// distinct surface — see TestEvidenceQuotesAreScoredVerbatim.)
func TestScoringDependsOnlyOnCompetenciesNotIdentity(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)

	rl := validRole(t)
	cBaseline, cProtected := kernel.NewID(), kernel.NewID()
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

	// Identical competencies; the perturbed profile differs ONLY in identity-
	// bearing fields (summary + passport status + id).
	baseProfile := profileWithSummary(t, cBaseline, "experienced backend engineer")
	protProfile := profileWithSummary(t, cProtected, protectedSummary)
	protProfile.MarkScreened()
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cBaseline).Return(baseProfile, nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cProtected).Return(protProfile, nil)

	var scoringPrompts []string
	d.scorer.EXPECT().Complete(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req app.LLMRequest) (app.LLMResponse, error) {
			scoringPrompts = append(scoringPrompts, req.Prompt)
			assert.Contains(t, strings.ToLower(req.System), "protected attributes",
				"system prompt must instruct the model to ignore protected attributes")
			return app.LLMResponse{Text: score06}, nil
		}).Times(2)
	d.matchRepo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	_, err := d.shortlister().GenerateShortlist(context.Background(), rl.ID, 10)
	require.NoError(t, err)

	require.Len(t, scoringPrompts, 2)
	assert.Equal(t, scoringPrompts[0], scoringPrompts[1],
		"identical competencies must produce identical scoring prompts regardless of identity fields")

	// The summary-borne protected terms must never reach the scorer or embedder.
	// Because the prompt is built only from competencies, perturbing the summary
	// changes nothing; a regression that fed the summary in would break BOTH the
	// equality assertion above and this leak check (which is why it is now sound,
	// unlike a version that only perturbs a field already known to be excluded).
	leaks := []string{"48", "married", "muslim", "ghanaian", "woman", "disability"}
	for _, prompt := range append(scoringPrompts, embedInput) {
		lower := strings.ToLower(prompt)
		for _, term := range leaks {
			assert.NotContainsf(t, lower, term,
				"summary-borne protected-attribute term %q must never reach the model", term)
		}
	}
}

// TestEvidenceQuotesAreScoredVerbatim pins the residual fairness surface honestly:
// a protected attribute that appears INSIDE a competency's CV evidence quote DOES
// reach the scorer (the candidate's own words are scored verbatim, never silently
// dropped), so bias-safety there is delegated to the system-prompt instruction —
// asserted present — rather than to input exclusion. This test exists so the
// guarantee is not overstated and any change to that boundary is deliberate.
func TestEvidenceQuotesAreScoredVerbatim(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)

	rl := validRole(t)
	cid := kernel.NewID()
	profile, err := talent.NewTalentProfile(cid, "engineer",
		[]talent.ProfileCompetency{{Name: "Go", Level: 4, EvidenceQuote: "led the women's Muslim coding guild in Accra"}})
	require.NoError(t, err)

	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	d.embedder.EXPECT().Embed(gomock.Any(), gomock.Any()).Return([]float32{0.1}, nil)
	d.recaller.EXPECT().Recall(gomock.Any(), gomock.Any(), gomock.Any()).Return([]kernel.ID{cid}, nil)
	d.candidates.EXPECT().ByID(gomock.Any(), cid).Return(candidateAt(t, "Accra"), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(profile, nil)

	var scoringPrompt, systemPrompt string
	d.scorer.EXPECT().Complete(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req app.LLMRequest) (app.LLMResponse, error) {
			scoringPrompt, systemPrompt = req.Prompt, req.System
			return app.LLMResponse{Text: score06}, nil
		})
	d.matchRepo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil)

	_, err = d.shortlister().GenerateShortlist(context.Background(), rl.ID, 10)
	require.NoError(t, err)

	// Evidence is scored verbatim (the candidate's words are not dropped)...
	assert.Contains(t, strings.ToLower(scoringPrompt), "muslim",
		"competency evidence reaches the scorer verbatim")
	// ...so the bias-safety defense for evidence content is the system instruction.
	assert.Contains(t, strings.ToLower(systemPrompt), "never on protected attributes")
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
