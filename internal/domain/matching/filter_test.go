package matching_test

import (
	"testing"

	"github.com/xcreativs/caliber/internal/domain/kernel"
	matchingdom "github.com/xcreativs/caliber/internal/domain/matching"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScreenLogisticsLocation(t *testing.T) {
	cid := kernel.NewID()
	cases := []struct {
		name        string
		req         matchingdom.Requirements
		location    string
		wantExclude bool
	}{
		{"shared token passes", matchingdom.Requirements{Location: "Accra"}, "Accra, Ghana", false},
		{"token either way", matchingdom.Requirements{Location: "Accra, Ghana"}, "Accra", false},
		{"distinct city excludes", matchingdom.Requirements{Location: "Accra"}, "Lagos", true},
		{"substring is not a token match", matchingdom.Requirements{Location: "Accra"}, "Accraville", true},
		{"remote skips gate", matchingdom.Requirements{Location: "Accra", RemoteAllowed: true}, "Lagos", false},
		{"unconstrained role passes", matchingdom.Requirements{Location: ""}, "Lagos", false},
		{"unknown candidate location passes", matchingdom.Requirements{Location: "Accra"}, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ex := tc.req.ScreenLogistics(cid, tc.location, 0, "")
			if !tc.wantExclude {
				assert.Empty(t, ex)
				return
			}
			require.Len(t, ex, 1)
			assert.Equal(t, matchingdom.GateLocation, ex[0].Gate)
			assert.Equal(t, cid, ex[0].CandidateID)
			assert.NotEmpty(t, ex[0].Reason)
		})
	}
}

func TestScreenLogisticsSalary(t *testing.T) {
	cid := kernel.NewID()
	req := matchingdom.Requirements{SalaryCeiling: 10000, SalaryCurrency: "GHS"}

	t.Run("floor above ceiling same currency excludes", func(t *testing.T) {
		ex := req.ScreenLogistics(cid, "", 12000, "GHS")
		require.Len(t, ex, 1)
		assert.Equal(t, matchingdom.GateSalaryFloor, ex[0].Gate)
		assert.Contains(t, ex[0].Reason, "GHS")
	})
	t.Run("cross-currency never gates", func(t *testing.T) {
		assert.Empty(t, req.ScreenLogistics(cid, "", 12000, "USD"))
	})
	t.Run("missing candidate currency never gates", func(t *testing.T) {
		assert.Empty(t, req.ScreenLogistics(cid, "", 12000, ""))
	})
	t.Run("floor within band passes", func(t *testing.T) {
		assert.Empty(t, req.ScreenLogistics(cid, "", 8000, "GHS"))
	})
	t.Run("unset floor passes", func(t *testing.T) {
		assert.Empty(t, req.ScreenLogistics(cid, "", 0, "GHS"))
	})
	t.Run("unknown ceiling passes", func(t *testing.T) {
		assert.Empty(t, matchingdom.Requirements{}.ScreenLogistics(cid, "", 99999, "GHS"))
	})
}

func mustHaveMatch(t *testing.T, items []matchingdom.MatchBreakdownItem) *matchingdom.Match {
	t.Helper()
	m, err := matchingdom.NewMatch(kernel.NewID(), kernel.NewID(), 0.5, kernel.ConfidenceMedium, items, "r", nil, false)
	require.NoError(t, err)
	return m
}

func TestScreenMatchMustHave(t *testing.T) {
	req := matchingdom.Requirements{MustHaves: []string{"Go", "SQL"}}

	t.Run("all must-haves met passes", func(t *testing.T) {
		m := mustHaveMatch(t, []matchingdom.MatchBreakdownItem{
			{Competency: "go", Score: 3}, {Competency: "SQL", Score: 2},
		})
		assert.Empty(t, req.ScreenMatch(m))
	})
	t.Run("present but under-scored excludes", func(t *testing.T) {
		m := mustHaveMatch(t, []matchingdom.MatchBreakdownItem{
			{Competency: "Go", Score: 1}, {Competency: "SQL", Score: 4},
		})
		ex := req.ScreenMatch(m)
		require.Len(t, ex, 1)
		assert.Equal(t, matchingdom.GateMustHave, ex[0].Gate)
		assert.Contains(t, ex[0].Reason, "Go")
	})
	t.Run("absent must-have does NOT exclude (routes to human review)", func(t *testing.T) {
		// SQL is missing from the breakdown: uncertainty, not evidence of a gap.
		m := mustHaveMatch(t, []matchingdom.MatchBreakdownItem{{Competency: "Go", Score: 4}})
		assert.Empty(t, req.ScreenMatch(m))
	})
	t.Run("naming drift is treated as absent, not a low score", func(t *testing.T) {
		// "Golang" does not token-match must-have "Go" -> absent -> not excluded
		// (a false accept that human review catches, never a false rejection).
		m := mustHaveMatch(t, []matchingdom.MatchBreakdownItem{{Competency: "Golang", Score: 1}})
		assert.Empty(t, req.ScreenMatch(m))
	})
	t.Run("token match against decorated competency name", func(t *testing.T) {
		// "SQL / Databases" token-matches must-have "SQL"; score 1 < 2 -> excluded.
		m := mustHaveMatch(t, []matchingdom.MatchBreakdownItem{
			{Competency: "Go", Score: 4}, {Competency: "SQL / Databases", Score: 1},
		})
		ex := req.ScreenMatch(m)
		require.Len(t, ex, 1)
		assert.Contains(t, ex[0].Reason, "SQL")
	})
	t.Run("duplicate must-haves yield a single exclusion", func(t *testing.T) {
		dup := matchingdom.Requirements{MustHaves: []string{"Go", "go", " GO "}}
		m := mustHaveMatch(t, []matchingdom.MatchBreakdownItem{{Competency: "Go", Score: 1}})
		assert.Len(t, dup.ScreenMatch(m), 1)
	})
	t.Run("blank must-have names are skipped", func(t *testing.T) {
		blank := matchingdom.Requirements{MustHaves: []string{"Go", "", "  "}}
		m := mustHaveMatch(t, []matchingdom.MatchBreakdownItem{{Competency: "Go", Score: 4}})
		assert.Empty(t, blank.ScreenMatch(m))
	})
	t.Run("nil match is safe", func(t *testing.T) {
		assert.Empty(t, req.ScreenMatch(nil))
	})
}
