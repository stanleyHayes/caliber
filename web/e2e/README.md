# Caliber e2e tests

End-to-end coverage for the three demo flows and the Talent Radar dashboard.

## Local setup

The suite expects the local stack to be running:

```bash
# Terminal 1 — backend
make run-api

# Terminal 2 — frontend (proxies /v1 to localhost:8080)
cd web && npm run dev
```

Seed the database with the deterministic demo dataset:

```bash
CALIBER_DATABASE_URL="postgres://caliber:caliber@localhost:5432/caliber?sslmode=disable" go run ./cmd/reseed
```

> If you are using the in-memory dev stack (`make run-api` without `CALIBER_DATABASE_URL`),
> reseeding is not required — the API seeds itself on boot.

## Run the tests

```bash
cd web
npx playwright test
```

To run a single spec:

```bash
npx playwright test e2e/radar.spec.ts
```

## Selectors

Prefer accessible selectors (`getByRole`, `getByLabel`, `getByText`) over CSS.
A minimal set of `data-testid` attributes is used only for dynamic content
(streamed interview turns, match cards) where text-based assertions would be
brittle.
