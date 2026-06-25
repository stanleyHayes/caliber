-- name: CreateRole :exec
INSERT INTO roles (id, employer_id, title, status, role_spec, rubric, salary_band, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetRole :one
SELECT id, employer_id, title, status, role_spec, rubric, salary_band, created_at
FROM roles
WHERE id = $1;

-- name: UpdateRole :execrows
UPDATE roles
SET title = $2, status = $3, role_spec = $4, rubric = $5, salary_band = $6
WHERE id = $1;

-- name: ListRolesByEmployer :many
SELECT id, employer_id, title, status, role_spec, rubric, salary_band, created_at
FROM roles
WHERE employer_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountRolesByEmployer :one
SELECT count(*) FROM roles WHERE employer_id = $1;
