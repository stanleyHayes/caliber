-- name: ListRetentionEligibleCandidates :many
-- Candidate-role users whose account was created on or before the cutoff — i.e.
-- whose personal data has aged past the retention window (CAL-158). Candidate age
-- is taken from the owning user because talent.Candidate carries no timestamp and
-- a registered candidate's id equals their user id.
SELECT id FROM users WHERE role = 'candidate' AND created_at <= $1 ORDER BY created_at;
