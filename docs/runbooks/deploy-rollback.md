<!--
  CAL-149 — Zero-downtime deploys & rollback runbook.

  VALIDATION NOTICE: This runbook describes Render deploy/health-check/rollback
  behavior and goose migration workflow as configured for Project Caliber. Render's
  dashboard/API surface and health-check semantics evolve; before relying on any
  concrete step below, validate it against Render's current documentation and this
  service's actual dashboard settings. It is operational documentation, not
  executable config, and cannot be applied or verified in this sandbox.

  Cross-references (single source of truth — do not duplicate here):
    - Environment topology & config parity ....... docs/environments.md
    - Secrets (never in repo; per-env store) ...... docs/runbooks/secret-rotation.md
    - Backup / restore / DR & restore drill ....... docs/runbooks/backup-restore.md
    - Migrate binary / image ...................... cmd/migrate, deploy/Dockerfile.migrate
    - Migrations (goose up/down pairs) ............ db/migrations/*.sql
    - Readiness checks (Postgres + Redis) ......... internal/platform/readiness/readiness.go
    - Health/readiness HTTP endpoints ............. internal/adapters/inbound/httpserver/router.go
-->

# Zero-downtime deploys & rollback (CAL-149)

How Project Caliber ships a new build without downtime, how a bad deploy is caught
and rolled back (automatically or by hand), and how to keep database migrations
safe and reversible so a rollback of the app is never blocked by the schema.

This follows the shared runbook structure: **Symptoms → Impact → Triage →
Mitigation → Escalation**, framed here around a rollout rather than an alert.

## Scope & the golden rule

- **App = code, DB = schema, and they roll independently.** The API/worker images
  (`deploy/Dockerfile.api`, `deploy/Dockerfile.worker`) are stateless and can be
  rolled forward or back at will. The database cannot: a rollback of the app must
  never require a rollback of the schema. That is what the **expand/contract**
  discipline below buys us.
- **Golden rule:** every deploy must leave the schema compatible with **both** the
  currently-running app version and the one being deployed. If a change can't be
  made that way in one step, split it across releases (see
  [Expand / contract](#db-migration-safety-expandcontract)).
- The three environments (`dev` / `staging` / `prod`) run the **same image and
  schema** with per-environment config and independent secrets — see
  [environments.md](../environments.md). Rehearse every rollout and rollback in
  `staging` first; it is a faithful prod rehearsal by design.

## How a zero-downtime rollout works (Render)

Each service (`caliber-api`, `caliber-worker`) is deployed from its Docker image on
Render. A rollout is **health-gated**: Render starts the new instance, waits for it
to pass its health check, and only then shifts traffic and retires the old
instance. The old version keeps serving until the new one is proven healthy, so a
successful deploy has no downtime and a failed one never receives traffic.

### Liveness vs. readiness (what Render probes)

The API exposes two endpoints on the HTTP port (`:8080`, see
[router.go](../../internal/adapters/inbound/httpserver/router.go)):

| Endpoint | Meaning | Behavior |
|---|---|---|
| `GET /healthz` | **Liveness** | Always `200 {"status":"ok"}` once the process is up. Says "the process is running," nothing about dependencies. |
| `GET /readyz` | **Readiness** | `200 {"status":"ready"}` only when every dependency check passes; `503 {"status":"not_ready"}` otherwise. Each check is bounded at 2s. |

`/readyz` runs the aggregate readiness checks in
[readiness.go](../../internal/platform/readiness/readiness.go) — currently a
Postgres reachability check and a Redis `PING`. So a new instance that boots but
**cannot reach its database or Redis reports itself not-ready** and never gets
traffic.

**Configure Render's health check path to `/readyz`, not `/healthz`.** Gating on
`/readyz` is what makes the rollout dependency-aware: a build shipped with a broken
`CALIBER_DATABASE_URL` / `CALIBER_REDIS_URL` (or a DB that's down) fails the health
gate instead of taking over and serving 503s. `/healthz` is for a coarser
"is the process alive" liveness probe only.

> The worker (`cmd/worker`) has no inbound HTTP surface; Render treats it as a
> background/worker service. Its "health" is observed via queue metrics and the
> queue runbook ([queue-operations.md](./queue-operations.md)) rather than an HTTP
> probe. Roll the worker with the same forward/rollback steps below.

### Ordering: migrate before the app

Deploy order for any release that includes a migration:

1. **Run the migration job first** (the `caliber-migrate` image,
   `deploy/Dockerfile.migrate`, which bakes `db/migrations` in at `/db/migrations`
   and runs `cmd/migrate`, i.e. `goose up`). Because migrations are **expand-only**
   (additive/backward-compatible — see below), applying them while the *old* app is
   still running is safe.
2. **Then roll the API and worker** to the new image. The new app finds the schema
   it expects; the old app kept working against the expanded schema in the gap.

This ordering is what makes the deploy zero-downtime: at no instant is a running
app version paired with a schema it can't use.

## Rollback

### A) Automatic rollback on a failed health check

Because traffic only shifts after `/readyz` passes, a new version that fails to
become ready **is never promoted** — Render keeps the last-known-good version live
and the deploy is marked failed. In effect the platform "rolls back" by never
rolling forward. No traffic hits the bad build; users see the previous version
throughout. Confirm the health-check path is `/readyz` and that the failure
threshold/timeout are tight enough to fail fast (validate current values in the
Render dashboard).

This covers the common failure modes: a crash-looping binary, a bad config/secret
that trips `Config.Validate()`/`ProdSafetyIssues()` at boot (see
[environments.md](../environments.md)), or a dependency the new build can't reach.

### B) Manual rollback to the previous deploy

When a deploy passes the health gate but is still bad in production (a functional
regression `/readyz` can't detect — wrong behavior, elevated 5xx, a broken flow),
roll back by hand to the previous good release:

1. **Decide fast.** Cross-check the 5xx / latency / AI-failure runbooks
   ([5xx-spike.md](./5xx-spike.md), [high-latency.md](./high-latency.md),
   [high-ai-failure-rate.md](./high-ai-failure-rate.md)) to confirm the regression
   correlates with the deploy timestamp.
2. **Roll back the app** to the immediately-preceding successful deploy. In Render
   this is **"Rollback" to the previous deploy** for `caliber-api` (and
   `caliber-worker` if it also changed) — it redeploys the previous image, which is
   itself health-gated on `/readyz`. (Validate the exact dashboard/API affordance
   against Render's current docs.)
3. **Do NOT roll the schema back by default.** Thanks to expand/contract, the
   previous app version is compatible with the current (expanded) schema, so
   reverting only the app is safe and sufficient. Leave the migration in place.
   Only run a `down` migration in the narrow case below.
4. **Verify recovery:** `/readyz` green on the restored instances, error rate and
   latency back to baseline, and a smoke test — a login plus one shortlist (the
   same smoke test used after a restore in
   [backup-restore.md](./backup-restore.md)).

### When (rarely) you must also undo a migration

If the failed release shipped a migration that is itself the problem, roll the app
back first (step B above), then reverse the last migration:

- Migrations are **reversible goose pairs**: every file in
  [db/migrations](../../db/migrations) has a `-- +goose Up` and a matching
  `-- +goose Down` (e.g. `00004_refresh_tokens.sql` drops the table it created;
  `00003_indexes.sql` drops each index it added). The migrate package exposes both
  directions ([internal/platform/migrate](../../internal/platform/migrate)); the
  deploy job runs `Up`, and `Down` reverses the most recent migration.
- **Reversing a purely additive (expand) migration is safe** — dropping a
  new-and-unused table/column/index the old app never touched loses no live data.
- **A contract step is destructive by design** (it drops a column/table the new app
  stopped using). Reversing forward past a contract, or hand-running a `down` that
  drops something still in use, can lose data — so **snapshot first**: take a backup
  per [backup-restore.md](./backup-restore.md) before any `down`, and prefer a
  point-in-time restore over a blind `down` when live data is at stake.

## DB migration safety: expand / contract

To keep the golden rule true, make every schema change in **two deploys** so the
schema is always compatible with the app version on either side of a rollout:

1. **Expand (this release):** additive, backward-compatible changes only — add a
   nullable column, a new table, a new index, a new enum value. The *old* app
   ignores it; the *new* app starts using it. Both versions run fine against the
   expanded schema, so this migration can be applied before the app rolls and a
   rollback of the app needs no schema change.
2. **Migrate data / dual-write** (as needed): backfill and, where a value moves,
   have the new app write both old and new locations until the old column is
   unused. Backfills should be batched/idempotent so they can be re-run.
3. **Contract (a *later* release, only after the expand version is fully rolled out
   and proven):** remove what nothing reads anymore — drop the old column/table,
   add the `NOT NULL`/constraint. Never expand and contract in the same deploy.

**Rules that keep migrations reversible and non-blocking:**

- One logical change per migration file, each with a real `-- +goose Down`.
- Never rename in place (rename = drop + add, which breaks the running app). Add
  the new name, dual-write, backfill, then drop the old name in a later contract.
- Never drop or `NOT NULL`-tighten a column the currently-deployed app still uses.
- Prefer index builds that don't hold long locks; a schema change must not stall
  live traffic (validate locking behavior for the specific statement).
- Keep migrations small and forward-only in normal operation — `down` is the
  break-glass path, gated on a backup, not the routine rollback mechanism.

## Pre-flight checklist (before promoting to prod)

- [ ] Rollout + **rollback rehearsed in `staging`** and both verified green
      (satisfies the CAL-149 AC: *rollback tested*).
- [ ] Any migration is **expand-only** and has a tested `-- +goose Down`
      (AC: *migrations reversible*); the migration integration test passes.
- [ ] Render health-check path is **`/readyz`** with a fail-fast threshold.
- [ ] Migrate job is ordered **before** the API/worker roll.
- [ ] A fresh backup exists per [backup-restore.md](./backup-restore.md) (only a
      `down`/contract that touches live data strictly needs it, but always have one).
- [ ] Config/secrets present for the target env — see
      [environments.md](../environments.md) and
      [secret-rotation.md](./secret-rotation.md).

## Escalation

- Automatic rollback held the line (bad build never promoted): no page — capture
  the failed-deploy logs, fix forward, redeploy.
- Manual rollback restored service: file a short incident note; keep the bad image
  for post-mortem.
- Rollback did **not** restore service, or a `down` migration is implicated in data
  loss: this is a data-integrity incident — escalate and move to disaster recovery
  ([backup-restore.md](./backup-restore.md), point-in-time restore) rather than
  further `down` migrations.
