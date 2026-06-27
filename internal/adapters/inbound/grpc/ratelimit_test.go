package grpcadapter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// fakeClock is a manually-advanced clock for deterministic limiter tests.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func TestRateLimiter_AllowsBurstThenDenies(t *testing.T) {
	clk := &fakeClock{t: time.Unix(1700000000, 0)}
	limiter := NewRateLimiter(1, 3, clk.now) // 1/sec sustained, burst 3

	// The burst is spent without any time passing.
	assert.True(t, limiter.Allow("user:a"))
	assert.True(t, limiter.Allow("user:a"))
	assert.True(t, limiter.Allow("user:a"))
	assert.False(t, limiter.Allow("user:a"), "fourth immediate request exceeds the burst")
}

func TestRateLimiter_RefillsOverTime(t *testing.T) {
	clk := &fakeClock{t: time.Unix(1700000000, 0)}
	limiter := NewRateLimiter(2, 2, clk.now) // 2/sec, burst 2

	require.True(t, limiter.Allow("k"))
	require.True(t, limiter.Allow("k"))
	require.False(t, limiter.Allow("k"))

	clk.advance(time.Second) // +2 tokens at 2/sec, capped at burst 2
	assert.True(t, limiter.Allow("k"))
	assert.True(t, limiter.Allow("k"))
	assert.False(t, limiter.Allow("k"), "refill is capped at the burst ceiling")
}

func TestRateLimiter_KeysAreIndependent(t *testing.T) {
	clk := &fakeClock{t: time.Unix(1700000000, 0)}
	limiter := NewRateLimiter(1, 1, clk.now)

	assert.True(t, limiter.Allow("user:a"))
	assert.False(t, limiter.Allow("user:a"))
	// A different principal has its own bucket and is unaffected.
	assert.True(t, limiter.Allow("user:b"))
}

func TestRateLimiter_ClampsNonPositiveConfig(t *testing.T) {
	clk := &fakeClock{t: time.Unix(1700000000, 0)}
	limiter := NewRateLimiter(0, 0, clk.now) // clamped so it never locks everyone out
	assert.True(t, limiter.Allow("k"), "a misconfigured limiter still admits some traffic")
}

func okHandler(_ context.Context, _ any) (any, error) { return "ok", nil }

func TestRateLimitInterceptor_RejectsOverLimit(t *testing.T) {
	clk := &fakeClock{t: time.Unix(1700000000, 0)}
	limiter := NewRateLimiter(1, 1, clk.now) // burst 1
	interceptor := NewRateLimitInterceptor(limiter)
	info := &grpc.UnaryServerInfo{FullMethod: "/caliber.v1.MatchingService/GenerateShortlist"}
	ctx := context.WithValue(context.Background(), principalKey{},
		app.Principal{UserID: kernel.NewID(), Role: identity.RoleEmployer.String()})

	resp, err := interceptor(ctx, nil, info, okHandler)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)

	// The same principal's next call is over the burst -> ResourceExhausted, and
	// the handler is never reached.
	_, err = interceptor(ctx, nil, info, okHandler)
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))
}

func TestRateLimitInterceptor_KeysAnonByMethod(t *testing.T) {
	clk := &fakeClock{t: time.Unix(1700000000, 0)}
	limiter := NewRateLimiter(1, 1, clk.now)
	interceptor := NewRateLimitInterceptor(limiter)

	a := &grpc.UnaryServerInfo{FullMethod: "/caliber.v1.IdentityService/Login"}
	b := &grpc.UnaryServerInfo{FullMethod: "/caliber.v1.IdentityService/Register"}

	// Anonymous (no principal) requests are bucketed per method, so exhausting one
	// method does not block a different one.
	_, err := interceptor(context.Background(), nil, a, okHandler)
	require.NoError(t, err)
	_, err = interceptor(context.Background(), nil, a, okHandler)
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
	_, err = interceptor(context.Background(), nil, b, okHandler)
	assert.NoError(t, err, "a different anonymous method has its own bucket")
}
