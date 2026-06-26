// Package dashboard holds the Talent Radar read-model use-cases (EPIC-11): it
// aggregates the candidate pool, supply/demand, the time-to-shortlist headline,
// and match alerts across the other contexts.
package dashboard

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	matchingdom "github.com/xcreativs/caliber/internal/domain/matching"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

// baselineHours is the manual "weeks to shortlist" reference (3 working weeks).
const baselineHours = 504.0

// demoCurrentHours is the platform's representative time-to-shortlist for the
// closing headline (weeks -> hours). It is a demo constant until per-role
// timing is tracked (CAL-079).
const demoCurrentHours = 2.0

// Two-way alert tuning. A pair must clear this fit to be alert-worthy; the scans
// bound how much of the open-role set and candidate pool a single feed builds.
const (
	strongFitThreshold = 0.7
	alertRoleScanLimit = 50
	alertCandidateScan = 200
)

// Two-way match alert type identifiers (CAL-078).
const (
	AlertCandidateForRole = "candidate_for_role"
	AlertRoleForCandidate = "role_for_candidate"
)

// PoolCandidate is a candidate row in the live pool view.
type PoolCandidate struct {
	CandidateID    kernel.ID
	Name           string
	PassportStatus talent.PassportStatus
	HeadlineScore  float64 // 0..1, derived from verified competency levels
}

// SupplyDemandItem is the open-roles-vs-available-candidates snapshot for a
// role family (seniority band).
type SupplyDemandItem struct {
	RoleFamily          string
	OpenRoles           int
	AvailableCandidates int
	Gap                 int
}

// MatchAlert is a two-way match notification.
type MatchAlert struct {
	ID          kernel.ID
	Type        string
	RoleID      kernel.ID
	CandidateID kernel.ID
	Message     string
}

// TimeToShortlist is the headline metric (manual weeks vs platform hours).
type TimeToShortlist struct {
	BaselineHours     float64
	CurrentHours      float64
	ImprovementFactor float64
}

// Aggregator builds the Talent Radar read models from the domain repositories.
type Aggregator struct {
	candidates talent.CandidateRepository
	profiles   talent.TalentProfileRepository
	users      identity.UserRepository
	roles      role.RoleRepository
}

// NewAggregator wires the read-model use-case.
func NewAggregator(
	candidates talent.CandidateRepository,
	profiles talent.TalentProfileRepository,
	users identity.UserRepository,
	roles role.RoleRepository,
) *Aggregator {
	return &Aggregator{candidates: candidates, profiles: profiles, users: users, roles: roles}
}

// Pool returns a paginated, enriched view of the candidate pool. Per-candidate
// user/profile lookups are best-effort: a missing name or profile yields a
// partial row rather than failing the whole view.
func (a *Aggregator) Pool(ctx context.Context, page kernel.Page) ([]PoolCandidate, int64, error) {
	candidates, total, err := a.candidates.List(ctx, page)
	if err != nil {
		return nil, 0, err
	}
	out := make([]PoolCandidate, 0, len(candidates))
	for _, c := range candidates {
		row := PoolCandidate{CandidateID: c.ID, PassportStatus: talent.PassportUnset}
		if u, uerr := a.users.ByID(ctx, c.UserID); uerr == nil {
			row.Name = u.Name
		}
		if p, perr := a.profiles.ByCandidateID(ctx, c.ID); perr == nil {
			row.PassportStatus = p.PassportStatus
			row.HeadlineScore = headlineScore(p)
		}
		out = append(out, row)
	}
	return out, total, nil
}

// SupplyDemand snapshots open roles by seniority band against the pool size.
func (a *Aggregator) SupplyDemand(ctx context.Context) ([]SupplyDemandItem, error) {
	roles, _, err := a.roles.ListOpen(ctx, kernel.NewPage(1, kernel.MaxPageSize))
	if err != nil {
		return nil, err
	}
	_, totalCandidates, err := a.candidates.List(ctx, kernel.NewPage(1, 1))
	if err != nil {
		return nil, err
	}
	byFamily := map[string]int{}
	for _, rl := range roles {
		byFamily[rl.Spec.Seniority.String()]++
	}
	items := make([]SupplyDemandItem, 0, len(byFamily))
	for family, open := range byFamily {
		items = append(items, SupplyDemandItem{
			RoleFamily:          family,
			OpenRoles:           open,
			AvailableCandidates: int(totalCandidates),
			Gap:                 open - int(totalCandidates),
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].RoleFamily < items[j].RoleFamily })
	return items, nil
}

// Alerts returns paginated two-way match alerts (CAL-078) sourced from two-way
// matching (CAL-053). For every passive candidate it computes a deterministic,
// bias-safe structural fit against each open role and emits a "strong candidate
// for an open role" alert per strong pair, plus one "new role fits this
// candidate" alert for the candidate's best fit. Alert IDs are deterministic
// (type:role:candidate) so the same match is stable across refreshes.
func (a *Aggregator) Alerts(ctx context.Context, page kernel.Page) ([]MatchAlert, int64, error) {
	open, _, err := a.roles.ListOpen(ctx, kernel.NewPage(1, alertRoleScanLimit))
	if err != nil {
		return nil, 0, err
	}
	candidates, _, err := a.candidates.List(ctx, kernel.NewPage(1, alertCandidateScan))
	if err != nil {
		return nil, 0, err
	}
	all := a.collectAlerts(ctx, candidates, open)
	return pageAlerts(all, page), int64(len(all)), nil
}

// candidateAlerts builds every strong-pair alert for one candidate and a single
// best-fit "new role" alert. A pair is emitted only when the candidate clears
// the role's logistics and meets every must-have at strong fit.
func candidateAlerts(c *talent.Candidate, name string, profile *talent.TalentProfile, open []*role.Role) []MatchAlert {
	signals := candidateSignalsFrom(profile)
	var out []MatchAlert
	var best *role.Role
	var bestFit matchingdom.Fit
	for _, rl := range open {
		fit, ok := strongFit(c, signals, rl)
		if !ok {
			continue
		}
		out = append(out, candidateForRoleAlert(c, name, rl, fit))
		if best == nil || fit.Score > bestFit.Score {
			best, bestFit = rl, fit
		}
	}
	if best != nil {
		out = append(out, roleForCandidateAlert(c, name, best, bestFit))
	}
	return out
}

// strongFit returns the candidate's fit for a role when it is alert-worthy:
// logistically compatible, no protected ranking signal, all must-haves met, and
// at or above the strong-fit threshold.
func strongFit(c *talent.Candidate, signals []matchingdom.CandidateSignal, rl *role.Role) (matchingdom.Fit, bool) {
	if !roleLogisticsClear(c, rl) {
		return matchingdom.Fit{}, false
	}
	rubric := rubricSignalsFrom(rl)
	if matchingdom.EnsureBiasSafe(signalNamesFrom(rubric)) != nil {
		return matchingdom.Fit{}, false
	}
	fit := matchingdom.ComputeFit(rubric, signals)
	if !fit.MustHavesMet || fit.Score < strongFitThreshold {
		return matchingdom.Fit{}, false
	}
	return fit, true
}

func candidateForRoleAlert(c *talent.Candidate, name string, rl *role.Role, fit matchingdom.Fit) MatchAlert {
	return MatchAlert{
		ID:          alertID(AlertCandidateForRole, rl.ID, c.ID),
		Type:        AlertCandidateForRole,
		RoleID:      rl.ID,
		CandidateID: c.ID,
		Message:     fmt.Sprintf("New strong candidate for %q: %s matches %d%% on the rubric.", rl.Spec.Title, name, pct(fit.Score)),
	}
}

func roleForCandidateAlert(c *talent.Candidate, name string, rl *role.Role, fit matchingdom.Fit) MatchAlert {
	return MatchAlert{
		ID:          alertID(AlertRoleForCandidate, rl.ID, c.ID),
		Type:        AlertRoleForCandidate,
		RoleID:      rl.ID,
		CandidateID: c.ID,
		Message:     fmt.Sprintf("New role fits %s: %q (%d%% match).", name, rl.Spec.Title, pct(fit.Score)),
	}
}

func alertID(alertType string, roleID, candidateID kernel.ID) kernel.ID {
	return kernel.ID(alertType + ":" + roleID.String() + ":" + candidateID.String())
}

func pct(score float64) int { return int(score*100 + 0.5) }

func candidateSignalsFrom(p *talent.TalentProfile) []matchingdom.CandidateSignal {
	out := make([]matchingdom.CandidateSignal, 0, len(p.Competencies))
	for _, c := range p.Competencies {
		out = append(out, matchingdom.CandidateSignal{Name: c.Name, Level: c.Level})
	}
	return out
}

func rubricSignalsFrom(rl *role.Role) []matchingdom.RubricSignal {
	out := make([]matchingdom.RubricSignal, 0, len(rl.Rubric.Competencies))
	for _, c := range rl.Rubric.Competencies {
		out = append(out, matchingdom.RubricSignal{Name: c.Name, Weight: c.Weight, MustHave: c.MustHave})
	}
	return out
}

func signalNamesFrom(rs []matchingdom.RubricSignal) []string {
	out := make([]string, 0, len(rs))
	for _, s := range rs {
		out = append(out, s.Name)
	}
	return out
}

// roleLogisticsClear reports whether the candidate passes a role's pre-scoring
// logistical gates (work location, salary floor) — never protected attributes.
func roleLogisticsClear(c *talent.Candidate, rl *role.Role) bool {
	req := matchingdom.NewRequirements(
		rl.Spec.Location, rl.Spec.Availability,
		rl.Spec.SalaryBand.High, rl.Spec.SalaryBand.Currency, nil)
	return len(req.ScreenLogistics(c.ID, c.Location, c.Intake.SalaryFloor, c.Intake.SalaryCurrency)) == 0
}

func pageAlerts(all []MatchAlert, page kernel.Page) []MatchAlert {
	start := min(page.Offset(), len(all))
	end := min(start+page.Limit(), len(all))
	return all[start:end]
}

// TimeToShortlist returns the headline weeks-to-hours metric.
func (a *Aggregator) TimeToShortlist(_ context.Context) TimeToShortlist {
	current := demoCurrentHours
	return TimeToShortlist{BaselineHours: baselineHours, CurrentHours: current, ImprovementFactor: baselineHours / current}
}

// headlineScore is the mean verified competency level normalized to 0..1.
func headlineScore(p *talent.TalentProfile) float64 {
	if len(p.Competencies) == 0 {
		return 0
	}
	var sum float64
	for _, c := range p.Competencies {
		sum += c.Level
	}
	return (sum / float64(len(p.Competencies))) / 5.0
}

func (a *Aggregator) collectAlerts(ctx context.Context, candidates []*talent.Candidate, open []*role.Role) []MatchAlert {
	var alerts []MatchAlert
	for _, c := range candidates {
		profile, err := a.profiles.ByCandidateID(ctx, c.ID)
		if err != nil {
			continue // no verified profile yet — nothing to match on
		}
		alerts = append(alerts, candidateAlerts(c, a.candidateName(ctx, c), profile, open)...)
	}
	return alerts
}

func (a *Aggregator) candidateName(ctx context.Context, c *talent.Candidate) string {
	if u, err := a.users.ByID(ctx, c.UserID); err == nil && strings.TrimSpace(u.Name) != "" {
		return u.Name
	}
	return "A candidate"
}
