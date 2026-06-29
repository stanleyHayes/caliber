package memory

import (
	"context"

	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

// TalentProfileRepo is an in-memory talent.TalentProfileRepository.
type TalentProfileRepo struct {
	store *keyedStore[talent.TalentProfile]
}

// NewTalentProfileRepo builds an empty in-memory talent profile repository.
func NewTalentProfileRepo() *TalentProfileRepo {
	return &TalentProfileRepo{store: newKeyedStore(
		func(p *talent.TalentProfile) kernel.ID { return p.ID },
		func(p *talent.TalentProfile) kernel.ID { return p.CandidateID },
	)}
}

// Create stores a new profile, rejecting a duplicate candidate profile.
func (r *TalentProfileRepo) Create(_ context.Context, p *talent.TalentProfile) error {
	return r.store.create(p, "memory: profile already exists for candidate")
}

// ByID returns the profile with the given id.
func (r *TalentProfileRepo) ByID(_ context.Context, id kernel.ID) (*talent.TalentProfile, error) {
	return r.store.get(id, "memory: talent profile not found")
}

// ByCandidateID returns the profile for a candidate.
func (r *TalentProfileRepo) ByCandidateID(_ context.Context, candidateID kernel.ID) (*talent.TalentProfile, error) {
	return r.store.getBySecondary(candidateID, "memory: talent profile not found")
}

// Update replaces an existing profile.
func (r *TalentProfileRepo) Update(_ context.Context, p *talent.TalentProfile) error {
	return r.store.update(p, "memory: talent profile not found")
}

// List returns a page of profiles and the total count.
func (r *TalentProfileRepo) List(_ context.Context, page kernel.Page) ([]*talent.TalentProfile, int64, error) {
	out, total := r.store.list(page)
	return out, total, nil
}

// DeleteByCandidate hard-removes a candidate's profile (right-to-erasure cascade,
// CAL-118). The profile is indexed by candidate id; a candidate with no profile
// is a no-op, so erasure is idempotent.
func (r *TalentProfileRepo) DeleteByCandidate(_ context.Context, candidateID kernel.ID) error {
	r.store.deleteBySecondary(candidateID)
	return nil
}

var _ talent.TalentProfileRepository = (*TalentProfileRepo)(nil)
