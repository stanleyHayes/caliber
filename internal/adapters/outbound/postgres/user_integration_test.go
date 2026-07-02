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
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/platform/migrate"
)

// TestUserRepoCreateEmployerIsAtomic verifies that creating an employer user
// writes both the users row and the mirroring employers row in one transaction —
// so the FK from roles.employer_id always resolves — while a candidate user
// creates no employers row.
func TestUserRepoCreateEmployerIsAtomic(t *testing.T) {
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

	users := pgrepo.NewUserRepo(pool)

	// Employer: both rows must exist after Create.
	empEmail, err := identity.NewEmail("employer.atomic@example.com")
	require.NoError(t, err)
	emp, err := identity.NewUser(empEmail, identity.RoleEmployer, "Acme Ltd", "hash", time.Now())
	require.NoError(t, err)
	require.NoError(t, users.Create(ctx, emp))

	assert.Equal(t, 1, countRows(t, pool, "users", emp.ID.String()))
	assert.Equal(t, 1, countRows(t, pool, "employers", emp.ID.String()),
		"an employer user must get a mirroring employers row (same tx)")

	// Candidate: users row only, no employers row.
	candEmail, err := identity.NewEmail("candidate.atomic@example.com")
	require.NoError(t, err)
	cand, err := identity.NewUser(candEmail, identity.RoleCandidate, "Ama", "hash", time.Now())
	require.NoError(t, err)
	require.NoError(t, users.Create(ctx, cand))

	assert.Equal(t, 1, countRows(t, pool, "users", cand.ID.String()))
	assert.Equal(t, 0, countRows(t, pool, "employers", cand.ID.String()),
		"a candidate must not create an employers row")
}

// countRows returns how many rows in table have id = the given id.
func countRows(t *testing.T, pool *pgxpool.Pool, table, id string) int {
	t.Helper()
	var n int
	// table is a test-controlled constant, not user input.
	require.NoError(t, pool.QueryRow(context.Background(),
		"SELECT count(*) FROM "+table+" WHERE id = $1", id).Scan(&n))
	return n
}
