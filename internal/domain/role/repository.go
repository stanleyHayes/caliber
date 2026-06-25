package role

import (
	"context"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// RoleRepository persists and retrieves roles.
type RoleRepository interface { //nolint:revive // domain name fixed by the role context spec
	Create(ctx context.Context, r *Role) error
	ByID(ctx context.Context, id kernel.ID) (*Role, error)
	Update(ctx context.Context, r *Role) error
	ListByEmployer(ctx context.Context, employerID kernel.ID, page kernel.Page) ([]*Role, int64, error)
}

//go:generate mockgen -source=repository.go -destination=../../mocks/role_repository.go -package=mocks
