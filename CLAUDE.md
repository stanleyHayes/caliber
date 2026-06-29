# CLAUDE.md — Project Caliber operating rules

Canonical AI + contributor rules for this repo. Read this before changing code.
Planning and progress live in [agent_plan.md](agent_plan.md) (we use it in lieu of Jira).

## What this is
Project Caliber is a **Talent Intelligence Platform** POC (see `Caliber_POC_Build_Spec.pdf`).
Three flows — employer explainable shortlisting (A), AI screening interview (B, centrepiece),
candidate autonomous agent (C) — plus a Talent Radar dashboard. Decisions: human-in-the-loop,
explainable, bias-safe, audited, Ghana DPA 2012 baseline, and a hard **no-fabrication** guardrail.

## Stack (locked — see agent_plan.md §3)
- **Backend:** Go · Hexagonal (ports & adapters) · gRPC + grpc-gateway (REST) · buf · chi · sqlc + pgx · PostgreSQL + pgvector · goose · Asynq (Redis) · Claude (Anthropic) · OpenAI embeddings · JWT + Argon2id · Render · OTel + Prometheus/Grafana/Loki.
- **Frontend:** React + Vite (SPA, prerendered public pages) · MUI v9 (Core only) · TanStack Query · Zustand · Fraunces/Outfit/JetBrains Mono · Motion · Vercel.
- **Quality:** SonarQube + **≥80% coverage on every push** · GitHub Actions.
- **Use the latest stable version of everything.**

## Architecture rules (non-negotiable)
- The **domain** (`internal/domain`) is pure: entities, value objects, domain services, and PORTS (interfaces). It imports nothing from `app`, `adapters`, or `platform`. Enforced by depguard in `.golangci.yml`.
- Use-cases live in `internal/app` and orchestrate the domain through ports only.
- Adapters implement ports: inbound (`adapters/inbound/{grpc,http,jobs}`), outbound (`adapters/outbound/{postgres,llm,embeddings,queue,auth}`).
- Wiring/DI happens in `internal/platform`. No global state; constructor injection.
- All model access goes through the AI orchestration port (`LLMClient`) — never call providers directly from domain/app.

## The API is protobuf-first
- `proto/caliber/v1/*` is the source of truth. Edit protos, then `make proto` to regenerate `internal/gen`.
- Generated code in `internal/gen` is committed; CI fails if it's stale.
- Message shapes for Role Spec/Rubric, Match, and Report Card are **locked contracts** (Appendix A) — don't rename fields.
- Every collection RPC is paginated.

## UX standards (frontend — firm)
Skeletons for content, animated dots for buttons, pagination everywhere, layout transitions,
circular-reveal theme toggle, marketing parallax/3D, and `prefers-reduced-motion` honored. (agent_plan.md §4.5)

## Definition of Done (every story)
Code within hexagonal boundaries · tests keep repo **≥80%** · `gofmt`/`golangci-lint`/`go vet` clean ·
SonarQube gate passes · security handled (validation, authz, secrets, PII) · PR reviewed + merged ·
`agent_plan.md` status + Sprint board updated · docs updated.

## Git conventions (project key CAL)
- **TEMPORARY (stabilization phase): push directly to `main`.** While the platform
  is being stabilized, commit your own work straight to `main` rather than opening
  feature-branch PRs (the solo owner can't self-approve the review gate). Before
  pushing: sync with origin (`git fetch` + fast-forward/rebase) and push **only the
  work you did** — never another agent's in-progress changes. Multiple agents work
  this repo concurrently, so keep each push small, green, and self-contained. Revert
  to the feature-branch + PR flow below once the platform is stable.
- Branch (post-stabilization): `feature/CAL-123-short-slug` (`fix/`, `chore/`, `docs/`).
- Commit: `CAL-123 imperative summary`.
- Trunk-based, squash-merge; CI + Sonar + 1 review required to merge (post-stabilization).
- End commit messages with: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.

## Common commands
```bash
make tools     # install codegen plugins (latest stable)
make proto     # buf dep update + lint + generate
make build     # go build ./...
make test      # go test -race -coverprofile
make cover     # total coverage
make lint      # golangci-lint (enforces hexagonal boundaries)
make run-api   # gRPC + REST gateway (:9090 / :8080, health on :8080)
make run-worker
```

## Security
Never commit secrets (env/secret store only). Validate all input; parameterized SQL via sqlc.
Treat all candidate/role text as untrusted (prompt-injection aware). Enforce the no-fabrication
invariant in code and prompt. Redact PII from logs/telemetry. (agent_plan.md §7, EPIC-16)

## Don't
- Don't let `domain` import infrastructure.
- Don't bypass the `LLMClient`/`Embedder` ports.
- Don't add spinners or unbounded lists in the UI.
- Don't fabricate candidate skills/experience anywhere in the agent path.
- Don't modify unrelated code in a story's PR.
