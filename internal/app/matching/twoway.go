package matching

import (
	"context"
	"sort"

	"github.com/xcreativs/caliber/internal/domain/kernel"
	matchingdom "github.com/xcreativs/caliber/internal/domain/matching"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

const (
	// twoWayScanLimit caps how many open roles a passive scan considers.
	twoWayScanLimit = 100
	// defaultRoleFitLimit is the default number of fitting roles returned.
	defaultRoleFitLimit = 10
)

// RoleFit pairs an open role with a candidate's explainable structural fit.
type RoleFit struct {
	Role *role.Role
	Fit  matchingdom.Fit
}

// PassiveMatcher answers the candidate→role direction of two-way matching
// (CAL-053): which open roles fit a passive candidate. It complements the
// Shortlister (role→candidate). Fit is deterministic and bias-safe — competency
// signals only, no LLM call — so it scales across the open-role set, which is
// what Talent Radar discovery and two-way alerts (CAL-078) need.
type PassiveMatcher struct {
	roles      role.RoleRepository
	profiles   talent.TalentProfileRepository
	candidates talent.CandidateRepository
}

// NewPassiveMatcher wires the two-way (candidate→role) matcher.
func NewPassiveMatcher(
	roles role.RoleRepository,
	profiles talent.TalentProfileRepository,
	candidates talent.CandidateRepository,
) *PassiveMatcher {
	return &PassiveMatcher{roles: roles, profiles: profiles, candidates: candidates}
}

// RolesForCandidate returns up to limit open roles the candidate fits, ranked by
// structural fit (highest first). A role is included only when the candidate
// clears its logistical gates (location/salary) AND meets every must-have
// competency — the same honesty bar the candidate agent applies, so Radar never
// surfaces a role the candidate could not pursue without fabrication. Returns an
// empty slice (no error) when the candidate has no profile yet.
func (m *PassiveMatcher) RolesForCandidate(ctx context.Context, candidateID kernel.ID, limit int) ([]RoleFit, error) {
	if limit <= 0 {
		limit = defaultRoleFitLimit
	}
	cand, err := m.candidates.ByID(ctx, candidateID)
	if err != nil {
		return nil, err
	}
	profile, err := m.profiles.ByCandidateID(ctx, candidateID)
	if err != nil {
		if kernel.KindOf(err) == kernel.KindNotFound {
			return nil, nil
		}
		return nil, err
	}
	open, _, err := m.roles.ListOpen(ctx, kernel.NewPage(1, twoWayScanLimit))
	if err != nil {
		return nil, err
	}

	fits := rankRoleFits(cand, candidateSignals(profile), open)
	if len(fits) > limit {
		fits = fits[:limit]
	}
	return fits, nil
}

// rankRoleFits keeps the roles the candidate genuinely fits, highest fit first.
func rankRoleFits(cand *talent.Candidate, signals []matchingdom.CandidateSignal, open []*role.Role) []RoleFit {
	fits := make([]RoleFit, 0, len(open))
	for _, rl := range open {
		if rf, ok := evaluateRoleFit(cand, signals, rl); ok {
			fits = append(fits, rf)
		}
	}
	sort.SliceStable(fits, func(i, j int) bool { return fits[i].Fit.Score > fits[j].Fit.Score })
	return fits
}

// evaluateRoleFit returns the candidate's fit for one role, or ok=false when the
// role is logistically incompatible, carries a biased ranking signal, or has an
// unmet must-have — the honesty bar that keeps Radar from surfacing dead ends.
func evaluateRoleFit(cand *talent.Candidate, signals []matchingdom.CandidateSignal, rl *role.Role) (RoleFit, bool) {
	if !logisticsClear(cand, rl) {
		return RoleFit{}, false
	}
	rubric := rubricSignals(rl)
	if matchingdom.EnsureBiasSafe(signalNames(rubric)) != nil {
		return RoleFit{}, false // never rank on a protected attribute (defensive)
	}
	fit := matchingdom.ComputeFit(rubric, signals)
	if !fit.MustHavesMet {
		return RoleFit{}, false
	}
	return RoleFit{Role: rl, Fit: fit}, true
}

func candidateSignals(p *talent.TalentProfile) []matchingdom.CandidateSignal {
	out := make([]matchingdom.CandidateSignal, 0, len(p.Competencies))
	for _, c := range p.Competencies {
		out = append(out, matchingdom.CandidateSignal{Name: c.Name, Level: c.Level})
	}
	return out
}

func rubricSignals(rl *role.Role) []matchingdom.RubricSignal {
	out := make([]matchingdom.RubricSignal, 0, len(rl.Rubric.Competencies))
	for _, c := range rl.Rubric.Competencies {
		out = append(out, matchingdom.RubricSignal{Name: c.Name, Weight: c.Weight, MustHave: c.MustHave})
	}
	return out
}

func signalNames(rs []matchingdom.RubricSignal) []string {
	out := make([]string, 0, len(rs))
	for _, s := range rs {
		out = append(out, s.Name)
	}
	return out
}

// logisticsClear reports whether the candidate passes a role's pre-scoring
// logistical gates (work location, salary floor) — never protected attributes.
func logisticsClear(cand *talent.Candidate, rl *role.Role) bool {
	req := requirementsFor(rl)
	return len(req.ScreenLogistics(cand.ID, cand.Location, cand.Intake.SalaryFloor, cand.Intake.SalaryCurrency)) == 0
}
