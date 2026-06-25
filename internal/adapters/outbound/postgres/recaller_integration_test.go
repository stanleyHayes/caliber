package postgres_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

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

func oneHot(idx int) string {
	parts := make([]string, 1536)
	for i := range parts {
		if i == idx {
			parts[i] = "1"
		} else {
			parts[i] = "0"
		}
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func TestRecallByEmbedding(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testcontainers integration test in -short mode")
	}
	ctx := context.Background()
	ctr, err := tcpostgres.Run(ctx, "pgvector/pgvector:pg17",
		tcpostgres.WithDatabase("caliber"), tcpostgres.WithUsername("caliber"), tcpostgres.WithPassword("caliber"),
		tcpostgres.BasicWaitStrategies())
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

	mustExec := func(qsql string, args ...any) {
		_, e := pool.Exec(ctx, qsql, args...)
		require.NoError(t, e)
	}
	seed := func(email string, hotIndex int) kernel.ID {
		uid, cid := kernel.NewID(), kernel.NewID()
		mustExec(`INSERT INTO users (id,email,role,name,password_hash) VALUES ($1,$2,'candidate','N','h')`, uid.String(), email)
		mustExec(`INSERT INTO candidates (id,user_id) VALUES ($1,$2)`, cid.String(), uid.String())
		mustExec(`INSERT INTO talent_profiles (id,candidate_id,profile_embedding) VALUES ($1,$2,$3::vector)`,
			kernel.NewID().String(), cid.String(), oneHot(hotIndex))
		return cid
	}

	cidA := seed("a@example.com", 0)
	_ = seed("b@example.com", 1)

	query := make([]float32, 1536)
	query[0] = 1 // aligned with candidate A's one-hot embedding

	ids, err := pgrepo.NewRecaller(pool).Recall(ctx, query, 10)
	require.NoError(t, err)
	require.Len(t, ids, 2)
	assert.Equal(t, cidA, ids[0], "the aligned candidate ranks first")
}
