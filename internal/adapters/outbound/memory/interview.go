package memory

import (
	"context"
	"sync"

	"github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// InterviewRepo is an in-memory interview.InterviewRepository for local/dev runs.
type InterviewRepo struct {
	mu   sync.RWMutex
	byID map[kernel.ID]interview.Interview
}

// NewInterviewRepo builds an empty in-memory interview repository.
func NewInterviewRepo() *InterviewRepo {
	return &InterviewRepo{byID: map[kernel.ID]interview.Interview{}}
}

// Reset clears every interview (test/dev reseed helper).
func (r *InterviewRepo) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID = map[kernel.ID]interview.Interview{}
}

// Create stores a new interview.
func (r *InterviewRepo) Create(_ context.Context, i *interview.Interview) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byID[i.ID]; exists {
		return kernel.Conflict("memory: interview already exists")
	}
	r.byID[i.ID] = *i
	return nil
}

// ByID returns a copy of the interview with the given id.
func (r *InterviewRepo) ByID(_ context.Context, id kernel.ID) (*interview.Interview, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	iv, ok := r.byID[id]
	if !ok {
		return nil, kernel.NotFound("memory: interview not found")
	}
	return &iv, nil
}

// Update replaces an existing interview.
func (r *InterviewRepo) Update(_ context.Context, i *interview.Interview) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[i.ID]; !ok {
		return kernel.NotFound("memory: interview not found")
	}
	r.byID[i.ID] = *i
	return nil
}

// ByCandidate returns a page of a candidate's interviews (insertion-agnostic).
func (r *InterviewRepo) ByCandidate(
	_ context.Context, candidateID kernel.ID, page kernel.Page,
) ([]*interview.Interview, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []*interview.Interview
	for id := range r.byID {
		if r.byID[id].CandidateID == candidateID {
			iv := r.byID[id]
			all = append(all, &iv)
		}
	}
	total := int64(len(all))
	start := page.Offset()
	if start >= len(all) {
		return nil, total, nil
	}
	end := min(start+page.Limit(), len(all))
	return all[start:end], total, nil
}

// DeleteByCandidate hard-removes every interview of a candidate, transcripts
// included (right-to-erasure cascade, CAL-118).
func (r *InterviewRepo) DeleteByCandidate(_ context.Context, candidateID kernel.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id := range r.byID {
		if r.byID[id].CandidateID == candidateID {
			delete(r.byID, id)
		}
	}
	return nil
}

var _ interview.InterviewRepository = (*InterviewRepo)(nil)
