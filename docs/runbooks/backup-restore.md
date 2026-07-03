# Backups & disaster recovery (CAL-151)

How Project Caliber backs up its Postgres database, how to restore it, and the
recovery objectives the process is designed to meet.

## Objectives

| Objective | Target | Rationale |
|---|---|---|
| **RPO** (max data loss) | ≤ 24h (POC) / ≤ 1h (prod) | Daily automated dumps for the POC; hourly WAL/PITR for production. |
| **RTO** (max downtime) | ≤ 1h | A custom-format dump of the POC dataset restores in minutes; the bound is provisioning + verification. |

The only durable state is Postgres (candidates, roles, matches, interviews,
applications, audit trail). Redis holds transient queue state and is rebuildable;
the LLM/embedding providers are stateless. So a Postgres backup is a complete
recovery point.

## What is backed up

A logical `pg_dump` in **custom format** (`-Fc`) of the whole database — schema,
data, the `vector` (pgvector) extension, and indexes. Custom format is compressed
and supports selective and parallel restore.

## Automated backups

Run [`scripts/db-backup.sh`](../../scripts/db-backup.sh) on a schedule (Render
Cron Job / k8s CronJob / cron). It writes a timestamped dump and prunes old ones.

```bash
CALIBER_DATABASE_URL=postgres://… \
CALIBER_BACKUP_DIR=/var/backups/caliber \
CALIBER_BACKUP_KEEP=14 \
  scripts/db-backup.sh          # or: make db-backup
```

For production, also enable the managed provider's **point-in-time recovery**
(continuous WAL archiving) to hit the 1h RPO between logical dumps; store dumps
off the database host (object storage) so a host loss doesn't take the backups.

## Restore

Restoring is **destructive** — it drops and recreates the objects in the dump.
Run it against the intended target during a maintenance window.

```bash
CALIBER_DATABASE_URL=postgres://… \
  scripts/db-restore.sh /var/backups/caliber/caliber-20260703T000000Z.dump
# or: make db-restore DUMP=/path/to/backup.dump
```

After restoring: run `make migrate` (a newer schema may post-date the dump — goose
is idempotent), confirm `/readyz` is green, and smoke-test a login + one shortlist.

## Restore drill (proving backups are restorable)

A backup you have never restored is a hope, not a backup. The drill is automated
as an integration test —
[`backup_restore_integration_test.go`](../../internal/adapters/outbound/postgres/backup_restore_integration_test.go),
run with `make restore-drill` — which:

1. spins up Postgres, migrates, and seeds a known row;
2. takes a `pg_dump` backup;
3. restores it into a **fresh** database with `pg_restore`;
4. asserts the seeded row survived.

It exercises the same `pg_dump`/`pg_restore` format the operational scripts
produce, so a green drill is evidence the real backups are restorable. Run it in
CI (Docker-backed) and, in production, schedule a quarterly manual restore of the
latest dump into a scratch database following the steps above.

## If the database is lost

1. Provision a fresh Postgres (same major version, `pgvector` available).
2. `scripts/db-restore.sh <latest-dump>` (or PITR to the desired timestamp).
3. `make migrate` to apply any schema newer than the dump.
4. Point `CALIBER_DATABASE_URL` at the restored instance and roll the API/worker.
5. Verify `/readyz`, a login, one shortlist, and that the audit trail is intact.
