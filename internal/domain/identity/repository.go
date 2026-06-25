package identity

import (
	"context"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// UserRepository persists and retrieves users.
type UserRepository interface {
	Create(ctx context.Context, u *User) error
	ByID(ctx context.Context, id kernel.ID) (*User, error)
	ByEmail(ctx context.Context, email Email) (*User, error)
	Update(ctx context.Context, u *User) error
}
