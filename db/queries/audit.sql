-- name: AppendAuditEntry :exec
INSERT INTO audit_log (id, actor_user_id, action, entity, entity_id, before_json, after_json, created_at, owner_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: ListAuditLog :many
SELECT id, actor_user_id, action, entity, entity_id, before_json, after_json, created_at, owner_id
FROM audit_log WHERE entity = $1 AND entity_id = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4;

-- name: CountAuditLog :one
SELECT count(*) FROM audit_log WHERE entity = $1 AND entity_id = $2;

-- name: ListAuditLogForOwner :many
SELECT id, actor_user_id, action, entity, entity_id, before_json, after_json, created_at, owner_id
FROM audit_log
WHERE entity = $1 AND entity_id = $2 AND ($3 = '' OR owner_id = $3)
ORDER BY created_at DESC LIMIT $4 OFFSET $5;

-- name: CountAuditLogForOwner :one
SELECT count(*) FROM audit_log
WHERE entity = $1 AND entity_id = $2 AND ($3 = '' OR owner_id = $3);

-- name: SearchAuditLog :many
SELECT id, actor_user_id, action, entity, entity_id, before_json, after_json, created_at, owner_id
FROM audit_log
WHERE created_at >= $1 AND created_at <= $2
  AND (cardinality($3::text[]) = 0 OR action = any($3::text[]))
  AND (cardinality($4::text[]) = 0 OR entity = any($4::text[]))
ORDER BY created_at DESC
LIMIT $5 OFFSET $6;

-- name: CountAuditLogForReport :one
SELECT count(*)
FROM audit_log
WHERE created_at >= $1 AND created_at <= $2
  AND (cardinality($3::text[]) = 0 OR action = any($3::text[]))
  AND (cardinality($4::text[]) = 0 OR entity = any($4::text[]));
