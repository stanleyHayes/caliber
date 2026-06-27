package memory

import (
	"context"
	"slices"
	"sync"

	"github.com/xcreativs/caliber/internal/domain/contest"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// ContestRepo is an in-memory contest.ContestRepository.
type ContestRepo struct {
	mu    sync.RWMutex
	byID  map[kernel.ID]contest.Contest
	order []kernel.ID // insertion order; listings return newest first
}

// NewContestRepo builds an empty in-memory contest repository.
func NewContestRepo() *ContestRepo {
	return &ContestRepo{byID: map[kernel.ID]contest.Contest{}}
}

// Create stores a new contest.
func (r *ContestRepo) Create(_ context.Context, c *contest.Contest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[c.ID] = *c
	r.order = append(r.order, c.ID)
	return nil
}

// ByID returns a contest by id.
func (r *ContestRepo) ByID(_ context.Context, id kernel.ID) (*contest.Contest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.byID[id]
	if !ok {
		return nil, kernel.NotFound("contest: not found")
	}
	return &c, nil
}

// Update persists a mutated contest.
func (r *ContestRepo) Update(_ context.Context, c *contest.Contest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[c.ID]; !ok {
		return kernel.NotFound("contest: not found")
	}
	r.byID[c.ID] = *c
	return nil
}

// ByCandidate lists a candidate's contests, newest first, paginated.
func (r *ContestRepo) ByCandidate(_ context.Context, candidateID kernel.ID, page kernel.Page) ([]*contest.Contest, int64, error) {
	return r.list(func(c contest.Contest) bool { return c.CandidateID == candidateID }, page)
}

// BySubject lists contests against an assessment, newest first, paginated.
func (r *ContestRepo) BySubject(_ context.Context, subjectID kernel.ID, page kernel.Page) ([]*contest.Contest, int64, error) {
	return r.list(func(c contest.Contest) bool { return c.SubjectID == subjectID }, page)
}

func (r *ContestRepo) list(keep func(contest.Contest) bool, page kernel.Page) ([]*contest.Contest, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []*contest.Contest
	for _, id := range slices.Backward(r.order) {
		if c := r.byID[id]; keep(c) {
			cp := c
			all = append(all, &cp)
		}
	}
	total := int64(len(all))
	start := min(page.Offset(), len(all))
	end := min(start+page.Limit(), len(all))
	return all[start:end], total, nil
}
