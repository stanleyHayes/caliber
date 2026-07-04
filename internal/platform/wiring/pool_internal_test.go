package wiring

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/xcreativs/caliber/internal/platform/config"
)

// TestNewPGPoolAppliesEfSearch proves the CAL-156 recall-tuning knob: when
// HNSWEfSearch is configured, every pooled connection has hnsw.ef_search set to
// it; when unset, the pool applies no override.
func TestNewPGPoolAppliesEfSearch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testcontainers integration test in -short mode")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available")
	}
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

	// Configured: each connection sets the GUC.
	tuned, err := newPGPool(ctx, config.Config{DatabaseURL: dsn, HNSWEfSearch: 77})
	require.NoError(t, err)
	t.Cleanup(tuned.Close)
	var ef string
	require.NoError(t, tuned.QueryRow(ctx, "SHOW hnsw.ef_search").Scan(&ef))
	assert.Equal(t, "77", ef, "ef_search applied per connection")

	// Unset (0): the pool adds no AfterConnect hook, so the GUC is untouched.
	plain, err := newPGPool(ctx, config.Config{DatabaseURL: dsn})
	require.NoError(t, err)
	t.Cleanup(plain.Close)
	require.NotNil(t, plain)
}
