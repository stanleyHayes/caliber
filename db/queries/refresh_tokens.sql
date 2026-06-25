-- name: SaveRefreshToken :exec
INSERT INTO refresh_tokens (id, user_id, expires_at, revoked)
VALUES ($1, $2, $3, false);

-- name: ConsumeRefreshToken :one
UPDATE refresh_tokens SET revoked = true
WHERE id = sqlc.arg(id) AND NOT revoked AND expires_at > sqlc.arg(now)
RETURNING id, user_id, expires_at;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked = true WHERE id = $1;
