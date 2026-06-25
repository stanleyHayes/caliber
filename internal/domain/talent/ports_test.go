package talent

import (
	"context"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// staticCandidateRepo is a compile-time assertion target for CandidateRepository.
type staticCandidateRepo struct{}

func (staticCandidateRepo) Create(context.Context, *Candidate) error            { return nil }
func (staticCandidateRepo) ByID(context.Context, kernel.ID) (*Candidate, error) { return nil, nil }
func (staticCandidateRepo) ByUserID(context.Context, kernel.ID) (*Candidate, error) {
	return nil, nil
}
func (staticCandidateRepo) Update(context.Context, *Candidate) error { return nil }
func (staticCandidateRepo) List(context.Context, kernel.Page) ([]*Candidate, int64, error) {
	return nil, 0, nil
}

// staticProfileRepo is a compile-time assertion target for TalentProfileRepository.
type staticProfileRepo struct{}

func (staticProfileRepo) Create(context.Context, *TalentProfile) error { return nil }
func (staticProfileRepo) ByID(context.Context, kernel.ID) (*TalentProfile, error) {
	return nil, nil
}
func (staticProfileRepo) ByCandidateID(context.Context, kernel.ID) (*TalentProfile, error) {
	return nil, nil
}
func (staticProfileRepo) Update(context.Context, *TalentProfile) error { return nil }
func (staticProfileRepo) List(context.Context, kernel.Page) ([]*TalentProfile, int64, error) {
	return nil, 0, nil
}

var (
	_ CandidateRepository     = staticCandidateRepo{}
	_ TalentProfileRepository = staticProfileRepo{}
)
