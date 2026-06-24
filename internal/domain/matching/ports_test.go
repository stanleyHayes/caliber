package matching

import (
	"context"
	"testing"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// stubRepo is a minimal in-memory MatchRepository used to assert the port is
// implementable and exercises the interface methods.
type stubRepo struct {
	stored []*Match
}

func (s *stubRepo) Upsert(_ context.Context, m *Match) error {
	s.stored = append(s.stored, m)
	return nil
}

func (s *stubRepo) ByRole(_ context.Context, roleID kernel.ID, _ kernel.Page) ([]*Match, int64, error) {
	var out []*Match
	for _, m := range s.stored {
		if m.RoleID == roleID {
			out = append(out, m)
		}
	}
	return out, int64(len(out)), nil
}

func (s *stubRepo) ForCandidate(_ context.Context, candidateID kernel.ID, _ kernel.Page) ([]*Match, int64, error) {
	var out []*Match
	for _, m := range s.stored {
		if m.CandidateID == candidateID {
			out = append(out, m)
		}
	}
	return out, int64(len(out)), nil
}

func TestMatchRepository_Port(t *testing.T) {
	var repo MatchRepository = &stubRepo{}
	ctx := context.Background()
	role := kernel.NewID()
	cand := kernel.NewID()

	m, err := NewMatch(role, cand, 0.7, kernel.ConfidenceHigh, nil, "r", nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := repo.Upsert(ctx, m); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	byRole, total, err := repo.ByRole(ctx, role, kernel.NewPage(1, 20))
	if err != nil || total != 1 || len(byRole) != 1 {
		t.Fatalf("ByRole: got %d/%d err=%v", len(byRole), total, err)
	}

	forCand, total, err := repo.ForCandidate(ctx, cand, kernel.NewPage(1, 20))
	if err != nil || total != 1 || len(forCand) != 1 {
		t.Fatalf("ForCandidate: got %d/%d err=%v", len(forCand), total, err)
	}

	none, total, err := repo.ByRole(ctx, kernel.NewID(), kernel.NewPage(1, 20))
	if err != nil || total != 0 || len(none) != 0 {
		t.Fatalf("ByRole unknown: got %d/%d err=%v", len(none), total, err)
	}
}
