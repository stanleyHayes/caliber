package matching

import (
	"fmt"
	"slices"
	"strings"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// Hard-filter gate identifiers (CAL-049 stage 3). Each names a structured,
// bias-safe constraint derived from the role; protected attributes are never
// gates (see EnsureBiasSafe). "Location" here is a logistical work-location
// constraint, distinct from the protected attribute "nationality": a remote
// role bypasses it entirely and unknown locations never exclude.
const (
	// GateLocation rejects a candidate whose work location is incompatible with
	// a non-remote role's required location.
	GateLocation = "location"
	// GateSalaryFloor rejects a candidate whose salary floor, in the same
	// currency, exceeds the top of the role's offered band.
	GateSalaryFloor = "salary_floor"
	// GateMustHave rejects a candidate who is present-but-underscored on a
	// must-have rubric competency.
	GateMustHave = "must_have_competency"
)

// MinMustHaveScore is the minimum 0..5 competency score a candidate must reach
// on every must-have rubric competency that the scorer actually evaluated.
const MinMustHaveScore = 2.0

// Exclusion records a candidate removed from a shortlist by a hard filter,
// naming the gate that rejected them and a plain-English reason. Surfacing the
// reason keeps the pipeline explainable: hard filters never drop silently.
type Exclusion struct {
	CandidateID kernel.ID
	Gate        string
	Reason      string
}

// Requirements are the structured, bias-safe hard constraints a candidate must
// satisfy to remain on a shortlist (CAL-049). They derive from a role's spec
// and rubric and consider only logistical and rubric facts — location, salary,
// and must-have competencies — never protected attributes. Every gate excludes
// only on POSITIVE evidence of a conflict: unknown or unscored data never
// excludes, favouring human review over false rejection (no-fabrication).
type Requirements struct {
	// Location is the required work location; empty means unconstrained.
	Location string
	// RemoteAllowed disables the location gate (the role can be done remotely).
	RemoteAllowed bool
	// SalaryCeiling is the top of the offered band; zero means unknown.
	SalaryCeiling float64
	// SalaryCurrency labels the band currency; the salary gate only fires when
	// the candidate's floor is quoted in the same currency.
	SalaryCurrency string
	// MustHaves are the rubric competencies flagged must-have.
	MustHaves []string
}

// NewRequirements builds the hard-constraint set from a role's logistical facts.
// RemoteAllowed is derived ONLY from the location field carrying "remote" as a
// whole token (e.g. "Remote" or "Accra / Remote"). It deliberately does NOT scan
// the free-text availability/start-date field ("e.g. within 1 month"), where an
// incidental mention ("remote teams experience required") would otherwise
// disable the location gate and let an incompatible candidate through. Token (not
// substring) matching also avoids false positives like "Remoteville".
// Centralizing this keeps every caller's gate identical — shortlist, the
// candidate agent, the two-way matcher, and the Radar alert feed.
func NewRequirements(location string, salaryCeiling float64, salaryCurrency string, mustHaves []string) Requirements {
	return Requirements{
		Location:       location,
		RemoteAllowed:  hasToken(strings.ReplaceAll(strings.ToLower(location), "-", " "), "remote"),
		SalaryCeiling:  salaryCeiling,
		SalaryCurrency: salaryCurrency,
		MustHaves:      mustHaves,
	}
}

// ScreenLogistics evaluates the pre-scoring gates that depend only on candidate
// facts (location, salary expectation). It returns one Exclusion per failed
// gate; an empty slice means the candidate clears these gates. Running before
// the LLM scoring step, it lets the caller skip expensive scoring for
// candidates that cannot qualify.
func (r Requirements) ScreenLogistics(candidateID kernel.ID, location string, salaryFloor float64, salaryCurrency string) []Exclusion {
	var ex []Exclusion
	if r.locationMismatch(location) {
		ex = append(ex, Exclusion{
			CandidateID: candidateID,
			Gate:        GateLocation,
			Reason:      fmt.Sprintf("role location %q is incompatible with candidate location %q", r.Location, location),
		})
	}
	if r.salaryFloorExceedsBand(salaryFloor, salaryCurrency) {
		ex = append(ex, Exclusion{
			CandidateID: candidateID,
			Gate:        GateSalaryFloor,
			Reason: fmt.Sprintf("candidate salary floor %.0f %s exceeds role ceiling %.0f %s",
				salaryFloor, salaryCurrency, r.SalaryCeiling, r.SalaryCurrency),
		})
	}
	return ex
}

// ScreenMatch evaluates the post-scoring must-have-competency gate: a must-have
// the scorer DID evaluate must reach MinMustHaveScore. A must-have that is
// absent from the breakdown (scorer omission, naming drift, or sparse evidence)
// is treated as uncertainty, not a gap — it never excludes, leaving the call to
// human review. This upholds the no-fabrication invariant: we reject only on the
// positive signal of a present-but-underscored competency. It returns one
// Exclusion per failed must-have; an empty slice means the match clears the gate.
func (r Requirements) ScreenMatch(m *Match) []Exclusion {
	if m == nil {
		return nil
	}
	var ex []Exclusion
	seen := make(map[string]struct{}, len(r.MustHaves))
	for _, name := range r.MustHaves {
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "" {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}

		score, found := scoreForCompetency(m.Breakdown, key)
		if found && score < MinMustHaveScore {
			ex = append(ex, Exclusion{
				CandidateID: m.CandidateID, Gate: GateMustHave,
				Reason: fmt.Sprintf("must-have competency %q scored %.1f/5, below the %.1f minimum",
					name, score, MinMustHaveScore),
			})
		}
	}
	return ex
}

// salaryFloorExceedsBand reports a positive salary conflict: the candidate's
// floor is above the role's ceiling AND both sides are quoted in the same known
// currency. Unknown or differing currencies never gate — a cross-currency
// comparison cannot prove a conflict, so the call goes to human review.
func (r Requirements) salaryFloorExceedsBand(salaryFloor float64, salaryCurrency string) bool {
	if r.SalaryCeiling <= 0 || salaryFloor <= 0 {
		return false
	}
	roleCur, candCur := strings.TrimSpace(r.SalaryCurrency), strings.TrimSpace(salaryCurrency)
	if roleCur == "" || candCur == "" || !strings.EqualFold(roleCur, candCur) {
		return false
	}
	return salaryFloor > r.SalaryCeiling
}

// locationMismatch reports whether a candidate location positively conflicts
// with the role's required location. Matching is token-based (comma/slash/space
// delimited, case-insensitive): a shared token means compatible. Remote roles,
// an unconstrained role location, or an unknown candidate location never
// conflict. Token matching avoids substring false-positives ("Accra" must not
// silently match "Accraville").
func (r Requirements) locationMismatch(location string) bool {
	if r.RemoteAllowed {
		return false
	}
	want, got := tokenSet(r.Location), tokenSet(location)
	if len(want) == 0 || len(got) == 0 {
		return false
	}
	for w := range want {
		if _, ok := got[w]; ok {
			return false
		}
	}
	return true
}

// scoreForCompetency finds a breakdown score for a normalized must-have name,
// matching an exact (normalized) competency name or one that carries the
// must-have as a whole separator-delimited token (so "SQL / Databases" matches
// must-have "SQL"). It returns false when no breakdown entry addresses it.
func scoreForCompetency(breakdown []MatchBreakdownItem, key string) (float64, bool) {
	for _, b := range breakdown {
		comp := strings.ToLower(strings.TrimSpace(b.Competency))
		if comp == key || hasToken(comp, key) {
			return b.Score, true
		}
	}
	return 0, false
}

// hasToken reports whether key appears as a whole token within an
// already-lower-cased text, splitting on separators and whitespace (avoids
// short-substring false matches like "go" inside "algorithms").
func hasToken(text, key string) bool {
	return slices.Contains(splitTokens(text), key)
}

// tokenSet lower-cases and tokenizes a string into a set of distinct tokens.
func tokenSet(s string) map[string]struct{} {
	toks := splitTokens(strings.ToLower(s))
	out := make(map[string]struct{}, len(toks))
	for _, tok := range toks {
		out[tok] = struct{}{}
	}
	return out
}

// splitTokens tokenizes via the shared kernel tokenizer so the must-have gate and
// the no-fabrication grounding check tokenize identically.
func splitTokens(s string) []string {
	return kernel.Tokens(s)
}
