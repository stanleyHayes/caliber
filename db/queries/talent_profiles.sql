-- name: CreateTalentProfile :exec
INSERT INTO talent_profiles (id, candidate_id, summary, profile, passport_status)
VALUES ($1, $2, $3, $4, $5);

-- name: GetTalentProfile :one
SELECT id, candidate_id, summary, profile, passport_status FROM talent_profiles WHERE id = $1;

-- name: GetTalentProfileByCandidateID :one
SELECT id, candidate_id, summary, profile, passport_status FROM talent_profiles WHERE candidate_id = $1;

-- name: UpdateTalentProfile :execrows
UPDATE talent_profiles SET summary = $2, profile = $3, passport_status = $4 WHERE id = $1;

-- name: ListTalentProfiles :many
SELECT id, candidate_id, summary, profile, passport_status FROM talent_profiles ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountTalentProfiles :one
SELECT count(*) FROM talent_profiles;
