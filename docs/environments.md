# Environment topology (CAL-146)

Project Caliber runs in three environments with the **same code and schema** but
per-environment configuration and **independent secrets**. This documents what
each environment is, what differs between them, and how parity + secret isolation
are enforced.

## Environments

| Env (`CALIBER_ENV`) | Purpose | Data | Exposure |
|---|---|---|---|
| `dev` | Local development | In-memory stores or a local Postgres; deterministic dev LLM/embedder | localhost only |
| `staging` | Pre-production verification; mirrors prod topology | Its **own** managed Postgres + Redis; real (staging) API keys | Internal / preview URLs |
| `prod` | Live | Managed Postgres (+ PITR) + Redis; production API keys | Public, TLS at the edge |

The app is environment-agnostic: `cmd/api` and `cmd/worker` read every
environment-specific value from configuration (see [config.go](../internal/platform/config/config.go)),
so promoting a build across environments changes only configuration, never code.

## What differs per environment

| Setting | dev | staging | prod |
|---|---|---|---|
| `CALIBER_DATABASE_URL` | local / in-memory | staging DB (managed) | prod DB (managed, PITR) |
| `CALIBER_REDIS_URL` | local / unset | staging Redis | prod Redis |
| `ANTHROPIC_API_KEY` / `OPENAI_API_KEY` | unset → deterministic dev provider | staging keys | prod keys |
| `CALIBER_JWT_SECRET` | unset → ephemeral dev secret | staging secret (≥32B) | prod secret (≥32B) |
| `CALIBER_CORS_ORIGINS` | localhost | staging origins | prod origins (required) |
| `CALIBER_LOG_LEVEL` | `debug`/`info` | `info` | `info` |
| gRPC reflection | on | on | **off** (`!IsProd`) |
| HSTS | off | edge-dependent | on |
| Demo seed (`CALIBER_SEED_DEMO`) | on | optional | off |
| Retention sweep (`CALIBER_RETENTION_WINDOW`) | off | optional | on |

Templates: [`.env.example`](../.env.example) (dev),
[`.env.staging.example`](../.env.staging.example),
[`.env.production.example`](../.env.production.example).

## Secrets: no sharing across environments

Every environment has its **own** database, cache, API keys, and JWT secret,
stored in that environment's platform secret store (Render/Vercel env), never in
the repo and never copied between environments. A secret is compromised for all
environments that share it, so sharing would collapse the isolation that lets
staging be exercised safely. Rotation is per-environment — see
[runbooks/secret-rotation.md](runbooks/secret-rotation.md).

## Parity & enforcement (how drift is caught)

Parity is enforced in code, not just documented:

- **Required settings** — `Config.Validate()` lists missing required values
  (`CALIBER_DATABASE_URL`, `CALIBER_REDIS_URL`, the API keys, `CALIBER_JWT_SECRET`,
  and CORS origins in prod). In `dev` this only warns; in `staging`/`prod` the
  process **fails fast at boot** rather than starting half-configured.
- **No dev config in prod** — `Config.ProdSafetyIssues()` fails prod boot if
  `CALIBER_DATABASE_URL` or `CALIBER_REDIS_URL` points at a **local** endpoint
  (localhost/127.0.0.1/::1), catching a dev config accidentally promoted.
- **Prod-only hardening** — `IsProd()` turns off gRPC reflection, requires
  explicit CORS origins, requires a database (no in-memory fallback), and
  requires a ≥32-byte JWT secret (no ephemeral dev secret).

Because the same binary runs everywhere and every difference is a configuration
value validated at boot, staging is a faithful rehearsal of production.
