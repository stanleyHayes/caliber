package postgres_test

// Container-backed chaos test for the DB path.
//
// Unlike the unit-level resilience tests (which mock the repository port), this
// exercises the REAL pgx-backed RoleRepo against a real Postgres container and
// then injects an infrastructure fault — it closes the connection pool out from
// under an in-flight repository, simulating a database that has died or become
// unreachable. It asserts the repository degrades gracefully: it returns a
// non-nil error and never panics or hangs.
//
// This adds signal a mock cannot: it proves the adapter's error handling copes
// with the actual pgx "closed pool" / connection-loss error shapes, not a
// hand-rolled errors.New stand-in.
//
// It is gated exactly like the other integration tests: skipped in -short mode
// and skipped fast when Docker is unavailable, so `go test -short ./...` and a
// Docker-less machine both stay green.

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
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/platform/migrate"
)

// TestRoleRepoDegradesWhenPoolClosed brings up a real Postgres, verifies a normal
// read works, then closes the pool (the "database is gone" fault) and asserts the
// next repository call returns a clean error instead of panicking.
func TestRoleRepoDegradesWhenPoolClosed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testcontainers chaos test in -short mode")
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
	t.Cleanup(pool.Close) // idempotent; safe even after we close below.

	emp := kernel.NewID()
	_, err = pool.Exec(ctx, `INSERT INTO employers (id, company_name) VALUES ($1, $2)`, emp.String(), "Acme")
	require.NoError(t, err)

	repo := pgrepo.NewRoleRepo(pool)
	rl := mkRole(t, emp, "Engineer", time.Unix(1000, 0).UTC())
	require.NoError(t, repo.Create(ctx, rl))

	// Baseline: a healthy read works.
	got, err := repo.ByID(ctx, rl.ID)
	require.NoError(t, err)
	require.Equal(t, "Engineer", got.Title)

	// CHAOS: the database connection is gone. Closing the pool models a dead DB /
	// severed network without needing to pause the container.
	pool.Close()

	// The repository must NOT panic and must surface a non-nil error. We wrap the
	// call so a panic fails the test with a clear message rather than crashing the
	// suite.
	assert.NotPanics(t, func() {
		_, readErr := repo.ByID(ctx, rl.ID)
		require.Error(t, readErr, "a read against a closed pool must return an error, not a nil result")
		// The error is unclassified infrastructure detail (KindInternal), which the
		// gRPC mapper renders as an opaque codes.Internal — never a false NotFound.
		assert.NotEqual(t, kernel.KindNotFound, kernel.KindOf(readErr),
			"a connection failure must not be misreported as a missing row")
	}, "the repository must degrade gracefully, never panic, on a dead connection")

	// A write against the dead pool must likewise error cleanly, with no panic and
	// no silent success (which would be data loss).
	assert.NotPanics(t, func() {
		writeErr := repo.Create(ctx, mkRole(t, emp, "Ghost", time.Unix(2000, 0).UTC()))
		require.Error(t, writeErr, "a write against a closed pool must fail loudly, never silently succeed")
	}, "a write against a dead connection must not panic")
}
