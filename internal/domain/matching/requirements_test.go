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
		candLocation string
		wantCleared  bool // does an out-of-area candidate clear the location gate?
	}{
		{"remote location token", "Remote", "Lagos", true},
		{"hybrid location token", "Accra / Remote", "Lagos", true},
		{"hyphenated remote-first", "Remote-first", "Lagos", true},
		{"hyphenated fully-remote", "Fully-Remote", "Lagos", true},
		{"case-insensitive remote", "REMOTE", "Lagos", true},
		{"onsite mismatch gates", "Accra", "Lagos", false},
		{"onsite match clears", "Accra", "Accra", true},
		{"remote substring is not a whole token", "Remoteville", "Lagos", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// availability is intentionally NOT a parameter: a remote role must
			// declare it via the location field, never the start-date text.
			req := matchingdom.NewRequirements(tc.location, 0, "", nil)
			ex := req.ScreenLogistics(cid, tc.candLocation, 0, "")
			assert.Equal(t, tc.wantCleared, len(ex) == 0)
		})
	}
}

func TestNewRequirements_CarriesSalaryAndMustHaves(t *testing.T) {
	req := matchingdom.NewRequirements("Accra", 5000, "GHS", []string{"Go"})
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
