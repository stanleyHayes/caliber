package roles

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
)

type fakeLLM struct {
	text string
	err  error
}

func (f fakeLLM) Complete(_ context.Context, _ app.LLMRequest) (app.LLMResponse, error) {
	return app.LLMResponse{Text: f.text}, f.err
}

// fakeRepo is an in-test implementation of the domain port (app tests must not
// depend on concrete adapters).
type fakeRepo struct {
	items map[kernel.ID]*role.Role
}

func newFakeRepo() *fakeRepo { return &fakeRepo{items: map[kernel.ID]*role.Role{}} }

func (r *fakeRepo) Create(_ context.Context, x *role.Role) error { r.items[x.ID] = x; return nil }
func (r *fakeRepo) ByID(_ context.Context, id kernel.ID) (*role.Role, error) {
	x, ok := r.items[id]
	if !ok {
		return nil, kernel.NotFound("not found")
	}
	return x, nil
}
func (r *fakeRepo) Update(_ context.Context, x *role.Role) error { r.items[x.ID] = x; return nil }
func (r *fakeRepo) ListByEmployer(_ context.Context, _ kernel.ID, _ kernel.Page) ([]*role.Role, int64, error) {
	return nil, 0, nil
}

var _ role.RoleRepository = (*fakeRepo)(nil)

const validSpecJSON = `{"title":"Backend Engineer","location":"Accra","seniority":"senior","availability":"now","responsibilities":["build"],"must_haves":["Go"],"nice_to_haves":[],"salary_band":{"currency":"GHS","low":1000,"high":2000},"rubric":[{"name":"Go","weight":0.6,"must_have":true},{"name":"SQL","weight":0.4,"must_have":false}]}`

func fixedClock() app.Clock { return func() time.Time { return time.Unix(1700000000, 0) } }

func TestGenerateHappyPath(t *testing.T) {
	repo := newFakeRepo()
	g := NewSpecGenerator(fakeLLM{text: validSpecJSON}, repo, fixedClock())
	emp := kernel.NewID()

	r, err := g.Generate(context.Background(), emp, "I need a senior Go engineer in Accra")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if r.Title != "Backend Engineer" || r.EmployerID != emp || r.ID.IsZero() {
		t.Errorf("unexpected role: %+v", r)
	}
	if tw := r.Rubric.TotalWeight(); tw < 0.999 || tw > 1.001 {
		t.Errorf("rubric weights not normalized: %v", tw)
	}
	if got, err := repo.ByID(context.Background(), r.ID); err != nil || got.ID != r.ID {
		t.Errorf("role not persisted: %v", err)
	}
}

func TestGenerateUnknownSeniorityDefaultsMid(t *testing.T) {
	g := NewSpecGenerator(fakeLLM{text: `{"title":"X","seniority":"wizard","rubric":[{"name":"A","weight":1}]}`}, newFakeRepo(), fixedClock())
	r, err := g.Generate(context.Background(), kernel.NewID(), "x")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if r.Spec.Seniority.String() != "mid" {
		t.Errorf("seniority = %q, want mid", r.Spec.Seniority.String())
	}
}

func TestGenerateErrors(t *testing.T) {
	repo := newFakeRepo()
	clock := fixedClock()

	g := NewSpecGenerator(fakeLLM{text: validSpecJSON}, repo, clock)
	if _, err := g.Generate(context.Background(), kernel.ID(""), "x"); kernel.KindOf(err) != kernel.KindInvalid {
		t.Error("zero employer should be invalid")
	}
	if _, err := g.Generate(context.Background(), kernel.NewID(), "   "); kernel.KindOf(err) != kernel.KindInvalid {
		t.Error("blank text should be invalid")
	}
	ge := NewSpecGenerator(fakeLLM{err: errors.New("boom")}, repo, clock)
	if _, err := ge.Generate(context.Background(), kernel.NewID(), "x"); err == nil {
		t.Error("llm error should propagate")
	}
	gb := NewSpecGenerator(fakeLLM{text: "not json"}, repo, clock)
	if _, err := gb.Generate(context.Background(), kernel.NewID(), "x"); kernel.KindOf(err) != kernel.KindInvalid {
		t.Error("bad json should be invalid")
	}
	gv := NewSpecGenerator(fakeLLM{text: `{"title":"X","seniority":"mid","rubric":[]}`}, repo, clock)
	if _, err := gv.Generate(context.Background(), kernel.NewID(), "x"); err == nil {
		t.Error("empty rubric should fail domain validation")
	}
}
