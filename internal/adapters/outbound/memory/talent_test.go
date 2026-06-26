package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

func TestMemoryCandidateRepo(t *testing.T) {
	ctx := context.Background()
	r := memory.NewCandidateRepo()
	uid := kernel.NewID()
	c, err := talent.NewCandidate(uid, "Accra", talent.CandidateIntake{})
	require.NoError(t, err)

	require.NoError(t, r.Create(ctx, c))
	dup, _ := talent.NewCandidate(uid, "Kumasi", talent.CandidateIntake{})
	assert.Equal(t, kernel.KindConflict, kernel.KindOf(r.Create(ctx, dup)), "one candidate per user")

	byID, err := r.ByID(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, "Accra", byID.Location)
	byUser, err := r.ByUserID(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, c.ID, byUser.ID)

	list, total, err := r.List(ctx, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)

	_, err = r.ByID(ctx, kernel.NewID())
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(r.Update(ctx, dup)))
}

func TestMemoryTalentProfileRepo(t *testing.T) {
	ctx := context.Background()
	r := memory.NewTalentProfileRepo()
	cid := kernel.NewID()
	p, err := talent.NewTalentProfile(cid, "summary", []talent.ProfileCompetency{{Name: "Go", Level: 4, EvidenceQuote: "x"}})
	require.NoError(t, err)

	require.NoError(t, r.Create(ctx, p))
	dup, _ := talent.NewTalentProfile(cid, "s2", nil)
	assert.Equal(t, kernel.KindConflict, kernel.KindOf(r.Create(ctx, dup)), "one profile per candidate")

	byCand, err := r.ByCandidateID(ctx, cid)
	require.NoError(t, err)
	assert.Equal(t, p.ID, byCand.ID)

	p.MarkScreened()
	require.NoError(t, r.Update(ctx, p))
	reloaded, err := r.ByID(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, talent.PassportScreened, reloaded.PassportStatus)

	list, total, err := r.List(ctx, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	_, err = r.ByCandidateID(ctx, kernel.NewID())
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
}
