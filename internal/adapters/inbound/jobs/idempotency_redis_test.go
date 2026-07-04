package jobs_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/inbound/jobs"
)

func TestRedisIdempotencyStore(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := jobs.NewRedisIdempotencyStore(client)
	ctx := context.Background()

	// First claim wins; a duplicate delivery / a second worker is rejected.
	ok, err := store.Claim(ctx, "task-1")
	require.NoError(t, err)
	assert.True(t, ok)
	ok, err = store.Claim(ctx, "task-1")
	require.NoError(t, err)
	assert.False(t, ok, "a claimed key cannot be re-claimed across the fleet")

	// Releasing an in-progress claim (a failed task) makes it retryable.
	require.NoError(t, store.Release(ctx, "task-1"))
	ok, err = store.Claim(ctx, "task-1")
	require.NoError(t, err)
	assert.True(t, ok, "a released key is re-claimable for retry")

	// Completing retains the key so duplicates stay suppressed — even Release
	// must not drop a completed key.
	require.NoError(t, store.Complete(ctx, "task-1"))
	ok, err = store.Claim(ctx, "task-1")
	require.NoError(t, err)
	assert.False(t, ok, "a completed key suppresses duplicate delivery")
	require.NoError(t, store.Release(ctx, "task-1"))
	ok, err = store.Claim(ctx, "task-1")
	require.NoError(t, err)
	assert.False(t, ok, "Release does not drop a completed key")

	// Empty keys are rejected on every method.
	_, err = store.Claim(ctx, "   ")
	require.Error(t, err)
	require.Error(t, store.Complete(ctx, ""))
	require.Error(t, store.Release(ctx, ""))
}
