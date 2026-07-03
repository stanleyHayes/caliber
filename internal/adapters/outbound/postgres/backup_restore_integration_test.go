package postgres_test

import (
	"context"
	"database/sql"
	"io"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	pgrepo "github.com/xcreativs/caliber/internal/adapters/outbound/postgres"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/platform/migrate"
)

// TestPostgresBackupRestoreDrill exercises the CAL-151 disaster-recovery drill
// end-to-end: it seeds a database, takes a pg_dump backup, restores it into a
// fresh database, and asserts the seeded row survived — proving a backup is
// actually restorable, not just producible. It runs the real pg_dump/pg_restore
// tooling inside the container, so it validates the same format the operational
// scripts (scripts/db-backup.sh, scripts/db-restore.sh) produce.
func TestPostgresBackupRestoreDrill(t *testing.T) {
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

	// Seed a known row we can look for after the restore.
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	email, err := identity.NewEmail("dr.drill@example.com")
	require.NoError(t, err)
	u, err := identity.NewUser(email, identity.RoleCandidate, "DR Drill", "hash", time.Now())
	require.NoError(t, err)
	require.NoError(t, pgrepo.NewUserRepo(pool).Create(ctx, u))
	pool.Close()

	// Local (unix-socket) connections in the postgres image use trust auth, so the
	// tools need no password. Back up, create a fresh DB, and restore into it.
	execOK(ctx, t, ctr, "pg_dump -U caliber --format=custom --no-owner --file=/tmp/drill.dump caliber")
	execOK(ctx, t, ctr, "createdb -U caliber caliber_restored")
	// pg_restore can exit non-zero on benign notices; the real success criterion is
	// the row count below, so run it without requiring a zero exit.
	execRaw(ctx, t, ctr, "pg_restore -U caliber --no-owner -d caliber_restored /tmp/drill.dump")

	// The seeded user must be present in the restored database.
	got := execOK(ctx, t, ctr,
		"psql -U caliber -d caliber_restored -tAc \"SELECT email FROM users WHERE id = '"+u.ID.String()+"'\"")
	assert.Equal(t, "dr.drill@example.com", strings.TrimSpace(got),
		"the seeded row was restored from the backup")
}

// execOK runs a shell command in the container and fails the test unless it exits 0.
func execOK(ctx context.Context, t *testing.T, ctr testcontainers.Container, cmd string) string {
	t.Helper()
	code, out := execRaw(ctx, t, ctr, cmd)
	require.Equalf(t, 0, code, "command failed (exit %d): %s\n%s", code, cmd, out)
	return out
}

// execRaw runs a shell command in the container and returns its exit code + output.
func execRaw(ctx context.Context, t *testing.T, ctr testcontainers.Container, cmd string) (int, string) {
	t.Helper()
	code, reader, err := ctr.Exec(ctx, []string{"sh", "-c", cmd}, tcexec.Multiplexed())
	require.NoError(t, err)
	b, err := io.ReadAll(reader)
	require.NoError(t, err)
	return code, string(b)
}
