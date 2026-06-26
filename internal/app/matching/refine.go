package matching

import (
	"context"

	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
)

// Refiner re-ranks a shortlist after applying edited spec/rubric overrides to
// the role (CAL-057 live re-rank): it revises and persists the role, then ranks.
type Refiner struct {
	roles       role.RoleRepository
	shortlister *Shortlister
}

// NewRefiner wires the use-case over the role repository and the shortlister.
func NewRefiner(roles role.RoleRepository, shortlister *Shortlister) *Refiner {
	return &Refiner{roles: roles, shortlister: shortlister}
}

// Refine applies the spec and re-normalized rubric to the role, persists the
// change, and returns a freshly ranked shortlist.
func (r *Refiner) Refine(
	ctx context.Context, roleID kernel.ID, spec role.RoleSpec, rubric role.Rubric, limit int,
) (*ShortlistResult, error) {
	rl, err := r.roles.ByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if err := rl.Revise(spec, rubric.Normalize()); err != nil {
		return nil, err
	}
	if err := r.roles.Update(ctx, rl); err != nil {
		return nil, err
	}
	return r.shortlister.GenerateShortlist(ctx, roleID, limit)
}
