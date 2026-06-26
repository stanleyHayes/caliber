package candidateagent

import (
	"slices"
	"strings"
	"unicode"
)

// GroundingResult reports whether an agent-authored summary stays within
// verified profile content, naming any fabricated claims found.
type GroundingResult struct {
	// Grounded is true when the summary asserts no skill the profile lacks.
	Grounded bool
	// Fabricated lists the role competencies the summary names but the profile
	// does not evidence — skills the agent would be claiming without proof.
	Fabricated []string
}

// CheckGrounding enforces the no-fabrication invariant on agent OUTPUT (CAL-071):
// a tailored application summary may surface only competencies the verified
// profile evidences. It flags any role competency the summary asserts that the
// profile does not cover — the agent claiming a skill the candidate cannot back
// up. Coverage mirrors the must-have gate (exact name or whole-token match, so
// "SQL / Databases" covers "SQL"); a competency is "asserted" only when its full
// name appears as a contiguous run of word tokens in the summary (token, not
// substring, matching — so "Go" is not found inside "ago"/"going"). An empty
// summary or empty role rubric is vacuously grounded.
func CheckGrounding(summary string, profileCompetencies, roleCompetencies []string) GroundingResult {
	summaryTokens := tokenize(summary)
	var fabricated []string
	for _, rc := range roleCompetencies {
		key := strings.ToLower(strings.TrimSpace(rc))
		if key == "" || coversCompetency(profileCompetencies, key) {
			continue
		}
		if mentions(summaryTokens, tokenize(rc)) {
			fabricated = append(fabricated, rc)
		}
	}
	return GroundingResult{Grounded: len(fabricated) == 0, Fabricated: fabricated}
}

// coversCompetency reports whether any profile competency matches key exactly or
// carries it as a whole token (mirrors the must-have coverage gate).
func coversCompetency(profileCompetencies []string, key string) bool {
	for _, pc := range profileCompetencies {
		if strings.ToLower(strings.TrimSpace(pc)) == key || slices.Contains(tokenize(pc), key) {
			return true
		}
	}
	return false
}

// mentions reports whether want appears as a contiguous run of word tokens in
// have (whole-token phrase match, avoiding short-substring false positives).
func mentions(have, want []string) bool {
	if len(want) == 0 {
		return false
	}
	for start := 0; start+len(want) <= len(have); start++ {
		matched := true
		for j := range want {
			if have[start+j] != want[j] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

// tokenize lower-cases s and splits it into alphanumeric word tokens.
func tokenize(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
}
