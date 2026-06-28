// Package interview holds the AI screening interview use-cases (Flow B): an
// adaptive question loop and an evidence-tagged report card. Every report score
// must cite a transcript quote (no fabrication, enforced by the domain).
package interview

import (
	"context"
	"fmt"
	"strings"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/app/prompts"
	"github.com/xcreativs/caliber/internal/domain/guard"
	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

const (
	defaultMaxTurns = 4
)

// Interviewer runs the screening interview over the domain ports.
type Interviewer struct {
	roles      role.RoleRepository
	interviews interviewdom.InterviewRepository
	llm        app.LLMClient
	maxTurns   int
	profiles   talent.TalentProfileRepository // optional: passport update on completion
}

// Option customizes an Interviewer.
type Option func(*Interviewer)

// WithPassportUpdater marks the candidate's Talent Passport as screened once an
// interview completes (best-effort).
func WithPassportUpdater(profiles talent.TalentProfileRepository) Option {
	return func(s *Interviewer) { s.profiles = profiles }
}

// NewInterviewer wires the use-case. A non-positive maxTurns defaults to 4.
func NewInterviewer(
	roles role.RoleRepository, interviews interviewdom.InterviewRepository, llm app.LLMClient, maxTurns int, opts ...Option,
) *Interviewer {
	if maxTurns <= 0 {
		maxTurns = defaultMaxTurns
	}
	s := &Interviewer{roles: roles, interviews: interviews, llm: llm, maxTurns: maxTurns}
	for _, opt := range opts {
		opt(s)
	}
	return s
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

// CandidateForInterview returns the candidate an interview belongs to, so inbound
// adapters can authorize that the caller owns it (CAL-116 IDOR protection).
func (s *Interviewer) CandidateForInterview(ctx context.Context, interviewID kernel.ID) (kernel.ID, error) {
	iv, err := s.interviews.ByID(ctx, interviewID)
	if err != nil {
		return "", err
	}
	return iv.CandidateID, nil
}

// EmployerForInterview returns the employer who owns the role an interview was
// screened against, so inbound adapters can authorize that a reviewer reading the
// report card owns the role (CAL-116 IDOR protection) — a report card must not be
// readable by employers who never posted the role nor ran the screening.
func (s *Interviewer) EmployerForInterview(ctx context.Context, interviewID kernel.ID) (kernel.ID, error) {
	iv, err := s.interviews.ByID(ctx, interviewID)
	if err != nil {
		return "", err
	}
	rl, err := s.roles.ByID(ctx, iv.RoleID)
	if err != nil {
		return "", err
	}
	return rl.EmployerID, nil
}

// ask generates the next adaptive question and records it as pending.
func (s *Interviewer) ask(ctx context.Context, rl *role.Role, iv *interviewdom.Interview) error {
	q, err := app.DecodeJSON[llmQuestion](ctx, s.llm,
		prompts.Get(prompts.IDInterviewQuestion).Request(questionPrompt(rl, iv)),
		app.DefaultLLMAttempts, "interview: question")
	if err != nil {
		return err
	}
	return iv.Ask(q.Question, q.CompetencyTag)
}

// finish scores the transcript and closes the interview.
func (s *Interviewer) finish(ctx context.Context, rl *role.Role, iv *interviewdom.Interview) error {
	parsed, err := app.DecodeJSON[llmReport](ctx, s.llm,
		prompts.Get(prompts.IDInterviewReport).Request(scorePrompt(rl, iv)),
		app.DefaultLLMAttempts, "interview: report")
	if err != nil {
		return err
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
	if err := s.interviews.Update(ctx, iv); err != nil {
		return err
	}
	s.markScreened(ctx, iv.CandidateID)
	return nil
}

// markScreened advances the candidate's passport to screened, best-effort: a
// missing profile or already-screened passport is a no-op.
func (s *Interviewer) markScreened(ctx context.Context, candidateID kernel.ID) {
	if s.profiles == nil {
		return
	}
	profile, err := s.profiles.ByCandidateID(ctx, candidateID)
	if err != nil {
		return
	}
	if profile.PassportStatus == talent.PassportScreened || profile.PassportStatus == talent.PassportVerified {
		return
	}
	profile.MarkScreened()
	_ = s.profiles.Update(ctx, profile)
}

// honestSignalDirective steers the next question to extract concrete evidence
// when the candidate's last answer was vague or evasive (CAL-063).
const honestSignalDirective = "The candidate's last answer was vague or lacked concrete detail. " +
	"Ask a focused follow-up that presses for a specific, real example — what they personally did, " +
	"the situation, and a measurable outcome — rather than moving on to a new topic.\n"

func questionPrompt(rl *role.Role, iv *interviewdom.Interview) string {
	var b strings.Builder
	fmt.Fprintf(&b, "ROLE: %s\nRUBRIC:\n", guard.Sanitize(rl.Spec.Title))
	for _, c := range rl.Rubric.Competencies {
		fmt.Fprintf(&b, "- %s\n", guard.Sanitize(c.Name))
	}
	b.WriteString(transcript(iv))
	if lastAnswerVague(iv) {
		b.WriteString(honestSignalDirective)
	}
	b.WriteString("Ask the next question.")
	return b.String()
}

// lastAnswerVague reports whether the most recent completed turn's answer reads
// as vague/evasive, so the next question can apply honest-signal pressure.
func lastAnswerVague(iv *interviewdom.Interview) bool {
	if len(iv.Turns) == 0 {
		return false
	}
	return interviewdom.VagueAnswer(iv.Turns[len(iv.Turns)-1].Answer)
}

func scorePrompt(rl *role.Role, iv *interviewdom.Interview) string {
	var b strings.Builder
	fmt.Fprintf(&b, "ROLE: %s\nRUBRIC:\n", guard.Sanitize(rl.Spec.Title))
	for _, c := range rl.Rubric.Competencies {
		fmt.Fprintf(&b, "- %s\n", guard.Sanitize(c.Name))
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
	for _, t := range iv.Turns {
		// Candidate answers are untrusted: sanitize each before it enters the
		// prompt, and fence the whole transcript as data (questions are ours).
		fmt.Fprintf(&b, "Q%d (%s): %s\nA: %s\n", t.Ordinal, guard.Sanitize(t.CompetencyTag), guard.Sanitize(t.Question), guard.Sanitize(t.Answer))
	}
	return "TRANSCRIPT:\n" + guard.Fence("INTERVIEW_TRANSCRIPT", b.String()) + "\n"
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
