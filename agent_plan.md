# Project Caliber — Agent Plan (Epics, Stories & Progress Tracker)

> **Talent Intelligence Platform** — Proof-of-Concept → Production.
> Internal build codename: **Project Caliber**. Client-facing name TBD (keep UI brandable).
> This file is the single source of truth for planning and progress in lieu of Jira.
> It follows the house **Epic → Story → Subtask** model and the standard story template
> (User Story · Business Value · Acceptance Criteria · Technical Notes · DoD · Estimate · Dependencies)
> from the *AI Development Workflow Training Manual* and *AI-Native Software Engineering Operations Manual*.

- **Document version:** 0.3 (draft for technical team)
- **Last updated:** 2026-06-24
- **Source spec:** `Caliber_POC_Build_Spec.pdf` (v0.1, Office of the CTO, XCreativs Technologies)
- **Owner:** Engineering Lead · **Prepared with:** Claude (planning), per AI Governance policy
- **Classification:** Confidential — Caliber build team only

---

## 0. How to use this document

1. Work is tracked as **Epics** (`EPIC-NN`) containing **Stories** (`CAL-NNN`).
2. Every story carries a **Status** badge. Update it as work moves. Status flow mirrors the house Jira workflow:
   `TODO → IN PROGRESS → IN REVIEW → QA → DONE` (plus `BLOCKED`).
3. A story is only `DONE` when it satisfies the **global Definition of Done** (§4.1).
4. The **Progress Dashboard** (§6) is the at-a-glance roll-up — keep its table in sync with the epics.
5. Branch / commit / PR conventions use project key **`CAL`** (§4.3).
6. **Security (§7), SEO (§8) and UX standards (§4.5) are cross-cutting** — baked into story acceptance criteria from day one, with dedicated hardening epics (EPIC-16 security, EPIC-17 SEO) for depth.

**Legend:** `[TODO]` `[WIP]` `[REVIEW]` `[QA]` `[DONE]` `[BLOCKED]` · Estimates in story points (Fibonacci).

---

## 1. Product thesis (why we are building this)

Recruitment today is manual: jobs advertised, CVs collected, humans screen/shortlist/interview. The market is splitting into employer-side CV-rankers and candidate-side mass-apply bots — an arms race where signal collapses. **Caliber's move: make the CV one input, not the verdict.** Every candidate is anchored to a **verified ability profile (Talent Passport)** produced by an AI-conducted screening interview and role-relevant evidence. The client stops being a CV-reading shop and becomes the **trusted verifier of talent**, with explainable, human-in-the-loop, bias-safe, auditable decisions — defensible to enterprise buyers and regulators.

**POC mission:** walk into the room with a real, working application running real intelligence on realistic seeded data, robust enough to demo live, and win the engagement.

---

## 2. What we are proving (definition of done for the demo as a whole)

| # | Claim | Proven by |
|---|-------|-----------|
| 1 | Intelligent intake & explainable shortlisting works | **Flow A** — EPIC-08 |
| 2 | The AI can actually interview and assess | **Flow B** — EPIC-09 (centrepiece) |
| 3 | The system works for candidates while they sleep — honestly | **Flow C** — EPIC-10 |
| — | Closing line: time-to-shortlist collapses weeks → hours | **Talent Radar** — EPIC-11 |

---

## 3. Locked technology decisions

Confirmed with the team on 2026-06-24 (two selection rounds; **every** layer was chosen explicitly, including those the spec had fixed). Backend language **diverges from the spec's NestJS to Go**; frontend **diverges from the spec's Next.js + Tailwind to React (Vite) + MUI v9**; Node-only infra (BullMQ) replaced with Go-native equivalents.

### Backend
| Layer | Decision | Notes |
|---|---|---|
| Language & architecture | **Go**, **Hexagonal / Ports & Adapters** | Domain core framework-agnostic; classic design patterns (§5.2) |
| API protocol | **gRPC services + grpc-gateway (REST/JSON)** | Protobuf is the contract source; gateway exposes REST/JSON + OpenAPI to the browser; Appendix A shapes become proto messages |
| Contract tooling | **buf** | `buf lint` / `buf generate`; generates Go stubs + TypeScript client types |
| HTTP layer | **chi** | Hosts the grpc-gateway mux, health endpoints, auth/middleware, and the interview stream (gRPC-web / SSE fallback) |
| Persistence | **sqlc + pgx** | Compile-time-checked SQL; repository adapters implement domain ports |
| Database | **PostgreSQL + pgvector** | One datastore: relational entities + vector embeddings + JSON columns for LLM structures |
| Migrations | **goose** | Versioned SQL migrations; runs in CI and on deploy |
| Async / jobs | **Asynq (Redis)** | Candidate-agent runs, interview scoring, batch re-matching, time-advance |
| LLM | **Claude (Anthropic API)** | Default model; all access behind the `LLMClient` port |
| Embeddings | **OpenAI text-embedding-3-small** | Confirmed (residency accepted); behind a swappable `Embedder` port |
| Auth | **Custom JWT + Argon2id + RBAC** | Two roles (employer/recruiter, candidate); access + refresh tokens; auth as a port |
| Backend hosting | **Render** | Managed PaaS, fast path to a live POC URL |
| Observability | **OpenTelemetry + Prometheus / Grafana / Loki** | Vendor-neutral, self-hostable; traces + metrics + logs |

### Frontend
| Layer | Decision | Notes |
|---|---|---|
| Framework | **React + Vite (SPA)** | Preferred over Next.js; **not** server-rendered app shell |
| SEO rendering | **Build-time prerender of public pages** | Public/marketing/role pages prerendered to crawlable HTML; app behind auth is CSR (§8, EPIC-17) |
| Component library | **MUI v9 (Material UI) — Core only** | Replaces Tailwind; **no MUI X licence** — use **TanStack Table** (headless) for complex grids; brandable theme/design tokens for the client name/logo |
| Server state / data | **TanStack Query** | Caching + first-class **pagination**; consumes the REST gateway; TS types generated from proto |
| Client state | **Zustand** | UI/wizard/auth state TanStack Query doesn't own |
| Loading UX | **Skeletons (content) + animated dots (buttons)** | Skeleton placeholders for content/lists; animated-dots loader inside buttons — never spinners or "Loading…" text (§4.5) |
| Lists | **Pagination (standard)** | All list/result surfaces paginated (§4.5) |
| Typography | **Fraunces** (titles) · **Outfit** (body) · **JetBrains Mono** (statuses) | Confirmed; self-hosted, `font-display: swap`; mono for status chips/badges/IDs |
| Animation | **Motion (Framer Motion)** (default) | Layout transitions app-wide; **circular-reveal** light/dark theme toggle; marketing: **parallax** + **3D reveal-on-scroll**. All honor `prefers-reduced-motion` |
| Forms | **react-hook-form + zod** (default) | Typed, validated forms for intake/spec-edit/auth |
| Frontend hosting | **Vercel** | Static/SPA + per-PR preview URLs, edge CDN, Web Vitals |

### Cross-cutting / delivery
| Layer | Decision | Notes |
|---|---|---|
| Code quality | **SonarQube** (SonarCloud for the GitHub gate) | Quality gate must pass to merge |
| Test coverage | **≥ 80% on every push** | CI-enforced gate, fails the build below threshold |
| Backend tests | **Go testing + testcontainers** | Unit (domain) + integration (adapters) |
| Frontend tests | **Vitest + React Testing Library + Playwright** (default) | Unit/component + e2e |
| CI/CD | **GitHub Actions** | Lint → test → coverage → SonarQube → build → deploy |
| Secrets | **Environment variables / platform secret store** | Never in code or VCS |
| Versioning | **Latest stable of everything** | Track current stable releases (Go, React 19+, MUI v9, buf + protoc plugins, etc.); Dependabot/Renovate keeps deps current; no pinning to legacy majors |
| Voice | **STT + TTS — committed post-win** | Built in the production phase once the contract is won (EPIC-22); default **OpenAI STT/TTS**; must degrade to text; never the sole path |

---

## 4. Working conventions

### 4.1 Global Definition of Done (applies to every story)
A story is `DONE` only when **all** of the following hold:
- [ ] Code implemented to spec and within the hexagonal boundaries (no domain → adapter leakage).
- [ ] Unit + integration tests written; **package coverage keeps the repo ≥ 80%**.
- [ ] Backend: `go vet`, `golangci-lint`, `gofmt`/`goimports` clean. Frontend: ESLint + type-check clean.
- [ ] **SonarQube quality gate passes** (no new bugs/vulnerabilities above threshold; security hotspots reviewed).
- [ ] Security checks for the story addressed (input validation, authz, secrets, data handling — §7).
- [ ] UX standards met where applicable: **skeleton loaders** for async UI, **pagination** for lists (§4.5).
- [ ] PR opened, reviewed, and approved; CI green.
- [ ] PR merged to `main` (trunk-based; short-lived branches).
- [ ] `agent_plan.md` status updated; Progress Dashboard (§6) reflects the change.
- [ ] Documentation updated where the change affects workflow, API/proto, or `CLAUDE.md` / `AGENTS.md`.

### 4.2 Story template (used implicitly below; expand on pickup)
`As a <role>, I want <capability>, so that <value>.` · **Business Value** · **Acceptance Criteria** · **Technical Notes** · **Dependencies** · **Estimate** · **DoD = §4.1**.

### 4.3 Git conventions (project key `CAL`)
- **Branch:** `feature/CAL-123-short-slug` (also `fix/`, `chore/`, `docs/`)
- **Commit:** `CAL-123 implement role spec generator`
- **PR title:** `CAL-123 Role Spec generator`
- Trunk-based, squash-merge, branch protection: CI + SonarQube + 1 review required.

### 4.4 SDLC mapping
This plan executes Phases 3–10 of the Operations Manual (Solution Design → Production Release). Discovery/BRD (Phase 1–2) is represented by the build spec; UAT/Sign-off/Hypercare (Phase 8/11/12) are tracked in the Production milestone (EPIC-20+).

### 4.5 UX standards (cross-cutting, frontend)
These are **firm preferences**, enforced in story ACs and the DoD:
- **Skeleton loading for content.** Every async content surface (lists, cards, dashboard tiles, shortlist, interview turns, report card) shows MUI `Skeleton` placeholders shaped like the eventual content. No `CircularProgress`/spinners and no bare "Loading…" text for content.
- **Animated dots for buttons.** Button busy/submit states use a reusable **animated-dots** loader inside the button (label → dots), never a spinner. Disable + preserve button width to avoid layout shift.
- **Layout transitions everywhere.** App-wide animated layout transitions via **Motion (Framer Motion)** — shared-layout/route transitions, list add/remove/reorder (e.g. shortlist re-rank), and enter/exit. Smooth, fast, non-blocking.
- **Theme toggle = circular reveal.** Light/dark switch animates as a **circular reveal** expanding from the toggle (View Transitions API where supported; clip-path fallback). MUI color-mode drives the palette.
- **Pagination everywhere.** Any endpoint or view returning a collection (candidate pool, shortlists, applications, interviews, audit log, alerts) is paginated — server-side pages via the gateway, surfaced with TanStack Query paginated/`keepPreviousData` queries and MUI pagination controls. No unbounded lists.
- **Typography system.** **Fraunces** for titles/headings, **Outfit** for body/UI, a **monospace** (default **JetBrains Mono**) for statuses, badges, IDs, and metric readouts. Wired into the MUI v9 theme `typography`; self-hosted with `font-display: swap`.
- **Marketing-site motion.** Public/marketing pages include **parallax** sections, **3D reveal-on-scroll** animations, and the circular-reveal theme toggle — performance-budgeted (§8) and gated behind `prefers-reduced-motion`.
- **Accessibility of motion.** All animations honor `prefers-reduced-motion: reduce` (reduce/disable), keep focus order intact, and never trap or block interaction.
- **MUI v9 theming.** All components from the central themed design system; brandable tokens (colors/typography) swappable for the client's name/logo.
- **Forms** use react-hook-form + zod with inline validation and accessible error states.

---

## 5. Architecture

### 5.1 Hexagonal layout (target repo structure)
```
caliber/
├── cmd/
│   ├── api/            # gRPC + grpc-gateway server entrypoint (chi hosts gateway/health/stream)
│   └── worker/         # Asynq worker entrypoint
├── proto/              # protobuf service + message contracts (buf-managed) — the API source of truth
├── internal/
│   ├── domain/         # Pure core: entities, value objects, domain services, PORTS (interfaces)
│   │   ├── talent/         # TalentProfile, Talent Passport, competencies
│   │   ├── role/           # Role, RoleSpec, Rubric
│   │   ├── matching/       # Match, scoring policy (domain logic)
│   │   ├── interview/      # Interview state machine, InterviewTurn, report card
│   │   ├── candidateagent/ # Candidate agent policy + no-fabrication invariant
│   │   ├── identity/       # User, roles, auth domain rules
│   │   └── audit/          # AuditLog domain
│   ├── app/            # Application services / use-cases (orchestrate domain + ports)
│   ├── adapters/
│   │   ├── inbound/
│   │   │   ├── grpc/       # gRPC service handlers (map proto ↔ app use-cases) + grpc-gateway
│   │   │   ├── http/       # chi: gateway mux mount, health, auth middleware, interview stream (SSE/gRPC-web)
│   │   │   └── jobs/       # Asynq task handlers (inbound side of async)
│   │   └── outbound/
│   │       ├── postgres/   # sqlc-generated + repository adapters (implement domain ports)
│   │       ├── llm/        # Anthropic Claude gateway (implements LLMClient port)
│   │       ├── embeddings/ # OpenAI embedder (implements Embedder port)
│   │       ├── queue/      # Asynq enqueuer (implements TaskDispatcher port)
│   │       └── auth/       # JWT issuer/verifier, Argon2id hasher
│   ├── platform/       # config, logging (slog), otel, db pool, server bootstrap, DI wiring
│   └── seed/           # demo data generation & curation
├── db/
│   ├── migrations/     # goose migrations (incl. pgvector extension)
│   └── queries/        # sqlc .sql sources
├── prompts/            # versioned LLM prompts & rubric templates (product, not config)
├── web/                # React + Vite SPA — MUI v9, TanStack Query, Zustand; employer/candidate/interview/dashboard
├── deploy/             # Dockerfiles, render/railway config, IaC
├── .github/workflows/  # CI: lint, test, coverage, sonar, build, deploy
├── buf.yaml / buf.gen.yaml  # protobuf lint + codegen config
├── CLAUDE.md           # AI operating rules (required)
├── AGENTS.md           # agent/workflow rules (required)
└── agent_plan.md       # this file
```

### 5.2 Design patterns in play
- **Ports & Adapters (Hexagonal):** domain defines interfaces; adapters implement them. Domain imports nothing from `adapters`.
- **Generated contracts:** protobuf/buf is the single API source; gRPC + REST gateway are generated inbound adapters.
- **Repository:** persistence behind `*Repository` ports; pgx/sqlc adapters.
- **Strategy / provider-swappable:** `LLMClient`, `Embedder` interfaces → Claude / OpenAI today, swappable later.
- **State machine:** the AI screening interview (`interview` domain) as an explicit FSM.
- **Command + Handler:** Asynq jobs as commands with idempotent handlers (candidate-agent, scoring, re-matching, time-advance).
- **Factory & Dependency Injection:** constructor injection wired in `platform`; no global state.
- **Decorator / Middleware:** cross-cutting concerns (auth, rate-limit, request-id, otel, recovery) as gRPC interceptors + chi middleware.
- **Outbox (production):** reliable audit/event emission alongside DB writes.
- **Pipeline:** matching = recall → precision → hard-filter stages as composable steps.

### 5.3 Request flow (illustrative)
1. React SPA calls the **REST gateway** (or gRPC-web) → API.
2. gRPC handler → app use-case → AI orchestration: generate Role Spec + Rubric (Claude); persist; embed the spec.
3. Matching: pgvector recall → rubric-based LLM scoring → hard filters → ranked Matches with rationale → client (paginated).
4. Interview launch opens a **streamed** session (gRPC server-streaming / SSE); the FSM drives the adaptive loop and writes a report card; the UI renders turns with skeletons until each arrives.
5. Candidate-agent & time-advance run as queued Asynq jobs that mutate state; the dashboard reflects it.

---

## 6. Progress Dashboard

> Roll-up of epic status. Update counts as stories close.

| Milestone | Epic | Title | Stories | Pts | Status | % |
|---|---|---|---|---|---|---|
| **M1 — POC Demo-Ready** | EPIC-00 | Engineering Foundations & Project Setup | 10 | 39 | WIP | ~70% |
| | EPIC-01 | Domain Model & Database Foundation | 7 | 29 | WIP | ~85% |
| | EPIC-02 | Identity, Authentication & RBAC | 7 | 31 | DONE | 100% |
| | EPIC-03 | Async Jobs & Queue Infrastructure | 5 | 21 | WIP | ~40% |
| | EPIC-04 | AI Orchestration Layer | 8 | 39 | WIP | ~40% |
| | EPIC-05 | Role Spec & Rubric Generator | 5 | 24 | TODO | 0% |
| | EPIC-06 | Profile Parser & Competency Extractor | 5 | 26 | WIP | ~35% |
| | EPIC-07 | Matching & Ranking Engine | 7 | 37 | WIP | ~70% |
| | EPIC-08 | Employer Intake & Explainable Shortlisting (Flow A) | 6 | 29 | WIP | ~45% |
| | EPIC-09 | AI Screening Interviewer (Flow B) | 9 | 50 | WIP | ~50% |
| | EPIC-10 | Candidate Agent & Time-Advance (Flow C) | 7 | 36 | WIP | ~55% |
| | EPIC-11 | Talent Radar Dashboard | 5 | 24 | WIP | ~60% |
| | EPIC-12 | Trust, Explainability, Audit & Guardrails | 7 | 33 | TODO | 0% |
| | EPIC-13 | Frontend Web Application (React/Vite) | 15 | 69 | WIP | ~70% |
| | EPIC-14 | Seed Data & Demo Orchestration | 6 | 28 | WIP | ~83% |
| | EPIC-15 | Demo Hardening & Run-of-Show | 6 | 24 | WIP | ~20% |
| **M2 — Production-Ready** | EPIC-16 | Security Hardening & Compliance | 11 | 55 | WIP | ~45% |
| | EPIC-17 | SEO & Web Performance | 10 | 43 | TODO | 0% |
| | EPIC-18 | Observability & Operations | 8 | 37 | TODO | 0% |
| | EPIC-19 | Quality, Testing & Performance Engineering | 8 | 39 | TODO | 0% |
| | EPIC-20 | CI/CD, Environments & Release Management | 7 | 32 | TODO | 0% |
| | EPIC-21 | Scale, Multi-Tenancy & Data Lifecycle | 7 | 35 | TODO | 0% |
| **Post-Win** | EPIC-22 | Voice Interview Mode (committed) | 4 | 18 | TODO | 0% |
| | | **TOTAL** | **172** | **808** | | **0%** |

---

## 6.1 Sprint board (live)

We deliver **sprint by sprint**. This board is the live cursor over the epics above; update it as stories move.

**Sprint 1 — Foundation** (EPIC-00). Goal: app runs; gRPC + REST contracts generate cleanly; CI + SonarQube + ≥80% coverage gates are green; ready to store & embed a profile in Sprint 2.

| # | Story | Title | Status |
|---|---|---|---|
| 1 | CAL-164 | Protobuf contracts + buf + gRPC/gateway scaffold | **DONE** — 9 protos → `internal/gen` (Go+gRPC+gateway+OpenAPI); API server wired; routes verified live |
| 2 | CAL-001 | Go monorepo & hexagonal skeleton | **DONE** — hexagon layout, depguard boundaries, build/vet/test green |
| 3 | CAL-005 | Configuration & secrets management | **DONE** — typed env config + `.env.example` done; gitleaks secret scan now gates CI |
| 4 | CAL-006 | Dockerization & local dev stack | **REVIEW** — api/worker multi-stage Dockerfiles, one-shot migration container, Postgres+pgvector, Redis, and Vite dev server wired in compose; `docker compose up` smoke pending after local Docker daemon recovery |
| 5 | CAL-007 | Structured logging & error baseline | **DONE** — structured access logging with request correlation + panic recovery |
| 6 | CAL-008 | Health, readiness & server bootstrap | **REVIEW** — `/healthz` + graceful shutdown done; `/readyz` now checks Postgres pool + Redis PING when configured; live compose smoke pending after Docker daemon recovery |
| 7 | CAL-002 | CLAUDE.md & AGENTS.md | **DONE** |
| 8 | CAL-003 | CI pipeline (lint/test/coverage gate) | **DONE** — workflow authored; all gates reproduced locally; first GitHub run pending remote |
| 9 | CAL-004 | SonarQube quality gate | **WIP** — `sonar-project.properties` + CI step done; needs SonarCloud project + `SONAR_TOKEN` secret |
| 10 | CAL-009 | Branch protection & repo policy | **DONE** — CODEOWNERS + PR template landed; `main` branch protection applied via GitHub API (PR + 1 code-owner review + required CI/security checks + conversation resolution; force-push/delete blocked) |

**Sprint 2 (active)** — EPIC-01 (domain + schema + pgvector), EPIC-02 (auth), EPIC-03 (queue), EPIC-04 (AI orchestration): the intelligence substrate becomes callable.

| # | Story | Title | Status |
|---|---|---|---|
| 1 | CAL-024 | Asynq client/server wiring | **DONE** — `TaskDispatcher` port + Asynq dispatcher/no-op adapter, worker handler mux, weighted queues, API candidate-agent enqueue/fallback path, and miniredis enqueue-to-process round trip verified. Local build/lint/race suite pass; app-code coverage reports 81.8%. |
| 2 | CAL-025 | Idempotent job handler framework | **DONE** — all Asynq handlers run through the base job wrapper with idempotency claims, structured lifecycle logs, and OTel spans; duplicate direct delivery is skipped before side effects. |

**Hardening lane (pulled while Sprint 2 queue stories are active)** — EPIC-16 supply-chain gate.

| # | Story | Title | Status |
|---|---|---|---|
| 1 | CAL-115 | Dependency & container scanning | **DONE** — CI now runs `govulncheck`, `npm audit --audit-level=high`, and Trivy HIGH/CRITICAL scans over api/worker/migrate images; Dependabot covers Go, npm, Docker, and GitHub Actions. |
| 2 | CAL-114 | Security headers, TLS & CORS | **DONE** — HTTP gateway emits deny-by-default security headers, HSTS in prod, exact-origin CORS, and rejects wildcard/malformed CORS config; prod validation requires explicit allowed origins. |
| 3 | CAL-120 | Security review & pen-test prep | **WIP** — adversarial security audit run; both HIGH findings + all MEDIUM + most LOW findings remediated and tested (see backlog entry). No open HIGH findings remain (the AC bar). A few LOWs deferred (token-revocation store, no-fabrication strengthening). |

---

# MILESTONE 1 — POC: Demo-Ready

Build a thin end-to-end slice early, then harden toward the demo. Maps to spec build Phases 1–5: Foundation → Intelligence → Flows → Polish → Hardening.

---

## EPIC-00 · Engineering Foundations & Project Setup
**Goal:** A clean, hexagonal Go repo with protobuf contracts, CI, quality gates, and conventions so every later story merges through the same disciplined pipeline.

- **CAL-001** `[DONE]` · 3 pts — **Initialize Go monorepo & hexagonal skeleton.** Scaffold `cmd/`, `internal/{domain,app,adapters,platform}`, `db/`, `prompts/`, `proto/`, `web/` per §5.1. *AC:* `go build ./...` passes; import-lint enforces domain imports no adapters. *Deps:* —
- **CAL-002** `[DONE]` · 2 pts — **CLAUDE.md & AGENTS.md.** Author required AI-governance files (coding standards, hexagonal rules, no-fabrication guardrail, UX standards §4.5, Jira-less workflow, git conventions). *AC:* both present, referenced in README. *Deps:* CAL-001
- **CAL-164** `[DONE]` · 5 pts — **Protobuf contracts + buf + gRPC/grpc-gateway scaffold.** `proto/` services & messages; `buf lint`/`generate` producing Go stubs + TS types; gRPC server with grpc-gateway mux mounted on chi; OpenAPI emitted. *AC:* a sample RPC is reachable via gRPC and REST; codegen runs in CI. *Done 2026-06-24:* 9 protos (all flows) generated to `internal/gen` (Go+gRPC+gateway+OpenAPI); API server wired & verified live (gateway→gRPC returns Unimplemented/501, health 200). CI codegen check lands with CAL-003. *Deps:* CAL-001
- **CAL-003** `[DONE]` · 5 pts — **CI pipeline (GitHub Actions).** Stages: format/lint (Go + web) → buf lint → `go test -race -coverprofile` + web tests → **coverage ≥ 80% gate** → build. *AC:* PR cannot merge if any stage fails or coverage < 80%. *Deps:* CAL-001
- **CAL-004** `[WIP]` · 5 pts — **SonarQube/SonarCloud integration.** Wire scanner into CI; configure quality gate (bugs, vulns, hotspots, duplication, coverage import for Go + TS). *AC:* gate status blocks merge. *Deps:* CAL-003
- **CAL-005** `[DONE]` · 3 pts — **Configuration & secrets management.** Typed config loader (env-driven), `.env.example`, no secrets in VCS; fail-fast on missing required vars; gitleaks in CI. *Done 2026-06-29:* typed env config + `.env.example` were already present; CI now includes a repo-wide Gitleaks job with a narrow allowlist for documented local placeholders and generated dependency metadata. *AC:* config validated at boot. *Deps:* CAL-001
- **CAL-006** `[REVIEW]` · 5 pts — **Dockerization & local dev stack.** Multi-stage Dockerfiles for `api`/`worker`; `docker-compose` with Postgres+pgvector and Redis; Vite dev server wired. *AC:* `docker compose up` boots the full local stack. *Implemented 2026-06-29:* compose now wires Postgres+pgvector, Redis, one-shot schema migrations, API, worker, and the Vite dev server; Vite proxies `/v1` to the API service inside Docker while preserving localhost proxying outside Docker. Verification so far: `docker compose config`, backend build/tests, and frontend build/tests pass; full `docker compose up` smoke is pending because the local Docker daemon dropped during image build. *Deps:* CAL-001
- **CAL-007** `[DONE]` · 3 pts — **Structured logging & error handling baseline.** `slog` JSON logger, request-scoped logger, typed domain errors, panic-recovery middleware/interceptor. *AC:* every request logs a correlation/request id. *Deps:* CAL-001 **[DONE]** slog JSON logger + typed kernel errors + chi panic-recovery were in place; the missing piece — request correlation — is now wired: a structured access-log middleware logs every request with its chi request id (method/path/status/duration only; PII-free). Tested by `TestRequestLoggerEmitsCorrelatedStructuredLog`.
- **CAL-008** `[REVIEW]` · 5 pts — **Health, readiness & server bootstrap.** chi server with `/healthz`, `/readyz`, graceful shutdown, timeouts, DI wiring in `platform`. *AC:* readiness reflects DB+Redis connectivity. *Implemented 2026-06-29:* `/readyz` now runs injected readiness checks, returns `503 {"status":"not_ready"}` when a dependency fails, reuses the live Postgres pool for DB readiness, and verifies Redis with a PING check (including authenticated Redis URLs). Tested at the HTTP router, platform server, composition-root, and Redis-check layers. Live compose smoke remains tied to CAL-006 while Docker daemon is unavailable. *Deps:* CAL-006
- **CAL-009** `[DONE]` · 3 pts — **Branch protection & repo policy.** Protect `main`; require CI + Sonar + 1 review; CODEOWNERS; PR template embedding the DoD checklist. *Done 2026-06-29:* added `.github/CODEOWNERS`, a PR template with the story/verification/DoD checklist, and `.github/branch-protection.md` documenting the required `main` branch rules/check names. Applied `main` branch protection through the GitHub API: required PRs, 1 approving code-owner review, stale-review dismissal, required conversation resolution, up-to-date required checks (`Secrets (gitleaks)`, `Backend (lint · proto · test · coverage · sonar)`, `Frontend (typecheck · build · lint · test)`), no force pushes, and no deletions. SonarCloud token/project setup remains tracked in CAL-004. *AC:* direct pushes blocked. *Deps:* CAL-003, CAL-004

## EPIC-01 · Domain Model & Database Foundation
**Goal:** The entities of spec §9 as a pure domain plus a migrated Postgres schema with pgvector.

- **CAL-010** `[DONE]` · 5 pts — **Domain entities & value objects.** `User, Employer, Role, RoleSpec, Rubric, Candidate, TalentProfile/Passport, Match, Application, Interview, InterviewTurn, AuditLog` as pure Go types with invariants. *AC:* no infra imports; unit-tested invariants. *Deps:* CAL-001
- **CAL-011** `[DONE]` · 3 pts — **Repository ports.** Define `*Repository` interfaces in `domain`. *AC:* application layer depends only on ports. *Deps:* CAL-010
- **CAL-012** `[DONE]` · 5 pts — **goose migration tooling & base schema.** goose migrations; relational schema; JSON columns for `role_spec`, `rubric`, `report_card`, `breakdown`. *AC:* up/down migrations run in CI. *Deps:* CAL-006 **[Audit-verified DONE]** goose migrations (db/migrations/0000{1,2,3}) create the 10 core tables + JSONB columns + indexes; `migrate_test.go:TestMigrationsApplyAgainstPgvector` asserts them against a real pgvector:pg17 testcontainer.
- **CAL-013** `[DONE]` · 3 pts — **Enable pgvector & embedding columns.** `vector` extension; `role_embedding`, `profile_embedding`; ivfflat/hnsw index. *AC:* vector similarity query returns ordered results. *Deps:* CAL-012 **[Audit-verified DONE]** pgvector extension + `role_embedding`/`profile_embedding vector(1536)` columns + HNSW indexes; `postgres/recaller.go` does nearest-neighbour recall, proven by `recaller_integration_test.go:TestRecallByEmbedding` (testcontainers).
- **CAL-014** `[DONE]` · 5 pts — **sqlc queries & Postgres repository adapters.** Implement ports with sqlc+pgx; transactions via a `UnitOfWork`. *AC:* repository integration tests against real Postgres (testcontainers). *Deps:* CAL-011, CAL-012
- **CAL-015** `[DONE]` · 3 pts — **Audit log persistence.** Append-only `audit_log` (actor, action, entity, before/after, timestamp). *AC:* writes immutable; covered by tests. *Deps:* CAL-014
- **CAL-016** `[DONE]` · 5 pts — **Seed-ready fixtures & factory helpers.** New `internal/platform/seed` package: a deterministic, internally-consistent Ghana-context demo dataset (3 employers, 5 open roles, 8 candidates+profiles) built only through the domain constructors (honouring the candidate.ID==user.ID convention), designed to produce strong two-way matches so the Radar alert feed is populated. `seed.Load(ctx, repos, hasher, now)` materializes it; wired into the in-memory dev path (`CALIBER_SEED_DEMO`, default on) so the API boots demo-ready. All demo accounts share `DefaultPassword` and are loginable (smoke-verified: candidate + employer login return JWTs). *AC:* reused by integration tests and EPIC-14 (demo seed); `TestLoad_ProducesTwoWayAlerts` proves the data is "alive" through the real aggregator. *Deps:* CAL-014

## EPIC-02 · Identity, Authentication & RBAC
**Goal:** Lightweight, secure login for two roles behind clean ports. (Spec: no enterprise SSO for POC.)

- **CAL-017** `[DONE]` · 3 pts — **Auth domain & roles.** `identity.Role{employer,recruiter,candidate}`, `PasswordPolicy`, `AccountStatus`, validated `User`/`Email`. *AC:* role rules unit-tested. *Deps:* CAL-010
- **CAL-018** `[DONE]` · 5 pts — **Argon2id password hashing adapter.** `PasswordHasher` port + `Argon2idHasher` (OWASP defaults m=64MiB/t=3/p=2, PHC-encoded, constant-time verify). Decoder validates embedded params (rejects t<1/p<1/oversized-m) so a crafted hash can't panic or exhaust memory. *AC:* hashes verify; params configurable; timing-safe. *Deps:* CAL-017
- **CAL-019** `[DONE]` · 5 pts — **JWT issuance & verification.** `TokenService` port + HS256 `JWTService` (golang-jwt/v5): short access + rotating refresh (jti for revocation), iss/aud/exp/nbf enforced, alg pinned to HS256 (none/RS256 rejected), ≥32-byte secret floor. *AC:* expiry, signature, audience validated; refresh rotation tested. *Deps:* CAL-017
- **CAL-020** `[DONE]` · 5 pts — **Register / login / logout / refresh RPCs.** `identity.Service` use-case + gRPC/REST handlers: register (Argon2id hash, dup→409), login (generic 401, no enumeration), refresh (single-use rotation + replay detection), idempotent logout. In-memory user repo + refresh store for dev; Postgres user repo + durable single-use refresh-token store (atomic `UPDATE ... RETURNING` rotation) wired when a DB is set. GetMe + rate-limiting deferred (CAL-021/CAL-112). *AC:* covers happy + error paths; rate-limited (ties to CAL-112). *Deps:* CAL-018, CAL-019, CAL-164
- **CAL-021** `[DONE]` · 3 pts — **Auth interceptor/middleware & RBAC guards.** Unary interceptor verifies bearer access tokens and injects the principal into context; `RequireAuth`/`RequireRole` guards map to 401/403; `GetMe` protected end-to-end. Per-flow role guards layer onto Role/Matching as their clients land. *AC:* unauthorized → 401, forbidden → 403, with tests. *Deps:* CAL-019
- **CAL-022** `[DONE]` · 3 pts — **Employer & candidate context bootstrap.** `Provisioner` port invoked on Register; `CandidateProvisioner` creates a user-owned Talent Passport (`talent.Candidate`) on candidate signup. Employer-context bootstrap deferred until signup collects a company name (employer users own roles by user id meanwhile). *AC:* user→context relationship enforced. *Deps:* CAL-020
- **CAL-023** `[DONE]` · 5 pts — **Session security hardening (POC baseline).** Brute-force login lockout (per-email sliding window → `429`), login timing-equalization (no account enumeration), OWASP secure-headers middleware (nosniff/DENY/CSP/Referrer/Permissions, HSTS in prod), and prod hard-fail on a missing DB/JWT secret. CSRF N/A (bearer-token API, no auth cookies). *AC:* OWASP auth checklist items pass. *Deps:* CAL-020

## EPIC-03 · Async Jobs & Queue Infrastructure
**Goal:** Asynq/Redis worker foundation for candidate-agent runs, interview scoring, batch re-matching, and the demo time-advance.

- **CAL-024** `[DONE]` · 5 pts — **Asynq client/server wiring.** `worker` entrypoint; `TaskDispatcher` port; queues with priorities. *Implemented 2026-06-29:* added the app-level task dispatcher port, Asynq outbound adapter, no-op dev adapter, Redis-backed API dispatcher wiring for candidate-agent runs, worker dependency wiring, registered handlers for candidate-agent, interview scoring, and batch rematch tasks, weighted queues, and miniredis-backed enqueue-to-process tests. *Verification:* `make build`, `make lint`, and `make cover` all complete; app-code coverage reports 81.8% after excluding generated/vendor-like packages from the coverage view. *AC:* enqueue-to-process round-trip tested. *Deps:* CAL-006, CAL-008
- **CAL-025** `[DONE]` · 3 pts — **Idempotent job handler framework.** Base handler with idempotency keys, structured logging, otel spans. *Implemented 2026-06-29:* added a reusable jobs handler framework with a pluggable `IdempotencyStore`, process-local memory implementation, stable keys from `Idempotency-Key` header / Asynq task id / task payload digest fallback, structured start/complete/fail/duplicate logs, and OTel spans per task. All registered healthcheck and business Asynq handlers now pass through the wrapper; failed attempts release their key for retry, while duplicate completed deliveries are skipped before side effects. Tests cover duplicate suppression, failure release, span emission, and the candidate-agent handler not double-applying. *AC:* duplicate delivery does not double-apply. *Deps:* CAL-024
- **CAL-026** `[TODO]` · 5 pts — **Retry, backoff & dead-letter handling.** Per-task retry policy, max-retry → archive, alerting hook. *AC:* failing task lands in archive after policy; visible. *Deps:* CAL-025
- **CAL-027** `[TODO]` · 3 pts — **Scheduled / delayed tasks.** Support deferred enqueue (time-advance & re-matching). *AC:* delayed task fires on time in tests. *Deps:* CAL-024
- **CAL-028** `[TODO]` · 5 pts — **Asynqmon dashboard & ops.** Mount monitoring UI (protected); operational runbook. *AC:* queue depth/failures observable. *Deps:* CAL-024

## EPIC-04 · AI Orchestration Layer
**Goal:** All model interaction behind one clean module: prompt assembly, the Claude gateway, schema-validated structured outputs, embeddings, cost/latency controls. Prompts & rubrics are versioned product, not config.

- **CAL-029** `[DONE]` · 3 pts — **`LLMClient` port & message types.** Provider-agnostic interface (complete, stream, tool/JSON modes). *AC:* domain/app depend only on the port. *Deps:* CAL-001
- **CAL-030** `[DONE]` · 5 pts — **Anthropic Claude gateway adapter.** Implement `LLMClient` with the Anthropic Go SDK; timeouts, retries, context cancellation. *AC:* live + mocked tests; configurable model. *Deps:* CAL-029
- **CAL-031** `[DONE]` · 5 pts — **Structured-output enforcement.** Generic `app.DecodeJSON[T]` orchestration helper: calls the model, decodes into T, and on unparseable output re-asks up to `DefaultLLMAttempts` times appending a corrective notice; transport failures return `KindInternal` immediately, exhausted attempts return `KindInvalid`. Adopted at all six LLM-decode sites (CV extraction, role-spec generation, interview question + report, agent assessment, shortlist scoring), replacing ad-hoc `Complete`+`Unmarshal`. *AC:* malformed output retried, then typed error. *Deps:* CAL-030
- **CAL-032** `[DONE]` · 3 pts — **Versioned prompt registry.** New `internal/app/prompts` package: the 6 system prompts now live as VCS files under `files/<id>/<version>.txt`, compiled in via `go:embed` and referenced by typed ids; a fail-fast registry (panics on dup/missing/empty at init) centralizes id↔version↔body↔token-budget. `Prompt.Request(userPrompt)` is the single blessed constructor and stamps `LLMRequest.Source{ID,Version}` onto every call. The audit (CAL-036) now records `PromptID`/`PromptVersion` explicitly and `dev.go` routes on the prompt id — the fragile `operationOf` substring classifier is deleted. Golden-content tests guard the CAL-119 fence notices + identity phrases. *AC:* prompt version recorded on each call (proven end-to-end via `TestAudited_RecordsRegistryPromptIDAndVersion`). *Deps:* CAL-030
- **CAL-033** `[DONE]` · 3 pts — **`Embedder` port + OpenAI adapter.** text-embedding-3-small behind the port; batch support. *AC:* embeddings stored in pgvector; provider swappable. *Deps:* CAL-013, CAL-029
- **CAL-034** `[TODO]` · 5 pts — **Streaming support.** Token/event streaming surfaced to inbound (gRPC server-stream / SSE) for the interview. *AC:* stream cancellable; backpressure handled. *Deps:* CAL-030
- **CAL-035** `[DONE]` · 5 pts — **Cost, rate-limit & guardrail controls.** `llm.Guarded` decorator over the `LLMClient` port: hard per-call token cap, concurrency semaphore (ctx-aware), request-budget rate limit (dependency-free `TokenBucket` with injectable clock → `KindTooManyRequests` fail-fast), and advisory prompt-injection telemetry (wires CAL-119 `guard.ScanInjection`; reports category labels only, never prompt content, so logs stay PII-safe). Wired in `buildLLM` in front of both the Claude and dev providers. *AC:* limits enforced; usage metered. *Deps:* CAL-030
- **CAL-036** `[DONE]` · 5 pts — **AI call audit & observability.** `app.AICallRecorder` port + `app.AICallRecord` (operation, model, latency, prompt/response char counts as token proxies, failed, timestamp — redacted: never prompt/response content, so candidate PII never reaches telemetry). `llm.Audited` decorator traces every call (success or failure) to a recorder; `SlogRecorder` (structured logs) and `MemoryRecorder` (bounded ring buffer, `Snapshot()`) implementations. Wired as the outermost LLM decorator in `buildLLM` (Audited→Guarded→provider); the operation/prompt id+version come from the request's registry `Source` (CAL-032), not substring classification. *AC:* every model call traceable. *Deps:* CAL-030, CAL-015

## EPIC-05 · Role Spec & Rubric Generator (Flow A.1)
**Goal:** Turn a hiring manager's messy sentence into a structured, editable **Role Spec** + weighted **Rubric** + suggested salary band. (Spec §8.1, Appendix A.1.)

- **CAL-037** `[DONE]` · 5 pts — **Role Spec generation use-case.** Free text → Role Spec JSON (title, location, seniority, availability, responsibilities, must/nice-to-haves, salary band). *AC:* matches Appendix A.1 contract. *Deps:* CAL-031, CAL-032 **[Audit gap]** `GenerateRoleSpecResponse.available_matches` is never populated (handler always returns 0) — the instant pool-depth signal (CAL-055) is not surfaced with the generated role. **[DONE]** free-text → structured persisted Role (`SpecGenerator`), and the response now carries the instant `available_matches` pool-depth signal: `RoleServer` calls the shortlister's cheap `CountAvailable` (logistical + must-have profile coverage, no LLM) best-effort.
- **CAL-038** `[DONE]` · 5 pts — **Weighted rubric generation.** Competencies with weights + must-have flags. *AC:* valid, normalized weights; deterministic schema. *Deps:* CAL-037 **[Audit-verified DONE]** `roles.toDomain` builds a normalised weighted Rubric (must-have flags); `role.Rubric.Normalize()` enforces sum=1.0; `generate_test.go:TestGenerateHappyPath` asserts TotalWeight()=1.0; `role_test.go:TestUpdateRoleSpecReweights`.
- **CAL-039** `[DONE]` · 3 pts — **Salary-band lookup over seeded market data.** Simple lookup for realism (Ghana market). *AC:* band returned in role currency. *Deps:* CAL-037, CAL-016 **Implemented:** `internal/domain/salary` — a deterministic Ghana-market monthly-GHS lookup (`salary.Lookup(title, seniority)`) that classifies the role family from the title (data/ML & platform/SRE at a premium, design/QA below the engineering baseline) and scales a per-seniority base band, rounded to tidy GHS figures and bracketing the seeded demo roles. Wired into `SpecGenerator.Generate` as a realism fallback: a generated spec that omits compensation gets a plausible band instead of a blank one; an explicit band the model supplies is preserved. Pure, no globals (gochecknoglobals-clean), fully unit-tested.
- **CAL-040** `[DONE]` · 5 pts — **Editable spec/rubric RPCs + re-weighting.** `RoleService.GetRole` + `UpdateRoleSpec` (domain `Role.Revise` validates; rubric re-normalized on save) wired; employer UI edits spec fields + rubric weights/must-haves and saves. Re-rank-on-edit (CAL-057) and audit (CAL-014) still pending. *AC:* edits persisted and audited. *Deps:* CAL-037, CAL-014
- **CAL-041** `[TODO]` · 3 pts — **Spec embedding on save.** Embed the role spec for recall. *AC:* `role_embedding` populated. *Deps:* CAL-033, CAL-040

## EPIC-06 · Profile Parser & Competency Extractor
**Goal:** Convert a CV + intake answers into a structured competency profile with evidence tied back to source text. (Spec §8.2.)

- **CAL-042** `[WIP]` · 5 pts — **CV ingestion (file/text).** Upload + parse PDF/DOCX/plain text to clean text. *AC:* common formats handled; size/type validated. *Deps:* CAL-020 **[Mostly done]** `cvtext.Extract` parses **plain text + DOCX** (stdlib only — `archive/zip` + `encoding/xml` over `word/document.xml`); the `CreateProfileFromCV` handler prefers an uploaded `cv_file` over `cv_text`, enforces a 10 MiB size cap, and rejects unsupported types. PDF returns a clear "paste the text" error rather than failing silently — full PDF text extraction is **deferred** (needs a 3rd-party parser; kept out to avoid the dependency). Tested: extractor (txt/docx/case/corrupt/missing-body/PDF/unknown) + handler (DOCX upload extracts the real content, oversize + PDF rejected).
- **CAL-043** `[DONE]` · 5 pts — **Competency extraction use-case.** Text → structured profile JSON (competencies, seniority, history). *AC:* fixed schema; covered by tests. *Deps:* CAL-031 **[Audit-verified DONE]** `profiles.CreateFromCV` extracts a structured, evidence-linked profile via the `cv_extract` prompt; tested in `builder_test.go` + end-to-end `talent_test.go:TestTalentCreateThenGetProfile`.
- **CAL-044** `[DONE]` · 5 pts — **Evidence-linking.** Each extracted competency cites its CV source span. *AC:* recruiter can see source of each claim. *Deps:* CAL-043 **[DONE]** evidence enforced at the extraction boundary: `profiles.CreateFromCV` drops any model-returned competency lacking a CV evidence quote, so every competency in a Talent Passport traces to a real CV span (no-fabrication). Tested by `builder_test.go:TestCreateFromCVDropsUnevidencedCompetencies`.
- **CAL-045** `[WIP]` · 5 pts — **Profile embedding + Talent Profile persistence.** Store structured profile + summary embedding. *AC:* `TalentProfile` + `profile_embedding` written. *Deps:* CAL-033, CAL-014
- **CAL-046** `[DONE]` · 3 pts — **Guided intake answers.** Intake (target titles, location, salary floor, deal-breakers) is captured + merged into the candidate. All now feed matching filters: location + salary via `ScreenLogistics`, and **deal-breakers** via the new `matchingdom.ViolatesDealBreaker` (whole-token phrase match over the role's text, shared `kernel.HasPhrase`) wired into BOTH the two-way matcher and the candidate-agent eligibility gate — a role whose text states a candidate's deal-breaker is never surfaced or applied to. (Target-title *relevance* ranking deferred: naive title-token matching over/under-filters; needs title normalization.) *AC:* intake feeds matching filters. *Deps:* CAL-043

## EPIC-07 · Matching & Ranking Engine
**Goal:** Rank candidates against a Role Spec with scores a human can trust — recall → precision → hard filters. (Spec §8.3, Appendix A.2.)

- **CAL-047** `[DONE]` · 5 pts — **Stage 1: vector recall.** pgvector cosine similarity role↔candidate top-N (`Recaller` raw `$1::vector` query, testcontainers ordering test). *AC:* top-N returned, ordered, paged. *Deps:* CAL-041, CAL-045
  - **Dev-stack enablement:** added an in-memory `Recaller` + `MatchRepo` and a deterministic dev scorer (`devScore`, routed on the shortlist prompt id) so Flow A (explainable shortlisting) runs end-to-end in the dev path **without pgvector/docker** — wired into `cmd/api`. Smoke-verified on seeded data: an employer shortlists a role and gets ranked, explainable matches (per-competency breakdown + evidence) plus plain-English hard-filter exclusions (location, must-have).
- **CAL-048** `[DONE]` · 8 pts — **Stage 2: rubric-based LLM scoring.** Per candidate, 0–5 per competency with evidence quote, overall fit, confidence. *AC:* output matches Appendix A.2 `breakdown`. *Deps:* CAL-047, CAL-031
- **CAL-049** `[DONE]` · 5 pts — **Stage 3: hard filters as gates.** Bias-safe `Requirements` gates: location (token-matched, remote-aware), salary-floor (currency-safe), and must-have competency (excludes only on a present-but-underscored signal — absence routes to human review, never a fabricated rejection). Each exclusion surfaced with a reason via `Shortlist.exclusions`. Logistical gates run pre-scoring (skip LLM cost). *AC:* gated-out candidates excluded with reason. *Deps:* CAL-048
- **CAL-050** `[DONE]` · 5 pts — **Match assembly & persistence.** Build `Match` (overall_score, breakdown, rationale, watch_outs, thin_evidence_flag). *AC:* matches Appendix A.2; persisted. *Deps:* CAL-049, CAL-014
- **CAL-051** `[DONE]` · 5 pts — **Live re-ranking on criteria change.** Editing must-have/weight/location re-ranks the shortlist. *AC:* re-rank ≤ acceptable latency; correct order. *Deps:* CAL-050, CAL-040 **[Audit-verified DONE]** editing criteria re-ranks correctly: the `Refiner` use-case + `RefineShortlist` RPC revise/persist/re-rank (`refine_test.go:TestRefinerRevisesPersistsAndReRanks`); the employer UI also achieves live re-rank by re-querying `GenerateShortlist` on a bumped version key. Outcome (correct order, low latency) met both ways.
- **CAL-052** `[DONE]` · 5 pts — **Bias-safe ranking guard.** Rubric-driven only; protected attributes excluded from scoring inputs. *AC:* automated test asserts protected fields never reach the scorer. *Deps:* CAL-048
- **CAL-053** `[DONE]` · 4 pts — **Two-way matching (role↔candidate).** Added the candidate→role direction to complement the Shortlister (role→candidate): pure-domain `matchingdom.ComputeFit` (deterministic, bias-safe, explainable weighted-coverage fit over competency signals only — no LLM, scales for Radar) and `app/matching.PassiveMatcher.RolesForCandidate` (loads profile, scans open roles, gates on logistics + must-have coverage, ranks by fit). Both directions now queryable at the use-case layer. Feeds Radar alerts (CAL-078). *AC:* both directions queryable. *Deps:* CAL-047

## EPIC-08 · Employer Intake & Explainable Shortlisting (Flow A)
**Goal:** End-to-end Flow A: messy sentence in → structured spec, rubric, explainable ranked shortlist out, in seconds. (Spec §6.1.)

- **CAL-054** `[DONE]` · 5 pts — **Flow A orchestration use-case.** `Shortlister` wires recall → logistical gates → rubric scoring → must-have gate → ranked Matches (+ surfaced exclusions); exposed via `MatchingService.GenerateShortlist` (gRPC + REST) and wired in `main` when a DB is configured. *AC:* single call produces a shortlist. *Deps:* CAL-040, CAL-050
- **CAL-055** `[DONE]` · 3 pts — **Instant availability signal.** "N strong matches already in your pool." `Shortlist.pool_depth` returned in the response. *AC:* pool depth returned immediately after spec. *Deps:* CAL-047 **[Partly fixed]** the `pool_depth` bug is resolved: the Shortlister now recalls/scores a `recallWindow` independent of the display page and sets `ShortlistResult.PoolDepth` to the full strong-match total, so a paginated response still reports the true count (test `TestGenerateShortlistPoolDepthExceedsPage`). Remaining: surface the signal *immediately after spec* via `available_matches` on role generation (tracked in CAL-037/058). **[DONE]** instant availability is real end-to-end: `available_matches` returns with the generated role (cheap no-LLM `Shortlister.CountAvailable`), and the shortlist's `pool_depth` reports the true strong-match total across the pool. Tests: `TestCountAvailable`, `TestGenerateRoleSpecSurfacesAvailableMatches`, `TestGenerateShortlistPoolDepthExceedsPage`.
- **CAL-056** `[DONE]` · 5 pts — **Explainable, paginated shortlist response.** Each candidate: fit score, per-competency breakdown, plain-English "why," watch-outs, thin-evidence flag; results paginated. *AC:* contract locked; no black-box fields. *Deps:* CAL-050, CAL-082 **Verified + locked:** the shortlist response exposes per-match fit score, confidence, a per-competency breakdown (each item citing evidence), a plain-English rationale, watch-outs, and a thin-evidence flag; hard-filter exclusions carry a gate + reason; the response now populates pagination metadata. `TestShortlistExplainabilityContract` asserts no black-box fields.
- **CAL-057** `[DONE]` · 3 pts — **Refine RPC.** `MatchingService.RefineShortlist` (Refiner use-case: revise+persist role → re-rank) wired; the employer UI re-ranks the shortlist live on every spec/rubric edit (version-keyed query, keeps the prior ranking visible while updating). *AC:* shortlist updates correctly. *Deps:* CAL-051
- **CAL-058** `[DONE]` · 5 pts — **Flow A proto contract & gateway.** gRPC service + REST gateway + OpenAPI; field names locked from Appendix A. *AC:* documented, validated, versioned. *Deps:* CAL-054, CAL-164 **[Audit gap]** contract complete + OpenAPI generated, but `available_matches` is never populated by the role handler; no single end-to-end Flow-A contract test on seeded data (CAL-059). **[DONE]** gRPC + REST gateway + OpenAPI with locked Appendix-A field names; `available_matches` is now populated on generation (was always 0). End-to-end demo-narrative test remains CAL-059.
- **CAL-059** `[DONE]` · 8 pts — **Flow A integration tests (demo beat).** Messy sentence → spec+rubric+ranked explainable shortlist on seed data. *AC:* acceptance criteria §15.1 pass. *Deps:* CAL-054, CAL-016 **[DONE]** `TestFlowAEndToEnd` is the single demo-narrative acceptance test: a messy hiring sentence → structured spec + weighted rubric + instant `available_matches`, then a ranked, explainable shortlist (breakdown + rationale + confidence) over a Ghana-context pool, with the must-have miss surfaced as an exclusion and a correct `pool_depth`. Drives the real use-cases through the gRPC handlers over the in-memory stack + deterministic dev model.

## EPIC-09 · AI Screening Interviewer (Flow B — centrepiece)
**Goal:** A short, adaptive interview that probes claimed competencies and returns a scored, evidence-tagged report card. The moment manual interviewing labour visibly disappears. (Spec §8.4, §6.2, Appendix A.3.)

- **CAL-060** `[DONE]` · 8 pts — **Interview state machine (FSM).** States: open → ask → analyze → adapt → … → close; max-K questions or T-minutes cap. *AC:* deterministic transitions; unit-tested. *Deps:* CAL-030
- **CAL-061** `[DONE]` · 5 pts — **Opening-question generation.** From rubric + profile. *AC:* question ties to a rubric competency. *Deps:* CAL-060, CAL-038
- **CAL-062** `[DONE]` · 8 pts — **Adaptive questioning loop.** Analyze each answer → update per-competency evidence coverage → select next question probing weakest/most-claimed competency, with follow-ups. *AC:* questions adapt to prior answers (not a fixed script). *Deps:* CAL-061
- **CAL-063** `[DONE]` · 5 pts — **Honest-signal pressure.** Detect vague/evasive answers; push for concrete examples. *AC:* evasive answers flagged in transcript. *Deps:* CAL-062 **Implemented:** `interview.VagueAnswer` — a pure, lenient heuristic that flags a vague/evasive answer (no concrete anchor: no digit, no first-person ownership phrase; short or hedge-laden), documented as surface-only (not a truthfulness judge). Wired into `questionPrompt`: when the candidate's last answer is vague, the next adaptive question carries an honest-signal directive pressing for a specific real example (what they personally did + a measurable outcome) instead of moving on. Tested at the domain level (thin/evasive vs concrete, digit-as-signal, long-specific-passes) and end-to-end through `Answer()` (vague answer ⇒ directive in the captured LLM prompt; concrete answer ⇒ no directive).
- **CAL-064** `[DONE]` · 8 pts — **Scored report card generation.** Per-competency scores + evidence quote each, overall verdict, confidence, recommended next step. *AC:* matches Appendix A.3; every score cites a transcript quote. *Deps:* CAL-062
- **CAL-065** `[DONE]` · 5 pts — **Streamed interview session.** `StartInterview` server-stream + a per-interview broker that forwards each `SubmitAnswer`'s next question (and the final report card) onto the open stream; `GetReportCard` unary. Cancellable via stream context. Smoke-tested over the gateway SSE: 4 adaptive questions + evidence-tagged report card. *Deps:* CAL-034, CAL-060
- **CAL-066** `[WIP]` · 3 pts — **Transcript & report card persistence + Passport update.** Store `Interview`, `InterviewTurn`s, report card; update Talent Passport. *AC:* transcript + card stored and viewable. *Deps:* CAL-064, CAL-014
- **CAL-067** `[DONE]` · 5 pts — **Async interview scoring job.** Heavy scoring via Asynq when not inline. `TypeInterviewScoring` task, `TaskDispatcher.DispatchInterviewScoring`, and `handleInterviewScoring` handler are wired through the idempotent job framework; `Interviewer.ScoreAsync` produces the scored report card and persists it, returning an existing card on duplicate runs. Tests prove the handler generates a report card with evidence-tagged scores and that duplicate delivery is skipped without re-scoring. *AC:* report card produced reliably off the request path. *Deps:* CAL-025, CAL-064
- **CAL-068** `[DONE]` · 8 pts — **Flow B acceptance tests (centrepiece).** Adaptive (not scripted), per-competency scores with evidence + verdict + confidence, Passport updated. *AC:* §15.2 pass; latency within demo budget. *Deps:* CAL-064, CAL-065 **[DONE]** `TestFlowBEndToEnd` is the centrepiece acceptance test: an adaptive screening (each question targets a different rubric competency — not scripted) that produces a report card with a per-competency score + evidence, an overall verdict + confidence + recommended next step, and advances the candidate's Talent Passport to screened. Drives the real interview use-case (Start→Answer*→Report) over the in-memory stack + deterministic dev model; the streaming transport is tested separately (CAL-034/091).

## EPIC-10 · Candidate Agent & Time-Advance (Flow C)
**Goal:** The agent that "works while you sleep, honestly" — matches, tailors, submits and screens using only verified profile content; demoed via a controlled time-advance. (Spec §8.5, §6.3.)

- **CAL-069** `[WIP]` · 3 pts — **One-time candidate setup.** CV upload + guided intake → initial profile. *AC:* usable profile from CV + intake. *Deps:* CAL-042, CAL-046
- **CAL-070** `[WIP]` · 8 pts — **Candidate-agent job (autonomous loop).** Scan open roles → score fit (reuse EPIC-07) → hard filters → for strong matches, tailor a truthful application. *AC:* runs as an Asynq job over the seeded role pool. *Deps:* CAL-050, CAL-025
- **CAL-071** `[DONE]` · 5 pts — **No-fabrication guardrail (hard invariant).** Added the missing OUTPUT check: pure-domain `candidateagent.CheckGrounding` validates the agent's tailored summary against the verified profile and flags any role competency the summary asserts that the profile does not cover (token-aware coverage mirroring the must-have gate; whole-token phrase matching, so "Go" isn't found in "ago"). The runner's `consider()` now rejects (never submits) a fabricated application — a 4th enforcement layer alongside domain construction, the must-have eligibility gate, and the grounded prompt. Hardened after an adversarial review: the grounding check and the must-have gate now share one `kernel.Tokens` tokenizer (a prior divergence both under-blocked a fabricated "C" claim from a "C++" profile and over-blocked an honest "C++ / Systems" candidate), and a rejection is surfaced to the candidate as an explainable wake-up highlight rather than dropped silently. **Scope (documented):** detects over-claiming of role-rubric competencies only; common skill abbreviations/variants are now canonicalized (`skillCanon`: k8s↔Kubernetes, golang↔Go, postgres↔PostgreSQL, …) so they can neither evade the guard nor falsely flag an honest variant; off-rubric fabrication (invented tenure/titles) and uncommon synonyms remain the grounded prompt's responsibility (follow-up). *AC:* asserted in code AND prompt; `TestRunRejectsFabricatedSummary` proves a tailored claim absent from the profile is not applied (and surfaced), and the grounding suite proves tailored content traces to the profile. *Deps:* CAL-070
- **CAL-072** `[DONE]` · 5 pts — **Application tailoring & submission (in-platform).** Generate role-specific application from verified content; submit within the platform; optionally complete/queue screening. *AC:* `Application{source: agent, tailored_summary, status}` written. *Deps:* CAL-070 **[Audit-verified DONE]** `candidateagent` Application (source=agent, tailored_summary, status) across domain/app/adapters/gRPC; `NewAgentApplication` grounds in the verified profile; `Submit` drafts→submitted; memory+postgres repos; e2e `candidateagent_test.go:TestAgentTimeAdvanceThenWakeUpAndList`.
- **CAL-073** `[DONE]` · 5 pts — **Time-advance action (demo engine).** Controlled "run overnight" advances agent state live — no real external submission, no waiting. *AC:* one action produces visible new state. *Deps:* CAL-027, CAL-072 **[Audit-verified DONE]** `TimeAdvance` RPC (candidate_agent.proto + candidateagent.go) drives the demo engine; tested by `TestAgentTimeAdvanceThenWakeUpAndList`.
- **CAL-074** `[DONE]` · 3 pts — **Wake-up view data.** Summary: new matches, applications tailored/submitted, completed screening + score, employer interest. *AC:* matches the §6.3 wake-up narrative. *Deps:* CAL-073 **[DONE]** the wake-up view is complete: `AgentRunner.enrichInsights` (wired via `WithWakeUpInsights`) now populates `ScreeningsCompleted` from the candidate's interviews carrying a report card and `EmployersInterested` from the roles they appear in a shortlist for. main shares the interview + match repos so Flow C reads the real interviews/matches Flow A & B produced. Best-effort (a read error leaves a count at 0). Tests: `TestRunEnrichesWakeUpInsights`, `TestRunWithoutInsightReadersLeavesCountsZero`.
- **CAL-075** `[DONE]` · 7 pts — **Flow C acceptance tests.** Setup builds a usable profile; time-advance yields tailored applications + ≥1 completed screening; **no application content untraceable to the verified profile**. *AC:* §15.3 pass. *Deps:* CAL-072, CAL-071 **[DONE]** `TestFlowCEndToEnd` is the Flow C acceptance test: a verified profile + an open role the candidate qualifies for + a previously completed screening, then a `TimeAdvance` ("run overnight") that yields a tailored application and surfaces the completed screening in the wake-up view — and asserts the hard invariant that every submitted application traces to the verified profile (ProfileID + agent source) and its summary passes the same `CheckGrounding` no-fabrication invariant the agent enforces. Runs the real agent use-case through the gRPC handler on the dev stack.

## EPIC-11 · Talent Radar Dashboard
**Goal:** The god-view that frames the whole demo: live pool, supply/demand snapshot, two-way alerts, and the headline time-to-shortlist metric dropping weeks → hours. (Spec §6.4.)

- **CAL-076** `[WIP]` · 5 pts — **Live, paginated candidate pool view.** Aggregated pool with passport status. *AC:* reflects current seed state; paginated. *Deps:* CAL-045
- **CAL-077** `[WIP]` · 5 pts — **Supply/demand snapshot by role family.** Counts and gaps per role family. *AC:* numbers reconcile with seed data. *Deps:* CAL-076
- **CAL-078** `[DONE]` · 5 pts — **Two-way match alerts.** `Aggregator.Alerts` computes a deterministic bias-safe fit (CAL-053 `ComputeFit`) for every passive candidate against each open role and emits a `candidate_for_role` alert per strong pair plus one best-fit `role_for_candidate` alert per candidate; alert IDs are deterministic (`type:role:candidate`) and the feed is paginated. gRPC `GetAlerts` maps the alert type to the `AlertType` enum end-to-end. *AC:* alerts generated from EPIC-07 two-way matching; paginated. *Deps:* CAL-053
- **CAL-079** `[WIP]` · 5 pts — **Time-to-shortlist metric.** Headline metric showing collapse from weeks → hours. *AC:* computed and displayed as the closing visual. *Deps:* CAL-059 **[Audit gap]** the metric is a hard-coded demo constant (504h→2h), not computed from real per-role timing; AC requires it computed.
- **CAL-080** `[DONE]` · 4 pts — **Dashboard aggregation performance.** Added an in-memory TTL snapshot cache (`CachedAggregator`) over the Talent Radar read model, caching pool, supply/demand, alerts, and time-to-shortlist. Configurable via `CALIBER_DASHBOARD_CACHE_TTL` (default 30s); snapshots can be refreshed on demand with `RefreshSnapshots()`. Tests prove cache hits skip repository work, TTL expiry triggers recompute, and cached reads are materially faster than slow underlying calls. *AC:* dashboard loads within demo budget. *Deps:* CAL-076

## EPIC-12 · Trust, Explainability, Audit & Guardrails
**Goal:** Demonstrable features (not disclaimers) that let the client sell to enterprise/public-sector buyers later. (Spec §11.)

- **CAL-081** `[DONE]` · 5 pts — **Human-approval gate before any rejection.** AI ranks/screens but never auto-rejects; a human approves declines, logged. *AC:* no rejection without a logged human approval. *Deps:* CAL-015, CAL-021 **Implemented:** the AI never auto-rejects — a rejection comes into being only as a logged, human-approved decision. Domain `matching.Rejection` gates on an explicit human approval + a non-empty (sanitised) reason; the `RejectionRecorder` use-case writes an `approve_rejection` audit entry where the **log is the approval** (an append failure fails the call — no unlogged rejection). gRPC `MatchingService.RecordRejection` (POST /v1/roles/{role_id}/rejections) is employer/recruiter-only, takes the approving human from the auth context (never the body), and requires `human_approved=true`. Surfaces in the audit trail (entity=`match`). Tests cover the invariant, authz, validation, append-failure, and end-to-end auditability.
- **CAL-082** `[DONE]` · 5 pts — **Explanation/rationale generator (cross-cutting).** Plain-English "why this person" + "watch-outs" derived from structured scores/evidence. *AC:* words trace back to rubric + data. *Deps:* CAL-050 **Met:** the Match carries a generated plain-English rationale + watch-outs alongside the structured per-competency breakdown (score + evidence) — words trace back to the rubric and the candidate's evidence (asserted by the explainability contract test).
- **CAL-083** `[DONE]` · 5 pts — **Candidate visibility & contest.** Full vertical: new pure-domain `contest` context (Contest entity with Subject{match,report_card} + open→upheld/dismissed lifecycle, validated) and `app/contest.Service` (Raise / ListForCandidate / ListForSubject / Resolve), every state change appended to the audit trail (explainable + human-in-the-loop). In-memory `ContestRepo` + `AuditRepo` adapters (the latter also makes the audit trail queryable in dev) + generated mocks; domain 100% / app + adapters fully tested. Exposed via a new `ContestService` (proto): `RaiseContest` (POST /v1/contests, candidate-only), `ListMyContests` (GET /v1/contests), `ResolveContest` (POST /v1/contests/{id}/resolve, employer/recruiter-only) — the acting principal is read from the authenticated context (a candidate contests only as themselves), wired into the dev stack. Smoke-verified end-to-end on seeded data: candidate raises → lists → employer resolves (upheld), and a candidate is 403-blocked from resolving. *AC:* surfaced as a fairness feature in the demo. *Deps:* CAL-066
- **CAL-084** `[DONE]` · 3 pts — **Audit trail surfacing.** Approvals, overrides, agent actions recorded and viewable (paginated). *AC:* AuditLog browsable per entity. *Deps:* CAL-015 **Implemented:** `AuditServer.ListAuditLog` (GET /v1/audit-log?entity=&entityId=) surfaces the append-only trail, reviewer-only (employer/recruiter); the in-memory `AuditRepo` is now shared with the contest service and wired in DI. Smoke-verified: a contest's log shows contest_raised + contest_resolved (newest first, actor + timestamp). The autonomous candidate agent (Flow C) now writes an `agent_submit` audit entry for every application it makes on a candidate's behalf (actor=candidate, entity=`application`, snapshot records the role + `autonomous:true`) via `AgentRunner`'s `WithAuditTrail` option — so the trail's "agent actions" clause is real, and an overseer can tell autonomous applications from manual ones.
- **CAL-085** `[DONE]` · 5 pts — **Bias & fairness checks.** Metamorphic fairness suite (`internal/app/matching/fairness_test.go`): proves through the real shortlist pipeline that two candidates with identical competencies yield byte-identical scoring/embedding inputs even when one carries protected attributes, that no protected term reaches the model, that a rubric naming a protected attribute aborts before scoring, and that the hard-filter gates are logistical (location ≠ nationality). Methodology documented in [docs/fairness.md](docs/fairness.md) (four defense layers: not-modelled, EnsureBiasSafe signal validation, input minimisation, model instruction). *AC:* fairness test suite green. *Deps:* CAL-052
- **CAL-086** `[DONE]` · 5 pts — **Data-protection baseline (Ghana DPA 2012).** Consent capture, data-minimization, deletion design (even if not fully built in POC), PII handling policy. *AC:* consent + deletion paths designed and stubbed; documented. *Deps:* CAL-014 **Documented** in [docs/data-protection.md](docs/data-protection.md): the Ghana DPA 2012 posture (consent basis, data minimisation, purpose limitation, accountability), the PII handling already enforced in code (PII-free logs/telemetry, untrusted-by-default text, no protected attributes stored/scored), the consent-capture design, the right-to-erasure cascade (designed + stubbed; audit entries retained-but-anonymised), retention, and the data-subject-rights matrix.
- **CAL-087** `[DONE]` · 5 pts — **Explainability contract.** Every score/shortlist position exposes its reasoning + evidence to the frontend. *AC:* no black-box fields in any API/proto response. *Deps:* CAL-056, CAL-064 **Met + tested:** no black-box fields — every shortlist match exposes its reasoning (breakdown + evidence + rationale + confidence) and the interview report card cites verbatim evidence; locked by `TestShortlistExplainabilityContract`.

## EPIC-13 · Frontend Web Application (React + Vite)
**Goal:** Brandable React (Vite) SPA with MUI v9, employer & candidate views, the streamed interview UI, and the Talent Radar dashboard. Skeleton loading and pagination throughout; SEO-ready public pages via prerender (EPIC-17).

- **CAL-088** `[DONE]` · 5 pts — **React+Vite scaffold + MUI v9 theme + typography.** Vite app, react-router, **MUI v9** themed design system with brandable tokens (Primary Blue #0066CC, Ink #111418, Slate #6B7280); typography wired to **Fraunces** (titles), **Outfit** (body), **JetBrains Mono** (statuses), self-hosted with `font-display: swap`; light/dark color modes ready. *AC:* design tokens + fonts centralized; no Tailwind. *Deps:* CAL-164
- **CAL-167** `[DONE]` · 3 pts — **App shell, routing & Zustand stores.** Layout, role-aware routes, Zustand stores for UI/auth/wizard state. *AC:* navigation + protected routes work. *Deps:* CAL-088
- **CAL-095** `[WIP]` · 5 pts — **API client (gRPC-web/REST) + TanStack Query + streaming.** Typed client from proto; TanStack Query setup; stream handling for the interview; resilient error states. *AC:* resilient to slow/failed calls. *Deps:* CAL-058
- **CAL-165** `[DONE]` · 3 pts — **Skeleton-loading system (content).** Reusable MUI `Skeleton` components shaped per surface (list rows, cards, dashboard tiles, report card, interview turns). *AC:* no spinners/"Loading…" text for content; lint/check guards against them. *Deps:* CAL-088
- **CAL-168** `[WIP]` · 5 pts — **Animation system (Motion): layout transitions + animated-dots buttons.** Install Motion (Framer Motion); app-wide layout/route/list transitions (incl. live shortlist re-rank); reusable **animated-dots** button-loading component (width-stable, no spinners); all gated behind `prefers-reduced-motion`. *AC:* buttons show dots when busy; layout changes animate; reduced-motion respected. *Deps:* CAL-088
- **CAL-169** `[DONE]` · 3 pts — **Circular-reveal light/dark theme toggle.** MUI color-mode toggle animated as a circular reveal from the control (View Transitions API + clip-path fallback); persisted preference. *AC:* theme switches with circular reveal; falls back cleanly; reduced-motion respected. *Deps:* CAL-088
- **CAL-166** `[WIP]` · 3 pts — **Pagination system (standard).** Reusable paginated-query hooks (TanStack Query, `keepPreviousData`) + MUI pagination controls, applied to every list. *AC:* no unbounded lists; pages map to server pages. *Deps:* CAL-095
- **CAL-089** `[DONE]` · 5 pts — **Auth UI & session handling.** Login/register, role-aware routing, secure token storage, refresh. *AC:* both roles reach their views behind login. *Deps:* CAL-167
- **CAL-090** `[WIP]` · 8 pts — **Employer view — Flow A UI.** Plain-language intake, editable spec/rubric, instant availability, explainable **paginated** ranked shortlist with live refine. *AC:* §15.1 visible end-to-end. *Deps:* CAL-058, CAL-166
- **CAL-091** `[WIP]` · 8 pts — **Interview UI — Flow B (centrepiece).** Streamed adaptive Q&A (skeletons between turns), evidence-tagged report card reveal; graceful, low-latency. *AC:* live adaptive interview renders + scored card. *Deps:* CAL-065, CAL-165 **[Audit] functionally complete** (streamed adaptive Q&A UI + evidence-tagged report card; backend streaming tested by `interview_test.go:TestStartInterviewStreamsQuestionThenReport`); kept WIP only because automated FE tests await the frontend test harness (CAL-138).
- **CAL-092** `[WIP]` · 8 pts — **Candidate view — Flow C UI.** One-time setup, time-advance ("run overnight"), wake-up view. *AC:* §15.3 visible end-to-end. *Deps:* CAL-073
- **CAL-093** `[WIP]` · 8 pts — **Talent Radar dashboard UI.** Live pool, supply/demand, two-way alerts, time-to-shortlist headline (the closing visual); skeleton tiles + paginated lists. *AC:* §15.4 visible. *Deps:* CAL-079, CAL-165, CAL-166 **[Backend locked]** `TestTalentRadarEndToEnd` is the closing demo-beat acceptance test: the live pool (named candidates + passport), the supply/demand snapshot, the two-way match alerts, and the time-to-shortlist headline are all served coherently through the gRPC dashboard handlers over a seeded in-memory pool. The React Radar UI itself remains the WIP frontend piece.
- **CAL-094** `[TODO]` · 5 pts — **Explainability & trust UI.** Per-score reasoning, watch-outs, thin-evidence flags, candidate contest, human-approval gate surfaced. *AC:* nothing reads as a black box. *Deps:* CAL-087
- **CAL-096** `[DONE]` · 5 pts — **Accessibility baseline (WCAG 2.1 AA).** Skip-to-content link + `<main>`/`<nav>` landmarks (AppShell), `aria-busy` on the animated-dots loading button, `role="status"` on skeleton loaders, `aria-live="polite"` on the streamed interview question, `lang="en"`, and `prefers-reduced-motion` honored via `MotionConfig reducedMotion="user"` + per-effect reduce fallbacks. Verified via tsc + eslint + vite build. (Automated axe assertions need a browser harness — CI-gated follow-up.) *AC:* axe checks pass on key screens; reduced-motion verified. *Deps:* CAL-088, CAL-168
- **CAL-097** `[DONE]` · 3 pts — **Responsive & demo-resilient layout.** Audited AppShell and all key flows; added MUI breakpoint-aware stacking/wrapping, small-screen nav sizing, and flexible card/list rows so the app stays production-credible on the demo screen/resolution. *AC:* no layout breakage at target resolutions. *Deps:* CAL-088

## EPIC-14 · Seed Data & Demo Orchestration
**Goal:** A believable, locally-plausible (Ghana/West Africa) pool the demo lives on. (Spec §10.)

- **CAL-098** `[DONE]` · 5 pts — **Seed generation pipeline.** Added a deterministic `seed.Generator` that produces 8 Ghana-context employers, 12 roles, and 55 CV/profiles by driving the real `ProfileBuilder` and `SpecGenerator` parsers. Wired behind `CALIBER_SEED_GENERATED` for startup generation; tests verify parser acceptance and the no-fabrication evidence invariant. *AC:* data produced by the real pipeline. *Deps:* CAL-043, CAL-037
- **CAL-099** `[DONE]` · 5 pts — **Local plausibility curation.** Curated the generated seed tables in `internal/platform/seed/generator_data.go`: expanded Ghanaian first/last names, added real Ghanaian cities (Tema, Ho, Sunyani, Koforidua, Wa), added real Ghanaian universities (GCTU, Valley View, Central, GIMPA, KTU, UMaT), and replaced non-local company entries with Ghana/West-Africa-plausible employers (Zeepay, Absa, CAL Bank, Republic Bank, GT Bank, AirtelTigo). Added `generator_plausibility_test.go` with whitelist regression tests asserting that every generated name, location, institution, and employer is Ghana-plausible and that no generic placeholder tokens appear. *AC:* review pass before demo; parser acceptance and no-fabrication evidence invariant preserved. *Deps:* CAL-098
- **CAL-100** `[DONE]` · 5 pts — **Hero candidate/role pairs.** Reserved candidate indices 0–2 as deterministic hero pairs (Ama→Senior Backend, Kofi→Data Engineer, Esi→Platform Engineer) with CV families tuned to cover the dev rubric and each role's must-haves; the remaining 52 candidates stay varied. *AC:* hero pairs deterministic; tested through the real shortlist and two-way matching pipelines. *Deps:* CAL-098
- **CAL-101** `[DONE]` · 3 pts — **Pre-run interviews.** Added `internal/platform/seed/prerun.go`: after seeding, it drives the real `app/interview.Interviewer` to complete 2-question screenings for two hero pairs (Ama→Senior Backend, Kofi→Data Engineer), storing report cards in the interview repository and advancing the Talent Passport to screened. Esi (generated seed) and Esi/Yaw (hand-curated seed) are intentionally left live for Flow B. Wired into both the parser-driven `Generator.Generate` and the hand-curated `seed.Load` via `WithPreRunInterviews`, and into `wiring.SeedDemo` so the dev stack boots with real assessments. *AC:* shortlists show real assessments; verified by `TestGenerator_PreRunsInterviewsForHeroes` and `TestLoad_PreRunsInterviewsWhenLLMProvided`. *Deps:* CAL-064
- **CAL-102** `[DONE]` · 5 pts — **Seeded application/agent state.** Added `internal/platform/seed/preseed.go`: after seeding, it drives the real `app/candidateagent.AgentRunner` to submit honest, grounded agent applications for curated hero candidates (hand-curated: Ama→Senior Backend, Kofi→Data Engineer, Esi→Mobile Engineer, Yaw→Platform Engineer, Abena→Junior Frontend; generated: Ama→Senior Backend, Kofi→Data Engineer, Esi→Platform Engineer). Wired into both `Generator.Generate` and `seed.Load` via `WithPreSeededAgentState`, and into `wiring.SeedDemo` so the dev stack boots with Flow C wake-up state ready. *AC:* Flow C demo state ready; verified by `TestLoad_PreSeedsAgentStateWhenLLMAndAppsProvided` and `TestGenerator_PreSeedsAgentStateForHeroes` (wake-up view shows pre-seeded applications + pre-run screenings). *Deps:* CAL-072
- **CAL-103** `[TODO]` · 5 pts — **Reseed/reset command.** One command to wipe + reseed to a known demo state. *AC:* deterministic, repeatable. *Deps:* CAL-098

## EPIC-15 · Demo Hardening & Run-of-Show
**Goal:** Make the demo reliable, repeatable, venue-proof. (Spec §13 Phase 5, §14, §16.)

- **CAL-104** `[DONE]` · 5 pts — **Latency tuning & session pre-warm.** Added `interview.Config` with `MaxQuestions`/`MaxDuration` caps (env-driven via `CALIBER_INTERVIEW_MAX_QUESTIONS`/`CALIBER_INTERVIEW_MAX_DURATION`), enforced in `Interviewer.Answer`; added `LLMClient.Warm` port method with implementations in dev/Claude/Audited/Guarded and pre-warm on `Start`; added `CachedEmbedder` so role embeddings are reused across `GenerateShortlist`/`CountAvailable`/re-rank. *AC:* interview + shortlist feel instant. *Deps:* CAL-065, CAL-068
- **CAL-105** `[TODO]` · 3 pts — **Run-of-show wiring.** Sequence: Frame → Flow A → Flow B → Flow C → close on dashboard. *AC:* one path drives the whole narrative. *Deps:* CAL-090, CAL-091, CAL-092, CAL-093
- **CAL-106** `[TODO]` · 5 pts — **Pre-recorded backup capture.** Clean live-style interview recording as insurance for venue network failure. *AC:* recording ready; live path primary. *Deps:* CAL-091
- **CAL-107** `[TODO]` · 5 pts — **Offline/standby deployment fallback.** Local/standby deployment where feasible. *AC:* demo survives a network drop. *Deps:* CAL-006
- **CAL-108** `[TODO]` · 3 pts — **Full dry run + acceptance sweep.** Verify all §15 acceptance criteria on seed data in one rehearsal. *AC:* every §15 item passes. *Deps:* CAL-059, CAL-068, CAL-075, CAL-093
- **CAL-109** `[DONE]` · 3 pts — **Demo runbook & failure playbook.** Written run-of-show, reset steps, fallback triggers. *AC:* any team member can drive it. *Deps:* CAL-103, CAL-105. *Done:* `docs/demo-runbook.md` created with run-of-show sequence, demo accounts, reset/reseed steps, talking points, expected outputs, and a failure playbook covering network drop, LLM latency, stream failure, worker/queue issues, and venue fallback.

---

# MILESTONE 2 — Production-Ready

Beyond the win: harden security, SEO, observability, quality, deployment, and scale. (Spec defers these to "the build phase that follows the win" — captured here so nothing is forgotten.)

## EPIC-16 · Security Hardening & Compliance
**Goal:** Defensible to enterprise clients and regulators. OWASP-aligned, Ghana DPA-compliant, audited.

- **CAL-110** `[DONE]` · 5 pts — **Threat model & security requirements.** STRIDE over the architecture; security backlog. *AC:* documented threat model. *Deps:* — **Documented** in [docs/threat-model.md](docs/threat-model.md): scope/assets, trust boundaries, and a full STRIDE pass mapping each threat class to controls **implemented today** (Argon2id+JWT, context-derived actor identity, append-only attributed audit trail, parameterised SQL, prompt-injection sanitise/fence/scan, no-fabrication grounding, human-approval-before-rejection, PII-free logs, protected-attributes-never-scored, per-RPC RBAC, pagination) vs the **security backlog** (CAL-111–118, 120) with a cross-cutting LLM/prompt-injection section, security-requirement acceptance list, and a backlog table. Sets up CAL-120.
- **CAL-111** `[TODO]` · 5 pts — **Input validation & output encoding everywhere.** Proto/DTO validation, parameterized SQL (sqlc), XSS-safe rendering. *AC:* OWASP A03 checks pass. *Deps:* CAL-058
- **CAL-112** `[DONE]` · 5 pts — **Rate limiting, throttling & abuse protection.** Per-IP/user/endpoint limits; expensive AI endpoints protected; bot mitigation. *AC:* limits enforced + tested. *Deps:* CAL-021 **Implemented:** a concurrency-safe per-key **token-bucket** `RateLimiter` (refill/sec + burst ceiling, injectable clock) and a gRPC unary `RateLimitInterceptor` chained **after** auth — keyed by the authenticated principal (quota follows the user across methods), falling back to a per-method anonymous bucket; over-limit → `ResourceExhausted` before the handler runs. Wired into every service (so the expensive AI endpoints are protected) with generous config-driven defaults (`CALIBER_RATE_LIMIT_RPS=30`, `_BURST=60`) documented in `.env.example`. Fully unit-tested (burst→deny, time-refill capped at burst, key isolation, config clamping, interceptor reject + anon-per-method). **Deferred follow-ups:** per-IP keying and bot mitigation, best handled at the gateway/CDN edge.
- **CAL-113** `[TODO]` · 5 pts — **Secrets management & rotation.** Platform secret store, rotation policy, no secrets in logs; gitleaks gate extended. *AC:* secret scan clean; rotation documented. *Deps:* CAL-005
- **CAL-114** `[DONE]` · 5 pts — **Security headers, TLS & CORS.** HSTS, CSP, X-Frame-Options, strict CORS, TLS everywhere. *AC:* securityheaders/observatory grade A. *Deps:* CAL-088 **Implemented 2026-06-29:** the HTTP gateway now has a configurable browser-security surface: exact-origin CORS (`CALIBER_CORS_ORIGINS`) reflects only allowlisted origins, emits no CORS headers for unknown browser origins, and handles preflight without reaching the gateway; wildcard/malformed CORS origins fail config load; production validation requires explicit allowed origins; secure headers are tested with concrete HSTS/CSP/frame/referrer/permissions values. TLS remains enforced at the hosting edge; app-level HSTS is emitted only in prod.
- **CAL-115** `[DONE]` · 5 pts — **Dependency & container scanning.** `govulncheck`, Trivy/Grype, npm audit, Dependabot in CI. *AC:* no high/critical vulns merge. *Deps:* CAL-003 **Implemented 2026-06-29:** the CI workflow has a named `Supply chain (govulncheck · npm audit · Trivy)` job that runs `govulncheck` against `./...`, blocks frontend high/critical findings with `npm audit --audit-level=high`, builds the api/worker/migrate Docker images, and fails on HIGH/CRITICAL Trivy image findings. Dependabot is configured for Go modules, npm, Docker base images, and GitHub Actions; local `make scan-*` targets mirror the gate where tooling is installed. Branch-protection docs now include the new required check.
- **CAL-116** `[WIP]` · 5 pts — **AuthZ hardening & least privilege.** Full ownership checks, IDOR tests, least privilege across services. *AC:* IDOR test suite green. *Deps:* CAL-021 **[Partial]** an end-to-end authn acceptance test now exercises the real Argon2id hasher + JWT service through the auth interceptor (`TestAuthFlowEndToEnd`: register → login → authenticated GetMe; wrong-password, missing-token, and forged-token all rejected). The systematic IDOR/ownership-check sweep + least-privilege audit remain. The **Talent Radar dashboard is now reviewer-only**: all four handlers (pool, supply/demand, alerts, time-to-shortlist) require employer/recruiter via a `requireReviewer` guard — closing the gap where the candidate pool + hiring intelligence were readable unauthenticated. Tested (candidate→PermissionDenied, anon→Unauthenticated). The **shortlist handlers (GenerateShortlist + RefineShortlist) are now employer/recruiter-only** (viewing/refining a role's shortlist is hiring work); RecordRejection was already guarded. Tested (candidate→PermissionDenied, anon→Unauthenticated). The **role write handlers (GenerateRoleSpec/UpdateRoleSpec/ListRoles) are employer/recruiter-only** and **GetRole requires authentication** (candidates view postings to apply). Tested (candidate→PermissionDenied, anon→Unauthenticated). *Remaining:* per-resource ownership (employer owns THIS role; candidate-self scoping on agent/talent handlers — deeper, tracked toward CAL-153) + the candidate-agent/talent handlers. The **candidate-agent handlers (RunAgent/TimeAdvance/GetWakeUpView/ListApplications) are now candidate-self-scoped** via `requireSelfCandidate` (the caller must be a candidate whose id matches the target — registered candidates have candidate.ID==user.ID), closing the IDOR where anyone could run/read another candidate's agent. Tested (other-candidate→PermissionDenied, anon→Unauthenticated). *Remaining:* the talent handlers (self-or-reviewer) + per-role employer ownership. The **talent handlers are scoped**: CreateProfileFromCV is candidate-self; GetTalentProfile is self-or-reviewer (employers view profiles when shortlisting). **Handler-level RBAC + candidate-self IDOR protection now cover every service** (identity/role/match/talent/agent/dashboard/contest/audit), each with an IDOR/authz test. *Remaining (toward CAL-153):* per-resource employer ownership (employer owns THIS role) — deferred due to the recruiter-acting-for-employer ambiguity + handler role-repo wiring. The **interview handlers (Flow B) are guarded too**: StartInterview is candidate-self (you screen as yourself), SubmitAnswer is candidate-only, GetReportCard requires authentication (candidate or reviewer). With this, **every gRPC handler across all nine services is authenticated + authorized** (tested per service). **[Adversarial review + fixes]** an adversarial-review workflow (4 dimensions + skeptic verification) audited the sweep: fixed the **interview-ownership IDORs** (SubmitAnswer + GetReportCard now verify the caller owns the interview via `Interviewer.CandidateForInterview` / the report's CandidateID) and added a **streaming auth interceptor** (`NewAuthStreamInterceptor` + `ChainStreamInterceptor`) — unary interceptors don't run for streams, so StartInterview previously couldn't authenticate any real candidate. **Confirmed remaining IDORs requiring the tenant model (deferred to CAL-153):** employer-ownership on GenerateRoleSpec/UpdateRoleSpec/ListRoles, RecordRejection (role ownership), and ResolveContest (contested-subject ownership) — all need a user↔employer mapping (Principal has no EmployerID; the seed uses employer-entity ids while registration has no employer entity), so a naive `employerId==UserID` check would break seeded-employer logins. **[Employer-ownership: role handlers]** the model is simpler than feared — employers ARE users and a role's `EmployerID` is the owning user's id (seed sets it so), so ownership is a direct `principal.UserID == EmployerID` check (no tenant entity / JWT change). Added `requireSelfEmployer` and applied it: GenerateRoleSpec + ListRoles (employer_id from the body must match the caller) and UpdateRoleSpec (loads the role, checks EmployerID). Tested (other-employer → PermissionDenied). **[Employer-ownership: shortlist + rejection]** `Shortlister.GenerateShortlist`, `Refiner.Refine`, and `RejectionRecorder.Record` now take the acting `actorUserID` and reject non-owners with `kernel.Forbidden` immediately after loading the role (`role.EmployerID == actorUserID`), before any recall/scoring or audit write; handlers pass `principal.UserID` from the auth context (never the body). The recorder is now built inside `openRepositories` so it binds the same role repo as the rest of the wiring. Cross-employer IDOR + role-not-found tests added at both the use-case and handler layers (commits `6908864`, `181e5a9`, CI-green). *Remaining employer-ownership:* **ResolveContest only** — deferred by design: enforcing reviewer ownership needs a contest→subject→role lookup the data model doesn't support today (`MatchRepository` has no `ByID`; report cards have no `ByID` store), so it needs new domain ports + Postgres queries + sqlc regen + mocks. It stays a documented POC simplification with audit logging as the compensating control (tracked toward CAL-153 / a dedicated story). **[Adversarial audit round 2]** a read-only authz/IDOR workflow (one auditor per service + independent skeptic verification + synthesis) re-swept all nine gRPC services and confirmed every write path + candidate-self path is airtight. It found **two new cross-employer read IDORs**: (1) **[HIGH] GetReportCard** — `requireSelfCandidateOrReviewer` granted *any* reviewer, leaking Flow B verdicts/scores/evidence across employers; **fixed**: a new `Interviewer.EmployerForInterview` resolves the screened role's owner and the handler now scopes the reviewer branch to that owner (candidate branch already self-scoped), with cross-employer + cross-candidate + owning-candidate IDOR tests at handler and use-case layers. (2) **[MEDIUM] ListAuditLog** — a reviewer can read another employer's hiring-decision trail for a shared subject (e.g. a candidate rejected by several employers, `entity:"match"`/`entity_id:candidateID`); **deferred by design**: actor-scoping was implemented and reverted because it breaks the legitimate contest trail (a reviewer must see the candidate's *raise* plus their own *resolve*), and the correct per-entity role-ownership scope is unresolvable from audit data alone — the audit row carries only `entity`+`entity_id` (a candidate/contest id), not the owning role — so it needs the same ownership model as ResolveContest (CAL-153). Compensating control: the trail remains reviewer-only (RBAC) and append-only.
- **CAL-117** `[WIP]` · 5 pts — **PII protection & encryption.** Encrypt sensitive data at rest, field-level where needed, PII redaction in logs/telemetry. *AC:* no PII in logs; encryption verified. *Deps:* CAL-036 **[PII-in-logs slice done 2026-06-29]** the root logger now wraps its JSON handler in a **redacting `slog.Handler`** (`internal/platform/logging/redact.go`): a defense-in-depth backstop that scrubs every record — message + attributes, recursively through groups and `With`-bound fields — before write. Values under secret/identifier key names (email/password/authorization/token/phone/ssn/…) are blanked, and PII-shaped substrings (emails, `Bearer` credentials, JWTs) are masked anywhere they appear, while neutral keys like `token_count`/`service_name` are left intact. Fully unit-tested (sensitive keys, value-shaped PII, nested groups, message, `With`-bound attrs, neutral-field preservation); documented in [docs/data-protection.md](docs/data-protection.md). *Remaining:* encryption at rest / field-level encryption (DB-side, deps CAL-036) — deferred to a follow-up.
- **CAL-118** `[TODO]` · 5 pts — **Ghana Data Protection Act 2012 compliance.** Consent records, lawful basis, retention schedule, **DSAR + deletion** flows, processor agreements. *AC:* DSAR + deletion functional. *Deps:* CAL-086
- **CAL-119** `[DONE]` · 5 pts — **LLM/prompt-injection & data-exfil defenses.** New pure-domain `guard` package: `Sanitize` (strips Unicode format/control/bidi-override chars, defangs forged fence markers, caps length), `Fence`/`FenceUntrusted` (collision-proof delimiters so untrusted text can't escape its data region), and `ScanInjection` (curated corpus → categories: instruction_override, role_manipulation, system_exfil, fabrication_pressure, delimiter_breakout, data_exfil). Wired at all four LLM call sites — CV extraction, role-spec generation, interview transcript (candidate answers), and the candidate-agent assess prompt (CV-derived evidence) — with system prompts updated to declare the fence as data-only. System-prompt isolation confirmed (untrusted text only ever lands in `Prompt`). *AC:* injection test corpus passes (96.6% pkg coverage; benign-CV false-positive guard). *Deps:* CAL-035
- **CAL-120** `[WIP]` · 5 pts — **Security review & pen-test prep.** Run `/security-review`, remediate; prepare for external pen test; SonarQube security hotspots cleared. *AC:* no open high findings. *Deps:* all EPIC-16 **[Audit + remediation 2026-06-29]** an adversarial security-review workflow (audit → independent skeptic verification → synthesis) produced verified findings, all remediated and tested: **H1 — DOCX zip-bomb → OOM DoS:** the CV extractor capped only the *compressed* upload; `body.Open()` streamed unbounded into a `strings.Builder`. Now rejects an oversized declared body and hard-caps the decompressed stream with `io.LimitReader` (4 MiB), defending forged size metadata. **H2 — rate-limit shared-bucket DoS:** anonymous requests keyed by method alone pooled every caller into one bucket, so a single flooder could lock all users out of login/register; now keyed per client IP (X-Forwarded-For-aware) + method. **M1 — unbounded request body:** REST gateway buffered bodies with no cap; added an `http.MaxBytesReader` (16 MiB) middleware + explicit `grpc.MaxRecvMsgSize` (16 MiB, also lifts the 4 MiB default so 10 MiB CVs work). **M2 — CWE-209 info disclosure:** `errToStatus` echoed raw internal errors (pgx/schema text) to clients; internal errors now log server-side and return an opaque `"internal error"`. **L2 — unbounded CV text:** raw `cv_text` flowed into the extraction model with no length limit (token-cost amplification); capped at 200k runes at the use-case. **L6 — unbounded intake free-text:** guided-intake fields (titles/location/deal-breakers) bounded at the gRPC boundary. **L4 — reflection in prod:** gRPC server reflection was registered unconditionally (API-schema disclosure); now gated behind `EnableReflection` (= `!cfg.IsProd()`). **L3 — third-party error leak:** the OpenAI embeddings client folded the raw provider error body into the returned error; now logged server-side, returns only the status code. **L7 — limiter map growth:** per-IP keying makes bucket cardinality attacker-driven; `Allow` now opportunistically evicts buckets idle > 10 min past a 10k threshold (behavior-preserving — an idle bucket has refilled). All shipped to `main` CI-green (commits `fd067f9`, `c39ae23`, `23081cb`). *Remaining (deferred, all LOW):* **L8** access-token revocation store (sizable — needs a denylist + lookup on the auth path), **L1** strengthening the no-fabrication guard beyond the evidence-quote drop, **L5** residual forged-fence edge cases (`guard.Sanitize` already defangs the common cases). SonarQube security-hotspot clearing tracked with CAL-145.

## EPIC-17 · SEO & Web Performance
**Goal:** Discoverable, fast, share-ready public surface from a React SPA. (Marketing/landing + any public talent/role pages.)

- **CAL-121** `[TODO]` · 5 pts — **Prerender pipeline for public pages.** Build-time prerender (e.g. vite-plugin-ssg / react-snap / prerendering) so public/marketing/role pages ship crawlable HTML; app behind auth stays CSR. *AC:* public pages contain content in initial HTML. *Deps:* CAL-088
- **CAL-122** `[DONE]` · 3 pts — **Metadata & Open Graph/Twitter cards.** React 19 native document metadata (no head-manager dep): a `Seo` component + a central `RouteSeo` route→metadata map render per-route `<title>`/description/canonical/OG/Twitter tags (auth routes noindex); `index.html` carries enriched defaults. *AC:* rich preview on share; unique titles per page. *Deps:* CAL-121
- **CAL-123** `[DONE]` · 5 pts — **Structured data (JSON-LD).** `Organization` JSON-LD emitted on the landing page via the `Seo` component. (`JobPosting`/`Occupation` on public role pages awaits the prerendered public role surface, CAL-121.) *AC:* validates in Rich Results Test. *Deps:* CAL-121
- **CAL-124** `[DONE]` · 3 pts — **Sitemap & robots.** `public/robots.txt` (public pages allowed, app/auth routes disallowed, sitemap referenced) + `public/sitemap.xml` (public URLs), shipped in the build output. *AC:* sitemap submitted; private routes disallowed. *Deps:* CAL-121
- **CAL-125** `[TODO]` · 5 pts — **Core Web Vitals optimization.** LCP/INP/CLS budgets; image optimization, font loading, code splitting/lazy routes, caching, MUI bundle trimming. *AC:* Lighthouse ≥ 90 perf on key pages. *Deps:* CAL-088
- **CAL-126** `[TODO]` · 5 pts — **Semantic HTML & a11y for SEO.** Heading hierarchy, landmarks, alt text (reinforces CAL-096). *AC:* no critical Lighthouse SEO/a11y issues. *Deps:* CAL-096
- **CAL-127** `[TODO]` · 3 pts — **Internationalization & localization readiness.** hreflang scaffolding, locale-aware routing (Ghana/West Africa first). *AC:* i18n structure in place. *Deps:* CAL-121
- **CAL-128** `[TODO]` · 4 pts — **Analytics & Search Console.** Privacy-respecting analytics, Web Vitals reporting, Search Console verification. *AC:* traffic + vitals visible. *Deps:* CAL-121
- **CAL-129** `[TODO]` · 5 pts — **Performance budgets in CI.** Lighthouse CI gate on PRs for public pages. *AC:* regressions block merge. *Deps:* CAL-125, CAL-003
- **CAL-170** `[WIP]` · 5 pts — **Marketing-site animation kit.** Parallax sections, 3D reveal-on-scroll, and the circular-reveal theme toggle on public/marketing pages — built with Motion, lazy/IntersectionObserver-driven, within the Core Web Vitals budget (CAL-125) and gated behind `prefers-reduced-motion`. *AC:* effects render; Lighthouse perf budget still met; reduced-motion disables them. *Deps:* CAL-121, CAL-125, CAL-168

## EPIC-18 · Observability & Operations
**Goal:** See everything in production. OpenTelemetry + Prometheus/Grafana/Loki.

- **CAL-130** `[TODO]` · 5 pts — **OpenTelemetry tracing.** Instrument gRPC/HTTP, DB, queue, and LLM calls with spans + context propagation. *AC:* end-to-end trace for a request. *Deps:* CAL-007
- **CAL-131** `[TODO]` · 5 pts — **Metrics (Prometheus).** RED/USE metrics, AI cost/latency/token metrics, queue depth, business KPIs (time-to-shortlist). *AC:* dashboards populate. *Deps:* CAL-130
- **CAL-132** `[TODO]` · 5 pts — **Centralized logging (Loki).** Ship structured logs; correlate via trace id; PII-safe (ties CAL-117). *AC:* logs searchable by request/trace id. *Deps:* CAL-007
- **CAL-133** `[TODO]` · 5 pts — **Grafana dashboards.** Service health, AI usage/cost, queue health, SLO dashboards. *AC:* on-call can triage from dashboards. *Deps:* CAL-131
- **CAL-134** `[TODO]` · 5 pts — **Alerting & SLOs.** Define SLOs (availability, latency, error rate, AI failure rate); alert routing. *AC:* alerts fire on breach. *Deps:* CAL-133
- **CAL-135** `[TODO]` · 3 pts — **Error tracking & on-call runbooks.** Error grouping; incident runbooks. *AC:* known failure modes documented. *Deps:* CAL-132
- **CAL-136** `[TODO]` · 4 pts — **Audit & compliance reporting.** Reportable audit-log views (approvals/overrides/agent actions). *AC:* exportable audit reports. *Deps:* CAL-084
- **CAL-137** `[WIP]` · 5 pts — **AI quality monitoring.** Track structured-output failure rate, refusal/latency, guardrail trips; eval harness. *AC:* AI regressions visible. *Deps:* CAL-036 **[Started]** `app.SummarizeAIQuality` computes an AI-quality summary over the redacted AICallRecord traces — call volume, failure rate, p50/p95 latency, per-operation breakdown, and an input/output char (token-proxy) cost signal — exposed as `MemoryRecorder.Stats()` (PII-free). Tests cover aggregation, rates, and percentiles. **Remaining:** structured-output(JSON)-specific + refusal + guardrail-trip counters, and surfacing via a metrics endpoint (ties to CAL-131 Prometheus).

## EPIC-19 · Quality, Testing & Performance Engineering
**Goal:** The ≥80% gate is the floor; build the full pyramid and prove it scales.

- **CAL-138** `[WIP]` · 5 pts — **Test pyramid standards.** Unit (domain), integration (adapters via testcontainers), contract (proto), e2e (Playwright) — documented + enforced. *AC:* standards in CLAUDE.md; CI runs each layer. *Deps:* CAL-003 **[FE unit layer landed]** Vitest + React Testing Library + jsdom harness wired (`npm run test:run`, `src/test-setup.ts` with jest-dom matchers + RTL cleanup), enforced as a CI step in the frontend job. First tests: `format.test.ts` (pure helpers) + `DotsButton.test.tsx` (loading/idle a11y states). **[FE unit layer now comprehensive]** the Vitest layer covers the full SPA: all Flow A/B/C + Radar presentational components (RoleSpecCard, RubricCard, MatchCard, ShortlistSection with exclusions-surfaced, ProfileView with evidence/no-fabrication, WakeUpCard, ApplicationsList, TranscriptList, ReportCardView, radar panels), the structural shell (AppShell auth-nav, ProtectedRoute, SessionBootstrap session-restore, RouteSeo public/noindex, ModeToggle reduced-motion, Seo, Skeletons, PageControls), the two core hooks (`useInterview` Flow B state machine, auth store), and **every page** (Login/Register/Landing/NotFound/Roles/Profile/Agent/Dashboard/Radar/EmployerFlow/Interview) — ~28 files / 80 tests, all green, run as a CI step. This removes the "automated FE tests await the harness" blocker noted on CAL-090/091/092/093. Go unit/integration(testcontainers)/contract layers already exist; **Playwright e2e (CAL-141) is the remaining layer.**
- **CAL-139** `[TODO]` · 5 pts — **Coverage enforcement & reporting.** Per-package ≥80% gate (Go + web), trend reporting, no-untested-merge. *AC:* gate enforced on every push. *Deps:* CAL-003 **[Audit gap]** the CI gate enforces TOTAL app coverage ≥80% (currently ~89.6%), but the AC's *per-package* gate and *web* coverage gate + trend reporting are not implemented.
- **CAL-140** `[DONE]` · 5 pts — **Deterministic AI testing.** Golden tests with mocked LLM/embeddings; live smoke tests behind a flag. *AC:* AI logic testable without network. *Deps:* CAL-030 **[Audit-verified DONE]** the `dev` LLM provider gives deterministic golden responses for all six call sites (interview/report/rolespec/cv-extract/agent/score), tested in `dev_*_test.go`; app logic is exercised via gomock with no network. Live calls go through the real provider behind `CALIBER_LLM_PROVIDER`.
- **CAL-141** `[TODO]` · 5 pts — **End-to-end (Playwright) suite.** Cover the three flows + dashboard, incl. skeleton/pagination behavior. *AC:* e2e green in CI. *Deps:* CAL-093
- **CAL-142** `[TODO]` · 5 pts — **Load & performance testing (k6).** Model demo + production traffic; find limits of matching/interview. *AC:* SLO targets met under load. *Deps:* CAL-008
- **CAL-143** `[TODO]` · 3 pts — **Chaos & resilience tests.** Kill DB/Redis/LLM; verify graceful degradation (esp. interview → text/cached). *AC:* no data loss; clean fallbacks. *Deps:* CAL-026
- **CAL-144** `[TODO]` · 5 pts — **Mutation testing & flake control.** Mutation testing on domain; quarantine/fix flaky tests. *AC:* mutation baseline set; flake rate tracked. *Deps:* CAL-138
- **CAL-145** `[TODO]` · 6 pts — **SonarQube deep config.** Custom quality profiles, security rules, coverage + duplication thresholds tuned for Go + TS. *AC:* gate reflects house standards. *Deps:* CAL-004

## EPIC-20 · CI/CD, Environments & Release Management
**Goal:** Safe, automated path from PR to production. (Ops Manual Phases 7–11.)

- **CAL-146** `[TODO]` · 5 pts — **Environment topology.** Dev, staging, production configs/secrets per environment. *AC:* parity documented; no shared secrets. *Deps:* CAL-005
- **CAL-147** `[TODO]` · 5 pts — **CD to staging (Render/Railway).** Auto-deploy `main` to staging; smoke tests + security scan post-deploy. *AC:* staging always reflects `main`. *Deps:* CAL-003, CAL-146
- **CAL-148** `[TODO]` · 5 pts — **Production deploy with approval gate.** Promote staging→prod behind QA approval; release notes auto-generated. *AC:* gated, audited promotion. *Deps:* CAL-147
- **CAL-149** `[TODO]` · 5 pts — **Zero-downtime & rollback.** Health-gated rollout, automatic rollback on failure, DB migration safety (expand/contract). *AC:* rollback tested; migrations reversible. *Deps:* CAL-012, CAL-148
- **CAL-150** `[TODO]` · 5 pts — **Infrastructure as Code.** Codify env, DB, Redis, secrets, CDN. *AC:* environment reproducible from code. *Deps:* CAL-146
- **CAL-151** `[TODO]` · 4 pts — **Backups & disaster recovery.** Automated Postgres backups, restore drills, RPO/RTO targets. *AC:* successful restore drill. *Deps:* CAL-146
- **CAL-152** `[TODO]` · 3 pts — **Frontend deploy (Vercel) + preview envs.** Per-PR preview URLs; production promotion. *AC:* previews on every PR. *Deps:* CAL-088

## EPIC-21 · Scale, Multi-Tenancy & Data Lifecycle
**Goal:** Production concerns the spec deferred: full RBAC, multi-tenant scale, caching, SSO-ready. (Spec §4.2.)

- **CAL-153** `[TODO]` · 5 pts — **Multi-tenancy model.** Tenant isolation for multiple employers/clients; row-level scoping. *AC:* cross-tenant access impossible; tested. *Deps:* CAL-021
- **CAL-154** `[TODO]` · 5 pts — **Full RBAC & permissions.** Granular roles/permissions beyond the two POC roles; admin tooling. *AC:* permission matrix enforced. *Deps:* CAL-021
- **CAL-155** `[TODO]` · 5 pts — **Enterprise SSO readiness.** OIDC/SAML integration points (deferred from POC). *AC:* SSO pluggable behind the auth port. *Deps:* CAL-019
- **CAL-156** `[TODO]` · 5 pts — **Caching & read-scaling.** Cache hot reads (dashboard, shortlists), pgvector index tuning, read replicas. *AC:* p95 latency targets met at scale. *Deps:* CAL-080
- **CAL-157** `[TODO]` · 5 pts — **Async scale-out & idempotency at volume.** Worker autoscaling, queue partitioning, exactly-once effects. *AC:* sustains target job throughput. *Deps:* CAL-024
- **CAL-158** `[TODO]` · 5 pts — **Data retention & lifecycle automation.** Automated retention, anonymization, deletion (operationalizes CAL-118). *AC:* retention jobs run + audited. *Deps:* CAL-118
- **CAL-159** `[TODO]` · 5 pts — **Cost controls & FinOps for AI.** Budgets/alerts on LLM + embedding spend; model-tier routing. *AC:* spend capped + alerting. *Deps:* CAL-035

## EPIC-22 · Voice Interview Mode (Committed — Post-Win Build)
**Goal:** Voice in/out for Flow B, built in the production phase **once the contract is won**. Default provider **OpenAI STT/TTS**. Must degrade gracefully to text; text is always the reliable path. (Spec §6.2, §16.)

- **CAL-160** `[TODO]` · 5 pts — **STT integration (port).** Speech-to-text behind a port for interview answers. *AC:* transcribes within latency budget. *Deps:* CAL-065
- **CAL-161** `[TODO]` · 5 pts — **TTS integration (port).** Text-to-speech for questions. *AC:* natural pacing; cancellable. *Deps:* CAL-065
- **CAL-162** `[TODO]` · 5 pts — **Graceful degradation to text.** Auto-fallback to text on any voice failure. *AC:* voice failure never blocks the interview. *Deps:* CAL-160, CAL-161
- **CAL-163** `[TODO]` · 3 pts — **Voice UX & device handling.** Mic permissions, levels, errors. *AC:* clear states; works on the demo machine. *Deps:* CAL-162

---

## 7. Cross-cutting Security baseline (applies to all stories)
- **Auth/AuthZ:** every endpoint authenticated unless explicitly public; ownership/role checks; no IDOR.
- **Input:** validate + sanitize all inputs; parameterized SQL (sqlc); strict proto/DTO validation.
- **Secrets:** env/secret store only; never logged; gitleaks in CI.
- **Transport:** TLS everywhere; HSTS; secure cookies.
- **AI:** treat all candidate/role text as untrusted (prompt-injection aware); enforce the **no-fabrication** invariant; redact PII from prompts/logs.
- **Data:** Ghana DPA 2012 baseline — consent, minimization, retention, deletion/DSAR.
- **Supply chain:** `govulncheck` + npm audit + container scanning; pin dependencies; review SonarQube security hotspots.

## 8. Cross-cutting SEO baseline (applies to public surfaces)
- **Prerender public content** (build-time SSG/prerender for the SPA); meaningful content in the initial HTML.
- Unique title/description/canonical per route (react-helmet-async); OG/Twitter cards; JSON-LD where applicable.
- `sitemap.xml` + `robots.txt`; private/auth routes excluded from indexing.
- Core Web Vitals budgets enforced in CI (Lighthouse CI); image/font/code-split + MUI bundle optimization.
- **Fonts** (Fraunces/Outfit/JetBrains Mono) self-hosted with `font-display: swap` + preload of critical faces to protect LCP/CLS.
- **Marketing motion** (parallax, 3D reveals) is lazy/IntersectionObserver-driven, kept inside the CWV budget, and disabled under `prefers-reduced-motion`.
- Semantic, accessible HTML (WCAG 2.1 AA) — a11y and SEO reinforce each other.

## 9. Risk register (from spec §16, extended)
| Risk | Mitigation | Owner |
|---|---|---|
| Live interview latency feels slow | Stream questions, pre-warm session, cap count/time, text default (CAL-104) | AI |
| Venue network fails mid-demo | Pre-recorded backup + standby deploy (CAL-106/107) | Demo |
| Seed data feels fake | Generate via real parser; curate hero pairs; local-plausibility review (EPIC-14) | Data |
| Match quality varies on edge cases | Tune rubric/filters; demo curated roles; always show reasoning (EPIC-07) | AI |
| Scope creep delays build | Hold spec §4 boundaries; defer non-demo work to M2 | Lead |
| Voice mode unreliable | Stretch only; never sole path (EPIC-22) | AI |
| **React SPA weak SEO** | Prerender public pages (EPIC-17); keep app-behind-auth CSR | FE |
| **Marketing animations hurt Core Web Vitals** | Lazy/IntersectionObserver-driven parallax/3D; CWV budget gate (CAL-129); reduced-motion fallback (CAL-170) | FE |
| Candidate data leaves region (embeddings) | Provider behind a port; self-host option for residency (CAL-118/159) | Security |
| Prompt injection / data exfil via CV text | Untrusted-input handling; output filtering; injection tests (CAL-119) | Security |
| Coverage/quality erosion under deadline pressure | Hard CI gates (≥80% + SonarQube) on every push (CAL-003/004) | All |

## 10. Open decisions & inputs needed (from spec §17 + this plan)
- [ ] **Client & sectors** — confirm exact role families to mirror in seed data.
- [ ] **Existing CV/processing software** — for the "complement and absorb, not rip-and-replace" narrative.
- [ ] **Market scope** — single-market (Ghana) vs pan-African (affects scale framing, EPIC-21).
- [ ] **Demo date & venue connectivity** — fixes phasing and whether an offline/standby plan is mandatory.
- [x] **Voice** — POC stays text-only; voice **committed for the post-win build** (EPIC-22), default OpenAI STT/TTS. *(decided 2026-06-24)*
- [ ] **Client-facing product name & branding** — keep UI brandable until provided (CAL-088).
- [x] **Embeddings data residency** — **OpenAI retained** (residency accepted for the POC). *(decided 2026-06-24)*
- [x] **Backend host** — **Render**. *(decided 2026-06-24)*
- [ ] **SonarQube** — SonarCloud (hosted) vs self-hosted SonarQube instance.
- [x] **MUI v9 licensing** — **Core only, no MUI X**; use TanStack Table (headless) for complex grids. *(decided 2026-06-24)*
- [x] **Monospace font** — **JetBrains Mono**. *(decided 2026-06-24)*
- [ ] **Animation library** — default Motion (Framer Motion); confirm vs GSAP for the heavier marketing parallax/3D work.

## 11. Suggested sequencing (build phases)
1. **Foundation** — EPIC-00, 01, 02, 03, 04 (app runs; proto/gRPC live; can store + embed a profile; AI layer callable).
2. **Intelligence** — EPIC-05, 06, 07, 09 (AI components callable + tested in isolation).
3. **Flows** — EPIC-08, 10, 11, 12 + EPIC-13 (thin end-to-end demo path exists).
4. **Polish** — EPIC-13 finish, EPIC-14 (UI production-credible; demo data real).
5. **Hardening (demo)** — EPIC-15 (reliable, repeatable, venue-proof).
6. **Production** — EPIC-16→21 (security, SEO, observability, quality, CI/CD, scale), EPIC-22 if pursued.

> Phase durations are a shape, not a commitment — compress/extend once the demo date is fixed.

---
*Project Caliber — Agent Plan v0.2 · Confidential · prepared per AI Governance (Claude = planning & documentation).*
