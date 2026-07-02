-- name: CreateEmployer :exec
INSERT INTO employers (id, company_name, contact_user_id, created_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO NOTHING;
