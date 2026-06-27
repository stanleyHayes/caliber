package memory

import (
	"context"

	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

// Recaller is an in-memory CandidateRecaller for the dev stack. With no real
// embeddings it ignores the role vector and returns the candidate pool; the hard
// filters and scorer then produce the shortlist. Production uses pgvector recall.
type Recaller struct {
	candidates talent.CandidateRepository
}

// NewRecaller builds an in-memory recaller over the candidate repository.
func NewRecaller(candidates talent.CandidateRepository) *Recaller {
	return &Recaller{candidates: candidates}
}

// Recall returns up to limit candidate ids from the pool (embedding ignored).
func (r *Recaller) Recall(ctx context.Context, _ []float32, limit int) ([]kernel.ID, error) {
	if limit <= 0 {
		limit = 100
	}
	cands, _, err := r.candidates.List(ctx, kernel.NewPage(1, limit))
	if err != nil {
		return nil, err
	}
	ids := make([]kernel.ID, 0, len(cands))
	for _, c := range cands {
		ids = append(ids, c.ID)
	}
	return ids, nil
}
