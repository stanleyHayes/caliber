package matching

import (
	"context"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/audit"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	matchingdom "github.com/xcreativs/caliber/internal/domain/matching"
	"github.com/xcreativs/caliber/internal/domain/role"
)

// RejectionRecorder records human-approved candidate rejections to the audit
// trail (CAL-081). It enforces the platform invariant that the AI never
// auto-rejects: a rejection exists only as a logged, human-approved decision.
type RejectionRecorder struct {
	roles role.RoleRepository
	audit audit.AuditRepository
	now   app.Clock
}

// NewRejectionRecorder wires the rejection use-case over the role repository
// (for ownership) and the audit trail.
func NewRejectionRecorder(roles role.RoleRepository, auditRepo audit.AuditRepository, now app.Clock) *RejectionRecorder {
	return &RejectionRecorder{roles: roles, audit: auditRepo, now: now}
}

// Record validates and durably logs a human-approved rejection, returning the
// id of the audit entry that now stands as the approval record.
//
// The log is the approval: unlike best-effort auditing of lower-stakes actions,
// if the entry cannot be written the rejection does not stand and the call
// fails — there must be no rejection without a logged human approval. actorUserID
// is the authenticated human who approved the decline (never a value the client
// supplies in the body).
func (r *RejectionRecorder) Record(
	ctx context.Context,
	actorUserID, roleID, candidateID kernel.ID,
	reason string,
	humanApproved bool,
) (kernel.ID, error) {
	// Ownership (CAL-116 IDOR guard): a reviewer may only reject a candidate
	// against their OWN role. Employers are users, so EmployerID is the owner's id.
	rl, err := r.roles.ByID(ctx, roleID)
	if err != nil {
		return "", err
	}
	if rl.EmployerID != actorUserID {
		return "", kernel.Forbidden("matching: may only reject candidates for your own roles")
	}
	rej, err := matchingdom.NewRejection(roleID, candidateID, reason, humanApproved)
	if err != nil {
		return "", err
	}
	snapshot, err := rej.SnapshotJSON()
	if err != nil {
		return "", err
	}
	entry, err := audit.NewAuditEntry(
		actorUserID, audit.ActionApproveRejection, "match", candidateID, "", snapshot, r.now(),
	)
	if err != nil {
		return "", err
	}
	if appendErr := r.audit.Append(ctx, entry); appendErr != nil {
		return "", appendErr
	}
	return entry.ID, nil
}
