// Package interview holds the AI screening interview use-cases (Flow B): an
// adaptive question loop and an evidence-tagged report card. Every report score
// must cite a transcript quote (no fabrication, enforced by the domain).
package interview

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xcreativs/caliber/internal/app"
	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
)

const (
	defaultMaxTurns   = 4
	questionMaxTokens = 512
	reportMaxTokens   = 1024
)

// QuestionSystemPrompt drives adaptive question generation.
const QuestionSystemPrompt = `You are an adaptive technical screening interviewer. Given the role rubric and the
interview so far, ask ONE focused follow-up question that probes an under-assessed rubric competency.
Respond ONLY with JSON: {"question": string, "competency_tag": string} where competency_tag is one of
the rubric competency names.`

// ReportSystemPrompt drives the evidence-tagged report card. No fabrication.
const ReportSystemPrompt = `You score a screening interview against the role rubric. For EACH rubric competency, give a
score 0-5 and an evidence quote taken VERBATIM from the candidate's answers — never invent evidence; if a
competency was not covered, score it low and say so in the evidence. Respond ONLY with JSON:
{"verdict":"advance|hold|decline","confidence":"low|medium|high",
"scores":[{"competency":string,"score":0..5,"evidence":string}],"recommended_next_step":string}.`

// Interviewer runs the screening interview over the domain ports.
type Interviewer struct {
	roles      role.RoleRepository
	interviews interviewdom.InterviewRepository
	llm        app.LLMClient
	maxTurns   int
}

// NewInterviewer wires the use-case. A non-positive maxTurns defaults to 4.
func NewInterviewer(
	roles role.RoleRepository, interviews interviewdom.InterviewRepository, llm app.LLMClient, maxTurns int,
) *Interviewer {
	if maxTurns <= 0 {
		maxTurns = defaultMaxTurns
	}
	return &Interviewer{roles: roles, interviews: interviews, llm: llm, maxTurns: maxTurns}
}

type llmQuestion struct {
	Question      string `json:"question"`
	CompetencyTag string `json:"competency_tag"`
}

type llmReport struct {
	Verdict    string `json:"verdict"`
	Confidence string `json:"confidence"`
	Scores     []struct {
		Competency string  `json:"competency"`
		Score      float64 `json:"score"`
		Evidence   string  `json:"evidence"`
	} `json:"scores"`
	RecommendedNextStep string `json:"recommended_next_step"`
}

// Start opens an interview for the candidate against the role and returns the
// first question.
func (s *Interviewer) Start(
	ctx context.Context, roleID, candidateID kernel.ID, mode interviewdom.InterviewMode,
) (*interviewdom.Interview, *interviewdom.PendingQuestion, error) {
	rl, err := s.roles.ByID(ctx, roleID)
	if err != nil {
		return nil, nil, err
	}
	iv, err := interviewdom.NewInterview(roleID, candidateID, mode)
	if err != nil {
		return nil, nil, err
	}
	if err := iv.Transition(interviewdom.StateAsking); err != nil {
		return nil, nil, err
	}
	if err := s.ask(ctx, rl, iv); err != nil {
		return nil, nil, err
	}
	if err := s.interviews.Create(ctx, iv); err != nil {
		return nil, nil, err
	}
	return iv, iv.Pending, nil
}

// Answer records the candidate's answer and returns either the next question or,
// once the interview is complete, the finished report card.
func (s *Interviewer) Answer(
	ctx context.Context, interviewID kernel.ID, answer string,
) (*interviewdom.PendingQuestion, *interviewdom.ReportCard, error) {
	iv, err := s.interviews.ByID(ctx, interviewID)
	if err != nil {
		return nil, nil, err
	}
	if err := iv.Answer(answer); err != nil {
		return nil, nil, err
	}
	rl, err := s.roles.ByID(ctx, iv.RoleID)
	if err != nil {
		return nil, nil, err
	}

	if len(iv.Turns) >= s.maxTurns {
		if err := s.finish(ctx, rl, iv); err != nil {
			return nil, nil, err
		}
		return nil, iv.Report, nil
	}
	if err := s.ask(ctx, rl, iv); err != nil {
		return nil, nil, err
	}
	if err := s.interviews.Update(ctx, iv); err != nil {
		return nil, nil, err
	}
	return iv.Pending, nil, nil
}

// Report returns a completed interview's report card.
func (s *Interviewer) Report(ctx context.Context, interviewID kernel.ID) (*interviewdom.ReportCard, error) {
	iv, err := s.interviews.ByID(ctx, interviewID)
	if err != nil {
		return nil, err
	}
	if iv.Report == nil {
		return nil, kernel.Invalid("interview: report card is not ready yet")
	}
	return iv.Report, nil
}

// ask generates the next adaptive question and records it as pending.
func (s *Interviewer) ask(ctx context.Context, rl *role.Role, iv *interviewdom.Interview) error {
	resp, err := s.llm.Complete(ctx, app.LLMRequest{
		System: QuestionSystemPrompt, Prompt: questionPrompt(rl, iv), MaxTokens: questionMaxTokens,
	})
	if err != nil {
		return kernel.Wrap(err, kernel.KindInternal, "interview: question generation failed")
	}
	var q llmQuestion
	if uerr := json.Unmarshal([]byte(resp.Text), &q); uerr != nil {
		return kernel.Wrap(uerr, kernel.KindInvalid, "interview: could not parse question output")
	}
	return iv.Ask(q.Question, q.CompetencyTag)
}

// finish scores the transcript and closes the interview.
func (s *Interviewer) finish(ctx context.Context, rl *role.Role, iv *interviewdom.Interview) error {
	resp, err := s.llm.Complete(ctx, app.LLMRequest{
		System: ReportSystemPrompt, Prompt: scorePrompt(rl, iv), MaxTokens: reportMaxTokens,
	})
	if err != nil {
		return kernel.Wrap(err, kernel.KindInternal, "interview: scoring failed")
	}
	var parsed llmReport
	if uerr := json.Unmarshal([]byte(resp.Text), &parsed); uerr != nil {
		return kernel.Wrap(uerr, kernel.KindInvalid, "interview: could not parse report output")
	}
	scores := make([]interviewdom.CompetencyScore, 0, len(parsed.Scores))
	for _, sc := range parsed.Scores {
		scores = append(scores, interviewdom.CompetencyScore{Competency: sc.Competency, Score: sc.Score, Evidence: sc.Evidence})
	}
	card := interviewdom.ReportCard{
		InterviewID:         iv.ID,
		RoleID:              iv.RoleID,
		CandidateID:         iv.CandidateID,
		Verdict:             parseVerdict(parsed.Verdict),
		Confidence:          parseConfidence(parsed.Confidence),
		Scores:              scores,
		RecommendedNextStep: parsed.RecommendedNextStep,
	}
	if err := iv.Transition(interviewdom.StateScoring); err != nil {
		return err
	}
	if err := iv.Complete(card); err != nil {
		return err
	}
	return s.interviews.Update(ctx, iv)
}

func questionPrompt(rl *role.Role, iv *interviewdom.Interview) string {
	var b strings.Builder
	fmt.Fprintf(&b, "ROLE: %s\nRUBRIC:\n", rl.Spec.Title)
	for _, c := range rl.Rubric.Competencies {
		fmt.Fprintf(&b, "- %s\n", c.Name)
	}
	b.WriteString(transcript(iv))
	b.WriteString("Ask the next question.")
	return b.String()
}

func scorePrompt(rl *role.Role, iv *interviewdom.Interview) string {
	var b strings.Builder
	fmt.Fprintf(&b, "ROLE: %s\nRUBRIC:\n", rl.Spec.Title)
	for _, c := range rl.Rubric.Competencies {
		fmt.Fprintf(&b, "- %s\n", c.Name)
	}
	b.WriteString(transcript(iv))
	b.WriteString("Score the interview now.")
	return b.String()
}

func transcript(iv *interviewdom.Interview) string {
	if len(iv.Turns) == 0 {
		return "TRANSCRIPT: (none yet)\n"
	}
	var b strings.Builder
	b.WriteString("TRANSCRIPT:\n")
	for _, t := range iv.Turns {
		fmt.Fprintf(&b, "Q%d (%s): %s\nA: %s\n", t.Ordinal, t.CompetencyTag, t.Question, t.Answer)
	}
	return b.String()
}

func parseVerdict(s string) interviewdom.InterviewVerdict {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "advance":
		return interviewdom.VerdictAdvance
	case "decline":
		return interviewdom.VerdictDecline
	default:
		return interviewdom.VerdictHold
	}
}

func parseConfidence(s string) kernel.Confidence {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "low":
		return kernel.ConfidenceLow
	case "high":
		return kernel.ConfidenceHigh
	default:
		return kernel.ConfidenceMedium
	}
}
