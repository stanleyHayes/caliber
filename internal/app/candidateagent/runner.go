// Package candidateagent holds the autonomous candidate-agent use-cases (Flow C):
// scan open roles, and for honest strong matches submit applications grounded in
// the candidate's VERIFIED profile. Hard invariant: no fabrication.
package candidateagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/app/prompts"
	agentdom "github.com/xcreativs/caliber/internal/domain/candidateagent"
	"github.com/xcreativs/caliber/internal/domain/guard"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	matchingdom "github.com/xcreativs/caliber/internal/domain/matching"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

const (
	defaultScanLimit = 20
	defaultMinFit    = 0.6
)

// AgentRunner scans the open-role pool on a candidate's behalf.
type AgentRunner struct {
	candidates talent.CandidateRepository
	profiles   talent.TalentProfileRepository
	roles      role.RoleRepository
	apps       agentdom.ApplicationRepository
	llm        app.LLMClient
	minFit     float64
}

// NewAgentRunner wires the use-case.
func NewAgentRunner(
	candidates talent.CandidateRepository,
	profiles talent.TalentProfileRepository,
	roles role.RoleRepository,
	apps agentdom.ApplicationRepository,
	llm app.LLMClient,
) *AgentRunner {
	return &AgentRunner{candidates: candidates, profiles: profiles, roles: roles, apps: apps, llm: llm, minFit: defaultMinFit}
}

type llmAssessment struct {
	FitScore        float64 `json:"fit_score"`
	Apply           bool    `json:"apply"`
	TailoredSummary string  `json:"tailored_summary"`
}

// Run scans up to scanLimit open roles and submits agent-authored applications
// for honest strong matches, returning a wake-up summary. Without a verified
// profile the agent does nothing (it cannot act honestly).
func (r *AgentRunner) Run(ctx context.Context, candidateID kernel.ID, scanLimit int) (agentdom.WakeUpView, error) {
	if scanLimit <= 0 {
		scanLimit = defaultScanLimit
	}
	cand, err := r.candidates.ByID(ctx, candidateID)
	if err != nil {
		return agentdom.WakeUpView{}, err
	}
	profile, err := r.profiles.ByCandidateID(ctx, candidateID)
	if err != nil {
		if kernel.KindOf(err) == kernel.KindNotFound {
			return agentdom.WakeUpView{}, nil
		}
		return agentdom.WakeUpView{}, err
	}
	roles, _, err := r.roles.ListOpen(ctx, kernel.NewPage(1, scanLimit))
	if err != nil {
		return agentdom.WakeUpView{}, err
	}
	return r.scanRoles(ctx, candidateID, cand, profile, roles)
}

// scanRoles considers each eligible role, applying for honest strong matches and
// accumulating the wake-up view (matches, applications, and explainable notes).
func (r *AgentRunner) scanRoles(
	ctx context.Context, candidateID kernel.ID,
	cand *talent.Candidate, profile *talent.TalentProfile, roles []*role.Role,
) (agentdom.WakeUpView, error) {
	view := agentdom.WakeUpView{}
	for _, rl := range roles {
		if !r.eligible(cand, profile, rl) {
			continue
		}
		view.NewMatches++
		applied, note, err := r.consider(ctx, candidateID, profile, rl)
		if err != nil {
			return agentdom.WakeUpView{}, err
		}
		if applied {
			view.ApplicationsSubmitted++
		}
		if note != "" {
			view.Highlights = append(view.Highlights, note)
		}
	}
	return view, nil
}

// eligible gates a role before applying: the candidate must be logistically
// compatible (location/salary) AND their verified profile must already cover the
// role's must-have competencies — the agent never applies where it would have to
// fabricate qualifications.
func (r *AgentRunner) eligible(cand *talent.Candidate, profile *talent.TalentProfile, rl *role.Role) bool {
	req := requirementsFor(rl)
	if len(req.ScreenLogistics(cand.ID, cand.Location, cand.Intake.SalaryFloor, cand.Intake.SalaryCurrency)) > 0 {
		return false
	}
	if matchingdom.ViolatesDealBreaker(cand.Intake.DealBreakers,
		rl.Spec.Title+" "+rl.Spec.Availability+" "+strings.Join(rl.Spec.Responsibilities, " ")) {
		return false // the agent never applies where the candidate declared a deal-breaker
	}
	return profileCoversMustHaves(profile, rl)
}

// consider asks the LLM to assess and (if a strong honest match) drafts and
// submits an agent application grounded in the verified profile.
// consider assesses, applies the no-fabrication guards, and (if honest) submits.
// It returns whether it applied and a short candidate-facing note worth
// surfacing (an application made, or a role skipped because its draft referenced
// unverified skills) — so a guardrail rejection is explainable, never silent.
func (r *AgentRunner) consider(
	ctx context.Context, candidateID kernel.ID, profile *talent.TalentProfile, rl *role.Role,
) (bool, string, error) {
	assessment, err := r.assess(ctx, rl, profile)
	if err != nil {
		return false, "", err
	}
	if !assessment.Apply || assessment.FitScore < r.minFit {
		return false, "", nil
	}
	if grounding := agentdom.CheckGrounding(
		assessment.TailoredSummary, profileCompetencyNames(profile), roleCompetencyNames(rl),
	); !grounding.Grounded {
		// No-fabrication (CAL-071): the summary asserts skills the verified
		// profile does not evidence; do not apply, and surface why.
		return false, fmt.Sprintf("Skipped %q: the drafted summary referenced unverified skills (%s).",
			rl.Title, strings.Join(grounding.Fabricated, ", ")), nil
	}
	application, err := agentdom.NewAgentApplication(rl.ID, candidateID, profile.ID, assessment.TailoredSummary)
	if err != nil {
		// A blank/ungrounded summary fails the no-fabrication invariant: skip, don't apply.
		if kernel.KindOf(err) == kernel.KindInvalid {
			return false, "", nil
		}
		return false, "", err
	}
	if err := application.Submit(); err != nil {
		return false, "", err
	}
	if err := r.apps.Create(ctx, application); err != nil {
		return false, "", err
	}
	return true, fmt.Sprintf("Applied to %q on your behalf.", rl.Title), nil
}

func (r *AgentRunner) assess(ctx context.Context, rl *role.Role, profile *talent.TalentProfile) (llmAssessment, error) {
	assessment, err := app.DecodeJSON[llmAssessment](ctx, r.llm,
		prompts.Get(prompts.IDAgentAssess).Request(assessPrompt(rl, profile)),
		app.DefaultLLMAttempts, "agent: assessment")
	if err != nil {
		return llmAssessment{}, err
	}
	return assessment, nil
}

func requirementsFor(rl *role.Role) matchingdom.Requirements {
	return matchingdom.NewRequirements(
		rl.Spec.Location, rl.Spec.SalaryBand.High, rl.Spec.SalaryBand.Currency, nil)
}

func profileCompetencyNames(p *talent.TalentProfile) []string {
	names := make([]string, 0, len(p.Competencies))
	for _, c := range p.Competencies {
		names = append(names, c.Name)
	}
	return names
}

func roleCompetencyNames(rl *role.Role) []string {
	names := make([]string, 0, len(rl.Rubric.Competencies))
	for _, c := range rl.Rubric.Competencies {
		names = append(names, c.Name)
	}
	return names
}

// profileCoversMustHaves reports whether the verified profile evidences every
// must-have rubric competency at or above the minimum level.
func profileCoversMustHaves(profile *talent.TalentProfile, rl *role.Role) bool {
	cand := make([]matchingdom.CandidateSignal, 0, len(profile.Competencies))
	for _, c := range profile.Competencies {
		cand = append(cand, matchingdom.CandidateSignal{Name: c.Name, Level: c.Level})
	}
	rubric := make([]matchingdom.RubricSignal, 0, len(rl.Rubric.Competencies))
	for _, c := range rl.Rubric.Competencies {
		rubric = append(rubric, matchingdom.RubricSignal{Name: c.Name, Weight: c.Weight, MustHave: c.MustHave})
	}
	// Token-aware, shared with the two-way matcher so Radar and agent agree.
	return matchingdom.CoversMustHaves(rubric, cand)
}

func assessPrompt(rl *role.Role, profile *talent.TalentProfile) string {
	var b strings.Builder
	fmt.Fprintf(&b, "ROLE: %s\nRUBRIC:\n", guard.Sanitize(rl.Spec.Title))
	for _, c := range rl.Rubric.Competencies {
		fmt.Fprintf(&b, "- %s\n", guard.Sanitize(c.Name))
	}
	// Evidence quotes originate from the candidate's CV (untrusted origin), so
	// sanitize and fence them before they re-enter a prompt.
	var prof strings.Builder
	for _, c := range profile.Competencies {
		fmt.Fprintf(&prof, "- %s (level %.1f): %s\n", guard.Sanitize(c.Name), c.Level, guard.Sanitize(c.EvidenceQuote))
	}
	b.WriteString("VERIFIED PROFILE COMPETENCIES:\n")
	b.WriteString(guard.Fence("VERIFIED_PROFILE", prof.String()))
	b.WriteString("\nDecide and draft.")
	return b.String()
}
