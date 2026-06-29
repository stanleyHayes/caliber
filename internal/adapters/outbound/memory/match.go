package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/xcreativs/caliber/internal/domain/kernel"
	matchingdom "github.com/xcreativs/caliber/internal/domain/matching"
)

// MatchRepo is an in-memory matchingdom.MatchRepository keyed by (role,
// candidate): re-scoring a pair replaces its match. ByRole returns a role's
// matches ranked by overall score (descending, candidate id breaking ties).
type MatchRepo struct {
	mu    sync.RWMutex
	byKey map[string]matchingdom.Match
}

// NewMatchRepo builds an empty in-memory match repository.
func NewMatchRepo() *MatchRepo { return &MatchRepo{byKey: map[string]matchingdom.Match{}} }

func matchKey(roleID, candidateID kernel.ID) string {
	return roleID.String() + ":" + candidateID.String()
}

// Upsert stores or replaces the match for its (role, candidate) pair.
func (r *MatchRepo) Upsert(_ context.Context, m *matchingdom.Match) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byKey[matchKey(m.RoleID, m.CandidateID)] = *m
	return nil
}

// ByRole returns a role's matches, highest overall score first, paginated.
func (r *MatchRepo) ByRole(_ context.Context, roleID kernel.ID, page kernel.Page) ([]*matchingdom.Match, int64, error) {
	return r.filter(func(m matchingdom.Match) bool { return m.RoleID == roleID }, page)
}

// ForCandidate returns a candidate's matches, highest overall score first, paginated.
func (r *MatchRepo) ForCandidate(_ context.Context, candidateID kernel.ID, page kernel.Page) ([]*matchingdom.Match, int64, error) {
	return r.filter(func(m matchingdom.Match) bool { return m.CandidateID == candidateID }, page)
}

// DeleteByCandidate hard-removes every match referencing a candidate (right-to-
// erasure cascade, CAL-118).
func (r *MatchRepo) DeleteByCandidate(_ context.Context, candidateID kernel.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, m := range r.byKey {
		if m.CandidateID == candidateID {
			delete(r.byKey, k)
		}
	}
	return nil
}

func (r *MatchRepo) filter(keep func(matchingdom.Match) bool, page kernel.Page) ([]*matchingdom.Match, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []*matchingdom.Match
	for _, m := range r.byKey {
		if keep(m) {
			cp := m
			all = append(all, &cp)
		}
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].OverallScore != all[j].OverallScore {
			return all[i].OverallScore > all[j].OverallScore
		}
		return all[i].CandidateID < all[j].CandidateID
	})
	total := int64(len(all))
	start := min(page.Offset(), len(all))
	end := min(start+page.Limit(), len(all))
	return all[start:end], total, nil
}
