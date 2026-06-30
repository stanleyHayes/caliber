package dashboard

import (
	"context"
	"fmt"
	"time"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// DefaultCacheTTL is the snapshot freshness window for the Talent Radar.
// A 30-second TTL keeps live demos snappy while still allowing updates to
// become visible within a reasonable refresh window.
const DefaultCacheTTL = 30 * time.Second

// poolCacheValue is the cached shape for a paginated pool view.
type poolCacheValue struct {
	candidates []PoolCandidate
	total      int64
}

// CachedAggregator wraps Aggregator with a TTL snapshot cache so the Talent
// Radar renders snappy live views without recomputing expensive aggregations
// on every request.
type CachedAggregator struct {
	agg   *Aggregator
	cache *snapshotCache
}

// NewCachedAggregator wires a cached decorator around the given aggregator.
// A nil now falls back to time.Now.
func NewCachedAggregator(agg *Aggregator, ttl time.Duration, now func() time.Time) *CachedAggregator {
	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}
	return &CachedAggregator{
		agg:   agg,
		cache: newSnapshotCache(ttl, now),
	}
}

// Pool returns a cached paginated pool view.
func (c *CachedAggregator) Pool(ctx context.Context, page kernel.Page) ([]PoolCandidate, int64, error) {
	key := fmt.Sprintf("pool:n:%d:s:%d", page.Number, page.Size)
	if v, ok := c.cache.get(key); ok {
		if res, ok := v.(poolCacheValue); ok {
			return res.candidates, res.total, nil
		}
	}
	candidates, total, err := c.agg.Pool(ctx, page)
	if err != nil {
		return nil, 0, err
	}
	c.cache.set(key, poolCacheValue{candidates: candidates, total: total})
	return candidates, total, nil
}

// SupplyDemand returns a cached supply/demand snapshot.
func (c *CachedAggregator) SupplyDemand(ctx context.Context) ([]SupplyDemandItem, error) {
	const key = "supplydemand"
	if v, ok := c.cache.get(key); ok {
		if res, ok := v.([]SupplyDemandItem); ok {
			return res, nil
		}
	}
	items, err := c.agg.SupplyDemand(ctx)
	if err != nil {
		return nil, err
	}
	c.cache.set(key, items)
	return items, nil
}

// Alerts caches the full alert list and serves paginated slices from memory.
func (c *CachedAggregator) Alerts(ctx context.Context, page kernel.Page) ([]MatchAlert, int64, error) {
	const key = "alerts"
	if v, ok := c.cache.get(key); ok {
		if all, ok := v.([]MatchAlert); ok {
			return pageAlerts(all, page), int64(len(all)), nil
		}
	}
	alerts, _, err := c.agg.Alerts(ctx, kernel.NewPage(1, alertCandidateScan))
	if err != nil {
		return nil, 0, err
	}
	c.cache.set(key, alerts)
	return pageAlerts(alerts, page), int64(len(alerts)), nil
}

// TimeToShortlist returns a cached headline metric.
func (c *CachedAggregator) TimeToShortlist(ctx context.Context) TimeToShortlist {
	const key = "tts"
	if v, ok := c.cache.get(key); ok {
		if m, ok := v.(TimeToShortlist); ok {
			return m
		}
	}
	m := c.agg.TimeToShortlist(ctx)
	c.cache.set(key, m)
	return m
}

// RefreshSnapshots clears cached snapshots so the next call recomputes fresh data.
func (c *CachedAggregator) RefreshSnapshots() {
	c.cache.invalidateAll()
}
