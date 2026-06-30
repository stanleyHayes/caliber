package dashboard

import (
	"sync"
	"time"
)

// clockFunc abstracts time.Now so tests can control expiry deterministically.
type clockFunc func() time.Time

// cacheEntry is a single TTL-stamped snapshot value.
type cacheEntry struct {
	value  any
	expiry time.Time
}

// snapshotCache is a simple in-memory TTL cache for dashboard read models.
// It is intentionally minimal (a single mutex + map) because the POC dashboard
// only holds small, bounded snapshot values.
type snapshotCache struct {
	mu    sync.RWMutex
	items map[string]cacheEntry
	ttl   time.Duration
	now   clockFunc
}

// newSnapshotCache creates a cache. A nil now falls back to time.Now.
func newSnapshotCache(ttl time.Duration, now clockFunc) *snapshotCache {
	if now == nil {
		now = time.Now
	}
	return &snapshotCache{
		items: make(map[string]cacheEntry),
		ttl:   ttl,
		now:   now,
	}
}

// get returns a cached value when it exists and has not expired.
func (c *snapshotCache) get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.items[key]
	if !ok || c.now().After(e.expiry) {
		return nil, false
	}
	return e.value, true
}

// set stores a value with the cache's TTL.
func (c *snapshotCache) set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = cacheEntry{value: value, expiry: c.now().Add(c.ttl)}
}

// invalidateAll clears every cached snapshot.
func (c *snapshotCache) invalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]cacheEntry)
}
