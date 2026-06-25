package memory

import (
	"context"
	"sync"
	"time"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// Default brute-force lockout policy.
const (
	DefaultMaxFailures = 5
	DefaultWindow      = 15 * time.Minute
	DefaultLockout     = 15 * time.Minute
)

type attemptState struct {
	count       int
	windowStart time.Time
	lockedUntil time.Time
}

// LoginThrottle is an in-memory app.LoginThrottle: after MaxFailures failed
// attempts within Window, a key is locked out for Lockout.
type LoginThrottle struct {
	mu      sync.Mutex
	now     func() time.Time
	maxFail int
	window  time.Duration
	lockout time.Duration
	byKey   map[string]*attemptState
}

// NewLoginThrottle builds a throttle. A nil clock defaults to time.Now;
// non-positive policy values fall back to the package defaults.
func NewLoginThrottle(now func() time.Time, maxFail int, window, lockout time.Duration) *LoginThrottle {
	if now == nil {
		now = time.Now
	}
	if maxFail <= 0 {
		maxFail = DefaultMaxFailures
	}
	if window <= 0 {
		window = DefaultWindow
	}
	if lockout <= 0 {
		lockout = DefaultLockout
	}
	return &LoginThrottle{now: now, maxFail: maxFail, window: window, lockout: lockout, byKey: map[string]*attemptState{}}
}

// Check returns a lockout error if the key is currently locked out.
func (t *LoginThrottle) Check(_ context.Context, key string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if st := t.byKey[key]; st != nil && t.now().Before(st.lockedUntil) {
		return kernel.TooManyRequests("auth: too many failed attempts; please try again later")
	}
	return nil
}

// Fail records a failed attempt, locking the key once the threshold is reached
// within the sliding window.
func (t *LoginThrottle) Fail(_ context.Context, key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	st := t.byKey[key]
	if st == nil || now.Sub(st.windowStart) > t.window {
		st = &attemptState{windowStart: now}
		t.byKey[key] = st
	}
	st.count++
	if st.count >= t.maxFail {
		st.lockedUntil = now.Add(t.lockout)
	}
}

// Reset clears the attempt counter for the key.
func (t *LoginThrottle) Reset(_ context.Context, key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.byKey, key)
}

var _ app.LoginThrottle = (*LoginThrottle)(nil)
