package postgres_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	pgrepo "github.com/xcreativs/caliber/internal/adapters/outbound/postgres"
	"github.com/xcreativs/caliber/internal/domain/audit"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/platform/migrate"
)

func TestAuditRepoSearchAndList(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testcontainers integration test in -short mode")
	}
	skipIfNoDocker(t)
	ctx := context.Background()
	ctr, err := tcpostgres.Run(ctx, "pgvector/pgvector:pg17",
		tcpostgres.WithDatabase("caliber"),
		tcpostgres.WithUsername("caliber"),
		tcpostgres.WithPassword("caliber"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Skipf("docker unavailable, skipping: %v", err)
	}
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(ctr) })

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	sqlDB, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	require.NoError(t, migrate.Up(sqlDB, migrationsDir(t)))
	require.NoError(t, sqlDB.Close())

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	repo := pgrepo.NewAuditRepo(pool)
	actor := kernel.NewID()
	matchID := kernel.NewID()
	appID := kernel.NewID()

	e1, err := audit.NewAuditEntry(actor, audit.ActionApproveRejection, "match", matchID, "", "", time.Unix(1000, 0).UTC())
	require.NoError(t, err)
	e2, err := audit.NewAuditEntry(actor, audit.ActionOverrideScore, "match", matchID, "", "", time.Unix(2000, 0).UTC())
	require.NoError(t, err)
	e3, err := audit.NewAuditEntry(actor, audit.ActionAgentSubmit, "application", appID, "", "", time.Unix(3000, 0).UTC())
	require.NoError(t, err)
	require.NoError(t, repo.Append(ctx, e1))
	require.NoError(t, repo.Append(ctx, e2))
	require.NoError(t, repo.Append(ctx, e3))

	// List by entity/entityID returns newest first.
	list, total, err := repo.List(ctx, "match", matchID, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, list, 2)
	assert.Equal(t, audit.ActionOverrideScore, list[0].Action)

	// Owner scoping (CAL-153): two employers act on a shared subject; ListForOwner
	// returns only the caller's entries, closing the cross-tenant read.
	ownerA, ownerB, subj := kernel.NewID(), kernel.NewID(), kernel.NewID()
	oa, err := audit.NewAuditEntry(actor, audit.ActionApproveRejection, "match", subj, "", "", time.Unix(5000, 0).UTC())
	require.NoError(t, err)
	oa.OwnerID = ownerA
	ob, err := audit.NewAuditEntry(actor, audit.ActionApproveRejection, "match", subj, "", "", time.Unix(6000, 0).UTC())
	require.NoError(t, err)
	ob.OwnerID = ownerB
	require.NoError(t, repo.Append(ctx, oa))
	require.NoError(t, repo.Append(ctx, ob))

	forA, totalA, err := repo.ListForOwner(ctx, "match", subj, ownerA, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(1), totalA)
	require.Len(t, forA, 1)
	assert.Equal(t, ownerA, forA[0].OwnerID, "owner A sees only their own entry")

	both, totalBoth, err := repo.ListForOwner(ctx, "match", subj, kernel.ID(""), kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(2), totalBoth, "a zero owner (admin) is unscoped")
	require.Len(t, both, 2)

	// Search across all actions/entities in the time range.
	filter := audit.ReportFilter{
		Start: time.Unix(0, 0).UTC(),
		End:   time.Unix(4000, 0).UTC(),
	}
	all, total, err := repo.Search(ctx, filter, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	require.Len(t, all, 3)
	assert.Equal(t, audit.ActionAgentSubmit, all[0].Action)

	// Search filtered by action and entity.
	filter.Actions = []string{audit.ActionApproveRejection, audit.ActionOverrideScore}
	filter.Entities = []string{"match"}
	matches, total, err := repo.Search(ctx, filter, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, matches, 2)
	for _, m := range matches {
		assert.Equal(t, "match", m.Entity)
	}

	// Search respects time boundaries.
	filter.Start = time.Unix(2500, 0).UTC()
	filter.End = time.Unix(3500, 0).UTC()
	filter.Actions = nil
	filter.Entities = nil
	bounded, total, err := repo.Search(ctx, filter, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, bounded, 1)
	assert.Equal(t, audit.ActionAgentSubmit, bounded[0].Action)
}
