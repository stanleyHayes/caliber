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
	mu        sync.Mutex
	buckets   map[string]*tokenBucket
	rate      float64 // tokens added per second
	burst     float64 // maximum tokens a bucket can hold
	now       func() time.Time
	lastSweep time.Time // last time evictIdle ran, to throttle the O(n) scan
}

type tokenBucket struct {
	tokens float64
	last   time.Time
}

// sweepThreshold is the bucket count past which Allow opportunistically evicts
// idle buckets. Keying anonymous traffic per client IP (and honoring a
// client-influenced X-Forwarded-For) means the map's cardinality is attacker-
// driven, so it must be bounded or it becomes a slow memory-exhaustion vector
// (CAL-120 L7).
const sweepThreshold = 10_000

// idleEvictAfter is how long a bucket may sit untouched before it is evicted. A
// bucket idle this long has fully refilled, so deleting it is behavior-
// preserving: the next request for that key re-creates an identical full bucket.
const idleEvictAfter = 10 * time.Minute

// maxBuckets is the hard ceiling on tracked buckets. Idle eviction is best-effort
// — it frees nothing when every bucket is hot — so under a spoofed-IP flood
// (X-Forwarded-For is client-influenced) the map could still grow without bound.
// This cap makes the bound absolute: at the ceiling, admitting a new key first
// reclaims headroom via evictBatch (CAL-120 L7).
const maxBuckets = 2 * sweepThreshold

// evictLowWater is the size evictBatch reclaims down to when the hard cap is hit,
// leaving ~10% headroom so the batch runs once per ~maxBuckets/10 new keys — an
// amortized O(1) cost per admitted key, versus an O(maxBuckets) scan every time.
const evictLowWater = maxBuckets - maxBuckets/10

// sweepInterval throttles idle eviction. The idle sweep is O(n), so running it on
// every new-key admission once the map is large would let a spoofed-IP flood pin
// every Allow() behind a full-map scan under the lock. evictBatch enforces the
// hard cap regardless; this sweep only keeps the map tidy under normal churn, so
// once per interval is ample and makes its amortized cost O(1) under a flood.
const sweepInterval = time.Second

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
		// Bound the map before admitting a brand-new key, so a flood of distinct
		// keys (per-IP anonymous traffic) cannot grow it without limit (L7). The
		// idle scan is throttled so a sustained flood does not pay it per request.
		if len(r.buckets) >= sweepThreshold && t.Sub(r.lastSweep) >= sweepInterval {
			r.evictIdle(t)
			r.lastSweep = t
		}
		// Idle eviction frees nothing when every bucket is hot; the hard cap then
		// batch-reclaims headroom so the map can never exceed it.
		if len(r.buckets) >= maxBuckets {
			r.evictBatch()
		}
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

// evictIdle removes buckets untouched for longer than idleEvictAfter. Such a
// bucket has fully refilled, so its removal is behavior-preserving — the next
// request for that key re-creates an identical full bucket. The caller holds the
// lock. If nothing is idle (every key is hot), the map simply stays at its high-
// water mark; that is the genuine working set, not a leak.
func (r *RateLimiter) evictIdle(now time.Time) {
	for key, b := range r.buckets {
		if now.Sub(b.last) > idleEvictAfter {
			delete(r.buckets, key)
		}
	}
}

// evictBatch reclaims headroom when the map is at its hard cap and idle eviction
// freed nothing (every bucket is hot). It drops buckets down to evictLowWater in
// a single partial pass, so the amortized cost per admitted key stays O(1) under
// a sustained flood rather than an O(maxBuckets) scan on every new key. Which
// specific buckets go is immaterial: a re-created bucket starts full, so a
// legitimate key is at worst granted a limit reset, never locked out. The caller
// holds the lock.
func (r *RateLimiter) evictBatch() {
	for key := range r.buckets {
		if len(r.buckets) <= evictLowWater {
			return
		}
		delete(r.buckets, key)
	}
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

// NewRateLimitStreamInterceptor is NewRateLimitInterceptor for streaming RPCs.
// Unary interceptors never run for streams, so without this the one streaming
// RPC (StartInterview) — which drives an LLM-backed interview loop — would bypass
// the limiter entirely, an LLM cost / goroutine amplification vector (CAL-112).
// It checks the bucket once at stream open, before the handler runs. Place it
// after the auth stream interceptor so the principal is available for keying.
func NewRateLimitStreamInterceptor(limiter *RateLimiter) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !limiter.Allow(rateLimitKey(ss.Context(), info.FullMethod)) {
			return status.Error(codes.ResourceExhausted, "rate limit exceeded; please slow down")
		}
		return handler(srv, ss)
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
