// Package interview holds the adaptive screening FSM and report card (Flow B).
package interview

// InterviewMode is the channel of the interview.
type InterviewMode int //nolint:revive // domain name fixed by the interview context spec

// Interview modes. Text is the reliable default; voice is a stretch.
const (
	ModeUnspecified InterviewMode = iota
	ModeText
	ModeVoice
)

// Valid reports whether the mode is known and non-zero.
func (m InterviewMode) Valid() bool { return m == ModeText || m == ModeVoice }

// String renders the mode.
func (m InterviewMode) String() string {
	switch m {
	case ModeText:
		return "text"
	case ModeVoice:
		return "voice"
	default:
		return "unspecified"
	}
}

// InterviewVerdict is the overall outcome of a screening interview.
type InterviewVerdict int //nolint:revive // domain name fixed by the interview context spec

// Interview verdicts.
const (
	VerdictUnspecified InterviewVerdict = iota
	VerdictAdvance
	VerdictHold
	VerdictDecline
)

// Valid reports whether the verdict is known and non-zero.
func (v InterviewVerdict) Valid() bool { return v >= VerdictAdvance && v <= VerdictDecline }

// String renders the verdict.
func (v InterviewVerdict) String() string {
	switch v {
	case VerdictAdvance:
		return "advance"
	case VerdictHold:
		return "hold"
	case VerdictDecline:
		return "decline"
	default:
		return "unspecified"
	}
}

// State is the interview finite-state-machine state.
type State int

// FSM states.
const (
	StateUnspecified State = iota
	StateOpen
	StateAsking
	StateScoring
	StateClosed
)

// CanTransition reports whether moving to "to" from the current state is legal.
func (s State) CanTransition(to State) bool {
	switch s {
	case StateOpen:
		return to == StateAsking
	case StateAsking:
		return to == StateAsking || to == StateScoring
	case StateScoring:
		return to == StateClosed
	default:
		return false
	}
}

// String renders the state.
func (s State) String() string {
	switch s {
	case StateOpen:
		return "open"
	case StateAsking:
		return "asking"
	case StateScoring:
		return "scoring"
	case StateClosed:
		return "closed"
	default:
		return "unspecified"
	}
}
