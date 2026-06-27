package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/domain/audit"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

func TestMemoryAuditRepo(t *testing.T) {
	ctx := context.Background()
	r := memory.NewAuditRepo()
	actor := kernel.NewID()
	entity := kernel.NewID()

	e1, err := audit.NewAuditEntry(actor, audit.ActionContestRaised, "contest", entity, "", "", time.Unix(1, 0))
	require.NoError(t, err)
	e2, err := audit.NewAuditEntry(actor, audit.ActionContestResolved, "contest", entity, "", "", time.Unix(2, 0))
	require.NoError(t, err)
	other, err := audit.NewAuditEntry(actor, audit.ActionAgentSubmit, "application", kernel.NewID(), "", "", time.Unix(3, 0))
	require.NoError(t, err)
	for _, e := range []*audit.AuditEntry{e1, e2, other} {
		require.NoError(t, r.Append(ctx, e))
	}

	// List filters by entity+entityID, newest first
	list, total, err := r.List(ctx, "contest", entity, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, list, 2)
	assert.Equal(t, audit.ActionContestResolved, list[0].Action, "newest first")

	// unrelated entity returns nothing
	none, total, err := r.List(ctx, "contest", kernel.NewID(), kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, none)
}
