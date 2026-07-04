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
		OwnerID:     e.OwnerID.String(),
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

// Search returns audit entries matching the report filter, newest first.
func (r *AuditRepo) Search(ctx context.Context, filter audit.ReportFilter, page kernel.Page) ([]*audit.AuditEntry, int64, error) {
	return r.SearchForOwner(ctx, filter, kernel.ID(""), page)
}

// SearchForOwner returns audit entries matching the report filter, scoped to an
// owning employer (CAL-153). A zero ownerID is unscoped (behaves like Search).
func (r *AuditRepo) SearchForOwner(
	ctx context.Context, filter audit.ReportFilter, ownerID kernel.ID, page kernel.Page,
) ([]*audit.AuditEntry, int64, error) {
	actions := filter.Actions
	if actions == nil {
		actions = []string{}
	}
	entities := filter.Entities
	if entities == nil {
		entities = []string{}
	}
	rows, err := r.q.SearchAuditLogForOwner(ctx, sqlcdb.SearchAuditLogForOwnerParams{
		CreatedAt:   pgtype.Timestamptz{Time: filter.Start, Valid: true},
		CreatedAt_2: pgtype.Timestamptz{Time: filter.End, Valid: true},
		Column3:     actions,
		Column4:     entities,
		Column5:     ownerID.String(),
		Limit:       clampInt32(page.Limit()),
		Offset:      clampInt32(page.Offset()),
	})
	if err != nil {
		return nil, 0, err
	}
	out := make([]*audit.AuditEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainAuditEntry(row))
	}
	total, err := r.q.CountAuditLogForOwnerReport(ctx, sqlcdb.CountAuditLogForOwnerReportParams{
		CreatedAt:   pgtype.Timestamptz{Time: filter.Start, Valid: true},
		CreatedAt_2: pgtype.Timestamptz{Time: filter.End, Valid: true},
		Column3:     actions,
		Column4:     entities,
		Column5:     ownerID.String(),
	})
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// ListForOwner returns a page of audit entries for an entity, scoped to an
// owning employer (CAL-153). A zero ownerID is unscoped (returns all).
func (r *AuditRepo) ListForOwner(
	ctx context.Context, entity string, entityID, ownerID kernel.ID, page kernel.Page,
) ([]*audit.AuditEntry, int64, error) {
	rows, err := r.q.ListAuditLogForOwner(ctx, sqlcdb.ListAuditLogForOwnerParams{
		Entity:   entity,
		EntityID: entityID.String(),
		Column3:  ownerID.String(),
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
	total, err := r.q.CountAuditLogForOwner(ctx, sqlcdb.CountAuditLogForOwnerParams{
		Entity: entity, EntityID: entityID.String(), Column3: ownerID.String(),
	})
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
		OwnerID:     kernel.ID(row.OwnerID),
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

// TombstoneActor de-identifies an erased subject in the append-only audit trail
// (CAL-118): the trail is itself a compliance record, so its entries are kept but
// the subject's actor id is replaced with a tombstone.
func (r *AuditRepo) TombstoneActor(ctx context.Context, actorID kernel.ID) error {
	return r.q.TombstoneAuditActor(ctx, actorID.String())
}

var _ audit.AuditRepository = (*AuditRepo)(nil)
