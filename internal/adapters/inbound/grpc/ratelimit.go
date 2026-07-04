package grpcadapter

import (
	"context"
	"net"
	"net/netip"
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
	// trustedProxies are the source CIDRs whose X-Forwarded-For is honored when
	// deriving the anonymous client IP. X-Forwarded-For is client-settable, so it
	// is trusted ONLY from a known proxy/LB hop; a direct (untrusted) peer's XFF is
	// ignored and its real connection IP used, closing rate-limit evasion by header
	// spoofing (CAL-120). Defaults to loopback — the co-located REST gateway.
	trustedProxies []netip.Prefix
}

// RateLimiterOption customizes a RateLimiter.
type RateLimiterOption func(*RateLimiter)

// loopbackProxies is the default trusted-proxy set: the REST gateway dials the
// gRPC server from localhost, so its relayed X-Forwarded-For is honored.
func loopbackProxies() []netip.Prefix {
	return []netip.Prefix{
		netip.MustParsePrefix("127.0.0.0/8"),
		netip.MustParsePrefix("::1/128"),
	}
}

// WithTrustedProxies adds proxy/LB source CIDRs (beyond loopback) whose
// X-Forwarded-For header is honored. Unparseable entries are skipped.
func WithTrustedProxies(cidrs []string) RateLimiterOption {
	return func(r *RateLimiter) {
		for _, c := range cidrs {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			if p, err := netip.ParsePrefix(c); err == nil {
				r.trustedProxies = append(r.trustedProxies, p)
				continue
			}
			// Accept a bare IP as a host route (/32 or /128).
			if a, err := netip.ParseAddr(c); err == nil {
				r.trustedProxies = append(r.trustedProxies, netip.PrefixFrom(a, a.BitLen()))
			}
		}
	}
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
func NewRateLimiter(ratePerSec, burst float64, now func() time.Time, opts ...RateLimiterOption) *RateLimiter {
	if ratePerSec <= 0 {
		ratePerSec = 1
	}
	if burst < 1 {
		burst = 1
	}
	r := &RateLimiter{
		buckets:        make(map[string]*tokenBucket),
		rate:           ratePerSec,
		burst:          burst,
		now:            now,
		trustedProxies: loopbackProxies(),
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Allow reports whether a request for key may proceed, consuming one token if so.
func (r *RateLimiter) Allow(key string) bool { return r.AllowN(key, 1) }

// AllowN reports whether a request for key may proceed at the given token cost,
// consuming that many tokens if so. A cost above the burst ceiling is clamped to
// it, so an expensive class is never permanently blocked — it just consumes a
// full bucket. Costs let pricier RPCs (LLM fan-out) drain a principal's bucket
// faster than cheap reads, so one caller cannot burst-fire them (CAL-112).
func (r *RateLimiter) AllowN(key string, cost float64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cost < 1 {
		cost = 1
	}
	if cost > r.burst {
		cost = r.burst
	}
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
		// A fresh key starts full, then immediately spends its cost below.
		b = &tokenBucket{tokens: r.burst, last: t}
		r.buckets[key] = b
	} else {
		elapsed := t.Sub(b.last).Seconds()
		if elapsed > 0 {
			b.tokens = min(r.burst, b.tokens+elapsed*r.rate)
			b.last = t
		}
	}
	if b.tokens >= cost {
		b.tokens -= cost
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

// llmMethodCost is the token cost charged for an LLM-heavy RPC — one that fans
// out to many/expensive model calls. It is far higher than a cheap read's cost of
// 1, so a single principal drains their bucket after only a few such calls and
// cannot burst-fire them to monopolize the shared LLM capacity (CAL-112).
const llmMethodCost = 12

// methodCost is the rate-limit token cost for a method: the RPCs whose handlers
// drive expensive model fan-out (shortlist scoring, the autonomous agent,
// role-spec generation, CV extraction, the interview loop) cost llmMethodCost,
// so they drain a caller's bucket far faster than a cheap read (cost 1).
func methodCost(fullMethod string) float64 {
	switch fullMethod {
	case "/caliber.v1.MatchingService/GenerateShortlist",
		"/caliber.v1.MatchingService/RefineShortlist",
		"/caliber.v1.CandidateAgentService/RunAgent",
		"/caliber.v1.RoleService/GenerateRoleSpec",
		"/caliber.v1.RoleService/UpdateRoleSpec",
		"/caliber.v1.TalentService/CreateProfileFromCV",
		"/caliber.v1.InterviewService/StartInterview",
		"/caliber.v1.InterviewService/SubmitAnswer":
		return llmMethodCost
	default:
		return 1
	}
}

// NewRateLimitInterceptor returns a unary interceptor that enforces the limiter.
// It keys by the authenticated principal when present (so a logged-in user's
// quota follows them across methods), falling back to a per-IP, per-method
// anonymous bucket otherwise. Expensive LLM RPCs are charged a higher token cost
// so they cannot be burst-fired. Over-limit requests are rejected with
// ResourceExhausted before reaching the handler. Place it after the auth
// interceptor so the principal is available.
func NewRateLimitInterceptor(limiter *RateLimiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !limiter.AllowN(limiter.rateLimitKey(ctx, info.FullMethod), methodCost(info.FullMethod)) {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded; please slow down")
		}
		return handler(ctx, req)
	}
}

// NewRateLimitStreamInterceptor is NewRateLimitInterceptor for streaming RPCs.
// Unary interceptors never run for streams, so without this the one streaming
// RPC (StartInterview) — which drives an LLM-backed interview loop — would bypass
// the limiter entirely, an LLM cost / goroutine amplification vector (CAL-112).
// It checks the bucket once at stream open (at the LLM method cost), before the
// handler runs. Place it after the auth stream interceptor so the principal is
// available for keying.
func NewRateLimitStreamInterceptor(limiter *RateLimiter) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !limiter.AllowN(limiter.rateLimitKey(ss.Context(), info.FullMethod), methodCost(info.FullMethod)) {
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
func (r *RateLimiter) rateLimitKey(ctx context.Context, fullMethod string) string {
	if p, ok := PrincipalFromContext(ctx); ok {
		return "user:" + p.UserID.String()
	}
	return "anon:" + r.clientIP(ctx) + ":" + fullMethod
}

// clientIP extracts the caller's IP for anonymous rate-limit keying. It uses the
// direct connection (peer) IP by default; the left-most X-Forwarded-For entry is
// honored ONLY when the peer is a trusted proxy/LB, because XFF is client-settable
// and would otherwise let an attacker forge a fresh source per request to evade
// the per-IP bucket. Only the IP is used — never the port — so a client cannot
// evade its bucket by opening fresh connections. Returns "unknown" when no IP is
// available, so those requests share one conservative bucket.
func (r *RateLimiter) clientIP(ctx context.Context) string {
	peer := peerIP(ctx)
	if r.peerIsTrusted(peer) {
		if fwd := forwardedForClient(ctx); fwd != "" {
			return fwd
		}
	}
	return peer
}

// peerIsTrusted reports whether ip parses and falls within a trusted-proxy CIDR.
func (r *RateLimiter) peerIsTrusted(ip string) bool {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	for _, p := range r.trustedProxies {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

// peerIP returns the direct connection's IP without the port, or "unknown".
func peerIP(ctx context.Context) string {
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		if host, _, err := net.SplitHostPort(p.Addr.String()); err == nil {
			return host
		}
		return p.Addr.String()
	}
	return "unknown"
}

// forwardedForClient returns the left-most X-Forwarded-For entry, or "".
func forwardedForClient(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	for _, h := range md.Get("x-forwarded-for") {
		if ip := strings.TrimSpace(strings.Split(h, ",")[0]); ip != "" {
			return ip
		}
	}
	return ""
}
