-- +goose Up
-- Core relational schema (spec section 9). IDs are app-generated text (kernel.ID).
-- LLM-produced structures (role_spec, rubric, breakdown, report_card) are JSONB.
-- Embedding dimension 1536 matches OpenAI text-embedding-3-small.

CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    role          TEXT NOT NULL,
    name          TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'active',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE employers (
    id              TEXT PRIMARY KEY,
    company_name    TEXT NOT NULL,
    contact_user_id TEXT REFERENCES users (id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE candidates (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    location    TEXT,
    preferences JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE roles (
    id             TEXT PRIMARY KEY,
    employer_id    TEXT NOT NULL REFERENCES employers (id) ON DELETE CASCADE,
    title          TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'draft',
    role_spec      JSONB NOT NULL DEFAULT '{}'::jsonb,
    rubric         JSONB NOT NULL DEFAULT '{}'::jsonb,
    salary_band    JSONB,
    role_embedding vector(1536),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE talent_profiles (
    id                TEXT PRIMARY KEY,
    candidate_id      TEXT NOT NULL REFERENCES candidates (id) ON DELETE CASCADE,
    cv_text           TEXT,
    summary           TEXT,
    profile           JSONB NOT NULL DEFAULT '{}'::jsonb,
    profile_embedding vector(1536),
    passport_status   TEXT NOT NULL DEFAULT 'cv_only',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE matches (
    id                 TEXT PRIMARY KEY,
    role_id            TEXT NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    candidate_id       TEXT NOT NULL REFERENCES candidates (id) ON DELETE CASCADE,
    overall_score      DOUBLE PRECISION NOT NULL,
    confidence         TEXT NOT NULL,
    breakdown          JSONB NOT NULL DEFAULT '[]'::jsonb,
    rationale          TEXT,
    watch_outs         JSONB NOT NULL DEFAULT '[]'::jsonb,
    thin_evidence_flag BOOLEAN NOT NULL DEFAULT false,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (role_id, candidate_id)
);

CREATE TABLE talent_interviews (
    id           TEXT PRIMARY KEY,
    role_id      TEXT NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    candidate_id TEXT NOT NULL REFERENCES candidates (id) ON DELETE CASCADE,
    mode         TEXT NOT NULL DEFAULT 'text',
    status       TEXT NOT NULL DEFAULT 'open',
    report_card  JSONB,
    confidence   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE interview_turns (
    id             TEXT PRIMARY KEY,
    interview_id   TEXT NOT NULL REFERENCES talent_interviews (id) ON DELETE CASCADE,
    ordinal        INTEGER NOT NULL,
    question       TEXT NOT NULL,
    answer         TEXT,
    competency_tag TEXT,
    UNIQUE (interview_id, ordinal)
);

CREATE TABLE applications (
    id               TEXT PRIMARY KEY,
    role_id          TEXT NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    candidate_id     TEXT NOT NULL REFERENCES candidates (id) ON DELETE CASCADE,
    profile_id       TEXT REFERENCES talent_profiles (id) ON DELETE SET NULL,
    source           TEXT NOT NULL,
    tailored_summary TEXT,
    status           TEXT NOT NULL DEFAULT 'drafted',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE audit_log (
    id            TEXT PRIMARY KEY,
    actor_user_id TEXT NOT NULL,
    action        TEXT NOT NULL,
    entity        TEXT NOT NULL,
    entity_id     TEXT NOT NULL,
    before_json   JSONB,
    after_json    JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS applications;
DROP TABLE IF EXISTS interview_turns;
DROP TABLE IF EXISTS talent_interviews;
DROP TABLE IF EXISTS matches;
DROP TABLE IF EXISTS talent_profiles;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS candidates;
DROP TABLE IF EXISTS employers;
DROP TABLE IF EXISTS users;
