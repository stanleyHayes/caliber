// Package provisioning bootstraps the bounded-context aggregate a newly
// registered user owns. It bridges the identity use-case (via the Provisioner
// port) to the talent context without coupling identity to talent.
package provisioning

import (
	"context"

	identityapp "github.com/xcreativs/caliber/internal/app/identity"
	identitydom "github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

// CandidateProvisioner creates a candidate's Talent Passport context on signup.
// Employer/recruiter accounts own roles directly by their user id, so they need
// no separate aggregate here (employer-profile bootstrap is a later story once
// signup collects a company name).
type CandidateProvisioner struct {
	candidates talent.CandidateRepository
}

// NewCandidateProvisioner builds the provisioner from the candidate repository.
func NewCandidateProvisioner(candidates talent.CandidateRepository) *CandidateProvisioner {
	return &CandidateProvisioner{candidates: candidates}
}

// Provision creates an empty Candidate aggregate owned by the user when the user
// registered as a candidate; it is a no-op for other roles.
func (p *CandidateProvisioner) Provision(ctx context.Context, user *identitydom.User) error {
	if user.Role != identitydom.RoleCandidate {
		return nil
	}
	candidate, err := talent.NewCandidate(user.ID, "", talent.CandidateIntake{})
	if err != nil {
		return err
	}
	// Use the user id as the candidate id so a candidate-role user is addressable
	// by a single id across identity, talent, agent, and dashboard surfaces.
	candidate.ID = user.ID
	return p.candidates.Create(ctx, candidate)
}

var _ identityapp.Provisioner = (*CandidateProvisioner)(nil)
