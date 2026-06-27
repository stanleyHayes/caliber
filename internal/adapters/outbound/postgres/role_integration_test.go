package postgres_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"runtime"
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
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/platform/migrate"
)

func migrationsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "db", "migrations")
}

func mkRole(t *testing.T, emp kernel.ID, title string, ts time.Time) *role.Role {
	t.Helper()
	rl, err := role.NewRole(emp,
		role.RoleSpec{Title: title, Seniority: role.SeniorityMid},
		role.Rubric{Competencies: []role.Competency{{Name: "Core", Weight: 1, MustHave: true}}},
		ts)
	require.NoError(t, err)
	return rl
}

func TestRoleRepoCRUDAndPagination(t *testing.T) {
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

	emp := kernel.NewID()
	_, err = pool.Exec(ctx, `INSERT INTO employers (id, company_name) VALUES ($1, $2)`, emp.String(), "Acme")
	require.NoError(t, err)

	repo := pgrepo.NewRoleRepo(pool)

	rl := mkRole(t, emp, "Engineer", time.Unix(1000, 0).UTC())
	require.NoError(t, repo.Create(ctx, rl))
	assert.Equal(t, kernel.KindConflict, kernel.KindOf(repo.Create(ctx, rl)), "duplicate create should conflict")

	got, err := repo.ByID(ctx, rl.ID)
	require.NoError(t, err)
	assert.Equal(t, "Engineer", got.Title)
	assert.Equal(t, emp, got.EmployerID)

	_, err = repo.ByID(ctx, kernel.NewID())
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))

	rl.Title = "Senior Engineer"
	require.NoError(t, repo.Update(ctx, rl))
	got, err = repo.ByID(ctx, rl.ID)
	require.NoError(t, err)
	assert.Equal(t, "Senior Engineer", got.Title)
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(repo.Update(ctx, mkRole(t, emp, "ghost", time.Unix(1, 0)))))

	require.NoError(t, repo.Create(ctx, mkRole(t, emp, "B", time.Unix(2000, 0).UTC())))
	require.NoError(t, repo.Create(ctx, mkRole(t, emp, "C", time.Unix(3000, 0).UTC())))

	page1, total, err := repo.ListByEmployer(ctx, emp, kernel.NewPage(1, 2))
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, page1, 2)
	page2, _, err := repo.ListByEmployer(ctx, emp, kernel.NewPage(2, 2))
	require.NoError(t, err)
	assert.Len(t, page2, 1)
	assert.True(t, page1[0].CreatedAt.After(page1[1].CreatedAt), "results should be newest-first")
}
