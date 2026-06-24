package talent

import (
	"context"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// CandidateRepository is the persistence port for Candidate aggregates.
type CandidateRepository interface {
	// Create persists a new candidate.
	Create(ctx context.Context, c *Candidate) error
	// ByID loads a candidate by its identifier.
	ByID(ctx context.Context, id kernel.ID) (*Candidate, error)
	// ByUserID loads the candidate belonging to a platform user.
	ByUserID(ctx context.Context, userID kernel.ID) (*Candidate, error)
	// Update persists changes to an existing candidate.
	Update(ctx context.Context, c *Candidate) error
	// List returns a page of candidates and the total count.
	List(ctx context.Context, page kernel.Page) ([]*Candidate, int64, error)
}

// TalentProfileRepository is the persistence port for TalentProfile aggregates.
type TalentProfileRepository interface { //nolint:revive // name fixed by domain spec
	// Create persists a new talent profile.
	Create(ctx context.Context, p *TalentProfile) error
	// ByID loads a talent profile by its identifier.
	ByID(ctx context.Context, id kernel.ID) (*TalentProfile, error)
	// ByCandidateID loads the talent profile for a candidate.
	ByCandidateID(ctx context.Context, candidateID kernel.ID) (*TalentProfile, error)
	// Update persists changes to an existing talent profile.
	Update(ctx context.Context, p *TalentProfile) error
	// List returns a page of talent profiles and the total count.
	List(ctx context.Context, page kernel.Page) ([]*TalentProfile, int64, error)
}
