package memory

import (
	"context"

	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

// CandidateRepo is an in-memory talent.CandidateRepository for local/dev runs.
type CandidateRepo struct {
	store *keyedStore[talent.Candidate]
}

// NewCandidateRepo builds an empty in-memory candidate repository.
func NewCandidateRepo() *CandidateRepo {
	return &CandidateRepo{store: newKeyedStore(
		func(c *talent.Candidate) kernel.ID { return c.ID },
		func(c *talent.Candidate) kernel.ID { return c.UserID },
	)}
}

// Create stores a new candidate, rejecting a duplicate user.
func (r *CandidateRepo) Create(_ context.Context, c *talent.Candidate) error {
	return r.store.create(c, "memory: candidate already exists for user")
}

// ByID returns the candidate with the given id.
func (r *CandidateRepo) ByID(_ context.Context, id kernel.ID) (*talent.Candidate, error) {
	return r.store.get(id, "memory: candidate not found")
}

// ByUserID returns the candidate belonging to a user.
func (r *CandidateRepo) ByUserID(_ context.Context, userID kernel.ID) (*talent.Candidate, error) {
	return r.store.getBySecondary(userID, "memory: candidate not found")
}

// Update replaces an existing candidate.
func (r *CandidateRepo) Update(_ context.Context, c *talent.Candidate) error {
	return r.store.update(c, "memory: candidate not found")
}

// List returns a page of candidates and the total count.
func (r *CandidateRepo) List(_ context.Context, page kernel.Page) ([]*talent.Candidate, int64, error) {
	out, total := r.store.list(page)
	return out, total, nil
}

// Delete hard-removes a candidate by id (right-to-erasure cascade, CAL-118).
// Removing an absent candidate is a no-op, so erasure is idempotent.
func (r *CandidateRepo) Delete(_ context.Context, id kernel.ID) error {
	r.store.deleteByPrimary(id)
	return nil
}

var _ talent.CandidateRepository = (*CandidateRepo)(nil)
