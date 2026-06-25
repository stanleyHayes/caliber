package migrate_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/xcreativs/caliber/internal/platform/migrate"
)

func migrationsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "db", "migrations")
}

func TestMigrationsApplyAgainstPgvector(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testcontainers integration test in -short mode")
	}
	ctx := context.Background()
	ctr, err := postgres.Run(ctx, "pgvector/pgvector:pg17",
		postgres.WithDatabase("caliber"),
		postgres.WithUsername("caliber"),
		postgres.WithPassword("caliber"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Skipf("docker unavailable, skipping integration test: %v", err)
	}
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(ctr) })

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, migrate.Up(db, migrationsDir(t)))

	var tables int
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT count(*) FROM information_schema.tables
		 WHERE table_schema = 'public' AND table_name IN
		 ('users','employers','candidates','roles','talent_profiles','matches',
		  'talent_interviews','interview_turns','applications','audit_log')`).Scan(&tables))
	assert.Equal(t, 10, tables, "all core tables should be created")

	var ext int
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT count(*) FROM pg_extension WHERE extname = 'vector'`).Scan(&ext))
	assert.Equal(t, 1, ext, "pgvector extension should be enabled")
}
