package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

func mkUser(t *testing.T, email string) *identity.User {
	t.Helper()
	e, err := identity.NewEmail(email)
	require.NoError(t, err)
	u, err := identity.NewUser(e, identity.RoleCandidate, "Test", "hash", time.Unix(1, 0))
	require.NoError(t, err)
	return u
}

func TestMemoryUserRepoCRUD(t *testing.T) {
	ctx := context.Background()
	r := memory.NewUserRepo()
	u := mkUser(t, "kofi@example.com")

	require.NoError(t, r.Create(ctx, u))

	byID, err := r.ByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, u.Email, byID.Email)

	byEmail, err := r.ByEmail(ctx, u.Email)
	require.NoError(t, err)
	assert.Equal(t, u.ID, byEmail.ID)

	u.Lock()
	require.NoError(t, r.Update(ctx, u))
	reloaded, err := r.ByID(ctx, u.ID)
	require.NoError(t, err)
	assert.False(t, reloaded.IsActive())
}

func TestMemoryUserRepoErrors(t *testing.T) {
	ctx := context.Background()
	r := memory.NewUserRepo()
	u := mkUser(t, "kofi@example.com")
	require.NoError(t, r.Create(ctx, u))

	dup := mkUser(t, "kofi@example.com")
	assert.Equal(t, kernel.KindConflict, kernel.KindOf(r.Create(ctx, dup)))

	_, err := r.ByID(ctx, kernel.NewID())
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
	_, err = r.ByEmail(ctx, identity.Email("missing@example.com"))
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(r.Update(ctx, mkUser(t, "ghost@example.com"))))
}
