package interview

import (
	"testing"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

func validCard() ReportCard {
	return ReportCard{
		Verdict:    VerdictAdvance,
		Confidence: kernel.ConfidenceHigh,
		Scores:     []CompetencyScore{{Competency: "Go", Score: 4.5, Evidence: "explained a goroutine leak"}},
	}
}

func TestModeAndVerdict(t *testing.T) {
	if ModeUnspecified.Valid() || !ModeText.Valid() || !ModeVoice.Valid() {
		t.Error("mode validity")
	}
	for m, want := range map[InterviewMode]string{ModeText: "text", ModeVoice: "voice", ModeUnspecified: "unspecified"} {
		if m.String() != want {
			t.Errorf("mode String(%d)=%q want %q", m, m.String(), want)
		}
	}
	if VerdictUnspecified.Valid() || !VerdictAdvance.Valid() || !VerdictDecline.Valid() {
		t.Error("verdict validity")
	}
	for v, want := range map[InterviewVerdict]string{VerdictAdvance: "advance", VerdictHold: "hold", VerdictDecline: "decline", VerdictUnspecified: "unspecified"} {
		if v.String() != want {
			t.Errorf("verdict String(%d)=%q want %q", v, v.String(), want)
		}
	}
}

func TestStateTransitions(t *testing.T) {
	allowed := map[State][]State{
		StateOpen:    {StateAsking},
		StateAsking:  {StateAsking, StateScoring},
		StateScoring: {StateClosed},
	}
	all := []State{StateOpen, StateAsking, StateScoring, StateClosed, StateUnspecified}
	for from := range allowed {
		ok := map[State]bool{}
		for _, to := range allowed[from] {
			ok[to] = true
			if !from.CanTransition(to) {
				t.Errorf("%s->%s should be allowed", from, to)
			}
		}
		for _, to := range all {
			if !ok[to] && from.CanTransition(to) {
				t.Errorf("%s->%s should be disallowed", from, to)
			}
		}
	}
	if StateClosed.CanTransition(StateOpen) {
		t.Error("closed is terminal")
	}
	for s, want := range map[State]string{StateOpen: "open", StateAsking: "asking", StateScoring: "scoring", StateClosed: "closed", StateUnspecified: "unspecified"} {
		if s.String() != want {
			t.Errorf("state String(%d)=%q want %q", s, s.String(), want)
		}
	}
}

func TestTurnValidate(t *testing.T) {
	if err := (InterviewTurn{Ordinal: 1, Question: "q"}).Validate(); err != nil {
		t.Errorf("valid turn rejected: %v", err)
	}
	if err := (InterviewTurn{Ordinal: 0, Question: "q"}).Validate(); err == nil {
		t.Error("ordinal < 1 should fail")
	}
	if err := (InterviewTurn{Ordinal: 1, Question: " "}).Validate(); err == nil {
		t.Error("blank question should fail")
	}
}

func TestCompetencyScoreValidate(t *testing.T) {
	good := CompetencyScore{Competency: "Go", Score: 3, Evidence: "x"}
	if err := good.Validate(); err != nil {
		t.Errorf("valid score rejected: %v", err)
	}
	for _, bad := range []CompetencyScore{
		{Competency: "", Score: 3, Evidence: "x"},
		{Competency: "Go", Score: 6, Evidence: "x"},
		{Competency: "Go", Score: -1, Evidence: "x"},
		{Competency: "Go", Score: 3, Evidence: " "},
	} {
		if err := bad.Validate(); err == nil {
			t.Errorf("bad score %+v should fail", bad)
		}
	}
}

func TestReportCardValidate(t *testing.T) {
	if err := validCard().Validate(); err != nil {
		t.Errorf("valid card rejected: %v", err)
	}
	c := validCard()
	c.Verdict = VerdictUnspecified
	if err := c.Validate(); err == nil {
		t.Error("invalid verdict should fail")
	}
	c = validCard()
	c.Confidence = kernel.ConfidenceUnknown
	if err := c.Validate(); err == nil {
		t.Error("invalid confidence should fail")
	}
	c = validCard()
	c.Scores = nil
	if err := c.Validate(); err == nil {
		t.Error("no scores should fail")
	}
	c = validCard()
	c.Scores = []CompetencyScore{{Competency: "", Score: 1, Evidence: "x"}}
	if err := c.Validate(); err == nil {
		t.Error("invalid score should fail")
	}
}

func TestInterviewLifecycle(t *testing.T) {
	role, cand := kernel.NewID(), kernel.NewID()
	iv, err := NewInterview(role, cand, ModeText)
	if err != nil {
		t.Fatalf("NewInterview: %v", err)
	}
	if iv.State != StateOpen || iv.ID.IsZero() {
		t.Error("unexpected initial interview")
	}

	// illegal transition + add turn before asking
	if err := iv.Transition(StateScoring); err == nil {
		t.Error("Open->Scoring should fail")
	}
	if err := iv.AddTurn(InterviewTurn{Ordinal: 1, Question: "q"}); err == nil {
		t.Error("AddTurn before asking should fail")
	}

	if err := iv.Transition(StateAsking); err != nil {
		t.Fatalf("Open->Asking: %v", err)
	}
	if err := iv.AddTurn(InterviewTurn{Ordinal: 0, Question: "q"}); err == nil {
		t.Error("invalid turn should fail")
	}
	if err := iv.AddTurn(InterviewTurn{Ordinal: 1, Question: "Tell me about Go"}); err != nil {
		t.Fatalf("AddTurn: %v", err)
	}
	if len(iv.Turns) != 1 {
		t.Error("turn not recorded")
	}

	// complete before scoring
	if err := iv.Complete(validCard()); err == nil {
		t.Error("Complete before scoring should fail")
	}
	if err := iv.Transition(StateScoring); err != nil {
		t.Fatalf("Asking->Scoring: %v", err)
	}
	// invalid card
	bad := validCard()
	bad.Scores = nil
	if err := iv.Complete(bad); err == nil {
		t.Error("Complete with invalid card should fail")
	}
	if err := iv.Complete(validCard()); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if iv.State != StateClosed || iv.Report == nil {
		t.Error("interview not completed")
	}

	// constructor error paths
	if _, err := NewInterview(kernel.ID(""), cand, ModeText); err == nil {
		t.Error("zero role should fail")
	}
	if _, err := NewInterview(role, kernel.ID(""), ModeText); err == nil {
		t.Error("zero candidate should fail")
	}
	if _, err := NewInterview(role, cand, ModeUnspecified); err == nil {
		t.Error("invalid mode should fail")
	}
}
