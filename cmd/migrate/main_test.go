package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrationsDirUsesDefault(t *testing.T) {
	t.Setenv("CALIBER_MIGRATIONS_DIR", "")

	require.Equal(t, defaultMigrationsDir, migrationsDir())
}

func TestMigrationsDirUsesOverride(t *testing.T) {
	t.Setenv("CALIBER_MIGRATIONS_DIR", "/app/db/migrations")

	require.Equal(t, "/app/db/migrations", migrationsDir())
}

func TestRunRequiresDatabaseURL(t *testing.T) {
	t.Setenv("CALIBER_DATABASE_URL", "")

	require.ErrorContains(t, run(), "CALIBER_DATABASE_URL is required")
}
