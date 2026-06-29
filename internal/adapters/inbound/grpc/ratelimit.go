package grpcadapter

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// RateLimiter is a per-key token-bucket limiter (CAL-112). Each key (an
// authenticated user, or an anonymous fallback per method) gets a bucket that
// refills at a steady rate up to a burst ceiling; a request consumes one token,
// and is denied when the bucket is empty. It is safe for concurrent use.
//
// This protects the API — especially the expensive AI endpoints — from flooding
// and runaway clients. It is deliberately coarse and in-memory; a distributed
// deployment would back it with Redis, but the algorithm is identical.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	rate    float64 // tokens added per second
	burst   float64 // maximum tokens a bucket can hold
	now     func() time.Time
}

type tokenBucket struct {
	tokens float64
	last   time.Time
}

// NewRateLimiter builds a limiter allowing ratePerSec sustained requests with a
// burst ceiling, using now as its clock (injectable for tests). A non-positive
// rate or burst is clamped to a small positive value so the limiter always
// admits some traffic rather than locking everyone out.
func NewRateLimiter(ratePerSec, burst float64, now func() time.Time) *RateLimiter {
	if ratePerSec <= 0 {
		ratePerSec = 1
	}
	if burst < 1 {
		burst = 1
	}
	return &RateLimiter{
		buckets: make(map[string]*tokenBucket),
		rate:    ratePerSec,
		burst:   burst,
		now:     now,
	}
}

// Allow reports whether a request for key may proceed, consuming a token if so.
func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	t := r.now()
	b, ok := r.buckets[key]
	if !ok {
		// A fresh key starts full, then immediately spends one token below.
		b = &tokenBucket{tokens: r.burst, last: t}
		r.buckets[key] = b
	} else {
		elapsed := t.Sub(b.last).Seconds()
		if elapsed > 0 {
			b.tokens = min(r.burst, b.tokens+elapsed*r.rate)
			b.last = t
		}
	}
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// NewRateLimitInterceptor returns a unary interceptor that enforces the limiter.
// It keys by the authenticated principal when present (so a logged-in user's
// quota follows them across methods), falling back to a per-IP, per-method
// anonymous bucket otherwise. Over-limit requests are rejected with
// ResourceExhausted before reaching the handler. Place it after the auth
// interceptor so the principal is available.
func NewRateLimitInterceptor(limiter *RateLimiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !limiter.Allow(rateLimitKey(ctx, info.FullMethod)) {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded; please slow down")
		}
		return handler(ctx, req)
	}
}

// rateLimitKey derives the limiter bucket key for a request. Authenticated calls
// key by principal. Anonymous calls (login, register, refresh — the flood-prone
// pre-auth surface) key by *client IP and method*: keying by method alone would
// pool every anonymous caller into one bucket, so a single attacker could drain
// it and lock all other users out of logging in (a self-inflicted DoS). Per-IP
// isolation gives each source its own quota.
func rateLimitKey(ctx context.Context, fullMethod string) string {
	if p, ok := PrincipalFromContext(ctx); ok {
		return "user:" + p.UserID.String()
	}
	return "anon:" + clientIP(ctx) + ":" + fullMethod
}

// clientIP extracts the caller's IP, preferring the left-most X-Forwarded-For
// entry set by a trusted proxy/load balancer (and by the REST gateway, which
// dials the gRPC server from localhost) and falling back to the peer address.
// Only the IP is used — never the port — so a client cannot evade its bucket by
// opening fresh connections. Returns "unknown" when neither is available, so all
// such requests share one conservative bucket rather than going unlimited.
func clientIP(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		for _, h := range md.Get("x-forwarded-for") {
			if ip := strings.TrimSpace(strings.Split(h, ",")[0]); ip != "" {
				return ip
			}
		}
	}
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		if host, _, err := net.SplitHostPort(p.Addr.String()); err == nil {
			return host
		}
		return p.Addr.String()
	}
	return "unknown"
}
