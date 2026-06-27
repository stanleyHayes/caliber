package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/domain/contest"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

func mkContest(t *testing.T, cid, sid kernel.ID, reason string) *contest.Contest {
	t.Helper()
	c, err := contest.NewContest(cid, sid, contest.SubjectMatch, reason, time.Unix(1, 0))
	require.NoError(t, err)
	return c
}

func TestMemoryContestRepo(t *testing.T) {
	ctx := context.Background()
	r := memory.NewContestRepo()
	cid, other := kernel.NewID(), kernel.NewID()
	subj := kernel.NewID()

	c1 := mkContest(t, cid, subj, "first")
	c2 := mkContest(t, cid, subj, "second")
	c3 := mkContest(t, other, kernel.NewID(), "other candidate")
	for _, c := range []*contest.Contest{c1, c2, c3} {
		require.NoError(t, r.Create(ctx, c))
	}

	// ByID
	got, err := r.ByID(ctx, c1.ID)
	require.NoError(t, err)
	assert.Equal(t, "first", got.Reason)
	_, err = r.ByID(ctx, kernel.NewID())
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))

	// ByCandidate: newest first, filtered
	list, total, err := r.ByCandidate(ctx, cid, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, list, 2)
	assert.Equal(t, "second", list[0].Reason, "newest first")

	// BySubject: both c1 and c2 share subj
	bySubj, total, err := r.BySubject(ctx, subj, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, bySubj, 2)

	// pagination
	page2, total, err := r.ByCandidate(ctx, cid, kernel.NewPage(2, 1))
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, page2, 1)
	assert.Equal(t, "first", page2[0].Reason)

	// Update (resolve), then re-read
	c1.Status = contest.StatusUpheld
	require.NoError(t, r.Update(ctx, c1))
	reread, err := r.ByID(ctx, c1.ID)
	require.NoError(t, err)
	assert.Equal(t, contest.StatusUpheld, reread.Status)

	// Update unknown -> NotFound
	err = r.Update(ctx, mkContest(t, cid, subj, "ghost"))
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
}
