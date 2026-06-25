-- name: CreateUser :exec
INSERT INTO users (id, email, role, name, password_hash, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetUser :one
SELECT id, email, role, name, password_hash, status, created_at FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, role, name, password_hash, status, created_at FROM users WHERE email = $1;

-- name: UpdateUser :execrows
UPDATE users SET email = $2, role = $3, name = $4, password_hash = $5, status = $6 WHERE id = $1;
