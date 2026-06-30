// Package memory provides in-memory implementations of domain repository ports,
// used for local development and tests until the Postgres adapters land.
package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
)

// RoleRepo is an in-memory role.RoleRepository.
type RoleRepo struct {
	mu    sync.RWMutex
	items map[kernel.ID]role.Role
}

// NewRoleRepo returns an empty in-memory role repository.
func NewRoleRepo() *RoleRepo { return &RoleRepo{items: make(map[kernel.ID]role.Role)} }

// Reset clears every role (test/dev reseed helper).
func (r *RoleRepo) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = make(map[kernel.ID]role.Role)
}

// Create stores a new role.
func (r *RoleRepo) Create(_ context.Context, rl *role.Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[rl.ID]; ok {
		return kernel.Conflict("memory: role already exists")
	}
	r.items[rl.ID] = *rl
	return nil
}

// ByID returns a role by id.
func (r *RoleRepo) ByID(_ context.Context, id kernel.ID) (*role.Role, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rl, ok := r.items[id]
	if !ok {
		return nil, kernel.NotFound("memory: role not found")
	}
	return &rl, nil
}

// Update replaces an existing role.
func (r *RoleRepo) Update(_ context.Context, rl *role.Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[rl.ID]; !ok {
		return kernel.NotFound("memory: role not found")
	}
	r.items[rl.ID] = *rl
	return nil
}

// ListOpen lists non-closed roles (the applyable pool), newest first.
func (r *RoleRepo) ListOpen(_ context.Context, page kernel.Page) ([]*role.Role, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []role.Role
	for _, rl := range r.items {
		if rl.Status != role.RoleClosed {
			all = append(all, rl)
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].CreatedAt.After(all[j].CreatedAt) })
	total := int64(len(all))
	start := min(page.Offset(), len(all))
	end := min(start+page.Limit(), len(all))
	out := make([]*role.Role, 0, end-start)
	for i := start; i < end; i++ {
		rl := all[i]
		out = append(out, &rl)
	}
	return out, total, nil
}

// ListByEmployer returns a page of an employer's roles, newest first.
func (r *RoleRepo) ListByEmployer(_ context.Context, employerID kernel.ID, page kernel.Page) ([]*role.Role, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []role.Role
	for _, rl := range r.items {
		if rl.EmployerID == employerID {
			all = append(all, rl)
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].CreatedAt.After(all[j].CreatedAt) })
	total := int64(len(all))
	start := min(page.Offset(), len(all))
	end := min(start+page.Limit(), len(all))
	out := make([]*role.Role, 0, end-start)
	for i := start; i < end; i++ {
		rl := all[i]
		out = append(out, &rl)
	}
	return out, total, nil
}

var _ role.RoleRepository = (*RoleRepo)(nil)
