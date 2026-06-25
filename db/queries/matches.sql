-- name: UpsertMatch :exec
INSERT INTO matches (id, role_id, candidate_id, overall_score, confidence, breakdown, rationale, watch_outs, thin_evidence_flag, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
ON CONFLICT (role_id, candidate_id) DO UPDATE SET
  overall_score = EXCLUDED.overall_score,
  confidence = EXCLUDED.confidence,
  breakdown = EXCLUDED.breakdown,
  rationale = EXCLUDED.rationale,
  watch_outs = EXCLUDED.watch_outs,
  thin_evidence_flag = EXCLUDED.thin_evidence_flag;

-- name: ListMatchesByRole :many
SELECT id, role_id, candidate_id, overall_score, confidence, breakdown, rationale, watch_outs, thin_evidence_flag, created_at
FROM matches WHERE role_id = $1 ORDER BY overall_score DESC LIMIT $2 OFFSET $3;

-- name: CountMatchesByRole :one
SELECT count(*) FROM matches WHERE role_id = $1;

-- name: ListMatchesByCandidate :many
SELECT id, role_id, candidate_id, overall_score, confidence, breakdown, rationale, watch_outs, thin_evidence_flag, created_at
FROM matches WHERE candidate_id = $1 ORDER BY overall_score DESC LIMIT $2 OFFSET $3;

-- name: CountMatchesByCandidate :one
SELECT count(*) FROM matches WHERE candidate_id = $1;
