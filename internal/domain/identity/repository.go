package identity

import (
	"context"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

//go:generate mockgen -source=repository.go -destination=../../mocks/identity.go -package=mocks

// UserRepository persists and retrieves users.
type UserRepository interface {
	Create(ctx context.Context, u *User) error
	ByID(ctx context.Context, id kernel.ID) (*User, error)
	ByEmail(ctx context.Context, email Email) (*User, error)
	Update(ctx context.Context, u *User) error
}
