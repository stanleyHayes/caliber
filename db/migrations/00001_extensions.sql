-- +goose Up
-- pgvector powers candidate/role embedding recall (matching stage 1).
CREATE EXTENSION IF NOT EXISTS vector;

-- +goose Down
DROP EXTENSION IF EXISTS vector;
