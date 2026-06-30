package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX matches the execution surface used by sqlc queries.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// TruncateAll removes all application data from Postgres while preserving the
// schema, extensions, and migration history. It is used by the reseed command
// to return the database to a blank state before reloading deterministic demo
// data.
func TruncateAll(ctx context.Context, db DBTX) error {
	_, err := db.Exec(ctx, `
		TRUNCATE TABLE
			interview_turns,
			talent_interviews,
			matches,
			applications,
			talent_profiles,
			candidates,
			refresh_tokens,
			roles,
			employers,
			users,
			audit_log
		RESTART IDENTITY CASCADE
	`)
	return err
}
