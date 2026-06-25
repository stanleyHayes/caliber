-- name: CreateCandidate :exec
INSERT INTO candidates (id, user_id, location, preferences) VALUES ($1, $2, $3, $4);

-- name: GetCandidate :one
SELECT id, user_id, location, preferences FROM candidates WHERE id = $1;

-- name: GetCandidateByUserID :one
SELECT id, user_id, location, preferences FROM candidates WHERE user_id = $1;

-- name: UpdateCandidate :execrows
UPDATE candidates SET location = $2, preferences = $3 WHERE id = $1;

-- name: ListCandidates :many
SELECT id, user_id, location, preferences FROM candidates ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountCandidates :one
SELECT count(*) FROM candidates;
