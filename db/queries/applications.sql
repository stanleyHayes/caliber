-- name: CreateApplication :exec
INSERT INTO applications (id, role_id, candidate_id, profile_id, source, tailored_summary, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, now());

-- name: GetApplication :one
SELECT id, role_id, candidate_id, profile_id, source, tailored_summary, status FROM applications WHERE id = $1;

-- name: UpdateApplication :execrows
UPDATE applications SET status = $2, tailored_summary = $3 WHERE id = $1;

-- name: ListApplicationsByCandidate :many
SELECT id, role_id, candidate_id, profile_id, source, tailored_summary, status
FROM applications WHERE candidate_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: CountApplicationsByCandidate :one
SELECT count(*) FROM applications WHERE candidate_id = $1;
