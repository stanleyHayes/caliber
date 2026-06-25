package interview

import "github.com/xcreativs/caliber/internal/domain/kernel"

// Interview is an adaptive screening session driven by the state machine.
type Interview struct {
	ID          kernel.ID
	RoleID      kernel.ID
	CandidateID kernel.ID
	Mode        InterviewMode
	State       State
	Turns       []InterviewTurn
	Report      *ReportCard
}

// NewInterview opens a new interview in the Open state.
func NewInterview(roleID, candidateID kernel.ID, mode InterviewMode) (*Interview, error) {
	if roleID.IsZero() {
		return nil, kernel.Invalid("interview: role id is required")
	}
	if candidateID.IsZero() {
		return nil, kernel.Invalid("interview: candidate id is required")
	}
	if !mode.Valid() {
		return nil, kernel.Invalid("interview: a valid mode is required")
	}
	return &Interview{
		ID:          kernel.NewID(),
		RoleID:      roleID,
		CandidateID: candidateID,
		Mode:        mode,
		State:       StateOpen,
	}, nil
}

// Transition moves the interview to a new state if the move is legal.
func (i *Interview) Transition(to State) error {
	if !i.State.CanTransition(to) {
		return kernel.Invalidf("interview: illegal transition %s -> %s", i.State, to)
	}
	i.State = to
	return nil
}

// AddTurn records a question/answer turn; only allowed while asking.
func (i *Interview) AddTurn(t InterviewTurn) error {
	if i.State != StateAsking {
		return kernel.Invalid("interview: can only add turns while asking")
	}
	if err := t.Validate(); err != nil {
		return err
	}
	i.Turns = append(i.Turns, t)
	return nil
}

// Complete attaches the report card and closes the interview; only from scoring.
func (i *Interview) Complete(card ReportCard) error {
	if i.State != StateScoring {
		return kernel.Invalid("interview: can only complete from the scoring state")
	}
	if err := card.Validate(); err != nil {
		return err
	}
	i.Report = &card
	i.State = StateClosed
	return nil
}
