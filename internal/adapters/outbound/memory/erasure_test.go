package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/domain/audit"
	agentdom "github.com/xcreativs/caliber/internal/domain/candidateagent"
	"github.com/xcreativs/caliber/internal/domain/identity"
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

func TestApplicationRepo_DeleteByCandidate(t *testing.T) {
	ctx := context.Background()
	r := memory.NewApplicationRepo()
	cid, other := kernel.NewID(), kernel.NewID()
	require.NoError(t, r.Create(ctx, &agentdom.Application{ID: kernel.NewID(), CandidateID: cid}))
	require.NoError(t, r.Create(ctx, &agentdom.Application{ID: kernel.NewID(), CandidateID: other}))

	require.NoError(t, r.DeleteByCandidate(ctx, cid))

	mine, total, err := r.ByCandidate(ctx, cid, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, mine)
	// Another candidate's applications are untouched.
	theirs, total, err := r.ByCandidate(ctx, other, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, theirs, 1)
}

func TestUserRepo_AnonymizeScrubsPII(t *testing.T) {
	ctx := context.Background()
	r := memory.NewUserRepo()
	u := &identity.User{
		ID: kernel.NewID(), Email: "ama@example.com", Name: "Ama Mensah",
		Role: identity.RoleCandidate, PasswordHash: "hash", Status: identity.StatusActive, CreatedAt: time.Unix(1, 0),
	}
	require.NoError(t, r.Create(ctx, u))

	require.NoError(t, r.Anonymize(ctx, u.ID))

	got, err := r.ByID(ctx, u.ID)
	require.NoError(t, err, "the account row is retained")
	assert.Empty(t, got.Name, "name is scrubbed")
	assert.Empty(t, got.PasswordHash, "credential is scrubbed")
	assert.NotEqual(t, identity.Email("ama@example.com"), got.Email, "email is replaced with a tombstone")
	// The original email no longer resolves to the account.
	_, err = r.ByEmail(ctx, "ama@example.com")
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
}

func TestAuditRepo_TombstoneActor(t *testing.T) {
	ctx := context.Background()
	r := memory.NewAuditRepo()
	actor, entity := kernel.NewID(), kernel.NewID()
	e, err := audit.NewAuditEntry(actor, audit.ActionContestRaised, "contest", entity, "", "", time.Unix(1, 0))
	require.NoError(t, err)
	require.NoError(t, r.Append(ctx, e))

	require.NoError(t, r.TombstoneActor(ctx, actor))

	entries, _, err := r.List(ctx, "contest", entity, kernel.NewPage(1, 10))
	require.NoError(t, err)
	require.Len(t, entries, 1, "the audit entry is retained as a compliance record")
	assert.Equal(t, kernel.ID("erased"), entries[0].ActorUserID, "the subject's identity is removed")
}
