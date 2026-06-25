-- name: AppendAuditEntry :exec
INSERT INTO audit_log (id, actor_user_id, action, entity, entity_id, before_json, after_json, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListAuditLog :many
SELECT id, actor_user_id, action, entity, entity_id, before_json, after_json, created_at
FROM audit_log WHERE entity = $1 AND entity_id = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4;

-- name: CountAuditLog :one
SELECT count(*) FROM audit_log WHERE entity = $1 AND entity_id = $2;
