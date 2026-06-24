package candidateagent

import "github.com/xcreativs/caliber/internal/domain/kernel"

// WakeUpView is a value object summarising what happened on a candidate's
// behalf while they were away: new matches found, applications submitted,
// screenings completed, employers expressing interest, and notable highlights.
type WakeUpView struct {
	// NewMatches is the number of newly discovered role matches.
	NewMatches int
	// ApplicationsSubmitted is the number of applications submitted by the agent.
	ApplicationsSubmitted int
	// ScreeningsCompleted is the number of screenings completed.
	ScreeningsCompleted int
	// EmployersInterested is the number of employers that expressed interest.
	EmployersInterested int
	// Highlights are short human-readable notes worth surfacing to the candidate.
	Highlights []string
}

// Validate reports whether the view is well-formed. All counts must be
// non-negative; a negative count indicates a programming error upstream.
func (v WakeUpView) Validate() error {
	if v.NewMatches < 0 {
		return kernel.Invalid("new matches must be non-negative")
	}
	if v.ApplicationsSubmitted < 0 {
		return kernel.Invalid("applications submitted must be non-negative")
	}
	if v.ScreeningsCompleted < 0 {
		return kernel.Invalid("screenings completed must be non-negative")
	}
	if v.EmployersInterested < 0 {
		return kernel.Invalid("employers interested must be non-negative")
	}
	return nil
}
