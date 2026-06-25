package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

func TestMemoryRefreshStoreRotation(t *testing.T) {
	ctx := context.Background()
	s := memory.NewRefreshStore()
	now := time.Unix(1700000000, 0)
	rec := app.RefreshRecord{ID: "jti-1", UserID: kernel.NewID(), ExpiresAt: now.Add(time.Hour)}
	require.NoError(t, s.Save(ctx, rec))

	got, err := s.Consume(ctx, "jti-1", now)
	require.NoError(t, err)
	assert.Equal(t, rec.UserID, got.UserID)

	// Single-use: a second consume (replay) is rejected.
	_, err = s.Consume(ctx, "jti-1", now)
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
}

func TestMemoryRefreshStoreExpiryAndUnknown(t *testing.T) {
	ctx := context.Background()
	s := memory.NewRefreshStore()
	now := time.Unix(1700000000, 0)
	require.NoError(t, s.Save(ctx, app.RefreshRecord{ID: "exp", ExpiresAt: now.Add(-time.Second)}))

	_, err := s.Consume(ctx, "exp", now)
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err), "expired rejected")
	_, err = s.Consume(ctx, "nope", now)
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err), "unknown rejected")
}

func TestMemoryRefreshStoreRevoke(t *testing.T) {
	ctx := context.Background()
	s := memory.NewRefreshStore()
	now := time.Unix(1700000000, 0)
	require.NoError(t, s.Save(ctx, app.RefreshRecord{ID: "jti", ExpiresAt: now.Add(time.Hour)}))
	require.NoError(t, s.Revoke(ctx, "jti"))

	_, err := s.Consume(ctx, "jti", now)
	assert.Equal(t, kernel.KindUnauthorized, kernel.KindOf(err))
	require.NoError(t, s.Revoke(ctx, "unknown-is-noop"))
}
