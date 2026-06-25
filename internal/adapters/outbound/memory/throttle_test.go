package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

func TestLoginThrottleLocksOutAndResets(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(1700000000, 0)
	th := memory.NewLoginThrottle(func() time.Time { return now }, 3, time.Hour, time.Hour)

	require.NoError(t, th.Check(ctx, "k"))
	th.Fail(ctx, "k")
	th.Fail(ctx, "k")
	require.NoError(t, th.Check(ctx, "k"), "still under threshold")
	th.Fail(ctx, "k") // third failure -> locked
	assert.Equal(t, kernel.KindTooManyRequests, kernel.KindOf(th.Check(ctx, "k")))

	th.Reset(ctx, "k")
	require.NoError(t, th.Check(ctx, "k"), "reset clears the lockout")
}

func TestLoginThrottleLockoutExpires(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(1700000000, 0)
	th := memory.NewLoginThrottle(func() time.Time { return now }, 2, time.Hour, time.Hour)

	th.Fail(ctx, "k")
	th.Fail(ctx, "k")
	assert.Equal(t, kernel.KindTooManyRequests, kernel.KindOf(th.Check(ctx, "k")))

	now = now.Add(2 * time.Hour) // past the lockout window
	require.NoError(t, th.Check(ctx, "k"))
}

func TestLoginThrottleWindowResetsCounter(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(1700000000, 0)
	th := memory.NewLoginThrottle(func() time.Time { return now }, 3, time.Minute, time.Hour)

	th.Fail(ctx, "k")
	th.Fail(ctx, "k")
	now = now.Add(2 * time.Minute) // outside the sliding window -> counter restarts
	th.Fail(ctx, "k")
	require.NoError(t, th.Check(ctx, "k"), "stale failures do not accumulate into a lockout")
}

func TestLoginThrottleDefaults(t *testing.T) {
	// non-positive policy values fall back to package defaults; nil clock is safe.
	th := memory.NewLoginThrottle(nil, 0, 0, 0)
	require.NoError(t, th.Check(context.Background(), "k"))
}
