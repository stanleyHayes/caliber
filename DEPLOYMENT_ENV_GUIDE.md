# Environment Configuration Guide

## Overview

Project Caliber uses environment-based configuration for all three deployments:
- **Development** (local): In-memory stack, optional Postgres/Redis, offline LLM stubs
- **Staging**: Managed infrastructure, real AI models, team testing
- **Production**: Strict safety checks, required encryption, deny-by-default CORS

Configuration is split between backend (Go) and frontend (React/Vite):
- Backend reads from environment variables (loaded from `.env.X` files or secret stores)
- Frontend environment variables are baked into the bundle at build time (prefixed `VITE_`)

## Files Structure

### Backend
```
.env                  (local development, ignored by git)
.env.staging          (staging environment template)
.env.production       (production environment template)
```

### Frontend
```
web/.env              (local development, ignored by git)
web/.env.staging      (staging environment template)
web/.env.production   (production environment template)
```

## Quick Start: Development

### Option A: In-Memory Stack (No External Services)
```bash
cd /path/to/caliber
set -a; . ./.env; set +a
go run ./cmd/api
# In another terminal:
cd web && npm run dev
```

The dev `.env` includes dummy API keys; the in-memory LLM returns canned responses.

### Option B: With PostgreSQL + Redis
Requires Docker Compose running:
```bash
docker-compose up -d
set -a; . ./.env; set +a
make migrate      # create schema
go run ./cmd/api
# In another terminal:
cd web && npm run dev
```

## Deployment Checklist

### 1. Backend (API + Worker)

#### Create Production `.env.production`
Copy the provided template and fill in:

| Variable | Source | Notes |
|----------|--------|-------|
| `CALIBER_DATABASE_URL` | Neon / your DB | Must use DIRECT endpoint, include `?sslmode=require` |
| `CALIBER_REDIS_URL` | Render Redis / your Redis | Use `rediss://` for TLS |
| `ANTHROPIC_API_KEY` | Anthropic console | OR set `ANTHROPIC_AUTH_TOKEN` + `ANTHROPIC_BASE_URL` for Kimi |
| `OPENAI_API_KEY` | OpenAI console | For embeddings (required) |
| `CALIBER_JWT_SECRET` | Generate: `openssl rand -base64 48` | ≥32 bytes, unique per env |
| `CALIBER_FIELD_ENCRYPTION_KEY` | Generate: `openssl rand -base64 32` | Base64 32-byte AES-256 key; REQUIRED in prod |
| `CALIBER_CORS_ORIGINS` | Your Vercel URL | e.g., `https://projectcaliber.vercel.app` |
| `CALIBER_SERVICE_VERSION` | Git SHA | e.g., `abc123def456` (Render injects `RENDER_GIT_COMMIT`) |

#### Load Secrets Securely (Render Example)
On Render, create a Blueprint or set environment variables in the dashboard:
```bash
# CLI:
render env create caliber-api --env prod --var CALIBER_DATABASE_URL="***neon-dsn***"
render env create caliber-api --secret ANTHROPIC_API_KEY="sk-ant-***"
# Dashboard: Settings → Environment → Add secret
```

#### Pre-deployment Checks
```bash
# Verify build
make build

# Run tests
make test

# Check linting
make lint

# Coverage gate (≥80% on app code)
make cover-check

# Security scan
make scan
```

#### Migrate Database
If using Render (free tier):
```bash
# Run once as a one-off job:
render jobs create caliber-api \
  --docker-file deploy/Dockerfile.migrate \
  --command '/migrate up'
```

Once schema is ready:
```bash
go run ./cmd/api      # /readyz will turn green
go run ./cmd/worker   # background job consumer
```

### 2. Frontend (React/Vite on Vercel)

#### Create Production `.env.production` in `web/`
```bash
cd web
# Fill in:
VITE_API_URL=https://api.projectcaliber.app        # Your backend URL
VITE_SITE_URL=https://projectcaliber.vercel.app    # Your frontend URL
VITE_PLAUSIBLE_DOMAIN=projectcaliber.app           # (optional)
```

#### Deploy to Vercel
```bash
cd web
# Option A: Vercel CLI
vercel --prod \
  --env VITE_API_URL="https://api.projectcaliber.app" \
  --env VITE_SITE_URL="https://projectcaliber.vercel.app"

# Option B: Vercel Dashboard
# Push to GitHub, Vercel auto-deploys. Set env vars in Project Settings → Environment Variables.
```

#### Verify Build
```bash
cd web
npm run build              # SSR + prerender public pages
npm run verify:prerender   # Check that /login, /register, / are crawlable
```

### 3. Connectivity Check

Once both backend and frontend are deployed:

```bash
# Check backend health
curl https://api.projectcaliber.app/healthz
# Expected: 200 OK, `{"status":"ready"}`

# Check frontend
curl https://projectcaliber.vercel.app
# Expected: 200 OK, prerendered /index.html

# Test API from frontend (browser console)
fetch('https://api.projectcaliber.app/v1/auth/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ email: 'test@example.com', password: 'test' })
})
.then(r => r.json())
.then(console.log)
```

## Environment Parity (CAL-146)

Production boot **FAILS FAST** if:
1. A REQUIRED setting is missing
2. A secret is weak (e.g., JWT < 32 bytes)
3. An endpoint is local (e.g., `localhost:5432` instead of a managed host)

Staging should mirror production configuration but may use:
- Lower-tier database/Redis
- Shared API keys (factored into budgets)
- Shorter retention windows

### Production Safety Issues (Checked at Startup)
```go
cfg.ProdSafetyIssues()  // Reports dev configs running in prod
```

Example failures:
```
FATAL: CALIBER_DATABASE_URL points at a local endpoint
FATAL: CALIBER_FIELD_ENCRYPTION_KEY is required in prod
FATAL: CALIBER_CORS_ORIGINS is required in prod
```

## Key Rotation

See `docs/runbooks/key-rotation.md` for:
- JWT secret rotation (access/refresh token signing)
- Field-encryption key rotation (candidate PII at rest)
- API key rotation (Anthropic, OpenAI)

Example field-encryption key rotation:
```bash
# 1. Set new key + list retiring keys:
export CALIBER_FIELD_ENCRYPTION_KEY="<new-base64-key>"
export CALIBER_FIELD_ENCRYPTION_KEY_PREVIOUS="<old-key-1>,<old-key-2>"

# 2. Run reencryption (prod data):
go run ./cmd/reencrypt  # decrypts old, encrypts with new

# 3. Clear PREVIOUS after verifying all rows:
unset CALIBER_FIELD_ENCRYPTION_KEY_PREVIOUS
```

## Architecture

### Request Flow
1. Frontend (Vercel) → API REST gateway (Render gRPC + REST)
2. API orchestrates domain logic, calls LLM (Claude/Anthropic) for screening
3. Matches stored in Postgres + pgvector embeddings
4. Background jobs (Asynq + Redis) handle scoring, candidate-agent, retention

### Secrets Security (CAL-035 / CAL-117)
- API keys: Platform secret store (Render dashboard, GitHub Secrets, etc.)
- JWT secret: ≥32 bytes, unique per environment
- Field-encryption key: Base64 32-byte AES-256; REQUIRED in prod
- Never in VCS; loaded at runtime only

### Observability (CAL-130/131/132)
- Traces: OpenTelemetry (noop/stdout/OTLP)
- Metrics: Prometheus format at `/metrics`
- Logs: stdout (dev) or Loki (prod, if `CALIBER_LOKI_URL` is set)
- Request ID: `X-Request-ID` header (for tracing across services)

## Testing

### Local Integration Tests
```bash
# Requires Docker + docker-compose
docker-compose up -d
make migrate
make test         # Go tests + integration tests
cd web && npm run test:run
```

### E2E Tests (Staging)
```bash
# Against a running staging API
cd web
VITE_API_URL=https://staging-api.projectcaliber.app npm run test:e2e
```

### Load Testing (k6)
```bash
make test-load            # Fresh stack, run k6 test, tear down
make test-load-smoke      # Quick validation
```

## Troubleshooting

### API won't start
```bash
# 1. Check logs
CALIBER_LOG_LEVEL=debug go run ./cmd/api

# 2. Verify config
go run ./cmd/api 2>&1 | grep -i "validation\|missing\|required"

# 3. Common issues:
#    - DATABASE_URL not set or unreachable
#    - JWT_SECRET missing or too short
#    - FIELD_ENCRYPTION_KEY missing in prod
#    - CORS_ORIGINS empty in prod
```

### Frontend doesn't connect to API
```bash
# 1. Check VITE_API_URL in build output
cat web/.env.production

# 2. Verify CORS on backend
# The API must list the frontend origin in CALIBER_CORS_ORIGINS

# 3. Check browser Network tab for preflight (OPTIONS) response
# Should be 200 OK with Access-Control-Allow-Origin header
```

### Performance issues
```bash
# Check pgvector tuning
export CALIBER_HNSW_EF_SEARCH=100  # Improve recall at latency cost

# Check LLM concurrency
export CALIBER_LLM_MAX_CONCURRENCY=16  # Increase capacity

# Check Redis persistence
# If Redis is evicting jobs, upgrade to a paid tier with persistence
```

## References

- **Architecture**: `CLAUDE.md`, `AGENTS.md`, `docs/architecture.md`
- **Environments**: `docs/environments.md`
- **Infrastructure**: `docs/infrastructure.md`
- **Key Rotation**: `docs/runbooks/key-rotation.md`
- **Deployment**: `docs/runbooks/deployment.md`
- **Render Blueprint**: `render.yaml`
- **Vercel Config**: `web/vercel.json`
