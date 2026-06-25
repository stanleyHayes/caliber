package interview

import (
	"context"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// InterviewRepository persists and retrieves interviews.
type InterviewRepository interface { //nolint:revive // domain name fixed by the interview context spec
	Create(ctx context.Context, i *Interview) error
	ByID(ctx context.Context, id kernel.ID) (*Interview, error)
	Update(ctx context.Context, i *Interview) error
	ByCandidate(ctx context.Context, candidateID kernel.ID, page kernel.Page) ([]*Interview, int64, error)
}
