-- name: CreateInterview :exec
INSERT INTO talent_interviews (id, role_id, candidate_id, mode, status, report_card, confidence, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: UpdateInterview :execrows
UPDATE talent_interviews
SET mode = $2, status = $3, report_card = $4, confidence = $5
WHERE id = $1;

-- name: GetInterview :one
SELECT id, role_id, candidate_id, mode, status, report_card, confidence, created_at
FROM talent_interviews WHERE id = $1;

-- name: ListInterviewsByCandidate :many
SELECT id, role_id, candidate_id, mode, status, report_card, confidence, created_at
FROM talent_interviews WHERE candidate_id = $1
ORDER BY created_at DESC, id LIMIT $2 OFFSET $3;

-- name: CountInterviewsByCandidate :one
SELECT count(*) FROM talent_interviews WHERE candidate_id = $1;

-- name: DeleteInterviewsByCandidate :exec
DELETE FROM talent_interviews WHERE candidate_id = $1;

-- name: InsertInterviewTurn :exec
INSERT INTO interview_turns (id, interview_id, ordinal, question, answer, competency_tag)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DeleteInterviewTurns :exec
DELETE FROM interview_turns WHERE interview_id = $1;

-- name: ListInterviewTurns :many
SELECT id, interview_id, ordinal, question, answer, competency_tag
FROM interview_turns WHERE interview_id = $1 ORDER BY ordinal;
