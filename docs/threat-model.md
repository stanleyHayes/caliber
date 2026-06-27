# Threat model & security requirements (STRIDE) — CAL-110

A STRIDE threat model over Project Caliber's architecture, mapping each threat
class to the controls that are **implemented** today and the ones still on the
**backlog** (with CAL ids). It is the security companion to
[data-protection.md](data-protection.md) (privacy) and [fairness.md](fairness.md)
(bias safety), and the input to the security-review prep (CAL-120).

## Scope & assets

What we are protecting, roughly in order of sensitivity:

- **Candidate PII & profiles** — CVs, contact details, evidenced competencies,
  interview transcripts, report cards.
- **Employer data** — role specs, rubrics, shortlists, rejection decisions.
- **Decision integrity** — match scores, screening verdicts, agent applications;
  the *explainable, non-fabricated, bias-safe* guarantees are themselves an asset.
- **The audit trail** — the append-only record of approvals, overrides, agent
  actions, and contests; its integrity underwrites every compliance claim.
- **Secrets** — Anthropic/OpenAI keys, JWT signing secret, DB/Redis credentials.

## Trust boundaries

```
[ Browser SPA ]  --HTTPS-->  [ REST gateway :8080 ]  -->  [ gRPC services :9090 ]
                                                              |  |        |
   untrusted candidate/role text ----------------------------+  |        |
                                                  [ Postgres+pgvector ] [ Redis/Asynq ]
                                                                        |
                                              [ Anthropic / OpenAI ]  <-+  (egress)
```

Boundaries crossed: client→gateway (authn), gateway→gRPC (internal), services→DB/
queue (credentialed), services→LLM/embeddings (egress with untrusted text in the
payload). **Every candidate/role string is untrusted** at the first boundary and
stays untrusted until sanitised+fenced.

## STRIDE analysis

### S — Spoofing (identity)

- **Threats:** stolen/forged tokens; logging in as another user; an agent acting
  as a user it isn't.
- **Implemented:** Argon2id password hashing; JWT access/refresh with issuer +
  audience claims; login throttling; per-RPC principal extraction where the actor
  identity comes from the **auth context, never the request body** (enforced in
  the rejection, contest, and agent paths). The candidate agent records the
  candidate as actor because it is their delegated proxy — never a third party.
- **Backlog:** secret rotation for the JWT signing key (CAL-113); short access-TTL
  is set (15m) — refresh-token rotation/replay detection to harden (CAL-116).

### T — Tampering (integrity)

- **Threats:** altering scores/verdicts; mutating the audit trail; SQL injection;
  prompt injection steering a model to fabricate or leak.
- **Implemented:** the audit trail is **append-only by design** (the port exposes
  only Append/List; entries have no mutators). Parameterised SQL via sqlc — no
  string-built queries. **Prompt-injection defence** (CAL-119): every model call
  site sanitises field-level input (`guard.Sanitize`) and fences untrusted blocks
  (`guard.Fence`/`FenceUntrusted`); injection scanning flags suspicious spans. The
  **no-fabrication** invariant (CAL-071) re-checks every agent-authored summary
  against the verified profile before it can become an application; a rejection
  cannot exist without an explicit human approval (CAL-081).
- **Backlog:** DB-level integrity (row checksums / WORM storage) for the audit
  trail in production; input-validation pass across every DTO (CAL-111).

### R — Repudiation (non-deniability)

- **Threats:** a reviewer denying a rejection; disputing an agent's application;
  "the AI did it" deflection.
- **Implemented:** every consequential action writes an attributed audit entry —
  rejections (`approve_rejection`, actor = approving human), agent submissions
  (`agent_submit`, actor = candidate), contests raised/resolved, score overrides.
  The trail is queryable per entity (CAL-084). The rejection log **is** the
  approval: if it can't be written, the action fails — no silent, unlogged decline.
- **Backlog:** exportable/reportable audit views (CAL-136); trace-id correlation
  across services (CAL-130).

### I — Information disclosure (confidentiality)

- **Threats:** PII in logs/telemetry; leaking one candidate's data to another
  (IDOR); a model echoing secrets or other users' data; over-broad audit reads.
- **Implemented:** logs/telemetry are **PII-free** — injection telemetry records
  only category labels; the AI-call record stores sizes/latency/model, never
  prompt or response text (CAL-035/036). Protected attributes are **not modelled**
  and never reach scoring (CAL-085, [fairness.md](fairness.md)). Audit reads are
  reviewer-only (employer/recruiter). Candidate-facing reads are scoped to the
  authenticated principal.
- **Backlog:** encryption at rest + field-level encryption for sensitive PII
  (CAL-117); full IDOR/ownership-check sweep with tests (CAL-116); TLS/HSTS/CSP +
  strict CORS (CAL-114); secret-scanning gate in CI — gitleaks (CAL-113).

### D — Denial of service (availability)

- **Threats:** flooding expensive AI endpoints (cost + latency DoS); unbounded
  pagination; a slow/hostile LLM hanging a request.
- **Implemented:** every collection RPC is **paginated** (bounded result sets);
  login throttling; LLM calls have bounded retry attempts.
- **Backlog:** per-IP/user/endpoint rate limiting with extra protection on the
  expensive AI endpoints (CAL-112); request timeouts/circuit-breaking on LLM
  egress; heavy scoring moved to the async queue so the request path can't be
  exhausted (CAL-067); queue-depth metrics + alerts (CAL-131/134).

### E — Elevation of privilege (authorization)

- **Threats:** a candidate acting as a reviewer; horizontal access to peers'
  records; an endpoint missing an authz check.
- **Implemented:** per-RPC RBAC via `RequireRole` — e.g. rejection/audit reads are
  employer/recruiter-only; contest listing is candidate-only; unauthenticated →
  Unauthenticated, wrong role → PermissionDenied. The hexagonal boundary
  (depguard) keeps authz decisions in the app/adapter layers, not leaked into the
  pure domain.
- **Backlog:** systematic least-privilege review across every service + IDOR test
  suite (CAL-116); ownership checks (does this employer own this role?) audited
  end-to-end.

## Prompt-injection & LLM-specific threats (cross-cutting)

The LLM is a confused-deputy risk: untrusted candidate/role text shares a channel
with our instructions. Defences in depth: (1) **sanitise** field input, (2)
**fence** untrusted blocks with explicit delimiters and "treat as data" framing,
(3) **scan** for injection markers, (4) **constrain** outputs to typed JSON and
re-validate against domain invariants, (5) **ground** every claim — the
no-fabrication checks reject any agent output asserting unverified skills, so a
successful injection still cannot manufacture qualifications or auto-apply.
Residual risk: a model could still produce a *plausible but wrong* rationale;
mitigated by human-in-the-loop (no auto-reject; contestable assessments).

## Security requirements (acceptance)

1. No secret in code or VCS; secrets only via env/secret store; CI secret-scan
   gate (CAL-113).
2. All input validated at the boundary; all SQL parameterised; all untrusted text
   sanitised+fenced before any model call (CAL-111, CAL-119 ✓).
3. Every state-changing action authenticated, authorised by role, and audited
   (CAL-081 ✓, CAL-084 ✓).
4. No PII in logs/telemetry; PII encrypted at rest in production (CAL-035/036 ✓,
   CAL-117).
5. Expensive AI endpoints rate-limited; all collections paginated (pagination ✓,
   CAL-112).
6. TLS everywhere; HSTS/CSP/X-Frame-Options; strict CORS (CAL-114).
7. Dependency + container scanning in CI: govulncheck, Trivy/Grype, npm audit,
   Dependabot (CAL-115).
8. `/security-review` run and hotspots cleared before any external pen test
   (CAL-120).

## Backlog summary (open security epics)

| Area | CAL |
|---|---|
| Input validation & output encoding everywhere | CAL-111 |
| Rate limiting / abuse protection | CAL-112 |
| Secrets management & rotation; gitleaks gate | CAL-113 |
| Security headers, TLS, CORS | CAL-114 |
| Dependency & container scanning | CAL-115 |
| AuthZ hardening & least privilege; IDOR tests | CAL-116 |
| PII protection & encryption at rest | CAL-117 |
| Ghana DPA 2012 compliance (consent/retention/DSAR) | CAL-118 |
| Security review & pen-test prep | CAL-120 |

Legend: ✓ = control implemented today; bare CAL id = tracked, not yet built.
