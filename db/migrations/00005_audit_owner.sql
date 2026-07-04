-- +goose Up
-- Owner (tenant) scoping for the audit trail (CAL-153): the employer that owns
-- the resource an entry concerns. Reviewer-facing reads (ListAuditLog) scope by
-- it so an employer sees only their own hiring-decision trail — closing the
-- cross-tenant read where a shared subject (e.g. a candidate rejected by several
-- employers) leaked another employer's decisions. Default '' = unowned; existing
-- rows are unowned and remain readable only via the unscoped/internal path.
ALTER TABLE audit_log ADD COLUMN owner_id TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_audit_log_owner ON audit_log (entity, entity_id, owner_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_audit_log_owner;
ALTER TABLE audit_log DROP COLUMN IF EXISTS owner_id;
