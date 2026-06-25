package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/xcreativs/caliber/internal/adapters/outbound/postgres/sqlcdb"
	"github.com/xcreativs/caliber/internal/domain/audit"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// AuditRepo is a Postgres-backed, append-only audit.AuditRepository.
type AuditRepo struct {
	q *sqlcdb.Queries
}

// NewAuditRepo builds the repository from a sqlc DBTX.
func NewAuditRepo(db sqlcdb.DBTX) *AuditRepo { return &AuditRepo{q: sqlcdb.New(db)} }

// Append durably stores a new audit entry.
func (r *AuditRepo) Append(ctx context.Context, e *audit.AuditEntry) error {
	err := r.q.AppendAuditEntry(ctx, sqlcdb.AppendAuditEntryParams{
		ID:          e.ID.String(),
		ActorUserID: e.ActorUserID.String(),
		Action:      e.Action,
		Entity:      e.Entity,
		EntityID:    e.EntityID.String(),
		BeforeJson:  jsonOrNil(e.BeforeJSON),
		AfterJson:   jsonOrNil(e.AfterJSON),
		CreatedAt:   pgtype.Timestamptz{Time: e.Timestamp, Valid: true},
	})
	if isUniqueViolation(err) {
		return kernel.Conflict("postgres: audit entry already exists")
	}
	return err
}

// List returns a page of audit entries for an entity and the total count.
func (r *AuditRepo) List(ctx context.Context, entity string, entityID kernel.ID, page kernel.Page) ([]*audit.AuditEntry, int64, error) {
	rows, err := r.q.ListAuditLog(ctx, sqlcdb.ListAuditLogParams{
		Entity:   entity,
		EntityID: entityID.String(),
		Limit:    clampInt32(page.Limit()),
		Offset:   clampInt32(page.Offset()),
	})
	if err != nil {
		return nil, 0, err
	}
	out := make([]*audit.AuditEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainAuditEntry(row))
	}
	total, err := r.q.CountAuditLog(ctx, sqlcdb.CountAuditLogParams{Entity: entity, EntityID: entityID.String()})
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func toDomainAuditEntry(row sqlcdb.AuditLog) *audit.AuditEntry {
	return &audit.AuditEntry{
		ID:          kernel.ID(row.ID),
		ActorUserID: kernel.ID(row.ActorUserID),
		Action:      row.Action,
		Entity:      row.Entity,
		EntityID:    kernel.ID(row.EntityID),
		BeforeJSON:  string(row.BeforeJson),
		AfterJSON:   string(row.AfterJson),
		Timestamp:   row.CreatedAt.Time,
	}
}

func jsonOrNil(s string) []byte {
	if s == "" {
		return nil
	}
	return []byte(s)
}

var _ audit.AuditRepository = (*AuditRepo)(nil)
