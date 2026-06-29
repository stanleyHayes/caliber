package privacy

import (
	"context"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// The Eraser declares the narrow removal ports it depends on (hexagonal: the
// application layer owns its port contracts). Repositories satisfy these once
// they gain hard-delete primitives — the remaining wiring step noted in
// docs/data-protection.md.

// CandidateScopedEraser hard-deletes every record of one kind that belongs to a
// candidate (talent profile + embeddings, applications, interviews + transcripts,
// matches referencing them, contests they raised).
type CandidateScopedEraser interface {
	EraseForCandidate(ctx context.Context, candidateID kernel.ID) error
}

// CandidateRemover hard-deletes the candidate aggregate itself.
type CandidateRemover interface {
	Delete(ctx context.Context, candidateID kernel.ID) error
}

// IdentityAnonymizer anonymises (or removes) the owning user account. A
// registered candidate's id equals their user id (the provisioner), so the
// candidate id is the user id.
type IdentityAnonymizer interface {
	Anonymize(ctx context.Context, userID kernel.ID) error
}

// AuditTombstoner retains the append-only audit trail but replaces the erased
// subject's actor id with a tombstone: the trail is itself a compliance record,
// so its existence is kept while the subject's identity is removed.
type AuditTombstoner interface {
	TombstoneActor(ctx context.Context, actorID kernel.ID) error
}

// Eraser orchestrates the right-to-erasure cascade (CAL-118, Ghana DPA 2012):
// a candidate-initiated hard delete of every record about them, with the audit
// trail retained-but-anonymised.
type Eraser struct {
	scoped    []CandidateScopedEraser
	candidate CandidateRemover
	identity  IdentityAnonymizer
	audit     AuditTombstoner
}

// NewEraser wires the erasure use-case. scoped are the per-candidate record
// erasers (profile, applications, interviews, matches, contests) — run before
// the candidate aggregate itself is removed.
func NewEraser(
	candidate CandidateRemover, identity IdentityAnonymizer, audit AuditTombstoner, scoped ...CandidateScopedEraser,
) *Eraser {
	return &Eraser{scoped: scoped, candidate: candidate, identity: identity, audit: audit}
}

// EraseCandidate runs the cascade in dependency order: the candidate's scoped
// records first, then the candidate aggregate, then the owning user account, and
// finally the audit trail is tombstoned (retained, but de-identified). The caller
// authorizes the subject (candidate-self). A failure at any step is surfaced;
// production erasure must additionally be idempotent so a retried request safely
// completes a partial cascade.
func (e *Eraser) EraseCandidate(ctx context.Context, candidateID kernel.ID) error {
	if candidateID.IsZero() {
		return kernel.Invalid("privacy: candidate id is required")
	}
	for _, s := range e.scoped {
		if err := s.EraseForCandidate(ctx, candidateID); err != nil {
			return err
		}
	}
	if err := e.candidate.Delete(ctx, candidateID); err != nil {
		return err
	}
	if err := e.identity.Anonymize(ctx, candidateID); err != nil {
		return err
	}
	return e.audit.TombstoneActor(ctx, candidateID)
}
