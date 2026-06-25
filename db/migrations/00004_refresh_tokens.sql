-- +goose Up
-- Refresh-token grants for rotation + revocation (CAL-019/020). Each row is a
-- single-use grant keyed by its jti; Consume flips revoked atomically.
CREATE TABLE refresh_tokens (
    id         TEXT PRIMARY KEY,                                   -- jti
    user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked    BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id);

-- +goose Down
DROP TABLE IF EXISTS refresh_tokens;
