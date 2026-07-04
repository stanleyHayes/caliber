package memory

import (
	"context"
	"slices"
	"sync"

	"github.com/xcreativs/caliber/internal/domain/audit"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// AuditRepo is an in-memory, append-only audit.AuditRepository.
type AuditRepo struct {
	mu      sync.RWMutex
	entries []audit.AuditEntry
}

// NewAuditRepo builds an empty in-memory audit trail.
func NewAuditRepo() *AuditRepo { return &AuditRepo{} }

// Reset clears every audit entry (test/dev reseed helper).
func (r *AuditRepo) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = nil
}

// Append stores a new audit entry.
func (r *AuditRepo) Append(_ context.Context, e *audit.AuditEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, *e)
	return nil
}

// List returns the entries for an entity/entityID, newest first, paginated.
func (r *AuditRepo) List(
	_ context.Context, entity string, entityID kernel.ID, page kernel.Page,
) ([]*audit.AuditEntry, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var matched []*audit.AuditEntry
	for _, e := range slices.Backward(r.entries) {
		if e.Entity == entity && e.EntityID == entityID {
			cp := e
			matched = append(matched, &cp)
		}
	}
	out, total := paginate(matched, page)
	return out, total, nil
}

// ListForOwner is List scoped to an owning employer (CAL-153). A zero ownerID is
// unscoped (returns all entries for the entity).
func (r *AuditRepo) ListForOwner(
	_ context.Context, entity string, entityID, ownerID kernel.ID, page kernel.Page,
) ([]*audit.AuditEntry, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var matched []*audit.AuditEntry
	for _, e := range slices.Backward(r.entries) {
		if e.Entity == entity && e.EntityID == entityID && (ownerID.IsZero() || e.OwnerID == ownerID) {
			cp := e
			matched = append(matched, &cp)
		}
	}
	out, total := paginate(matched, page)
	return out, total, nil
}

// Search returns audit entries matching the report filter, newest first.
func (r *AuditRepo) Search(_ context.Context, filter audit.ReportFilter, page kernel.Page) ([]*audit.AuditEntry, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var matched []*audit.AuditEntry
	for _, e := range slices.Backward(r.entries) {
		if !e.Timestamp.IsZero() && (e.Timestamp.Before(filter.Start) || e.Timestamp.After(filter.End)) {
			continue
		}
		if len(filter.Actions) > 0 && !slices.Contains(filter.Actions, e.Action) {
			continue
		}
		if len(filter.Entities) > 0 && !slices.Contains(filter.Entities, e.Entity) {
			continue
		}
		cp := e
		matched = append(matched, &cp)
	}
	out, total := paginate(matched, page)
	return out, total, nil
}

func paginate(matched []*audit.AuditEntry, page kernel.Page) ([]*audit.AuditEntry, int64) {
	total := int64(len(matched))
	start := min(page.Offset(), len(matched))
	end := min(start+page.Limit(), len(matched))
	return matched[start:end], total
}

// TombstoneActor replaces an erased subject's actor id with a tombstone across
// the trail (right-to-erasure cascade, CAL-118): the append-only audit record is
// retained as a compliance artifact, but the subject's identity is removed.
func (r *AuditRepo) TombstoneActor(_ context.Context, actorID kernel.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.entries {
		if r.entries[i].ActorUserID == actorID {
			r.entries[i].ActorUserID = kernel.ID("erased")
		}
	}
	return nil
}
