#!/usr/bin/env bash
# Restore a Project Caliber Postgres backup (CAL-151).
#
# Usage: scripts/db-restore.sh <backup.dump>
# Restores the given custom-format dump into the database named by
# CALIBER_DATABASE_URL. This is destructive — it drops and recreates the objects
# in the dump — so run it against the intended target during a maintenance window.
# See docs/runbooks/backup-restore.md for the full restore + drill procedure.
set -euo pipefail

: "${CALIBER_DATABASE_URL:?CALIBER_DATABASE_URL must be set}"
dump="${1:?usage: db-restore.sh <backup.dump>}"
[ -f "$dump" ] || { echo "backup file not found: $dump" >&2; exit 1; }

# --clean --if-exists drops existing objects first so the restore is idempotent;
# --no-owner ignores the dump's ownership so it lands under the target role.
pg_restore --dbname="$CALIBER_DATABASE_URL" --clean --if-exists --no-owner "$dump"
echo "restore complete from: $dump"
