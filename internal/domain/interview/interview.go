package interview

import (
	"strings"
	"time"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// MaxAnswerLen bounds a candidate's answer (CAL-111): the answer is untrusted
// text that is transcribed and sent to the model for scoring, so it is
// length-capped at the domain boundary to cap cost and resource use.
const MaxAnswerLen = 8000

// PendingQuestion is a question that has been asked but not yet answered.
type PendingQuestion struct {
	Ordinal       int
	Text          string
	CompetencyTag string
}

// Interview is an adaptive screening session driven by the state machine.
type Interview struct {
	ID          kernel.ID
	RoleID      kernel.ID
	CandidateID kernel.ID
	Mode        InterviewMode
	State       State
	Turns       []InterviewTurn
	Pending     *PendingQuestion
	Report      *ReportCard
	StartedAt   time.Time // set by the use-case when the interview begins
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

// Ask records the next question as pending (awaiting an answer). It is valid
// only while asking and when no question is already outstanding.
func (i *Interview) Ask(text, competencyTag string) error {
	if i.State != StateAsking {
		return kernel.Invalid("interview: can only ask a question while asking")
	}
	if i.Pending != nil {
		return kernel.Invalid("interview: a question is already awaiting an answer")
	}
	if strings.TrimSpace(text) == "" {
		return kernel.Invalid("interview: question text is required")
	}
	i.Pending = &PendingQuestion{Ordinal: len(i.Turns) + 1, Text: text, CompetencyTag: competencyTag}
	return nil
}

// Answer records the answer to the pending question as a completed turn and
// clears the pending question.
func (i *Interview) Answer(answer string) error {
	if i.Pending == nil {
		return kernel.Invalid("interview: no question is awaiting an answer")
	}
	if len([]rune(answer)) > MaxAnswerLen {
		return kernel.Invalidf("interview: answer exceeds %d characters", MaxAnswerLen)
	}
	turn := InterviewTurn{
		ID:            kernel.NewID(),
		Ordinal:       i.Pending.Ordinal,
		Question:      i.Pending.Text,
		Answer:        answer,
		CompetencyTag: i.Pending.CompetencyTag,
	}
	if err := i.AddTurn(turn); err != nil {
		return err
	}
	i.Pending = nil
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
