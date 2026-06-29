# Data-protection baseline (Ghana DPA 2012) — CAL-086

Project Caliber processes employment-relevant personal data (CVs, profiles,
interview transcripts, assessments). This document records the data-protection
posture for the POC against the **Data Protection Act, 2012 (Act 843)** of
Ghana, and the design of the consent and erasure paths. It is the privacy
companion to [fairness.md](fairness.md).

## Principles applied

| DPA 2012 principle | How Caliber applies it |
|---|---|
| **Lawfulness / consent** | Accounts are created explicitly (register flow); a candidate's CV and intake are submitted by the candidate themselves. Consent is the lawful basis for processing a candidate's profile and running the screening/agent flows on their behalf. |
| **Data minimisation** | Protected attributes (age, disability, ethnicity, gender, marital status, nationality, religion) are **not modelled** as candidate fields and never become ranking signals ([fairness.md](fairness.md)). Profiles store only evidenced competencies (name, level, evidence quote, source span) plus intake preferences. |
| **Purpose limitation** | Profile data feeds only matching, screening, and the candidate agent. The agent applies on the candidate's behalf only to roles their verified profile already qualifies for (no-fabrication, CAL-071). |
| **Accuracy** | Every competency carries an evidence quote and source span; a candidate can view and **contest** an assessment (CAL-083), which a human reviewer resolves and the audit trail records. |
| **Security** | Passwords are Argon2id-hashed; access is JWT-gated with per-RPC RBAC; SQL is parameterised (sqlc); secrets live only in env/secret store, never in VCS. |
| **Accountability** | Every consequential action (contest raise/resolve, score override, agent submission) is recorded in an append-only audit trail, queryable via the AuditService (CAL-084). |

## PII handling — already enforced in code

- **Logs/telemetry are PII-free.** The prompt-injection telemetry hook records
  only category labels, never prompt content (CAL-035). The AI-call audit record
  (CAL-036) stores sizes, latency, model, and prompt id/version — never prompt or
  response text. The audit trail stores actor id, action, entity, and entity id;
  it does not retain *candidate-authored* text (a contest's dispute wording is
  kept on the contest, not copied into the trail). A rejection is the deliberate
  exception: it records the employer's own justification (CAL-081), because an
  unexplained decline is exactly what the human-approval gate exists to prevent —
  the decider's words, not candidate PII.
- **Defense-in-depth log redaction (CAL-117).** The root structured logger wraps
  its JSON handler in a redacting handler (`internal/platform/logging/redact.go`)
  that scrubs every record — message and attributes, recursively through groups
  and `With`-bound fields — before it is written: values under secret/identifier
  key names (`email`, `password`, `authorization`, `token`, `phone`, …) are
  blanked, and PII-shaped substrings (email addresses, `Bearer` credentials, JWTs)
  are masked wherever they appear, even inside an otherwise neutral field. This is
  a backstop, not a licence to log PII; call sites still avoid it deliberately.
- **Untrusted-by-default.** All candidate/role text is sanitised and fenced
  before it reaches a model (CAL-119), and treated as data, never instructions.
- **No protected attributes in scoring inputs** (CAL-085) — they are not even
  stored on the candidate model.

## Consent capture (design)

- **Sign-up consent.** Registration is the consent event: a candidate/employer
  agrees to processing when they create an account. A `consent` record (version
  of the terms accepted, timestamp) attaches to the user. *POC status:* the
  account itself is the consent artefact; an explicit versioned consent record is
  a small additive field on the user aggregate (designed, not yet persisted).
- **Purpose-specific consent.** Running the autonomous agent on a candidate's
  behalf is an opt-in; the agent only acts for candidates with a verified profile
  and never fabricates. *POC status:* gated by the candidate initiating the agent.

## Right of access (DSAR — implemented)

A candidate can obtain a complete, structured copy of every record held about
them. `privacy.Exporter` (CAL-118) aggregates — read-only, over the existing
repository ports — the candidate, their talent passport (omitted if never built),
and every application, interview, and contest, paging through each repository
until all records are collected (no silent truncation). *Remaining:* the
candidate-self `GET /v1/me/data` endpoint over this use-case.

## Right to erasure (deletion path — use-case implemented)

The DPA 2012 grants data subjects the right to have their data deleted. The
design:

1. A `DeleteMyData` use-case (candidate-initiated) cascades a hard delete across
   the candidate's aggregates: `Candidate`, `TalentProfile` (+ embeddings),
   `Application`s, `Interview`s + transcripts, `Match`es referencing the
   candidate, and `Contest`s they raised. The owning `User` is anonymised or
   removed.
2. Audit-trail entries are **retained but anonymised** (the actor id is replaced
   with a tombstone), because the append-only trail is itself a compliance record
   — its existence, not the subject's identity, is what is retained.
3. Exposed as `DELETE /v1/me/data` (candidate-only), audited as a deletion event.

*POC status:* **cascade use-case implemented + tested** — `privacy.Eraser`
(CAL-118) orchestrates exactly the order above (scoped records → candidate
aggregate → owning user anonymised → audit trail tombstoned), declaring the
narrow removal ports it depends on (hexagonal). *Remaining:* the repositories'
hard-delete primitives that satisfy those ports and the `DELETE /v1/me/data`
endpoint, gated on the Postgres persistence work (EPIC-02).

## Retention

- Demo/dev data is in-memory and ephemeral (reset on restart).
- In production, profile data is retained while the account is active; on erasure
  or account closure the cascade above runs. Audit entries are retained
  (anonymised) for the statutory period.

## Data subject rights summary

| Right | Mechanism | Status |
|---|---|---|
| Access | In-app views + `privacy.Exporter` DSAR aggregation (CAL-118); endpoint pending | Use-case built |
| Rectify / contest | Contest an assessment (CAL-083); human reviewer resolves; audited | Built |
| Erasure | `privacy.Eraser` cascade use-case (CAL-118, above); repo delete primitives + endpoint pending | Use-case built |
| Object / restrict | Agent is opt-in; deal-breakers exclude roles (CAL-046) | Built |
| Portability | Profile is structured JSON behind the API | Built (API) |

## Cross-border note (West Africa)

The platform targets Ghana and West Africa; the location gate is logistical
(work location), deliberately distinct from the protected attribute
*nationality* ([fairness.md](fairness.md)). No cross-border transfer of personal
data occurs in the POC (single-region in-memory/Postgres).
