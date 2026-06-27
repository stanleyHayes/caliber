package contest

import (
	"context"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

//go:generate mockgen -source=ports.go -destination=../../mocks/contest.go -package=mocks

// ContestRepository is the persistence port for assessment contests.
type ContestRepository interface { //nolint:revive // XRepository naming is the project convention
	// Create durably stores a new contest.
	Create(ctx context.Context, c *Contest) error
	// ByID returns a contest by id (kernel.NotFound when absent).
	ByID(ctx context.Context, id kernel.ID) (*Contest, error)
	// ByCandidate lists a candidate's contests, newest first, paginated.
	ByCandidate(ctx context.Context, candidateID kernel.ID, page kernel.Page) ([]*Contest, int64, error)
	// BySubject lists contests against a given assessment, newest first
	// (for the employer/reviewer side), paginated.
	BySubject(ctx context.Context, subjectID kernel.ID, page kernel.Page) ([]*Contest, int64, error)
	// Update persists a mutated contest (e.g. after resolution).
	Update(ctx context.Context, c *Contest) error
}
