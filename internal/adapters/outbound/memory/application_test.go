package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/domain/candidateagent"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

func agentApp(t *testing.T, candidateID kernel.ID, summary string) *candidateagent.Application {
	t.Helper()
	a, err := candidateagent.NewAgentApplication(kernel.NewID(), candidateID, kernel.NewID(), summary)
	require.NoError(t, err)
	return a
}

func TestMemoryApplicationRepo(t *testing.T) {
	ctx := context.Background()
	r := memory.NewApplicationRepo()
	cid := kernel.NewID()
	a1 := agentApp(t, cid, "first")
	a2 := agentApp(t, cid, "second")

	require.NoError(t, r.Create(ctx, a1))
	require.NoError(t, r.Create(ctx, a2))
	assert.Equal(t, kernel.KindConflict, kernel.KindOf(r.Create(ctx, a1)))

	got, err := r.ByID(ctx, a1.ID)
	require.NoError(t, err)
	assert.Equal(t, "first", got.TailoredSummary)

	require.NoError(t, a1.Submit())
	require.NoError(t, r.Update(ctx, a1))
	reloaded, err := r.ByID(ctx, a1.ID)
	require.NoError(t, err)
	assert.Equal(t, candidateagent.StatusSubmitted, reloaded.Status)

	list, total, err := r.ByCandidate(ctx, cid, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, list, 2)
	assert.Equal(t, a2.ID, list[0].ID, "newest first")
}

func TestMemoryApplicationRepoErrors(t *testing.T) {
	ctx := context.Background()
	r := memory.NewApplicationRepo()
	_, err := r.ByID(ctx, kernel.NewID())
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(r.Update(ctx, agentApp(t, kernel.NewID(), "x"))))

	list, total, err := r.ByCandidate(ctx, kernel.NewID(), kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, list)
}
