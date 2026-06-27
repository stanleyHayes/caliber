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
	total := int64(len(matched))
	start := min(page.Offset(), len(matched))
	end := min(start+page.Limit(), len(matched))
	return matched[start:end], total, nil
}
