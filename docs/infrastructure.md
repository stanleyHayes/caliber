# Infrastructure as Code (CAL-150)

> **Validation note.** The provider configs referenced here (Render Blueprint,
> Vercel project config) **cannot be applied or validated inside the build
> sandbox** and provider schemas drift over time. Validate
> [`render.yaml`](../render.yaml) against Render's current
> [Blueprint spec](https://render.com/docs/blueprint-spec) — and the Vercel
> config against Vercel's current schema — before applying. Treat the plan
> names, regions, and any `preDeployCommand` behavior as **placeholders to
> confirm**, not as verified values.

Everything needed to stand up an environment is codified in the repo so an
environment is **reproducible from code**. This page maps *what* is codified
*where* and *how* to apply it. It does not restate the env/secret/backup docs —
it links them.

## What is codified where

| Layer | Source of truth | Covers | Applied via |
|---|---|---|---|
| **Backend** (Postgres+pgvector, Redis, `api`, `worker`) | [`render.yaml`](../render.yaml) — Render Blueprint | Managed DB + Key-Value store, the `api` web service (from [`deploy/Dockerfile.api`](../deploy/Dockerfile.api), health `/readyz`) and the `worker` (from [`deploy/Dockerfile.worker`](../deploy/Dockerfile.worker)), plus env wiring | Render Blueprint sync |
| **Migrations** | [`deploy/Dockerfile.migrate`](../deploy/Dockerfile.migrate) (goose + `db/migrations`) | Schema (incl. `CREATE EXTENSION vector`) | Pre-deploy / CD step (see below) |
| **Frontend** (SPA) | Vercel project config — **CAL-152** (separate story) | React+Vite SPA build, preview envs, prod promotion | Vercel |
| **Local** (full stack incl. observability) | [`docker-compose.yml`](../docker-compose.yml) | Postgres, Redis, migrate, api, worker, web, Prometheus/Loki/Grafana | `docker compose up --build` |

The **same binary and schema** run in every environment; only configuration
differs (see [environments.md](environments.md)). The Blueprint therefore
encodes topology + non-secret config; per-environment secrets are injected at
apply time, never committed.

## Backend: `render.yaml`

Codifies, for the backend only:

- **`caliber-postgres`** — managed Postgres with **pgvector**. The extension is
  created idempotently by our goose migrations
  (`CREATE EXTENSION IF NOT EXISTS vector`), matching the local
  `pgvector/pgvector:pg17` image.
- **`caliber-redis`** — managed Key-Value (Redis) store backing Asynq
  (`maxmemoryPolicy: noeviction` so queued jobs are never evicted).
- **`caliber-api`** — `type: web`, built from `deploy/Dockerfile.api`,
  `healthCheckPath: /readyz` (served on the HTTP port by
  `internal/adapters/inbound/httpserver`; `/healthz` is the liveness sibling).
- **`caliber-worker`** — `type: worker`, built from `deploy/Dockerfile.worker`.
  No inbound HTTP; it exposes Prometheus metrics on
  `CALIBER_WORKER_METRICS_ADDR` (`:8081`).

### Env wiring

- **Injected (never hand-entered):** `CALIBER_DATABASE_URL` via
  `fromDatabase: caliber-postgres` and `CALIBER_REDIS_URL` via
  `fromService: caliber-redis` — both use `property: connectionString`.
- **Real secrets (`sync: false`):** `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`,
  `CALIBER_JWT_SECRET`, and (api only) `CALIBER_CORS_ORIGINS`. These are set once
  per environment in the Render dashboard / secret store and rotated per
  [runbooks/secret-rotation.md](runbooks/secret-rotation.md). Value shapes:
  [`.env.staging.example`](../.env.staging.example) /
  [`.env.production.example`](../.env.production.example); full tunable list:
  [`.env.example`](../.env.example).
- **Non-secret config** (`CALIBER_ENV`, `CALIBER_LOG_LEVEL`, addresses) is set
  inline in the Blueprint.

A Blueprint declares **one** environment's resources. Staging and production are
separate Render environments (own DB, Redis, and secrets — no sharing, per
[environments.md](environments.md)); reuse this Blueprint per environment rather
than pointing both at one set of backends.

### Migrations (pre-deploy / CD step)

Migrations do **not** run at api/worker boot. They run from
`deploy/Dockerfile.migrate` (goose, with `db/migrations` baked in and
`CALIBER_MIGRATIONS_DIR` set), against `CALIBER_DATABASE_URL`, **before** new
app code serves traffic:

1. `render.yaml` declares a `preDeployCommand: "/migrate up"` on `caliber-api`
   to document the intent. **Validate this:** Render runs `preDeployCommand`
   inside the *service* image, which is the api image — not the migrate image —
   so `/migrate` and `db/migrations` may not be present there. If so, drop the
   `preDeployCommand` and instead run the migrate image as an explicit CD job
   (e.g. a GitHub Actions step, or a Render one-off job) that builds/pulls
   `deploy/Dockerfile.migrate` and runs `/migrate up` against the environment's
   `CALIBER_DATABASE_URL` ahead of the app deploy.
2. Either way the ordering is: **migrate → deploy api/worker**, so schema is
   ready before traffic shifts.

## How to apply

**Backend (Render):**
1. Validate `render.yaml` against the current Blueprint spec; confirm plan names
   and region are real for the account.
2. Create/point a Render Blueprint at the repo. Render provisions
   `caliber-postgres` + `caliber-redis` and wires their connection strings.
3. Set the `sync: false` secrets for the environment (see the `.env.*.example`
   templates + secret-rotation runbook).
4. Ensure the migrate step runs before app rollout (see Migrations above).
5. Deploy. Health is gated on `/readyz`.

**Frontend (Vercel):** tracked in **CAL-152** — preview URL per PR + production
promotion.

**Local:** `docker compose up --build` brings up the whole stack (backend +
web + Prometheus/Loki/Grafana). See [demo-runbook.md](demo-runbook.md).

## Related docs

- [environments.md](environments.md) — dev/staging/prod topology, parity,
  secret isolation, boot-time validation.
- [runbooks/secret-rotation.md](runbooks/secret-rotation.md) — rotating the
  `sync: false` secrets.
- [runbooks/backup-restore.md](runbooks/backup-restore.md) — Postgres backups,
  restore drills, RPO/RTO (CAL-151); managed-provider PITR/WAL is deploy-side
  config for the DB provisioned by this Blueprint.
- [observability.md](observability.md) — metrics/traces/logs the deployed
  services emit.
