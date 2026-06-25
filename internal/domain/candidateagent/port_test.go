package candidateagent

import (
	"context"
	"testing"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// fakeRepo is an in-memory ApplicationRepository used to assert the port is
// implementable and to exercise it through the domain types.
type fakeRepo struct {
	store map[kernel.ID]*Application
}

func newFakeRepo() *fakeRepo { return &fakeRepo{store: map[kernel.ID]*Application{}} }

func (r *fakeRepo) Create(_ context.Context, app *Application) error {
	if _, ok := r.store[app.ID]; ok {
		return kernel.Conflict("already exists")
	}
	r.store[app.ID] = app
	return nil
}

func (r *fakeRepo) ByID(_ context.Context, id kernel.ID) (*Application, error) {
	app, ok := r.store[id]
	if !ok {
		return nil, kernel.NotFound("application not found")
	}
	return app, nil
}

func (r *fakeRepo) Update(_ context.Context, app *Application) error {
	if _, ok := r.store[app.ID]; !ok {
		return kernel.NotFound("application not found")
	}
	r.store[app.ID] = app
	return nil
}

func (r *fakeRepo) ByCandidate(_ context.Context, candidateID kernel.ID, page kernel.Page) ([]*Application, int64, error) {
	var all []*Application
	for _, app := range r.store {
		if app.CandidateID == candidateID {
			all = append(all, app)
		}
	}
	total := int64(len(all))
	start := page.Offset()
	if start > len(all) {
		start = len(all)
	}
	end := start + page.Limit()
	if end > len(all) {
		end = len(all)
	}
	return all[start:end], total, nil
}

// ensure the fake satisfies the port.
var _ ApplicationRepository = (*fakeRepo)(nil)

func TestApplicationRepositoryPort(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	cand := kernel.NewID()

	app, err := NewAgentApplication(kernel.NewID(), cand, kernel.NewID(), "summary")
	if err != nil {
		t.Fatalf("construct: %v", err)
	}
	if err := repo.Create(ctx, app); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.Create(ctx, app); kernel.KindOf(err) != kernel.KindConflict {
		t.Fatalf("duplicate create kind = %v, want Conflict", kernel.KindOf(err))
	}

	got, err := repo.ByID(ctx, app.ID)
	if err != nil || got.ID != app.ID {
		t.Fatalf("byID: %v", err)
	}
	if _, err := repo.ByID(ctx, kernel.NewID()); kernel.KindOf(err) != kernel.KindNotFound {
		t.Fatalf("missing byID kind = %v, want NotFound", kernel.KindOf(err))
	}

	if err := app.Submit(); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if err := repo.Update(ctx, app); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := repo.Update(ctx, &Application{ID: kernel.NewID()}); kernel.KindOf(err) != kernel.KindNotFound {
		t.Fatalf("update missing kind = %v, want NotFound", kernel.KindOf(err))
	}

	list, total, err := repo.ByCandidate(ctx, cand, kernel.NewPage(1, 10))
	if err != nil {
		t.Fatalf("byCandidate: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("byCandidate total=%d len=%d, want 1/1", total, len(list))
	}

	// Page past the end returns empty slice but correct total.
	list2, total2, err := repo.ByCandidate(ctx, cand, kernel.NewPage(5, 10))
	if err != nil {
		t.Fatalf("byCandidate page: %v", err)
	}
	if total2 != 1 || len(list2) != 0 {
		t.Fatalf("paged total=%d len=%d, want 1/0", total2, len(list2))
	}
}
