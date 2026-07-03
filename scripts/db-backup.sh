#!/usr/bin/env bash
# Automated Postgres backup for Project Caliber (CAL-151).
#
# Writes a timestamped, compressed custom-format dump of the database named by
# CALIBER_DATABASE_URL into the backup directory, then prunes old backups. Run it
# from a scheduler (cron / Render Cron Job / k8s CronJob) to meet the RPO target
# documented in docs/runbooks/backup-restore.md.
#
# Env:
#   CALIBER_DATABASE_URL  (required) postgres connection string
#   CALIBER_BACKUP_DIR    (default ./backups) where dumps are written
#   CALIBER_BACKUP_KEEP   (default 14) how many recent dumps to retain
set -euo pipefail

: "${CALIBER_DATABASE_URL:?CALIBER_DATABASE_URL must be set}"
backup_dir="${CALIBER_BACKUP_DIR:-./backups}"
keep="${CALIBER_BACKUP_KEEP:-14}"

mkdir -p "$backup_dir"
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
out="$backup_dir/caliber-$stamp.dump"

# --format=custom is compressed and restorable selectively/in parallel with
# pg_restore; --no-owner keeps the dump portable across roles and environments.
pg_dump --dbname="$CALIBER_DATABASE_URL" --format=custom --no-owner --file="$out"
echo "backup written: $out"

# Retention: delete all but the newest $keep dumps.
# shellcheck disable=SC2012 # filenames are timestamped and shell-safe.
ls -1t "$backup_dir"/caliber-*.dump 2>/dev/null | tail -n +"$((keep + 1))" | while read -r old; do
  rm -f -- "$old"
  echo "pruned old backup: $old"
done
