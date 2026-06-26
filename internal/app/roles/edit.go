package roles

import (
	"context"

	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
)

// SpecEditor reads and edits a persisted Role's spec and rubric (Flow A.1,
// CAL-040). Re-weighting normalizes the rubric so weights sum to 1.0.
type SpecEditor struct {
	roles role.RoleRepository
}

// NewSpecEditor wires the use-case.
func NewSpecEditor(repo role.RoleRepository) *SpecEditor { return &SpecEditor{roles: repo} }

// Get loads a role by id.
func (e *SpecEditor) Get(ctx context.Context, roleID kernel.ID) (*role.Role, error) {
	if roleID.IsZero() {
		return nil, kernel.Invalid("roles: role id is required")
	}
	return e.roles.ByID(ctx, roleID)
}

// Update applies an edited spec and rubric to an existing role and persists it.
// The rubric is normalized (re-weighting) before validation.
func (e *SpecEditor) Update(ctx context.Context, roleID kernel.ID, spec role.RoleSpec, rubric role.Rubric) (*role.Role, error) {
	if roleID.IsZero() {
		return nil, kernel.Invalid("roles: role id is required")
	}
	r, err := e.roles.ByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if err := r.Revise(spec, rubric.Normalize()); err != nil {
		return nil, err
	}
	if err := e.roles.Update(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

// List returns a page of an employer's roles, newest first, with the total.
func (e *SpecEditor) List(ctx context.Context, employerID kernel.ID, page kernel.Page) ([]*role.Role, int64, error) {
	if employerID.IsZero() {
		return nil, 0, kernel.Invalid("roles: employer id is required")
	}
	return e.roles.ListByEmployer(ctx, employerID, page)
}
