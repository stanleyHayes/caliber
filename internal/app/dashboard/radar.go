package dashboard

import (
	"context"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// TalentRadar is the read-model surface exposed to inbound adapters. Both the
// raw Aggregator and the CachedAggregator implement it, so callers can swap
// caching in without changing the server wiring.
type TalentRadar interface {
	Pool(ctx context.Context, page kernel.Page) ([]PoolCandidate, int64, error)
	SupplyDemand(ctx context.Context) ([]SupplyDemandItem, error)
	Alerts(ctx context.Context, page kernel.Page) ([]MatchAlert, int64, error)
	TimeToShortlist(ctx context.Context) TimeToShortlist
}

// Ensure the concrete types satisfy the interface.
var (
	_ TalentRadar = (*Aggregator)(nil)
	_ TalentRadar = (*CachedAggregator)(nil)
)
