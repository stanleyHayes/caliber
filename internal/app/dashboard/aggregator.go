// Package dashboard holds the Talent Radar read-model use-cases (EPIC-11): it
// aggregates the candidate pool, supply/demand, the time-to-shortlist headline,
// and match alerts across the other contexts.
package dashboard

import (
	"context"
	"sort"

	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

// baselineHours is the manual "weeks to shortlist" reference (3 working weeks).
const baselineHours = 504.0

// demoCurrentHours is the platform's representative time-to-shortlist for the
// closing headline (weeks -> hours). It is a demo constant until per-role
// timing is tracked (CAL-079).
const demoCurrentHours = 2.0

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

// Alerts returns two-way match alerts. The alert source is two-way matching
// (CAL-053/078); until that lands the feed is empty.
func (a *Aggregator) Alerts(_ context.Context, _ kernel.Page) ([]MatchAlert, int64, error) {
	return nil, 0, nil
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
