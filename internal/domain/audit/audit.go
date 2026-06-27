// Package audit implements the append-only audit-trail bounded context.
//
// It records human approvals, score overrides, and agent actions as immutable
// AuditEntry records. Entries are never mutated once created; the repository
// port only supports appending and listing.
package audit

import (
	"context"
	"strings"
	"time"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// Action constants enumerate the recognized audit actions captured in the trail.
const (
	// ActionApproveRejection records a human approving a rejection decision.
	ActionApproveRejection = "approve_rejection"
	// ActionOverrideScore records a human overriding an automated score.
	ActionOverrideScore = "override_score"
	// ActionAgentSubmit records an automated agent submitting an action.
	ActionAgentSubmit = "agent_submit"
	// ActionContestRaised records a candidate disputing an assessment.
	ActionContestRaised = "contest_raised"
	// ActionContestResolved records a reviewer resolving a contest.
	ActionContestResolved = "contest_resolved"
)

// AuditEntry is an immutable, append-only record of a single auditable action.
//
// It captures who acted (ActorUserID), what they did (Action), the affected
// entity (Entity / EntityID), and the before/after state snapshots as opaque
// JSON strings. AuditEntry has no mutator methods by design.
type AuditEntry struct { //nolint:revive // domain name is fixed by the audit context spec
	ID          kernel.ID
	ActorUserID kernel.ID
	Action      string
	Entity      string
	EntityID    kernel.ID
	BeforeJSON  string
	AfterJSON   string
	Timestamp   time.Time
}

// NewAuditEntry validates its inputs and constructs an AuditEntry with a fresh ID.
//
// It returns a kernel.Invalid error when actorUserID is zero, or when action or
// entity are empty/whitespace. The entityID, beforeJSON, and afterJSON values
// are accepted as-is (an entry may concern a not-yet-persisted entity and the
// JSON snapshots are optional).
func NewAuditEntry(
	actorUserID kernel.ID,
	action string,
	entity string,
	entityID kernel.ID,
	beforeJSON string,
	afterJSON string,
	ts time.Time,
) (*AuditEntry, error) {
	if actorUserID.IsZero() {
		return nil, kernel.Invalid("audit: actorUserID is required")
	}
	if strings.TrimSpace(action) == "" {
		return nil, kernel.Invalid("audit: action is required")
	}
	if strings.TrimSpace(entity) == "" {
		return nil, kernel.Invalid("audit: entity is required")
	}
	return &AuditEntry{
		ID:          kernel.NewID(),
		ActorUserID: actorUserID,
		Action:      action,
		Entity:      entity,
		EntityID:    entityID,
		BeforeJSON:  beforeJSON,
		AfterJSON:   afterJSON,
		Timestamp:   ts,
	}, nil
}

//go:generate mockgen -source=audit.go -destination=../../mocks/audit.go -package=mocks

// AuditRepository is the persistence port for the append-only audit trail.
type AuditRepository interface { //nolint:revive // domain name is fixed by the audit context spec
	// Append durably stores a new audit entry.
	Append(ctx context.Context, entry *AuditEntry) error
	// List returns a page of audit entries for a given entity and entityID,
	// along with the total count of matching entries.
	List(ctx context.Context, entity string, entityID kernel.ID, page kernel.Page) ([]*AuditEntry, int64, error)
}
