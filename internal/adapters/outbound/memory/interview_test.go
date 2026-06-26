package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

func mkInterview(t *testing.T, candidateID kernel.ID) *interview.Interview {
	t.Helper()
	iv, err := interview.NewInterview(kernel.NewID(), candidateID, interview.ModeText)
	require.NoError(t, err)
	return iv
}

func TestMemoryInterviewRepoCRUD(t *testing.T) {
	ctx := context.Background()
	r := memory.NewInterviewRepo()
	cid := kernel.NewID()
	iv := mkInterview(t, cid)

	require.NoError(t, r.Create(ctx, iv))
	assert.Equal(t, kernel.KindConflict, kernel.KindOf(r.Create(ctx, iv)), "duplicate create conflicts")

	got, err := r.ByID(ctx, iv.ID)
	require.NoError(t, err)
	assert.Equal(t, interview.StateOpen, got.State)

	require.NoError(t, iv.Transition(interview.StateAsking))
	require.NoError(t, r.Update(ctx, iv))
	reloaded, err := r.ByID(ctx, iv.ID)
	require.NoError(t, err)
	assert.Equal(t, interview.StateAsking, reloaded.State)

	list, total, err := r.ByCandidate(ctx, cid, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, iv.ID, list[0].ID)
}

func TestMemoryInterviewRepoErrors(t *testing.T) {
	ctx := context.Background()
	r := memory.NewInterviewRepo()
	_, err := r.ByID(ctx, kernel.NewID())
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(r.Update(ctx, mkInterview(t, kernel.NewID()))))

	list, total, err := r.ByCandidate(ctx, kernel.NewID(), kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, list)
}
