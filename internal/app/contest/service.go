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
	"github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/matching"
	"github.com/xcreativs/caliber/internal/domain/role"
)

// Service orchestrates the contest lifecycle through the domain. The match,
// interview, and role repositories are read-only here — they resolve a
// contested assessment to its owning employer so only that employer (the tenant
// that owns the role) can resolve the contest (CAL-153 multi-tenant isolation).
type Service struct {
	contests   contestdom.ContestRepository
	audit      audit.AuditRepository
	matches    matching.MatchRepository
	interviews interview.InterviewRepository
	roles      role.RoleRepository
	now        app.Clock
}

// NewService wires the contest use-case. matches/interviews/roles are used to
// resolve a contest's owning employer for the tenant-ownership check on Resolve.
func NewService(
	contests contestdom.ContestRepository,
	auditRepo audit.AuditRepository,
	matches matching.MatchRepository,
	interviews interview.InterviewRepository,
	roles role.RoleRepository,
	now app.Clock,
) *Service {
	return &Service{
		contests:   contests,
		audit:      auditRepo,
		matches:    matches,
		interviews: interviews,
		roles:      roles,
		now:        now,
	}
}

// Raise opens a contest on behalf of a candidate against an assessment. The
// candidate may only contest their OWN assessment: the subject (match/report
// card) is resolved and its candidate must equal the raiser, closing the
// candidate-side IDOR where anyone could dispute (and audit-pollute) another
// candidate's assessment (CAL-153). A subject_id that references no real
// assessment surfaces NotFound rather than silently creating a dangling contest.
func (s *Service) Raise(
	ctx context.Context, candidateID, subjectID kernel.ID, subject contestdom.Subject, reason string,
) (*contestdom.Contest, error) {
	c, err := contestdom.NewContest(candidateID, subjectID, subject, reason, s.now())
	if err != nil {
		return nil, err
	}
	roleID, subjectCandidate, err := s.subjectRef(ctx, subject, subjectID)
	if err != nil {
		return nil, err
	}
	if subjectCandidate != candidateID {
		return nil, kernel.Forbidden("contest: may only contest your own assessment")
	}
	if err := s.contests.Create(ctx, c); err != nil {
		return nil, err
	}
	// Own the audit entry to the employer that owns the contested assessment
	// (CAL-153), so both the candidate's raise and the reviewer's resolve are
	// visible in that employer's scoped audit read. Best-effort: an unresolvable
	// owner leaves the entry unowned rather than failing the raise.
	owner := kernel.ID("")
	if rl, rerr := s.roles.ByID(ctx, roleID); rerr == nil {
		owner = rl.EmployerID
	}
	s.record(ctx, candidateID, audit.ActionContestRaised, c.ID, owner)
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
// The reviewer must own the contested assessment: the contest's subject resolves
// to a role, and only that role's employer (the owning tenant) may resolve it.
// A non-owner is rejected with kernel.Forbidden before any state change, closing
// the cross-tenant IDOR where any reviewer could resolve any contest (CAL-153).
func (s *Service) Resolve(
	ctx context.Context, reviewerID, contestID kernel.ID, upheld bool, note string,
) (*contestdom.Contest, error) {
	c, err := s.contests.ByID(ctx, contestID)
	if err != nil {
		return nil, err
	}
	ownerID, err := s.ownerOf(ctx, c)
	if err != nil {
		return nil, err
	}
	if ownerID != reviewerID {
		return nil, kernel.Forbidden("contest: reviewer does not own the contested assessment")
	}
	if rerr := c.Resolve(upheld, note, s.now()); rerr != nil {
		return nil, rerr
	}
	if uerr := s.contests.Update(ctx, c); uerr != nil {
		return nil, uerr
	}
	// ownerID == reviewerID here (verified above), and owns the entry so the
	// resolve joins the raise in the owner's scoped audit trail (CAL-153).
	s.record(ctx, reviewerID, audit.ActionContestResolved, c.ID, ownerID)
	return c, nil
}

// subjectRef resolves a contested assessment to its owning role and the candidate
// it concerns. A match and a report card both carry a role id and a candidate id.
// Returns kernel.KindNotFound if the assessment no longer exists, which also
// validates that a contest's subject_id references a real assessment.
func (s *Service) subjectRef(
	ctx context.Context, subject contestdom.Subject, subjectID kernel.ID,
) (kernel.ID, kernel.ID, error) {
	switch subject {
	case contestdom.SubjectMatch:
		m, err := s.matches.ByID(ctx, subjectID)
		if err != nil {
			return "", "", err
		}
		return m.RoleID, m.CandidateID, nil
	case contestdom.SubjectReportCard:
		iv, err := s.interviews.ByID(ctx, subjectID)
		if err != nil {
			return "", "", err
		}
		return iv.RoleID, iv.CandidateID, nil
	default:
		return "", "", kernel.Invalid("contest: unknown subject")
	}
}

// ownerOf resolves the employer (tenant) that owns the assessment a contest
// disputes: the assessment's role carries its owning employer id.
func (s *Service) ownerOf(ctx context.Context, c *contestdom.Contest) (kernel.ID, error) {
	roleID, _, err := s.subjectRef(ctx, c.Subject, c.SubjectID)
	if err != nil {
		return "", err
	}
	rl, err := s.roles.ByID(ctx, roleID)
	if err != nil {
		return "", err
	}
	return rl.EmployerID, nil
}

// record appends an audit entry, best-effort: an audit failure never blocks the
// contest action it describes.
func (s *Service) record(ctx context.Context, actorID kernel.ID, action string, contestID, ownerID kernel.ID) {
	entry, err := audit.NewAuditEntry(actorID, action, "contest", contestID, "", "", s.now())
	if err != nil {
		return
	}
	entry.OwnerID = ownerID
	_ = s.audit.Append(ctx, entry)
}
