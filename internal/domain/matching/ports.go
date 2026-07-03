package matching

import (
	"context"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

//go:generate mockgen -source=ports.go -destination=../../mocks/matching.go -package=mocks

// MatchRepository is the persistence port for Match aggregates. List methods
// are paginated via kernel.Page and return the page slice together with the
// total count of matching records.
type MatchRepository interface {
	// Upsert inserts or updates the given match.
	Upsert(ctx context.Context, m *Match) error
	// ByID returns a single match by its id, or a kernel.KindNotFound error when
	// absent. Used to resolve a match's owning role for tenant-ownership checks.
	ByID(ctx context.Context, id kernel.ID) (*Match, error)
	// ByRole returns the matches for a role, ordered by the adapter, paginated.
	ByRole(ctx context.Context, roleID kernel.ID, page kernel.Page) ([]*Match, int64, error)
	// ForCandidate returns the matches for a candidate, paginated.
	ForCandidate(ctx context.Context, candidateID kernel.ID, page kernel.Page) ([]*Match, int64, error)
}
