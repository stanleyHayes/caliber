package llm

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/guard"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// RateLimiter decides whether a model call may proceed now (request budget).
type RateLimiter interface {
	Allow() bool
}

// Guarded wraps an app.LLMClient with cost and safety controls (CAL-035): a hard
// per-call token cap, a concurrency limit, a request-budget rate limit, and
// prompt-injection telemetry over the untrusted prompt. It implements
// app.LLMClient so it composes transparently in front of any provider.
type Guarded struct {
	inner       app.LLMClient
	maxTokens   int
	sem         chan struct{}
	limiter     RateLimiter
	onInjection func(categories []string)
}

// GuardOption configures a Guarded client.
type GuardOption func(*Guarded)

// WithMaxTokens caps MaxTokens on every request (<=0 leaves requests untouched).
func WithMaxTokens(n int) GuardOption { return func(g *Guarded) { g.maxTokens = n } }

// WithConcurrency bounds simultaneous in-flight model calls (<=0 = unbounded).
func WithConcurrency(n int) GuardOption {
	return func(g *Guarded) {
		if n > 0 {
			g.sem = make(chan struct{}, n)
		}
	}
}

// WithRateLimiter enforces a request budget; denied calls fail fast with
// kernel.KindTooManyRequests before any provider cost is incurred.
func WithRateLimiter(rl RateLimiter) GuardOption { return func(g *Guarded) { g.limiter = rl } }

// WithInjectionHook is invoked (best-effort, before the call) with the detected
// prompt-injection categories when the prompt looks adversarial. It never blocks
// the call — a candidate's words are data, not a reason to refuse service — and
// receives only category labels, never prompt content, so logging stays
// PII-safe.
func WithInjectionHook(h func(categories []string)) GuardOption {
	return func(g *Guarded) { g.onInjection = h }
}

// NewGuarded wraps inner with the given controls.
func NewGuarded(inner app.LLMClient, opts ...GuardOption) *Guarded {
	g := &Guarded{inner: inner}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// Complete enforces the request budget, records injection telemetry, caps the
// token budget, and delegates under the concurrency limit.
func (g *Guarded) Complete(ctx context.Context, req app.LLMRequest) (app.LLMResponse, error) {
	if g.limiter != nil && !g.limiter.Allow() {
		return app.LLMResponse{}, kernel.TooManyRequests("llm: request budget exceeded; retry later")
	}
	g.reportInjection(req.Prompt)
	req.MaxTokens = g.cappedTokens(req.MaxTokens)

	release, err := g.acquire(ctx)
	if err != nil {
		return app.LLMResponse{}, err
	}
	defer release()
	return g.inner.Complete(ctx, req)
}

// reportInjection runs the advisory injection scan and hands any detected
// categories to the telemetry hook. It never blocks the call.
func (g *Guarded) reportInjection(prompt string) {
	if g.onInjection == nil {
		return
	}
	if cats := guard.ScanInjection(prompt); len(cats) > 0 {
		g.onInjection(cats)
	}
}

// cappedTokens clamps a requested token budget to the configured ceiling,
// treating an unset (<=0) budget as "use the cap".
func (g *Guarded) cappedTokens(requested int) int {
	if g.maxTokens > 0 && (requested <= 0 || requested > g.maxTokens) {
		return g.maxTokens
	}
	return requested
}

// acquire takes a concurrency slot, returning a release func. With no limit it
// returns a no-op release; if the context is canceled while waiting it errors.
func (g *Guarded) acquire(ctx context.Context) (func(), error) {
	if g.sem == nil {
		return func() {}, nil
	}
	select {
	case g.sem <- struct{}{}:
		return func() { <-g.sem }, nil
	case <-ctx.Done():
		return nil, kernel.Wrap(ctx.Err(), kernel.KindInternal, "llm: canceled awaiting a slot")
	}
}

// TokenBucket is a minimal, dependency-free token-bucket RateLimiter, safe for
// concurrent use. It refills at ratePerSec up to burst; the clock is injectable
// for deterministic tests.
type TokenBucket struct {
	mu         sync.Mutex
	ratePerSec float64
	burst      float64
	tokens     float64
	last       time.Time
	now        func() time.Time
}

// NewTokenBucket builds a token bucket that starts full. now defaults to
// time.Now when nil.
func NewTokenBucket(ratePerSec float64, burst int, now func() time.Time) *TokenBucket {
	if now == nil {
		now = time.Now
	}
	return &TokenBucket{
		ratePerSec: ratePerSec,
		burst:      float64(burst),
		tokens:     float64(burst),
		last:       now(),
		now:        now,
	}
}

// Allow consumes one token if available, refilling based on elapsed time.
func (b *TokenBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()
	if elapsed := now.Sub(b.last).Seconds(); elapsed > 0 {
		b.tokens = math.Min(b.burst, b.tokens+elapsed*b.ratePerSec)
		b.last = now
	}
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}
