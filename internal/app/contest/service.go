// Package contest holds the assessment-contest use-cases (CAL-083): a candidate
// raises a dispute over an assessment, lists their disputes, and a human reviewer
// resolves them. Every state change appends an audit entry (explainable, audited
// fairness control).
package contest

import (
	"context"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/audit"
	contestdom "github.com/xcreativs/caliber/internal/domain/contest"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// Service orchestrates the contest lifecycle through the domain.
type Service struct {
	contests contestdom.ContestRepository
	audit    audit.AuditRepository
	now      app.Clock
}

// NewService wires the contest use-case.
func NewService(contests contestdom.ContestRepository, auditRepo audit.AuditRepository, now app.Clock) *Service {
	return &Service{contests: contests, audit: auditRepo, now: now}
}

// Raise opens a contest on behalf of a candidate against an assessment.
func (s *Service) Raise(
	ctx context.Context, candidateID, subjectID kernel.ID, subject contestdom.Subject, reason string,
) (*contestdom.Contest, error) {
	c, err := contestdom.NewContest(candidateID, subjectID, subject, reason, s.now())
	if err != nil {
		return nil, err
	}
	if err := s.contests.Create(ctx, c); err != nil {
		return nil, err
	}
	s.record(ctx, candidateID, audit.ActionContestRaised, c.ID)
	return c, nil
}

// ListForCandidate returns a candidate's contests, newest first.
func (s *Service) ListForCandidate(
	ctx context.Context, candidateID kernel.ID, page kernel.Page,
) ([]*contestdom.Contest, int64, error) {
	return s.contests.ByCandidate(ctx, candidateID, page)
}

// ListForSubject returns the contests raised against an assessment (reviewer side).
func (s *Service) ListForSubject(
	ctx context.Context, subjectID kernel.ID, page kernel.Page,
) ([]*contestdom.Contest, int64, error) {
	return s.contests.BySubject(ctx, subjectID, page)
}

// Resolve resolves an open contest as a human reviewer (uphold or dismiss).
func (s *Service) Resolve(
	ctx context.Context, reviewerID, contestID kernel.ID, upheld bool, note string,
) (*contestdom.Contest, error) {
	c, err := s.contests.ByID(ctx, contestID)
	if err != nil {
		return nil, err
	}
	if rerr := c.Resolve(upheld, note, s.now()); rerr != nil {
		return nil, rerr
	}
	if uerr := s.contests.Update(ctx, c); uerr != nil {
		return nil, uerr
	}
	s.record(ctx, reviewerID, audit.ActionContestResolved, c.ID)
	return c, nil
}

// record appends an audit entry, best-effort: an audit failure never blocks the
// contest action it describes.
func (s *Service) record(ctx context.Context, actorID kernel.ID, action string, contestID kernel.ID) {
	entry, err := audit.NewAuditEntry(actorID, action, "contest", contestID, "", "", s.now())
	if err != nil {
		return
	}
	_ = s.audit.Append(ctx, entry)
}
