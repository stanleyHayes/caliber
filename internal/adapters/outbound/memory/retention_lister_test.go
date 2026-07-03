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

func mkRetentionUser(t *testing.T, addr string, role identity.Role, createdAt time.Time) *identity.User {
	t.Helper()
	email, err := identity.NewEmail(addr)
	require.NoError(t, err)
	u, err := identity.NewUser(email, role, "Test", "hash", createdAt)
	require.NoError(t, err)
	return u
}

func TestUserRepo_ListEligibleForRetention(t *testing.T) {
	ctx := context.Background()
	r := memory.NewUserRepo()
	cutoff := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	oldCand := mkRetentionUser(t, "old.candidate@example.com", identity.RoleCandidate, cutoff.Add(-24*time.Hour))
	atCutoff := mkRetentionUser(t, "boundary.candidate@example.com", identity.RoleCandidate, cutoff)
	newCand := mkRetentionUser(t, "new.candidate@example.com", identity.RoleCandidate, cutoff.Add(24*time.Hour))
	oldEmployer := mkRetentionUser(t, "old.employer@example.com", identity.RoleEmployer, cutoff.Add(-48*time.Hour))
	for _, u := range []*identity.User{oldCand, atCutoff, newCand, oldEmployer} {
		require.NoError(t, r.Create(ctx, u))
	}

	ids, err := r.ListEligibleForRetention(ctx, cutoff)
	require.NoError(t, err)

	// Candidates created on or before the cutoff — never the recent candidate, and
	// never a non-candidate (employers have no candidate data to erase).
	assert.ElementsMatch(t, []kernel.ID{oldCand.ID, atCutoff.ID}, ids)
	assert.NotContains(t, ids, newCand.ID)
	assert.NotContains(t, ids, oldEmployer.ID)
}
