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
	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/platform/migrate"
)

func TestRefreshStoreLifecycle(t *testing.T) {
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

	// A refresh grant requires an owning user (FK).
	uid := kernel.NewID()
	_, err = pool.Exec(ctx, `INSERT INTO users (id,email,role,name,password_hash) VALUES ($1,'r@example.com','candidate','R','h')`, uid.String())
	require.NoError(t, err)

	store := pgrepo.NewRefreshStore(pool)
	now := time.Unix(1700000000, 0)
	require.NoError(t, store.Save(ctx, app.RefreshRecord{ID: "jti-1", UserID: uid, ExpiresAt: now.Add(time.Hour)}))

	got, err := store.Consume(ctx, "jti-1", now)
	require.NoError(t, err)
	assert.Equal(t, uid, got.UserID)

	// Single-use: a replay of the same jti is rejected.
	_, err = store.Consume(ctx, "jti-1", now)
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))

	// Expired and unknown tokens are rejected.
	require.NoError(t, store.Save(ctx, app.RefreshRecord{ID: "jti-exp", UserID: uid, ExpiresAt: now.Add(-time.Second)}))
	_, err = store.Consume(ctx, "jti-exp", now)
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
	_, err = store.Consume(ctx, "nope", now)
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))

	// Revoke makes a live token unusable.
	require.NoError(t, store.Save(ctx, app.RefreshRecord{ID: "jti-2", UserID: uid, ExpiresAt: now.Add(time.Hour)}))
	require.NoError(t, store.Revoke(ctx, "jti-2"))
	_, err = store.Consume(ctx, "jti-2", now)
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
}
