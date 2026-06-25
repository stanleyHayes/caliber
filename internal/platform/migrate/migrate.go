// Package migrate applies database schema migrations using goose.
package migrate

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

// Up applies all pending migrations from dir against db (postgres dialect).
func Up(db *sql.DB, dir string) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db, dir)
}

// Down rolls back the most recent migration (used by tests).
func Down(db *sql.DB, dir string) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Down(db, dir)
}
