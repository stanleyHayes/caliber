// Package matching is the matching bounded context: it models explainable,
// ranked matches between roles and candidates (see Appendix A.2).
//
// The context is bias-safe by construction: scoring and ranking inputs MUST
// never include protected attributes. Callers feeding signals into a ranking
// pipeline MUST validate the signal keys with EnsureBiasSafe before scoring;
// any protected attribute causes a kernel.Invalid error.
//
// This package is pure domain code: it depends only on the shared kernel and
// the Go standard library, and references sibling entities (roles, candidates)
// by kernel.ID rather than importing their packages.
package matching

import (
	"strings"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// MatchBreakdownItem is a value object describing a single competency's
// contribution to an overall match, together with supporting evidence.
type MatchBreakdownItem struct {
	// Competency is the name of the scored competency (non-empty).
	Competency string
	// Score is the competency score on a 0..5 scale.
	Score float64
	// Evidence is human-readable supporting evidence for the score.
	Evidence string
}

// Validate reports whether the breakdown item is well-formed: the competency
// must be non-empty and the score must lie within the inclusive range [0, 5].
func (i MatchBreakdownItem) Validate() error {
	if strings.TrimSpace(i.Competency) == "" {
		return kernel.Invalid("matching: breakdown competency must not be empty")
	}
	if i.Score < 0 || i.Score > 5 {
		return kernel.Invalidf("matching: breakdown score %.2f out of range [0,5]", i.Score)
	}
	return nil
}

// Match is the aggregate root of the matching context: an explainable, ranked
// pairing of a role and a candidate with an overall score, confidence level,
// per-competency breakdown, narrative rationale, and watch-outs.
type Match struct {
	// ID uniquely identifies the match.
	ID kernel.ID
	// RoleID references the role being matched against.
	RoleID kernel.ID
	// CandidateID references the candidate being matched.
	CandidateID kernel.ID
	// OverallScore is the normalized overall score in the inclusive range [0,1].
	OverallScore float64
	// Confidence is the coarse confidence attached to the match.
	Confidence kernel.Confidence
	// Breakdown holds the per-competency contributions to the score.
	Breakdown []MatchBreakdownItem
	// Rationale is a narrative explanation of the match.
	Rationale string
	// WatchOuts lists caveats reviewers should be aware of.
	WatchOuts []string
	// ThinEvidence flags matches computed from sparse evidence.
	ThinEvidence bool
}

// NewMatch validates its inputs and constructs a Match. It returns a
// kernel.Invalid error when the role or candidate ID is zero, when the overall
// score falls outside [0,1], when the confidence is not a valid level, or when
// any breakdown item is invalid.
func NewMatch(
	roleID, candidateID kernel.ID,
	overall float64,
	conf kernel.Confidence,
	breakdown []MatchBreakdownItem,
	rationale string,
	watchOuts []string,
	thinEvidence bool,
) (*Match, error) {
	if roleID.IsZero() {
		return nil, kernel.Invalid("matching: roleID must not be zero")
	}
	if candidateID.IsZero() {
		return nil, kernel.Invalid("matching: candidateID must not be zero")
	}
	if overall < 0 || overall > 1 {
		return nil, kernel.Invalidf("matching: overall score %.4f out of range [0,1]", overall)
	}
	if !conf.Valid() {
		return nil, kernel.Invalidf("matching: confidence %q is not valid", conf.String())
	}
	for idx := range breakdown {
		if err := breakdown[idx].Validate(); err != nil {
			return nil, err
		}
	}

	items := make([]MatchBreakdownItem, len(breakdown))
	copy(items, breakdown)

	outs := make([]string, len(watchOuts))
	copy(outs, watchOuts)

	return &Match{
		ID:           kernel.NewID(),
		RoleID:       roleID,
		CandidateID:  candidateID,
		OverallScore: overall,
		Confidence:   conf,
		Breakdown:    items,
		Rationale:    rationale,
		WatchOuts:    outs,
		ThinEvidence: thinEvidence,
	}, nil
}
