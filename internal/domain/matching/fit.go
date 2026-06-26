package matching

import "strings"

// maxCompetencyLevel is the top of the 0..5 evidenced-competency scale.
const maxCompetencyLevel = 5.0

// RubricSignal is one bias-safe ranking input derived from a role's rubric: a
// competency name, its weight, and whether it is a must-have. Protected
// attributes must never appear here (validate names with EnsureBiasSafe).
type RubricSignal struct {
	Name     string
	Weight   float64
	MustHave bool
}

// CandidateSignal is one evidenced competency on a candidate's profile: a name
// and a 0..5 level. It carries no identity or protected-attribute data.
type CandidateSignal struct {
	Name  string
	Level float64
}

// Fit is an explainable two-way fit between a role rubric and a candidate,
// computed only from competency signals (bias-safe by construction). Score is a
// weight-normalized coverage in [0,1]; MustHavesMet reports whether every
// must-have is evidenced at or above MinMustHaveScore; Covered and Missing make
// the result explainable.
type Fit struct {
	Score        float64
	MustHavesMet bool
	Covered      []string
	Missing      []string
}

// ComputeFit scores how well a candidate's competencies cover a role's rubric.
// It is deterministic, bias-safe (competency signals only), and explainable —
// suitable for passive two-way matching at scale (Talent Radar) where an LLM
// assessment per pair would be needless cost. Matching is exact-or-token on
// normalized names, mirroring the must-have gate (so "SQL" matches a candidate
// competency "SQL / Databases"). An empty rubric yields a zero score with
// must-haves vacuously met.
func ComputeFit(rubric []RubricSignal, candidate []CandidateSignal) Fit {
	fit := Fit{MustHavesMet: true}
	var totalWeight float64
	for _, rs := range rubric {
		key := strings.ToLower(strings.TrimSpace(rs.Name))
		if key == "" {
			continue
		}
		totalWeight += rs.Weight
		level, found := candidateLevel(candidate, key)
		if found {
			fit.Score += rs.Weight * clampUnit(level/maxCompetencyLevel)
			fit.Covered = append(fit.Covered, rs.Name)
		}
		if rs.MustHave && (!found || level < MinMustHaveScore) {
			fit.MustHavesMet = false
			fit.Missing = append(fit.Missing, rs.Name)
		}
	}
	if totalWeight > 0 {
		fit.Score /= totalWeight
	}
	return fit
}

// candidateLevel finds a candidate's level for a normalized rubric key, matching
// an exact normalized name or one that carries the key as a whole token.
func candidateLevel(candidate []CandidateSignal, key string) (float64, bool) {
	for _, c := range candidate {
		name := strings.ToLower(strings.TrimSpace(c.Name))
		if name == key || hasToken(name, key) {
			return c.Level, true
		}
	}
	return 0, false
}

func clampUnit(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}
