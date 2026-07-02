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

func seedAuditRepo(t *testing.T, r *memory.AuditRepo) kernel.ID {
	t.Helper()
	actor := kernel.NewID()
	contestID := kernel.NewID()

	entries := []*audit.AuditEntry{
		mustEntry(t, actor, audit.ActionContestRaised, "contest", contestID, time.Unix(1, 0)),
		mustEntry(t, actor, audit.ActionContestResolved, "contest", contestID, time.Unix(2, 0)),
		mustEntry(t, actor, audit.ActionAgentSubmit, "application", kernel.NewID(), time.Unix(3, 0)),
	}
	for _, e := range entries {
		require.NoError(t, r.Append(context.Background(), e))
	}
	return contestID
}

func mustEntry(t *testing.T, actor kernel.ID, action, entity string, entityID kernel.ID, ts time.Time) *audit.AuditEntry {
	t.Helper()
	e, err := audit.NewAuditEntry(actor, action, entity, entityID, "", "", ts)
	require.NoError(t, err)
	return e
}

func TestMemoryAuditRepo(t *testing.T) {
	ctx := context.Background()
	r := memory.NewAuditRepo()
	contestID := seedAuditRepo(t, r)

	// List filters by entity+entityID, newest first
	list, total, err := r.List(ctx, "contest", contestID, kernel.NewPage(1, 10))
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

func TestMemoryAuditRepo_SearchFiltersByTimeActionAndEntity(t *testing.T) {
	ctx := context.Background()
	r := memory.NewAuditRepo()
	seedAuditRepo(t, r)

	// All entries fall between Unix(0) and Unix(4,0).
	filter := audit.ReportFilter{Start: time.Unix(0, 0), End: time.Unix(4, 0)}
	all, total, err := r.Search(ctx, filter, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, all, 3)
	assert.Equal(t, audit.ActionAgentSubmit, all[0].Action, "newest first")

	// Filter by action.
	filter.Actions = []string{audit.ActionContestRaised, audit.ActionContestResolved}
	matches, total, err := r.Search(ctx, filter, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, matches, 2)

	// Filter by entity.
	filter.Actions = nil
	filter.Entities = []string{"application"}
	apps, total, err := r.Search(ctx, filter, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, apps, 1)
	assert.Equal(t, "application", apps[0].Entity)

	// Time range excludes everything.
	filter.Entities = nil
	filter.Start = time.Unix(10, 0)
	filter.End = time.Unix(20, 0)
	none, total, err := r.Search(ctx, filter, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, none)
}

func TestMemoryAuditRepo_SearchPagination(t *testing.T) {
	ctx := context.Background()
	r := memory.NewAuditRepo()
	actor := kernel.NewID()
	entityID := kernel.NewID()
	for i := range 5 {
		require.NoError(t, r.Append(ctx, mustEntry(t, actor, audit.ActionAgentSubmit, "application", entityID, time.Unix(int64(i+1), 0))))
	}

	filter := audit.ReportFilter{Start: time.Unix(0, 0), End: time.Unix(10, 0)}
	page1, total, err := r.Search(ctx, filter, kernel.NewPage(1, 2))
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	require.Len(t, page1, 2)
	assert.Equal(t, time.Unix(5, 0), page1[0].Timestamp)

	page2, _, err := r.Search(ctx, filter, kernel.NewPage(2, 2))
	require.NoError(t, err)
	require.Len(t, page2, 2)
	assert.Equal(t, time.Unix(3, 0), page2[0].Timestamp)
}
