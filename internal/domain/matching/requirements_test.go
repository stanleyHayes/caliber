package matching_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xcreativs/caliber/internal/domain/kernel"
	matchingdom "github.com/xcreativs/caliber/internal/domain/matching"
)

func TestNewRequirements_RemoteDetection(t *testing.T) {
	cid := kernel.NewID()
	cases := []struct {
		name         string
		location     string
		availability string
		candLocation string
		wantCleared  bool // does an Accra candidate clear the location gate?
	}{
		{"remote in location", "Remote", "", "Lagos", true},
		{"remote in availability", "Accra", "remote-friendly, start in 1 month", "Lagos", true},
		{"case-insensitive remote", "REMOTE", "", "Lagos", true},
		{"onsite mismatch gates", "Accra", "onsite", "Lagos", false},
		{"onsite match clears", "Accra", "onsite", "Accra", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := matchingdom.NewRequirements(tc.location, tc.availability, 0, "", nil)
			ex := req.ScreenLogistics(cid, tc.candLocation, 0, "")
			assert.Equal(t, tc.wantCleared, len(ex) == 0)
		})
	}
}

func TestNewRequirements_CarriesSalaryAndMustHaves(t *testing.T) {
	req := matchingdom.NewRequirements("Accra", "onsite", 5000, "GHS", []string{"Go"})
	assert.Equal(t, "Accra", req.Location)
	assert.False(t, req.RemoteAllowed)
	assert.InDelta(t, 5000.0, req.SalaryCeiling, 1e-9)
	assert.Equal(t, "GHS", req.SalaryCurrency)
	assert.Equal(t, []string{"Go"}, req.MustHaves)
}

func TestCoversMustHaves_MatchesComputeFit(t *testing.T) {
	rubric := []matchingdom.RubricSignal{
		{Name: "Go", Weight: 0.6, MustHave: true},
		{Name: "SQL", Weight: 0.4, MustHave: true},
	}
	met := []matchingdom.CandidateSignal{{Name: "Go", Level: 4}, {Name: "SQL / Databases", Level: 3}}
	assert.True(t, matchingdom.CoversMustHaves(rubric, met), "token match satisfies the SQL must-have")
	assert.Equal(t, matchingdom.ComputeFit(rubric, met).MustHavesMet, matchingdom.CoversMustHaves(rubric, met))

	unmet := []matchingdom.CandidateSignal{{Name: "Go", Level: 4}}
	assert.False(t, matchingdom.CoversMustHaves(rubric, unmet), "missing SQL must-have")

	underscored := []matchingdom.CandidateSignal{{Name: "Go", Level: 4}, {Name: "SQL", Level: 1}}
	assert.False(t, matchingdom.CoversMustHaves(rubric, underscored), "SQL below MinMustHaveScore")
}
