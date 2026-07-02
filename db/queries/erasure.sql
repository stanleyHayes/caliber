-- Right-to-erasure + retention hard-delete primitives (CAL-118 / CAL-158).
-- Deleting a candidate cascades to its profiles, matches, interviews, and
-- applications via ON DELETE CASCADE; the scoped deletes below are still run
-- explicitly so the erasure cascade is honoured even where a store is not the
-- database. The user row is retained-but-anonymised (it may be referenced), and
-- the append-only audit trail is retained-but-tombstoned.

-- name: DeleteCandidate :exec
DELETE FROM candidates WHERE id = $1;

-- name: DeleteTalentProfilesByCandidate :exec
DELETE FROM talent_profiles WHERE candidate_id = $1;

-- name: DeleteApplicationsByCandidate :exec
DELETE FROM applications WHERE candidate_id = $1;

-- name: DeleteMatchesByCandidate :exec
DELETE FROM matches WHERE candidate_id = $1;

-- name: AnonymizeUser :exec
UPDATE users SET name = '', password_hash = '', email = $2 WHERE id = $1;

-- name: TombstoneAuditActor :exec
UPDATE audit_log SET actor_user_id = 'erased' WHERE actor_user_id = $1;
