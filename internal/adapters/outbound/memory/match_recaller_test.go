package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	matchingdom "github.com/xcreativs/caliber/internal/domain/matching"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

func TestMemoryRecaller(t *testing.T) {
	ctx := context.Background()
	cands := memory.NewCandidateRepo()
	for range 3 {
		c, err := talent.NewCandidate(kernel.NewID(), "Accra", talent.CandidateIntake{})
		require.NoError(t, err)
		require.NoError(t, cands.Create(ctx, c))
	}
	r := memory.NewRecaller(cands)

	ids, err := r.Recall(ctx, []float32{0.1}, 10)
	require.NoError(t, err)
	assert.Len(t, ids, 3, "recall returns the pool (embedding ignored in dev)")

	limited, err := r.Recall(ctx, nil, 2)
	require.NoError(t, err)
	assert.Len(t, limited, 2, "respects the limit")
}

func mkMatch(t *testing.T, roleID, candID kernel.ID, score float64) *matchingdom.Match {
	t.Helper()
	m, err := matchingdom.NewMatch(roleID, candID, score, kernel.ConfidenceMedium, nil, "r", nil, false)
	require.NoError(t, err)
	return m
}

func TestMemoryMatchRepo(t *testing.T) {
	ctx := context.Background()
	r := memory.NewMatchRepo()
	role := kernel.NewID()
	cA, cB := kernel.NewID(), kernel.NewID()

	require.NoError(t, r.Upsert(ctx, mkMatch(t, role, cA, 0.6)))
	require.NoError(t, r.Upsert(ctx, mkMatch(t, role, cB, 0.9)))
	require.NoError(t, r.Upsert(ctx, mkMatch(t, kernel.NewID(), cA, 0.5))) // other role

	// ByRole: ranked by score desc
	byRole, total, err := r.ByRole(ctx, role, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, byRole, 2)
	assert.Equal(t, cB, byRole[0].CandidateID, "highest score first")
	assert.Equal(t, cA, byRole[1].CandidateID)

	// Upsert replaces (re-score cA higher)
	require.NoError(t, r.Upsert(ctx, mkMatch(t, role, cA, 0.95)))
	byRole, total, err = r.ByRole(ctx, role, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(2), total, "re-score replaces, not appends")
	assert.Equal(t, cA, byRole[0].CandidateID, "cA now ranks first")

	// ForCandidate
	forCand, total, err := r.ForCandidate(ctx, cA, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(2), total, "cA has matches in two roles")
	assert.Len(t, forCand, 2)
}
