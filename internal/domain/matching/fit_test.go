package matching_test

import (
	"testing"

	matchingdom "github.com/xcreativs/caliber/internal/domain/matching"

	"github.com/stretchr/testify/assert"
)

func TestComputeFit_FullCoverage(t *testing.T) {
	rubric := []matchingdom.RubricSignal{
		{Name: "Go", Weight: 0.6, MustHave: true},
		{Name: "SQL", Weight: 0.4},
	}
	cand := []matchingdom.CandidateSignal{
		{Name: "Go", Level: 5},
		{Name: "SQL", Level: 5},
	}
	fit := matchingdom.ComputeFit(rubric, cand)
	assert.InDelta(t, 1.0, fit.Score, 1e-9)
	assert.True(t, fit.MustHavesMet)
	assert.ElementsMatch(t, []string{"Go", "SQL"}, fit.Covered)
	assert.Empty(t, fit.Missing)
}

func TestComputeFit_WeightNormalizedPartial(t *testing.T) {
	rubric := []matchingdom.RubricSignal{
		{Name: "Go", Weight: 0.6, MustHave: true},
		{Name: "SQL", Weight: 0.4},
	}
	// Go at level 5 (full), SQL absent → score = 0.6*1 / (0.6+0.4) = 0.6.
	cand := []matchingdom.CandidateSignal{{Name: "Go", Level: 5}}
	fit := matchingdom.ComputeFit(rubric, cand)
	assert.InDelta(t, 0.6, fit.Score, 1e-9)
	assert.True(t, fit.MustHavesMet, "Go must-have is met; SQL is not must-have")
	assert.Equal(t, []string{"Go"}, fit.Covered)
}

func TestComputeFit_MissingMustHave(t *testing.T) {
	rubric := []matchingdom.RubricSignal{
		{Name: "Go", Weight: 0.5, MustHave: true},
		{Name: "Kubernetes", Weight: 0.5, MustHave: true},
	}
	cand := []matchingdom.CandidateSignal{{Name: "Go", Level: 4}}
	fit := matchingdom.ComputeFit(rubric, cand)
	assert.False(t, fit.MustHavesMet)
	assert.Equal(t, []string{"Kubernetes"}, fit.Missing)
}

func TestComputeFit_UnderscoredMustHaveFails(t *testing.T) {
	rubric := []matchingdom.RubricSignal{{Name: "Go", Weight: 1, MustHave: true}}
	// Level 1 is below MinMustHaveScore (2.0): present but underscored.
	cand := []matchingdom.CandidateSignal{{Name: "Go", Level: 1}}
	fit := matchingdom.ComputeFit(rubric, cand)
	assert.False(t, fit.MustHavesMet)
	assert.Equal(t, []string{"Go"}, fit.Missing)
}

func TestComputeFit_TokenMatch(t *testing.T) {
	rubric := []matchingdom.RubricSignal{{Name: "SQL", Weight: 1, MustHave: true}}
	cand := []matchingdom.CandidateSignal{{Name: "SQL / Databases", Level: 4}}
	fit := matchingdom.ComputeFit(rubric, cand)
	assert.True(t, fit.MustHavesMet, "must-have SQL matches the 'SQL / Databases' competency token")
	assert.InDelta(t, 0.8, fit.Score, 1e-9) // 4/5
}

func TestComputeFit_LevelClampedAndCaseInsensitive(t *testing.T) {
	rubric := []matchingdom.RubricSignal{{Name: "go", Weight: 1}}
	// An out-of-range level is clamped to the unit interval.
	cand := []matchingdom.CandidateSignal{{Name: "GO", Level: 9}}
	fit := matchingdom.ComputeFit(rubric, cand)
	assert.InDelta(t, 1.0, fit.Score, 1e-9)
}

func TestComputeFit_EmptyRubric(t *testing.T) {
	fit := matchingdom.ComputeFit(nil, []matchingdom.CandidateSignal{{Name: "Go", Level: 5}})
	assert.Zero(t, fit.Score)
	assert.True(t, fit.MustHavesMet, "no must-haves are vacuously met")
	assert.Empty(t, fit.Covered)
}

func TestComputeFit_BlankRubricNameSkipped(t *testing.T) {
	rubric := []matchingdom.RubricSignal{
		{Name: "  ", Weight: 0.5, MustHave: true},
		{Name: "Go", Weight: 0.5, MustHave: true},
	}
	cand := []matchingdom.CandidateSignal{{Name: "Go", Level: 5}}
	fit := matchingdom.ComputeFit(rubric, cand)
	assert.True(t, fit.MustHavesMet, "a blank rubric entry is ignored, not treated as an unmet must-have")
	assert.InDelta(t, 1.0, fit.Score, 1e-9)
}
