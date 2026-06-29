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

func TestCandidateRepo_DeleteRemovesAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	r := memory.NewCandidateRepo()
	c, err := talent.NewCandidate(kernel.NewID(), "Accra", talent.CandidateIntake{})
	require.NoError(t, err)
	require.NoError(t, r.Create(ctx, c))

	require.NoError(t, r.Delete(ctx, c.ID))
	_, err = r.ByID(ctx, c.ID)
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err), "the candidate is gone after erasure")

	// Deleting again (or an unknown id) is a no-op — erasure is idempotent.
	require.NoError(t, r.Delete(ctx, c.ID))
	require.NoError(t, r.Delete(ctx, kernel.NewID()))
}

func TestTalentProfileRepo_DeleteByCandidate(t *testing.T) {
	ctx := context.Background()
	r := memory.NewTalentProfileRepo()
	cid := kernel.NewID()
	p, err := talent.NewTalentProfile(cid, "summary", []talent.ProfileCompetency{
		{Name: "Go", Level: 4, EvidenceQuote: "x", SourceSpan: "CV"},
	})
	require.NoError(t, err)
	require.NoError(t, r.Create(ctx, p))

	require.NoError(t, r.DeleteByCandidate(ctx, cid))
	_, err = r.ByCandidateID(ctx, cid)
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err), "the profile is gone after erasure")

	// Idempotent: a candidate with no profile is a no-op.
	require.NoError(t, r.DeleteByCandidate(ctx, kernel.NewID()))
}
