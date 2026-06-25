package matching

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	matchingdom "github.com/xcreativs/caliber/internal/domain/matching"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

const scoringMaxTokens = 1024

// ScoringSystemPrompt instructs the model to score a candidate against a rubric.
const ScoringSystemPrompt = `You score a candidate against a role rubric. Respond ONLY with JSON:
{"overall_score":0..1,"confidence":"low|medium|high","breakdown":[{"competency":string,"score":0..5,"evidence":string}],
"rationale":string,"watch_outs":[string],"thin_evidence":bool}. Score only on the rubric competencies and the
candidate's evidence — never on protected attributes.`

// ShortlistResult is the outcome of a shortlist run: the surviving ranked
// matches and the candidates removed by hard filters, each with a reason.
type ShortlistResult struct {
	Matches    []*matchingdom.Match
	Exclusions []matchingdom.Exclusion
}

// Shortlister produces an explainable ranked shortlist for a role (Flow A):
// vector recall -> hard filters -> rubric-based LLM scoring -> hard filters ->
// ranked, persisted Matches.
type Shortlister struct {
	roles      role.RoleRepository
	candidates talent.CandidateRepository
	profiles   talent.TalentProfileRepository
	recaller   CandidateRecaller
	embedder   app.Embedder
	scorer     app.LLMClient
	matches    matchingdom.MatchRepository
}

// NewShortlister wires the use-case.
func NewShortlister(
	roles role.RoleRepository,
	candidates talent.CandidateRepository,
	profiles talent.TalentProfileRepository,
	recaller CandidateRecaller,
	embedder app.Embedder,
	scorer app.LLMClient,
	matches matchingdom.MatchRepository,
) *Shortlister {
	return &Shortlister{
		roles:      roles,
		candidates: candidates,
		profiles:   profiles,
		recaller:   recaller,
		embedder:   embedder,
		scorer:     scorer,
		matches:    matches,
	}
}

type llmScore struct {
	OverallScore float64        `json:"overall_score"`
	Confidence   string         `json:"confidence"`
	Breakdown    []llmScoreItem `json:"breakdown"`
	Rationale    string         `json:"rationale"`
	WatchOuts    []string       `json:"watch_outs"`
	ThinEvidence bool           `json:"thin_evidence"`
}

type llmScoreItem struct {
	Competency string  `json:"competency"`
	Score      float64 `json:"score"`
	Evidence   string  `json:"evidence"`
}

// GenerateShortlist recalls candidates for the role, applies the hard filters,
// scores survivors against the rubric, ranks by overall fit, persists the
// surviving matches, and returns them alongside any filter exclusions.
func (s *Shortlister) GenerateShortlist(ctx context.Context, roleID kernel.ID, limit int) (*ShortlistResult, error) {
	rl, err := s.roles.ByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	// Bias-safety: the ranking and gating signals are the rubric competencies only.
	if err := matchingdom.EnsureBiasSafe(competencyNames(rl.Rubric)); err != nil {
		return nil, err
	}

	emb, err := s.embedder.Embed(ctx, roleText(rl))
	if err != nil {
		return nil, err
	}
	candidateIDs, err := s.recaller.Recall(ctx, emb, limit)
	if err != nil {
		return nil, err
	}

	result, err := s.screenAndScore(ctx, rl, requirementsFor(rl), candidateIDs)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(result.Matches, func(i, j int) bool {
		return result.Matches[i].OverallScore > result.Matches[j].OverallScore
	})
	return result, s.persist(ctx, result.Matches)
}

// screenAndScore evaluates every recalled candidate through the hard filters
// and scores the survivors, collecting matches and exclusions.
func (s *Shortlister) screenAndScore(
	ctx context.Context, rl *role.Role, req matchingdom.Requirements, candidateIDs []kernel.ID,
) (*ShortlistResult, error) {
	res := &ShortlistResult{Matches: make([]*matchingdom.Match, 0, len(candidateIDs))}
	for _, cid := range candidateIDs {
		m, excluded, err := s.evaluateCandidate(ctx, rl, req, cid)
		if err != nil {
			return nil, err
		}
		switch {
		case len(excluded) > 0:
			res.Exclusions = append(res.Exclusions, excluded...)
		case m != nil:
			res.Matches = append(res.Matches, m)
		}
	}
	return res, nil
}

// evaluateCandidate screens one recalled candidate through the pre-scoring
// (logistical) gates and, if it clears them, scores it and applies the
// post-scoring (must-have) gate. It returns the surviving match (nil when the
// candidate is gated out or its data has gone missing) and any exclusions.
func (s *Shortlister) evaluateCandidate(
	ctx context.Context, rl *role.Role, req matchingdom.Requirements, cid kernel.ID,
) (*matchingdom.Match, []matchingdom.Exclusion, error) {
	cand, err := s.candidates.ByID(ctx, cid)
	if err != nil {
		return nil, nil, skipIfMissing(err)
	}
	if ex := req.ScreenLogistics(cid, cand.Location, cand.Intake.SalaryFloor, cand.Intake.SalaryCurrency); len(ex) > 0 {
		return nil, ex, nil
	}

	profile, err := s.profiles.ByCandidateID(ctx, cid)
	if err != nil {
		return nil, nil, skipIfMissing(err)
	}
	m, err := s.score(ctx, rl, profile)
	if err != nil {
		return nil, nil, err
	}
	if ex := req.ScreenMatch(m); len(ex) > 0 {
		return nil, ex, nil
	}
	return m, nil, nil
}

// skipIfMissing turns a NotFound into a nil error (the caller skips the
// candidate) and propagates anything else.
func skipIfMissing(err error) error {
	if kernel.KindOf(err) == kernel.KindNotFound {
		return nil
	}
	return err
}

// persist upserts each surviving match.
func (s *Shortlister) persist(ctx context.Context, matches []*matchingdom.Match) error {
	for _, m := range matches {
		if err := s.matches.Upsert(ctx, m); err != nil {
			return err
		}
	}
	return nil
}

func (s *Shortlister) score(ctx context.Context, rl *role.Role, profile *talent.TalentProfile) (*matchingdom.Match, error) {
	resp, err := s.scorer.Complete(ctx, app.LLMRequest{
		System:    ScoringSystemPrompt,
		Prompt:    scoringPrompt(rl, profile),
		MaxTokens: scoringMaxTokens,
	})
	if err != nil {
		return nil, kernel.Wrap(err, kernel.KindInternal, "matching: scoring failed")
	}
	var parsed llmScore
	if uerr := json.Unmarshal([]byte(resp.Text), &parsed); uerr != nil {
		return nil, kernel.Wrap(uerr, kernel.KindInvalid, "matching: could not parse scoring output")
	}
	breakdown := make([]matchingdom.MatchBreakdownItem, 0, len(parsed.Breakdown))
	for _, b := range parsed.Breakdown {
		breakdown = append(breakdown, matchingdom.MatchBreakdownItem{Competency: b.Competency, Score: b.Score, Evidence: b.Evidence})
	}
	return matchingdom.NewMatch(rl.ID, profile.CandidateID, clamp01(parsed.OverallScore),
		confidence(parsed.Confidence), breakdown, parsed.Rationale, parsed.WatchOuts, parsed.ThinEvidence)
}

// requirementsFor derives the bias-safe hard constraints from a role's spec and
// rubric (location, salary ceiling, must-have competencies).
func requirementsFor(rl *role.Role) matchingdom.Requirements {
	return matchingdom.Requirements{
		Location:       rl.Spec.Location,
		RemoteAllowed:  isRemote(rl.Spec),
		SalaryCeiling:  rl.Spec.SalaryBand.High,
		SalaryCurrency: rl.Spec.SalaryBand.Currency,
		MustHaves:      mustHaveNames(rl.Rubric),
	}
}

// isRemote detects a remote-friendly role from the available free-text fields
// (there is no structured work-arrangement field yet); a remote role disables
// the location gate.
func isRemote(spec role.RoleSpec) bool {
	return strings.Contains(strings.ToLower(spec.Location+" "+spec.Availability), "remote")
}

func mustHaveNames(r role.Rubric) []string {
	names := make([]string, 0, len(r.Competencies))
	for _, c := range r.Competencies {
		if c.MustHave {
			names = append(names, c.Name)
		}
	}
	return names
}

func roleText(rl *role.Role) string {
	parts := make([]string, 0, 2+len(rl.Spec.Responsibilities)+len(rl.Spec.MustHaves))
	parts = append(parts, rl.Spec.Title, rl.Spec.Location)
	parts = append(parts, rl.Spec.Responsibilities...)
	parts = append(parts, rl.Spec.MustHaves...)
	return strings.Join(parts, " ")
}

func scoringPrompt(rl *role.Role, p *talent.TalentProfile) string {
	var b strings.Builder
	fmt.Fprintf(&b, "ROLE: %s\nRUBRIC:\n", rl.Spec.Title)
	for _, c := range rl.Rubric.Competencies {
		fmt.Fprintf(&b, "- %s (weight %.2f, must_have %v)\n", c.Name, c.Weight, c.MustHave)
	}
	b.WriteString("CANDIDATE COMPETENCIES:\n")
	for _, c := range p.Competencies {
		fmt.Fprintf(&b, "- %s (level %.1f): %s\n", c.Name, c.Level, c.EvidenceQuote)
	}
	return b.String()
}

func competencyNames(r role.Rubric) []string {
	names := make([]string, 0, len(r.Competencies))
	for _, c := range r.Competencies {
		names = append(names, c.Name)
	}
	return names
}

func confidence(s string) kernel.Confidence {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "low":
		return kernel.ConfidenceLow
	case "high":
		return kernel.ConfidenceHigh
	default:
		return kernel.ConfidenceMedium
	}
}

func clamp01(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}
