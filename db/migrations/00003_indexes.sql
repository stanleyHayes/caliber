-- +goose Up
-- HNSW cosine indexes for vector recall (matching stage 1).
CREATE INDEX idx_roles_embedding ON roles USING hnsw (role_embedding vector_cosine_ops);
CREATE INDEX idx_profiles_embedding ON talent_profiles USING hnsw (profile_embedding vector_cosine_ops);

-- Foreign-key / lookup indexes.
CREATE INDEX idx_roles_employer ON roles (employer_id);
CREATE INDEX idx_roles_status ON roles (status);
CREATE INDEX idx_candidates_user ON candidates (user_id);
CREATE INDEX idx_profiles_candidate ON talent_profiles (candidate_id);
CREATE INDEX idx_matches_role ON matches (role_id);
CREATE INDEX idx_matches_candidate ON matches (candidate_id);
CREATE INDEX idx_interviews_candidate ON talent_interviews (candidate_id);
CREATE INDEX idx_interviews_role ON talent_interviews (role_id);
CREATE INDEX idx_turns_interview ON interview_turns (interview_id);
CREATE INDEX idx_applications_candidate ON applications (candidate_id);
CREATE INDEX idx_applications_role ON applications (role_id);
CREATE INDEX idx_audit_entity ON audit_log (entity, entity_id);

-- +goose Down
DROP INDEX IF EXISTS idx_audit_entity;
DROP INDEX IF EXISTS idx_applications_role;
DROP INDEX IF EXISTS idx_applications_candidate;
DROP INDEX IF EXISTS idx_turns_interview;
DROP INDEX IF EXISTS idx_interviews_role;
DROP INDEX IF EXISTS idx_interviews_candidate;
DROP INDEX IF EXISTS idx_matches_candidate;
DROP INDEX IF EXISTS idx_matches_role;
DROP INDEX IF EXISTS idx_profiles_candidate;
DROP INDEX IF EXISTS idx_candidates_user;
DROP INDEX IF EXISTS idx_roles_status;
DROP INDEX IF EXISTS idx_roles_employer;
DROP INDEX IF EXISTS idx_profiles_embedding;
DROP INDEX IF EXISTS idx_roles_embedding;
