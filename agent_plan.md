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
| **M1 — POC Demo-Ready** | EPIC-00 | Engineering Foundations & Project Setup | 10 | 39 | WIP | ~45% |
| | EPIC-01 | Domain Model & Database Foundation | 7 | 29 | WIP | ~85% |
| | EPIC-02 | Identity, Authentication & RBAC | 7 | 31 | DONE | 100% |
| | EPIC-03 | Async Jobs & Queue Infrastructure | 5 | 21 | TODO | 0% |
| | EPIC-04 | AI Orchestration Layer | 8 | 39 | WIP | ~40% |
| | EPIC-05 | Role Spec & Rubric Generator | 5 | 24 | TODO | 0% |
| | EPIC-06 | Profile Parser & Competency Extractor | 5 | 26 | TODO | 0% |
| | EPIC-07 | Matching & Ranking Engine | 7 | 37 | WIP | ~70% |
| | EPIC-08 | Employer Intake & Explainable Shortlisting (Flow A) | 6 | 29 | WIP | ~30% |
| | EPIC-09 | AI Screening Interviewer (Flow B) | 9 | 50 | TODO | 0% |
| | EPIC-10 | Candidate Agent & Time-Advance (Flow C) | 7 | 36 | TODO | 0% |
| | EPIC-11 | Talent Radar Dashboard | 5 | 24 | TODO | 0% |
| | EPIC-12 | Trust, Explainability, Audit & Guardrails | 7 | 33 | TODO | 0% |
| | EPIC-13 | Frontend Web Application (React/Vite) | 15 | 69 | TODO | 0% |
| | EPIC-14 | Seed Data & Demo Orchestration | 6 | 28 | TODO | 0% |
| | EPIC-15 | Demo Hardening & Run-of-Show | 6 | 24 | TODO | 0% |
| **M2 — Production-Ready** | EPIC-16 | Security Hardening & Compliance | 11 | 55 | TODO | 0% |
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
| 3 | CAL-005 | Configuration & secrets management | **WIP** — typed env config + `.env.example` done; gitleaks secret-scan pending (CI) |
| 4 | CAL-006 | Dockerization & local dev stack | TODO |
| 5 | CAL-007 | Structured logging & error baseline | **WIP** — slog JSON + recovery middleware done; typed domain errors pending |
| 6 | CAL-008 | Health, readiness & server bootstrap | **WIP** — `/healthz` `/readyz` + graceful shutdown done; readiness→DB/Redis pending |
| 7 | CAL-002 | CLAUDE.md & AGENTS.md | **DONE** |
| 8 | CAL-003 | CI pipeline (lint/test/coverage gate) | **DONE** — workflow authored; all gates reproduced locally; first GitHub run pending remote |
| 9 | CAL-004 | SonarQube quality gate | **WIP** — `sonar-project.properties` + CI step done; needs SonarCloud project + `SONAR_TOKEN` secret |
| 10 | CAL-009 | Branch protection & repo policy | TODO — needs GitHub remote |

**Sprint 2 (next)** — EPIC-01 (domain + schema + pgvector), EPIC-02 (auth), EPIC-03 (queue), EPIC-04 (AI orchestration): the intelligence substrate becomes callable.

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
- **CAL-005** `[WIP]` · 3 pts — **Configuration & secrets management.** Typed config loader (env-driven), `.env.example`, no secrets in VCS; fail-fast on missing required vars; gitleaks in CI. *AC:* config validated at boot. *Deps:* CAL-001
- **CAL-006** `[TODO]` · 5 pts — **Dockerization & local dev stack.** Multi-stage Dockerfiles for `api`/`worker`; `docker-compose` with Postgres+pgvector and Redis; Vite dev server wired. *AC:* `docker compose up` boots the full local stack. *Deps:* CAL-001
- **CAL-007** `[WIP]` · 3 pts — **Structured logging & error handling baseline.** `slog` JSON logger, request-scoped logger, typed domain errors, panic-recovery middleware/interceptor. *AC:* every request logs a correlation/request id. *Deps:* CAL-001
- **CAL-008** `[WIP]` · 5 pts — **Health, readiness & server bootstrap.** chi server with `/healthz`, `/readyz`, graceful shutdown, timeouts, DI wiring in `platform`. *AC:* readiness reflects DB+Redis connectivity. *Deps:* CAL-006
- **CAL-009** `[TODO]` · 3 pts — **Branch protection & repo policy.** Protect `main`; require CI + Sonar + 1 review; CODEOWNERS; PR template embedding the DoD checklist. *AC:* direct pushes blocked. *Deps:* CAL-003, CAL-004

## EPIC-01 · Domain Model & Database Foundation
**Goal:** The entities of spec §9 as a pure domain plus a migrated Postgres schema with pgvector.

- **CAL-010** `[DONE]` · 5 pts — **Domain entities & value objects.** `User, Employer, Role, RoleSpec, Rubric, Candidate, TalentProfile/Passport, Match, Application, Interview, InterviewTurn, AuditLog` as pure Go types with invariants. *AC:* no infra imports; unit-tested invariants. *Deps:* CAL-001
- **CAL-011** `[DONE]` · 3 pts — **Repository ports.** Define `*Repository` interfaces in `domain`. *AC:* application layer depends only on ports. *Deps:* CAL-010
- **CAL-012** `[WIP]` · 5 pts — **goose migration tooling & base schema.** goose migrations; relational schema; JSON columns for `role_spec`, `rubric`, `report_card`, `breakdown`. *AC:* up/down migrations run in CI. *Deps:* CAL-006
- **CAL-013** `[WIP]` · 3 pts — **Enable pgvector & embedding columns.** `vector` extension; `role_embedding`, `profile_embedding`; ivfflat/hnsw index. *AC:* vector similarity query returns ordered results. *Deps:* CAL-012
- **CAL-014** `[DONE]` · 5 pts — **sqlc queries & Postgres repository adapters.** Implement ports with sqlc+pgx; transactions via a `UnitOfWork`. *AC:* repository integration tests against real Postgres (testcontainers). *Deps:* CAL-011, CAL-012
- **CAL-015** `[DONE]` · 3 pts — **Audit log persistence.** Append-only `audit_log` (actor, action, entity, before/after, timestamp). *AC:* writes immutable; covered by tests. *Deps:* CAL-014
- **CAL-016** `[TODO]` · 5 pts — **Seed-ready fixtures & factory helpers.** Deterministic data factories for every entity. *AC:* reused by integration tests and EPIC-14. *Deps:* CAL-014

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

- **CAL-024** `[TODO]` · 5 pts — **Asynq client/server wiring.** `worker` entrypoint; `TaskDispatcher` port; queues with priorities. *AC:* enqueue→process round-trip tested. *Deps:* CAL-006, CAL-008
- **CAL-025** `[TODO]` · 3 pts — **Idempotent job handler framework.** Base handler with idempotency keys, structured logging, otel spans. *AC:* duplicate delivery does not double-apply. *Deps:* CAL-024
- **CAL-026** `[TODO]` · 5 pts — **Retry, backoff & dead-letter handling.** Per-task retry policy, max-retry → archive, alerting hook. *AC:* failing task lands in archive after policy; visible. *Deps:* CAL-025
- **CAL-027** `[TODO]` · 3 pts — **Scheduled / delayed tasks.** Support deferred enqueue (time-advance & re-matching). *AC:* delayed task fires on time in tests. *Deps:* CAL-024
- **CAL-028** `[TODO]` · 5 pts — **Asynqmon dashboard & ops.** Mount monitoring UI (protected); operational runbook. *AC:* queue depth/failures observable. *Deps:* CAL-024

## EPIC-04 · AI Orchestration Layer
**Goal:** All model interaction behind one clean module: prompt assembly, the Claude gateway, schema-validated structured outputs, embeddings, cost/latency controls. Prompts & rubrics are versioned product, not config.

- **CAL-029** `[DONE]` · 3 pts — **`LLMClient` port & message types.** Provider-agnostic interface (complete, stream, tool/JSON modes). *AC:* domain/app depend only on the port. *Deps:* CAL-001
- **CAL-030** `[DONE]` · 5 pts — **Anthropic Claude gateway adapter.** Implement `LLMClient` with the Anthropic Go SDK; timeouts, retries, context cancellation. *AC:* live + mocked tests; configurable model. *Deps:* CAL-029
- **CAL-031** `[TODO]` · 5 pts — **Structured-output enforcement.** Strict JSON-schema validation of model output with bounded re-ask on violation. *AC:* malformed output retried, then typed error. *Deps:* CAL-030
- **CAL-032** `[TODO]` · 3 pts — **Versioned prompt registry.** `prompts/` loaded with version tags; prompts in VCS, referenced by id. *AC:* prompt version recorded on each call. *Deps:* CAL-030
- **CAL-033** `[DONE]` · 3 pts — **`Embedder` port + OpenAI adapter.** text-embedding-3-small behind the port; batch support. *AC:* embeddings stored in pgvector; provider swappable. *Deps:* CAL-013, CAL-029
- **CAL-034** `[TODO]` · 5 pts — **Streaming support.** Token/event streaming surfaced to inbound (gRPC server-stream / SSE) for the interview. *AC:* stream cancellable; backpressure handled. *Deps:* CAL-030
- **CAL-035** `[TODO]` · 5 pts — **Cost, rate-limit & guardrail controls.** Per-call token caps, request budgets, concurrency limits, prompt-injection-aware input handling. *AC:* limits enforced; usage metered. *Deps:* CAL-030
- **CAL-036** `[TODO]` · 5 pts — **AI call audit & observability.** Persist prompt id/version, model, latency, tokens, redacted I/O for explainability & debugging. *AC:* every model call traceable. *Deps:* CAL-030, CAL-015

## EPIC-05 · Role Spec & Rubric Generator (Flow A.1)
**Goal:** Turn a hiring manager's messy sentence into a structured, editable **Role Spec** + weighted **Rubric** + suggested salary band. (Spec §8.1, Appendix A.1.)

- **CAL-037** `[TODO]` · 5 pts — **Role Spec generation use-case.** Free text → Role Spec JSON (title, location, seniority, availability, responsibilities, must/nice-to-haves, salary band). *AC:* matches Appendix A.1 contract. *Deps:* CAL-031, CAL-032
- **CAL-038** `[TODO]` · 5 pts — **Weighted rubric generation.** Competencies with weights + must-have flags. *AC:* valid, normalized weights; deterministic schema. *Deps:* CAL-037
- **CAL-039** `[TODO]` · 3 pts — **Salary-band lookup over seeded market data.** Simple lookup for realism (Ghana market). *AC:* band returned in role currency. *Deps:* CAL-037, CAL-016
- **CAL-040** `[TODO]` · 5 pts — **Editable spec/rubric RPCs + re-weighting.** Persist; allow field edits & weight changes that trigger re-rank. *AC:* edits persisted and audited. *Deps:* CAL-037, CAL-014
- **CAL-041** `[TODO]` · 3 pts — **Spec embedding on save.** Embed the role spec for recall. *AC:* `role_embedding` populated. *Deps:* CAL-033, CAL-040

## EPIC-06 · Profile Parser & Competency Extractor
**Goal:** Convert a CV + intake answers into a structured competency profile with evidence tied back to source text. (Spec §8.2.)

- **CAL-042** `[TODO]` · 5 pts — **CV ingestion (file/text).** Upload + parse PDF/DOCX/plain text to clean text. *AC:* common formats handled; size/type validated. *Deps:* CAL-020
- **CAL-043** `[TODO]` · 5 pts — **Competency extraction use-case.** Text → structured profile JSON (competencies, seniority, history). *AC:* fixed schema; covered by tests. *Deps:* CAL-031
- **CAL-044** `[TODO]` · 5 pts — **Evidence-linking.** Each extracted competency cites its CV source span. *AC:* recruiter can see source of each claim. *Deps:* CAL-043
- **CAL-045** `[TODO]` · 5 pts — **Profile embedding + Talent Profile persistence.** Store structured profile + summary embedding. *AC:* `TalentProfile` + `profile_embedding` written. *Deps:* CAL-033, CAL-014
- **CAL-046** `[TODO]` · 3 pts — **Guided intake answers.** Capture target titles, location, salary floor, deal-breakers; merge into profile. *AC:* intake feeds matching filters. *Deps:* CAL-043

## EPIC-07 · Matching & Ranking Engine
**Goal:** Rank candidates against a Role Spec with scores a human can trust — recall → precision → hard filters. (Spec §8.3, Appendix A.2.)

- **CAL-047** `[DONE]` · 5 pts — **Stage 1: vector recall.** pgvector cosine similarity role↔candidate top-N (`Recaller` raw `$1::vector` query, testcontainers ordering test). *AC:* top-N returned, ordered, paged. *Deps:* CAL-041, CAL-045
- **CAL-048** `[DONE]` · 8 pts — **Stage 2: rubric-based LLM scoring.** Per candidate, 0–5 per competency with evidence quote, overall fit, confidence. *AC:* output matches Appendix A.2 `breakdown`. *Deps:* CAL-047, CAL-031
- **CAL-049** `[DONE]` · 5 pts — **Stage 3: hard filters as gates.** Bias-safe `Requirements` gates: location (token-matched, remote-aware), salary-floor (currency-safe), and must-have competency (excludes only on a present-but-underscored signal — absence routes to human review, never a fabricated rejection). Each exclusion surfaced with a reason via `Shortlist.exclusions`. Logistical gates run pre-scoring (skip LLM cost). *AC:* gated-out candidates excluded with reason. *Deps:* CAL-048
- **CAL-050** `[DONE]` · 5 pts — **Match assembly & persistence.** Build `Match` (overall_score, breakdown, rationale, watch_outs, thin_evidence_flag). *AC:* matches Appendix A.2; persisted. *Deps:* CAL-049, CAL-014
- **CAL-051** `[TODO]` · 5 pts — **Live re-ranking on criteria change.** Editing must-have/weight/location re-ranks the shortlist. *AC:* re-rank ≤ acceptable latency; correct order. *Deps:* CAL-050, CAL-040
- **CAL-052** `[DONE]` · 5 pts — **Bias-safe ranking guard.** Rubric-driven only; protected attributes excluded from scoring inputs. *AC:* automated test asserts protected fields never reach the scorer. *Deps:* CAL-048
- **CAL-053** `[TODO]` · 4 pts — **Two-way matching (role↔candidate).** Surface roles fitting a passive candidate (feeds Radar alerts). *AC:* both directions queryable. *Deps:* CAL-047

## EPIC-08 · Employer Intake & Explainable Shortlisting (Flow A)
**Goal:** End-to-end Flow A: messy sentence in → structured spec, rubric, explainable ranked shortlist out, in seconds. (Spec §6.1.)

- **CAL-054** `[DONE]` · 5 pts — **Flow A orchestration use-case.** `Shortlister` wires recall → logistical gates → rubric scoring → must-have gate → ranked Matches (+ surfaced exclusions); exposed via `MatchingService.GenerateShortlist` (gRPC + REST) and wired in `main` when a DB is configured. *AC:* single call produces a shortlist. *Deps:* CAL-040, CAL-050
- **CAL-055** `[WIP]` · 3 pts — **Instant availability signal.** "N strong matches already in your pool." `Shortlist.pool_depth` returned in the response. *AC:* pool depth returned immediately after spec. *Deps:* CAL-047
- **CAL-056** `[TODO]` · 5 pts — **Explainable, paginated shortlist response.** Each candidate: fit score, per-competency breakdown, plain-English "why," watch-outs, thin-evidence flag; results paginated. *AC:* contract locked; no black-box fields. *Deps:* CAL-050, CAL-082
- **CAL-057** `[TODO]` · 3 pts — **Refine RPC.** Tighten criteria / add skill → live re-rank. *AC:* shortlist updates correctly. *Deps:* CAL-051
- **CAL-058** `[TODO]` · 5 pts — **Flow A proto contract & gateway.** gRPC service + REST gateway + OpenAPI; field names locked from Appendix A. *AC:* documented, validated, versioned. *Deps:* CAL-054, CAL-164
- **CAL-059** `[TODO]` · 8 pts — **Flow A integration tests (demo beat).** Messy sentence → spec+rubric+ranked explainable shortlist on seed data. *AC:* acceptance criteria §15.1 pass. *Deps:* CAL-054, CAL-016

## EPIC-09 · AI Screening Interviewer (Flow B — centrepiece)
**Goal:** A short, adaptive interview that probes claimed competencies and returns a scored, evidence-tagged report card. The moment manual interviewing labour visibly disappears. (Spec §8.4, §6.2, Appendix A.3.)

- **CAL-060** `[TODO]` · 8 pts — **Interview state machine (FSM).** States: open → ask → analyze → adapt → … → close; max-K questions or T-minutes cap. *AC:* deterministic transitions; unit-tested. *Deps:* CAL-030
- **CAL-061** `[TODO]` · 5 pts — **Opening-question generation.** From rubric + profile. *AC:* question ties to a rubric competency. *Deps:* CAL-060, CAL-038
- **CAL-062** `[TODO]` · 8 pts — **Adaptive questioning loop.** Analyze each answer → update per-competency evidence coverage → select next question probing weakest/most-claimed competency, with follow-ups. *AC:* questions adapt to prior answers (not a fixed script). *Deps:* CAL-061
- **CAL-063** `[TODO]` · 5 pts — **Honest-signal pressure.** Detect vague/evasive answers; push for concrete examples. *AC:* evasive answers flagged in transcript. *Deps:* CAL-062
- **CAL-064** `[TODO]` · 8 pts — **Scored report card generation.** Per-competency scores + evidence quote each, overall verdict, confidence, recommended next step. *AC:* matches Appendix A.3; every score cites a transcript quote. *Deps:* CAL-062
- **CAL-065** `[TODO]` · 5 pts — **Streamed interview session.** Stream questions/turns (gRPC server-stream / SSE); low-latency; pre-warm session. *AC:* turns render live; cancellable. *Deps:* CAL-034, CAL-060
- **CAL-066** `[TODO]` · 3 pts — **Transcript & report card persistence + Passport update.** Store `Interview`, `InterviewTurn`s, report card; update Talent Passport. *AC:* transcript + card stored and viewable. *Deps:* CAL-064, CAL-014
- **CAL-067** `[TODO]` · 5 pts — **Async interview scoring job.** Heavy scoring via Asynq when not inline. *AC:* report card produced reliably off the request path. *Deps:* CAL-025, CAL-064
- **CAL-068** `[TODO]` · 8 pts — **Flow B acceptance tests (centrepiece).** Adaptive (not scripted), per-competency scores with evidence + verdict + confidence, Passport updated. *AC:* §15.2 pass; latency within demo budget. *Deps:* CAL-064, CAL-065

## EPIC-10 · Candidate Agent & Time-Advance (Flow C)
**Goal:** The agent that "works while you sleep, honestly" — matches, tailors, submits and screens using only verified profile content; demoed via a controlled time-advance. (Spec §8.5, §6.3.)

- **CAL-069** `[TODO]` · 3 pts — **One-time candidate setup.** CV upload + guided intake → initial profile. *AC:* usable profile from CV + intake. *Deps:* CAL-042, CAL-046
- **CAL-070** `[TODO]` · 8 pts — **Candidate-agent job (autonomous loop).** Scan open roles → score fit (reuse EPIC-07) → hard filters → for strong matches, tailor a truthful application. *AC:* runs as an Asynq job over the seeded role pool. *Deps:* CAL-050, CAL-025
- **CAL-071** `[TODO]` · 5 pts — **No-fabrication guardrail (hard invariant).** Agent may only surface/rephrase verified profile content; never invents skills/experience. *AC:* asserted in code AND prompt; test proves tailored content traces to profile. *Deps:* CAL-070
- **CAL-072** `[TODO]` · 5 pts — **Application tailoring & submission (in-platform).** Generate role-specific application from verified content; submit within the platform; optionally complete/queue screening. *AC:* `Application{source: agent, tailored_summary, status}` written. *Deps:* CAL-070
- **CAL-073** `[TODO]` · 5 pts — **Time-advance action (demo engine).** Controlled "run overnight" advances agent state live — no real external submission, no waiting. *AC:* one action produces visible new state. *Deps:* CAL-027, CAL-072
- **CAL-074** `[TODO]` · 3 pts — **Wake-up view data.** Summary: new matches, applications tailored/submitted, completed screening + score, employer interest. *AC:* matches the §6.3 wake-up narrative. *Deps:* CAL-073
- **CAL-075** `[TODO]` · 7 pts — **Flow C acceptance tests.** Setup builds a usable profile; time-advance yields tailored applications + ≥1 completed screening; **no application content untraceable to the verified profile**. *AC:* §15.3 pass. *Deps:* CAL-072, CAL-071

## EPIC-11 · Talent Radar Dashboard
**Goal:** The god-view that frames the whole demo: live pool, supply/demand snapshot, two-way alerts, and the headline time-to-shortlist metric dropping weeks → hours. (Spec §6.4.)

- **CAL-076** `[TODO]` · 5 pts — **Live, paginated candidate pool view.** Aggregated pool with passport status. *AC:* reflects current seed state; paginated. *Deps:* CAL-045
- **CAL-077** `[TODO]` · 5 pts — **Supply/demand snapshot by role family.** Counts and gaps per role family. *AC:* numbers reconcile with seed data. *Deps:* CAL-076
- **CAL-078** `[TODO]` · 5 pts — **Two-way match alerts.** "New strong candidate for an open role" / "new role fits a passive candidate." *AC:* alerts generated from EPIC-07 two-way matching; paginated. *Deps:* CAL-053
- **CAL-079** `[TODO]` · 5 pts — **Time-to-shortlist metric.** Headline metric showing collapse from weeks → hours. *AC:* computed and displayed as the closing visual. *Deps:* CAL-059
- **CAL-080** `[TODO]` · 4 pts — **Dashboard aggregation performance.** Cache/precompute snapshots for snappy live rendering. *AC:* dashboard loads within demo budget. *Deps:* CAL-076

## EPIC-12 · Trust, Explainability, Audit & Guardrails
**Goal:** Demonstrable features (not disclaimers) that let the client sell to enterprise/public-sector buyers later. (Spec §11.)

- **CAL-081** `[TODO]` · 5 pts — **Human-approval gate before any rejection.** AI ranks/screens but never auto-rejects; a human approves declines, logged. *AC:* no rejection without a logged human approval. *Deps:* CAL-015, CAL-021
- **CAL-082** `[TODO]` · 5 pts — **Explanation/rationale generator (cross-cutting).** Plain-English "why this person" + "watch-outs" derived from structured scores/evidence. *AC:* words trace back to rubric + data. *Deps:* CAL-050
- **CAL-083** `[TODO]` · 5 pts — **Candidate visibility & contest.** A candidate can view their assessment and flag/contest it. *AC:* surfaced as a fairness feature in the demo. *Deps:* CAL-066
- **CAL-084** `[TODO]` · 3 pts — **Audit trail surfacing.** Approvals, overrides, agent actions recorded and viewable (paginated). *AC:* AuditLog browsable per entity. *Deps:* CAL-015
- **CAL-085** `[TODO]` · 5 pts — **Bias & fairness checks.** Tests + UI assertion that scores never depend on protected attributes; document methodology. *AC:* fairness test suite green. *Deps:* CAL-052
- **CAL-086** `[TODO]` · 5 pts — **Data-protection baseline (Ghana DPA 2012).** Consent capture, data-minimization, deletion design (even if not fully built in POC), PII handling policy. *AC:* consent + deletion paths designed and stubbed; documented. *Deps:* CAL-014
- **CAL-087** `[TODO]` · 5 pts — **Explainability contract.** Every score/shortlist position exposes its reasoning + evidence to the frontend. *AC:* no black-box fields in any API/proto response. *Deps:* CAL-056, CAL-064

## EPIC-13 · Frontend Web Application (React + Vite)
**Goal:** Brandable React (Vite) SPA with MUI v9, employer & candidate views, the streamed interview UI, and the Talent Radar dashboard. Skeleton loading and pagination throughout; SEO-ready public pages via prerender (EPIC-17).

- **CAL-088** `[TODO]` · 5 pts — **React+Vite scaffold + MUI v9 theme + typography.** Vite app, react-router, **MUI v9** themed design system with brandable tokens (Primary Blue #0066CC, Ink #111418, Slate #6B7280); typography wired to **Fraunces** (titles), **Outfit** (body), **JetBrains Mono** (statuses), self-hosted with `font-display: swap`; light/dark color modes ready. *AC:* design tokens + fonts centralized; no Tailwind. *Deps:* CAL-164
- **CAL-167** `[TODO]` · 3 pts — **App shell, routing & Zustand stores.** Layout, role-aware routes, Zustand stores for UI/auth/wizard state. *AC:* navigation + protected routes work. *Deps:* CAL-088
- **CAL-095** `[TODO]` · 5 pts — **API client (gRPC-web/REST) + TanStack Query + streaming.** Typed client from proto; TanStack Query setup; stream handling for the interview; resilient error states. *AC:* resilient to slow/failed calls. *Deps:* CAL-058
- **CAL-165** `[TODO]` · 3 pts — **Skeleton-loading system (content).** Reusable MUI `Skeleton` components shaped per surface (list rows, cards, dashboard tiles, report card, interview turns). *AC:* no spinners/"Loading…" text for content; lint/check guards against them. *Deps:* CAL-088
- **CAL-168** `[TODO]` · 5 pts — **Animation system (Motion): layout transitions + animated-dots buttons.** Install Motion (Framer Motion); app-wide layout/route/list transitions (incl. live shortlist re-rank); reusable **animated-dots** button-loading component (width-stable, no spinners); all gated behind `prefers-reduced-motion`. *AC:* buttons show dots when busy; layout changes animate; reduced-motion respected. *Deps:* CAL-088
- **CAL-169** `[TODO]` · 3 pts — **Circular-reveal light/dark theme toggle.** MUI color-mode toggle animated as a circular reveal from the control (View Transitions API + clip-path fallback); persisted preference. *AC:* theme switches with circular reveal; falls back cleanly; reduced-motion respected. *Deps:* CAL-088
- **CAL-166** `[TODO]` · 3 pts — **Pagination system (standard).** Reusable paginated-query hooks (TanStack Query, `keepPreviousData`) + MUI pagination controls, applied to every list. *AC:* no unbounded lists; pages map to server pages. *Deps:* CAL-095
- **CAL-089** `[TODO]` · 5 pts — **Auth UI & session handling.** Login/register, role-aware routing, secure token storage, refresh. *AC:* both roles reach their views behind login. *Deps:* CAL-167
- **CAL-090** `[TODO]` · 8 pts — **Employer view — Flow A UI.** Plain-language intake, editable spec/rubric, instant availability, explainable **paginated** ranked shortlist with live refine. *AC:* §15.1 visible end-to-end. *Deps:* CAL-058, CAL-166
- **CAL-091** `[TODO]` · 8 pts — **Interview UI — Flow B (centrepiece).** Streamed adaptive Q&A (skeletons between turns), evidence-tagged report card reveal; graceful, low-latency. *AC:* live adaptive interview renders + scored card. *Deps:* CAL-065, CAL-165
- **CAL-092** `[TODO]` · 8 pts — **Candidate view — Flow C UI.** One-time setup, time-advance ("run overnight"), wake-up view. *AC:* §15.3 visible end-to-end. *Deps:* CAL-073
- **CAL-093** `[TODO]` · 8 pts — **Talent Radar dashboard UI.** Live pool, supply/demand, two-way alerts, time-to-shortlist headline (the closing visual); skeleton tiles + paginated lists. *AC:* §15.4 visible. *Deps:* CAL-079, CAL-165, CAL-166
- **CAL-094** `[TODO]` · 5 pts — **Explainability & trust UI.** Per-score reasoning, watch-outs, thin-evidence flags, candidate contest, human-approval gate surfaced. *AC:* nothing reads as a black box. *Deps:* CAL-087
- **CAL-096** `[TODO]` · 5 pts — **Accessibility baseline (WCAG 2.1 AA).** Semantic HTML, keyboard nav, focus, contrast, ARIA for streaming + skeletons, and **`prefers-reduced-motion`** honored across all transitions/parallax/3D effects. *AC:* axe checks pass on key screens; reduced-motion verified. *Deps:* CAL-088, CAL-168
- **CAL-097** `[TODO]` · 3 pts — **Responsive & demo-resilient layout.** Production-credible on the demo screen/resolution. *AC:* no layout breakage at target resolutions. *Deps:* CAL-088

## EPIC-14 · Seed Data & Demo Orchestration
**Goal:** A believable, locally-plausible (Ghana/West Africa) pool the demo lives on. (Spec §10.)

- **CAL-098** `[TODO]` · 5 pts — **Seed generation pipeline.** LLM-generate ~50–60 realistic CVs/profiles, 6–8 employers, 8–12 roles; run through the *real* parser. *AC:* data produced by the real pipeline. *Deps:* CAL-043, CAL-037
- **CAL-099** `[TODO]` · 5 pts — **Local plausibility curation.** Names, institutions, locations, roles read as locally credible (Ghana / West Africa). *AC:* review pass before demo. *Deps:* CAL-098
- **CAL-100** `[TODO]` · 5 pts — **Hero candidate/role pairs.** Engineer pairs that produce excellent, legible matches so Flow A always lands; keep the rest varied. *AC:* hero pairs deterministic. *Deps:* CAL-098
- **CAL-101** `[TODO]` · 3 pts — **Pre-run interviews.** Pre-generate report cards for several candidates; leave 1–2 to run live in Flow B. *AC:* shortlists show real assessments. *Deps:* CAL-064
- **CAL-102** `[TODO]` · 5 pts — **Seeded application/agent state.** Pre-seed agent state so time-advance produces a crisp wake-up view. *AC:* Flow C demo state ready. *Deps:* CAL-072
- **CAL-103** `[TODO]` · 5 pts — **Reseed/reset command.** One command to wipe + reseed to a known demo state. *AC:* deterministic, repeatable. *Deps:* CAL-098

## EPIC-15 · Demo Hardening & Run-of-Show
**Goal:** Make the demo reliable, repeatable, venue-proof. (Spec §13 Phase 5, §14, §16.)

- **CAL-104** `[TODO]` · 5 pts — **Latency tuning & session pre-warm.** Cap question count/time; pre-warm LLM sessions; stream everything. *AC:* interview + shortlist feel instant. *Deps:* CAL-065, CAL-068
- **CAL-105** `[TODO]` · 3 pts — **Run-of-show wiring.** Sequence: Frame → Flow A → Flow B → Flow C → close on dashboard. *AC:* one path drives the whole narrative. *Deps:* CAL-090, CAL-091, CAL-092, CAL-093
- **CAL-106** `[TODO]` · 5 pts — **Pre-recorded backup capture.** Clean live-style interview recording as insurance for venue network failure. *AC:* recording ready; live path primary. *Deps:* CAL-091
- **CAL-107** `[TODO]` · 5 pts — **Offline/standby deployment fallback.** Local/standby deployment where feasible. *AC:* demo survives a network drop. *Deps:* CAL-006
- **CAL-108** `[TODO]` · 3 pts — **Full dry run + acceptance sweep.** Verify all §15 acceptance criteria on seed data in one rehearsal. *AC:* every §15 item passes. *Deps:* CAL-059, CAL-068, CAL-075, CAL-093
- **CAL-109** `[TODO]` · 3 pts — **Demo runbook & failure playbook.** Written run-of-show, reset steps, fallback triggers. *AC:* any team member can drive it. *Deps:* CAL-103, CAL-105

---

# MILESTONE 2 — Production-Ready

Beyond the win: harden security, SEO, observability, quality, deployment, and scale. (Spec defers these to "the build phase that follows the win" — captured here so nothing is forgotten.)

## EPIC-16 · Security Hardening & Compliance
**Goal:** Defensible to enterprise clients and regulators. OWASP-aligned, Ghana DPA-compliant, audited.

- **CAL-110** `[TODO]` · 5 pts — **Threat model & security requirements.** STRIDE over the architecture; security backlog. *AC:* documented threat model. *Deps:* —
- **CAL-111** `[TODO]` · 5 pts — **Input validation & output encoding everywhere.** Proto/DTO validation, parameterized SQL (sqlc), XSS-safe rendering. *AC:* OWASP A03 checks pass. *Deps:* CAL-058
- **CAL-112** `[TODO]` · 5 pts — **Rate limiting, throttling & abuse protection.** Per-IP/user/endpoint limits; expensive AI endpoints protected; bot mitigation. *AC:* limits enforced + tested. *Deps:* CAL-021
- **CAL-113** `[TODO]` · 5 pts — **Secrets management & rotation.** Platform secret store, rotation policy, no secrets in logs; gitleaks gate extended. *AC:* secret scan clean; rotation documented. *Deps:* CAL-005
- **CAL-114** `[TODO]` · 5 pts — **Security headers, TLS & CORS.** HSTS, CSP, X-Frame-Options, strict CORS, TLS everywhere. *AC:* securityheaders/observatory grade A. *Deps:* CAL-088
- **CAL-115** `[TODO]` · 5 pts — **Dependency & container scanning.** `govulncheck`, Trivy/Grype, npm audit, Dependabot in CI. *AC:* no high/critical vulns merge. *Deps:* CAL-003
- **CAL-116** `[TODO]` · 5 pts — **AuthZ hardening & least privilege.** Full ownership checks, IDOR tests, least privilege across services. *AC:* IDOR test suite green. *Deps:* CAL-021
- **CAL-117** `[TODO]` · 5 pts — **PII protection & encryption.** Encrypt sensitive data at rest, field-level where needed, PII redaction in logs/telemetry. *AC:* no PII in logs; encryption verified. *Deps:* CAL-036
- **CAL-118** `[TODO]` · 5 pts — **Ghana Data Protection Act 2012 compliance.** Consent records, lawful basis, retention schedule, **DSAR + deletion** flows, processor agreements. *AC:* DSAR + deletion functional. *Deps:* CAL-086
- **CAL-119** `[TODO]` · 5 pts — **LLM/prompt-injection & data-exfil defenses.** Treat candidate/role text as untrusted; system-prompt isolation; output filtering; no-fabrication invariant tests (extends CAL-071). *AC:* injection test corpus passes. *Deps:* CAL-035
- **CAL-120** `[TODO]` · 5 pts — **Security review & pen-test prep.** Run `/security-review`, remediate; prepare for external pen test; SonarQube security hotspots cleared. *AC:* no open high findings. *Deps:* all EPIC-16

## EPIC-17 · SEO & Web Performance
**Goal:** Discoverable, fast, share-ready public surface from a React SPA. (Marketing/landing + any public talent/role pages.)

- **CAL-121** `[TODO]` · 5 pts — **Prerender pipeline for public pages.** Build-time prerender (e.g. vite-plugin-ssg / react-snap / prerendering) so public/marketing/role pages ship crawlable HTML; app behind auth stays CSR. *AC:* public pages contain content in initial HTML. *Deps:* CAL-088
- **CAL-122** `[TODO]` · 3 pts — **Metadata & Open Graph/Twitter cards.** Per-route titles/descriptions/canonical via a head manager (react-helmet-async); OG/Twitter tags. *AC:* rich preview on share; unique titles per page. *Deps:* CAL-121
- **CAL-123** `[TODO]` · 5 pts — **Structured data (JSON-LD).** `Organization`, and where applicable `JobPosting`/`Occupation` schema for role pages. *AC:* validates in Rich Results Test. *Deps:* CAL-121
- **CAL-124** `[TODO]` · 3 pts — **Sitemap & robots.** Dynamic `sitemap.xml`, `robots.txt`; auth routes excluded from indexing. *AC:* sitemap submitted; private routes disallowed. *Deps:* CAL-121
- **CAL-125** `[TODO]` · 5 pts — **Core Web Vitals optimization.** LCP/INP/CLS budgets; image optimization, font loading, code splitting/lazy routes, caching, MUI bundle trimming. *AC:* Lighthouse ≥ 90 perf on key pages. *Deps:* CAL-088
- **CAL-126** `[TODO]` · 5 pts — **Semantic HTML & a11y for SEO.** Heading hierarchy, landmarks, alt text (reinforces CAL-096). *AC:* no critical Lighthouse SEO/a11y issues. *Deps:* CAL-096
- **CAL-127** `[TODO]` · 3 pts — **Internationalization & localization readiness.** hreflang scaffolding, locale-aware routing (Ghana/West Africa first). *AC:* i18n structure in place. *Deps:* CAL-121
- **CAL-128** `[TODO]` · 4 pts — **Analytics & Search Console.** Privacy-respecting analytics, Web Vitals reporting, Search Console verification. *AC:* traffic + vitals visible. *Deps:* CAL-121
- **CAL-129** `[TODO]` · 5 pts — **Performance budgets in CI.** Lighthouse CI gate on PRs for public pages. *AC:* regressions block merge. *Deps:* CAL-125, CAL-003
- **CAL-170** `[TODO]` · 5 pts — **Marketing-site animation kit.** Parallax sections, 3D reveal-on-scroll, and the circular-reveal theme toggle on public/marketing pages — built with Motion, lazy/IntersectionObserver-driven, within the Core Web Vitals budget (CAL-125) and gated behind `prefers-reduced-motion`. *AC:* effects render; Lighthouse perf budget still met; reduced-motion disables them. *Deps:* CAL-121, CAL-125, CAL-168

## EPIC-18 · Observability & Operations
**Goal:** See everything in production. OpenTelemetry + Prometheus/Grafana/Loki.

- **CAL-130** `[TODO]` · 5 pts — **OpenTelemetry tracing.** Instrument gRPC/HTTP, DB, queue, and LLM calls with spans + context propagation. *AC:* end-to-end trace for a request. *Deps:* CAL-007
- **CAL-131** `[TODO]` · 5 pts — **Metrics (Prometheus).** RED/USE metrics, AI cost/latency/token metrics, queue depth, business KPIs (time-to-shortlist). *AC:* dashboards populate. *Deps:* CAL-130
- **CAL-132** `[TODO]` · 5 pts — **Centralized logging (Loki).** Ship structured logs; correlate via trace id; PII-safe (ties CAL-117). *AC:* logs searchable by request/trace id. *Deps:* CAL-007
- **CAL-133** `[TODO]` · 5 pts — **Grafana dashboards.** Service health, AI usage/cost, queue health, SLO dashboards. *AC:* on-call can triage from dashboards. *Deps:* CAL-131
- **CAL-134** `[TODO]` · 5 pts — **Alerting & SLOs.** Define SLOs (availability, latency, error rate, AI failure rate); alert routing. *AC:* alerts fire on breach. *Deps:* CAL-133
- **CAL-135** `[TODO]` · 3 pts — **Error tracking & on-call runbooks.** Error grouping; incident runbooks. *AC:* known failure modes documented. *Deps:* CAL-132
- **CAL-136** `[TODO]` · 4 pts — **Audit & compliance reporting.** Reportable audit-log views (approvals/overrides/agent actions). *AC:* exportable audit reports. *Deps:* CAL-084
- **CAL-137** `[TODO]` · 5 pts — **AI quality monitoring.** Track structured-output failure rate, refusal/latency, guardrail trips; eval harness. *AC:* AI regressions visible. *Deps:* CAL-036

## EPIC-19 · Quality, Testing & Performance Engineering
**Goal:** The ≥80% gate is the floor; build the full pyramid and prove it scales.

- **CAL-138** `[TODO]` · 5 pts — **Test pyramid standards.** Unit (domain), integration (adapters via testcontainers), contract (proto), e2e (Playwright) — documented + enforced. *AC:* standards in CLAUDE.md; CI runs each layer. *Deps:* CAL-003
- **CAL-139** `[TODO]` · 5 pts — **Coverage enforcement & reporting.** Per-package ≥80% gate (Go + web), trend reporting, no-untested-merge. *AC:* gate enforced on every push. *Deps:* CAL-003
- **CAL-140** `[TODO]` · 5 pts — **Deterministic AI testing.** Golden tests with mocked LLM/embeddings; live smoke tests behind a flag. *AC:* AI logic testable without network. *Deps:* CAL-030
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
