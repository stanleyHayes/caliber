# Testing standards — the test pyramid (CAL-138)

This is the canonical description of Project Caliber's test pyramid and the exact
CI enforcement for each layer. It documents what the pipeline **actually does**
today — see the "Honest caveats" section for enforcement that is conditional.

CI is defined in [`.github/workflows/ci.yml`](../.github/workflows/ci.yml). Local
equivalents live in the [`Makefile`](../Makefile). Sonar wiring is in
[`sonar-project.properties`](../sonar-project.properties).

## The four layers

| Layer | What it tests | Where it lives | Run locally | Enforced by (CI job → step) |
|---|---|---|---|---|
| **Unit** (Go domain/app) | Pure domain logic and use-cases: entities, value objects, domain services, matching/interview/candidate-agent rules — no I/O. | `internal/domain/**/*_test.go`, `internal/app/**/*_test.go` (table tests beside the code). | `make test` (or `make test-short` to skip Docker-gated tests). | Job `backend` → step **"Test with race + coverage"** (`go test -race -coverprofile=coverage.out -covermode=atomic ./...`). |
| **Integration** (adapters) | Outbound adapters against real infrastructure via [testcontainers-go](https://golang.testcontainers.org/): Postgres/pgvector repositories (roles, users, audit, erasure, backup/restore, refresh tokens), plus the migrate and queue adapters. | `internal/adapters/outbound/postgres/*_integration_test.go` (+ `docker_test.go` helper), `internal/platform/migrate/*_test.go`, `internal/adapters/outbound/queue/*_test.go`. | `make test` **with Docker running** (they self-skip if Docker is down; `make test-short` skips them by design). | Same job/step as Unit — job `backend` → **"Test with race + coverage"**. CI runs `./...` **without `-short`**, and the GitHub runner provides Docker, so these execute in CI. See caveat 1. |
| **Contract** (proto/buf) | The protobuf API contract — the source of truth in `proto/caliber/v1/*`. Lint enforces buf `STANDARD` + `breaking: FILE` rules; a freshness check proves generated Go in `internal/gen` matches the protos (and sqlc output matches its queries). | `proto/caliber/v1/*.proto`, `buf.yaml`, `buf.gen.yaml`; generated code in `internal/gen` (committed). | `make proto` (buf lint + generate), then `git diff --exit-code`. | Job `backend` → step **"buf lint"** (`buf lint`) and step **"Verify generated code is up to date"** (`buf generate` + `git diff --exit-code -- internal/gen`, then `sqlc generate` + diff). |
| **E2E** (Playwright) | Full browser flows against the real stack (Postgres + migrate + api + web via Docker Compose): auth, employer shortlisting (Flow A), AI screening interview (Flow B), candidate agent (Flow C), role gates, Talent Radar. | `web/e2e/*.spec.ts` (`web/playwright.config.ts`, `testDir: ./e2e`, `baseURL: http://localhost:5173`). | `make test-e2e` against an already-running stack, or `make test-e2e-ci` for a fresh Compose stack. | Job `e2e` (`needs: [frontend, backend]`) → step **"Install Playwright browsers"** (`npx playwright install --with-deps chromium`) + step **"Run Playwright tests"** (`npm run test:e2e`), after "Start the app stack", health waits, and "Seed the database". |

### Frontend unit layer (Vitest)

The SPA has its own unit layer alongside the Go pyramid:

- **What / where:** component, store, and SSR-entry tests in `web/src/**/*.test.ts[x]`
  (jsdom), configured in `web/vite.config.ts`.
- **Run locally:** `npm run test:run` (or `npm run test:coverage` for the gated run)
  from `web/`.
- **Enforced by:** job `frontend` → step **"Test"** (`npm run test:run`, i.e. `vitest run`)
  and step **"Enforce frontend coverage floor"** (`npm run test:coverage`, i.e.
  `vitest run --coverage`).

## The ≥80% coverage gate

The house standard is **≥80% coverage on every push** (CLAUDE.md, Quality). It is
enforced independently on each side:

**Go (backend job):**
- **"Test with race + coverage"** writes `coverage.out` (atomic mode).
- **"Enforce coverage >= 80% (app code)"** strips generated/vendor/infra packages
  (`internal/gen/`, `internal/mocks/`, `internal/platform/migrate/`,
  `internal/adapters/outbound/postgres/`, `/cmd/`, `/web/`) into `coverage.app.out`
  and fails if the total is `< 80`.
- **"Enforce per-package coverage >= 80%"** runs `scripts/check-go-coverage.sh 80`,
  which fails if **any** non-excluded package with tests is below 80% (and also fails
  the gate if `go test` itself fails to build/run).

**Frontend (frontend job):**
- **"Enforce frontend coverage floor"** runs `vitest run --coverage`; the floor is
  set in `web/vite.config.ts` under `coverage.thresholds` — `statements`, `branches`,
  `functions`, and `lines` all at **80** (v8 provider). Vitest exits non-zero if any
  threshold is unmet.

**Sonar import (backend job → "SonarQube scan"):** `sonar-project.properties` imports
Go coverage from `coverage.out` (`sonar.go.coverage.reportPaths`) and TS/JS coverage
from `web/coverage/lcov.info` (`sonar.javascript.lcov.reportPaths` /
`sonar.typescript.lcov.reportPaths`). `sonar.qualitygate.wait=true` makes the scan
block on the gate result. The gate **definition** (thresholds, new-code period) lives
in the SonarCloud UI, not in the repo — see [`docs/sonarqube.md`](sonarqube.md). The
scan step is gated on the `SONAR_TOKEN` secret (`if: ${{ env.SONAR_TOKEN != '' }}`),
so it is skipped when the token is absent (e.g. on forks). See caveat 2.

## Honest caveats — where enforcement is conditional

1. **Integration tests are Docker-conditional, not `-short`-conditional.** Each
   `*_integration_test.go` guards with **both** `if testing.Short()` **and**
   `skipIfNoDocker(t)` (`internal/adapters/outbound/postgres/docker_test.go`). CI's
   test step runs `./...` **without `-short`**, so the short guard does not fire in
   CI; the GitHub-hosted `ubuntu-latest` runner ships Docker, so `skipIfNoDocker`
   passes and the integration tests run. But the guard is real: on any runner (or
   local `make test`) **without** a reachable Docker daemon, these tests **silently
   skip** rather than fail. There is no CI assertion that Docker was present, so a
   silent skip would not, by itself, redden the pipeline. `make test-short` /
   `make test-short` intentionally skips them.

2. **Sonar's frontend LCOV import is wired.** `sonar-project.properties` reads web
   coverage from `web/coverage/lcov.info`, and `web/vite.config.ts` now includes
   `'lcov'` in `coverage.reporter` (`['text', 'json', 'html', 'lcov']`), so
   `npm run test:coverage` emits that file for the scanner. The Go side uses a native
   `coverage.out` coverprofile. Note the Sonar scan step itself is secret-gated (below),
   so a fork/PR without `SONAR_TOKEN` still relies on the in-CI 80% floors, which run
   unconditionally on both sides.

3. **The SonarQube scan is secret-gated.** With no `SONAR_TOKEN` the scan step is
   skipped entirely, so the Sonar quality gate does not run on that push. The
   in-workflow 80% gates (Go total, Go per-package, Vitest thresholds) are **not**
   secret-gated and always run.

## Quick reference — pyramid → CI job

- **Unit + Integration (Go):** job `backend`, step "Test with race + coverage" (+ two
  coverage-gate steps).
- **Frontend unit (Vitest):** job `frontend`, steps "Test" + "Enforce frontend
  coverage floor".
- **Contract (proto/buf):** job `backend`, steps "buf lint" + "Verify generated code
  is up to date".
- **E2E (Playwright):** job `e2e`, steps "Install Playwright browsers" + "Run
  Playwright tests".
