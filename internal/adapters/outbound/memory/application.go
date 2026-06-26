package memory

import (
	"context"
	"sync"

	"github.com/xcreativs/caliber/internal/domain/candidateagent"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// ApplicationRepo is an in-memory candidateagent.ApplicationRepository.
type ApplicationRepo struct {
	mu    sync.RWMutex
	byID  map[kernel.ID]candidateagent.Application
	order []kernel.ID // insertion order; ByCandidate returns newest first
}

// NewApplicationRepo builds an empty in-memory application repository.
func NewApplicationRepo() *ApplicationRepo {
	return &ApplicationRepo{byID: map[kernel.ID]candidateagent.Application{}}
}

// Create stores a new application.
func (r *ApplicationRepo) Create(_ context.Context, app *candidateagent.Application) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byID[app.ID]; exists {
		return kernel.Conflict("memory: application already exists")
	}
	r.byID[app.ID] = *app
	r.order = append(r.order, app.ID)
	return nil
}

// ByID returns a copy of the application with the given id.
func (r *ApplicationRepo) ByID(_ context.Context, id kernel.ID) (*candidateagent.Application, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.byID[id]
	if !ok {
		return nil, kernel.NotFound("memory: application not found")
	}
	return &a, nil
}

// Update replaces an existing application.
func (r *ApplicationRepo) Update(_ context.Context, app *candidateagent.Application) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[app.ID]; !ok {
		return kernel.NotFound("memory: application not found")
	}
	r.byID[app.ID] = *app
	return nil
}

// ByCandidate lists a candidate's applications, newest first, paginated.
func (r *ApplicationRepo) ByCandidate(
	_ context.Context, candidateID kernel.ID, page kernel.Page,
) ([]*candidateagent.Application, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []*candidateagent.Application
	for i := len(r.order) - 1; i >= 0; i-- {
		a := r.byID[r.order[i]]
		if a.CandidateID == candidateID {
			app := a
			all = append(all, &app)
		}
	}
	total := int64(len(all))
	start := min(page.Offset(), len(all))
	end := min(start+page.Limit(), len(all))
	return all[start:end], total, nil
}

var _ candidateagent.ApplicationRepository = (*ApplicationRepo)(nil)
