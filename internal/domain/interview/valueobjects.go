package interview

import (
	"strings"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// InterviewTurn is one question/answer exchange in the interview.
type InterviewTurn struct { //nolint:revive // domain name fixed by the interview context spec
	ID            kernel.ID
	Ordinal       int
	Question      string
	Answer        string
	CompetencyTag string
}

// Validate checks the turn is well-formed.
func (t InterviewTurn) Validate() error {
	if t.Ordinal < 1 {
		return kernel.Invalid("interview: turn ordinal must be >= 1")
	}
	if strings.TrimSpace(t.Question) == "" {
		return kernel.Invalid("interview: turn question is required")
	}
	return nil
}

// CompetencyScore is a per-competency score with its supporting evidence.
type CompetencyScore struct {
	Competency string
	Score      float64
	Evidence   string
}

// Validate checks the score is well-formed; every score must cite evidence.
func (c CompetencyScore) Validate() error {
	if strings.TrimSpace(c.Competency) == "" {
		return kernel.Invalid("interview: competency is required")
	}
	if c.Score < 0 || c.Score > 5 {
		return kernel.Invalidf("interview: score for %q must be in [0,5]", c.Competency)
	}
	if strings.TrimSpace(c.Evidence) == "" {
		return kernel.Invalid("interview: every score must cite evidence")
	}
	return nil
}

// ReportCard is the scored, evidence-tagged result of an interview (Appendix A.3).
type ReportCard struct {
	InterviewID         kernel.ID
	RoleID              kernel.ID
	CandidateID         kernel.ID
	Verdict             InterviewVerdict
	Confidence          kernel.Confidence
	Scores              []CompetencyScore
	RecommendedNextStep string
}

// Validate checks the report card is well-formed.
func (r ReportCard) Validate() error {
	if !r.Verdict.Valid() {
		return kernel.Invalid("interview: a valid verdict is required")
	}
	if !r.Confidence.Valid() {
		return kernel.Invalid("interview: a valid confidence is required")
	}
	if len(r.Scores) == 0 {
		return kernel.Invalid("interview: report card needs at least one score")
	}
	for _, s := range r.Scores {
		if err := s.Validate(); err != nil {
			return err
		}
	}
	return nil
}
