<!--
  VALIDATE BEFORE APPLYING. This document describes the intended SonarCloud
  project configuration (quality profiles, security rules, quality gate,
  new-code period). Those are configured in the SonarCloud PROJECT UI, not in
  version-controlled files — the wiring in sonar-project.properties only tells the
  scanner what to analyze and where to read coverage. Confirm every menu path,
  rule key, and setting name against SonarCloud's current UI/docs
  (https://docs.sonarsource.com/sonarcloud/) before applying; SonarSource renames
  navigation and rule keys between releases.
-->

# SonarQube / SonarCloud configuration (CAL-145)

House quality standards enforced by static analysis for Project Caliber. This
page is the human-readable spec for what lives in the SonarCloud **UI**;
`sonar-project.properties` (repo root) is the machine-readable wiring for the
scanner. Keep the two in sync.

## What lives where

| Concern | Configured in | Source of truth |
|---|---|---|
| Sources / tests / exclusions | `sonar-project.properties` | this repo |
| Coverage import paths | `sonar-project.properties` | this repo |
| Duplication exclusions | `sonar-project.properties` | this repo |
| Quality **profiles** (rule sets per language) | SonarCloud UI → *Quality Profiles* | this doc |
| Security **hotspot / vulnerability** rules | part of the profile + gate | this doc |
| Quality **gate** conditions | SonarCloud UI → *Quality Gates* | this doc |
| New-Code Period | SonarCloud UI → *Project Settings → New Code* | this doc |

## Account & token (required to enforce)

- A **SonarCloud** account and the **`xcreativs`** organization (matching
  `sonar.organization`) with the project `xcreativs_caliber` bound to this repo.
- A **project analysis token** exposed to CI as the GitHub Actions secret
  **`SONAR_TOKEN`**. The scan step in `.github/workflows/ci.yml` (backend job) is
  gated on it (`if: env.SONAR_TOKEN != ''`); with no token, analysis is skipped
  and the gate is **not** enforced. Set the secret to turn enforcement on.
- `SONAR_TOKEN` is a CI/third-party credential, not a runtime app secret, so it is
  deliberately absent from `.env*.example` and `docs/environments.md`. Rotate and
  handle it per the third-party-credential guidance in
  `docs/runbooks/secret-rotation.md`.

## Quality gate — "Caliber house standard"

Set as the project's gate (clone SonarCloud's built-in *Sonar way* and tighten).
Conditions apply to **New Code** (Clean as You Code) unless noted:

| Metric | Condition (new code) | Rationale |
|---|---|---|
| Coverage | ≥ **80%** | Matches the repo-wide push gate (CLAUDE.md, `make cover-check`, web Vitest thresholds). |
| Duplicated lines density | ≤ **3%** | Sonar default; generated/mocks/DB code excluded so this measures authored code only. |
| Maintainability rating | **A** | No new code smells above the profile threshold. |
| Reliability rating | **A** | Zero new bugs. |
| Security rating | **A** | Zero new vulnerabilities. |
| Security hotspots reviewed | **100%** | Every new hotspot must be triaged (ties to CAL-120 hotspot clearing). |
| Security review rating | **A** | No unreviewed hotspots ship. |

Overall-code coverage should also hold ≥ 80% (informational condition) so a large
low-coverage legacy area cannot hide behind new-code-only checks.

**New-Code Period:** reference branch **`main`** (Previous version). PRs are gated
on their changed lines/branch; `sonar.qualitygate.wait=true` makes the CI scan
block and fail red, so the gate is enforcing rather than advisory.

## Quality profiles

Two custom profiles, each a copy of *Sonar way* set as the project default for its
language, then adjusted. Record any rule activation/deactivation here so the UI
state is reproducible.

### Go — "Caliber Go"
- Base: **Sonar way (Go)**.
- Keep cognitive-complexity, error-handling (unchecked errors), and
  hardcoded-credential rules at default or stricter — they reinforce the hexagonal
  and security rules in `.golangci.yml` (this is a second, independent gate, not a
  replacement for `golangci-lint`).
- Security/PII-relevant rules to keep **active**: hardcoded secrets/credentials,
  weak crypto/hashing (we standardize on Argon2id + JWT — flag anything weaker),
  SQL built by string concatenation (all SQL must go through sqlc; raw-string SQL
  is a smell to review), and logging of sensitive data (supports the PII-redaction
  invariant, EPIC-16).
- Generated code (`internal/gen`, `*.pb.go`, sqlc `sqlcdb`) and `internal/mocks`
  are excluded at the scanner level, so their rules never fire regardless of
  profile — no need to deactivate rules to silence them.

### TypeScript/React — "Caliber Web"
- Base: **Sonar way (TypeScript)** + React-specific rules active.
- Keep accessibility (jsx-a11y-equivalent), `no-explicit-any` discouragement, and
  React hooks-correctness rules active to back the frontend UX/quality standards in
  CLAUDE.md. ESLint remains the primary FE linter; Sonar is the cross-cutting gate.
- Security rules to keep active: DOM-based XSS / `dangerouslySetInnerHTML` review,
  hardcoded secrets, and insecure-randomness. Public prerendered pages and the SPA
  handle untrusted candidate/role text, so injection-class rules matter.

## Security hotspots & rules

- Security **hotspots** are review items, not auto-fails; the gate requires
  **100% reviewed on new code** so nothing ships un-triaged. This is the SonarCloud
  side of the CAL-120 "security hotspots cleared" AC.
- Treat all candidate/role free-text as untrusted (prompt-injection aware, per
  CLAUDE.md security section); hotspots touching input handling, deserialization,
  or output encoding must be reviewed against `docs/threat-model.md` before being
  marked *Safe*.
- Do **not** globally disable a security rule to clear a hotspot. Either fix the
  code or mark the specific hotspot *Safe/Acknowledged* with a justification in the
  UI so the audit trail (EPIC-16) is preserved.

## Coverage plumbing (what the scanner imports)

- **Go:** `make test` writes `coverage.out` at repo root; imported via
  `sonar.go.coverage.reportPaths`. CI produces it before the scan step.
- **Web:** Vitest (v8 provider) must emit an **LCOV** report at
  `web/coverage/lcov.info`; imported via `sonar.javascript.lcov.reportPaths` /
  `sonar.typescript.lcov.reportPaths`.

  **Action required:** the current `web/vite.config.ts` coverage reporters are
  `['text','json','html']` — add **`'lcov'`** so `web/coverage/lcov.info` exists,
  and run `npm run test:coverage` (`vitest run --coverage`) in CI's frontend job
  before/alongside the scan. Until then, frontend coverage imports as 0% and the
  coverage gate condition would be wrong for TS. (The `sonar-project.properties`
  coverage exclusions already mirror the Vitest `coverage.exclude` list so the two
  agree once LCOV is wired.)

  Note: the SonarCloud scan currently runs only in the **backend** CI job over the
  whole repo. For frontend coverage to reach Sonar, the LCOV file must be present
  in the workspace at scan time — either generate web coverage before the backend
  scan step, or move the scan to a job that has both `coverage.out` and
  `web/coverage/lcov.info` available. Decide and reflect the choice in
  `.github/workflows/ci.yml` (that workflow edit is out of scope for CAL-145).

## Keeping this in sync

When you change analysis scope, coverage sources, or exclusions, edit
`sonar-project.properties` **and** this doc together. When you change gate
conditions or profile rules in the SonarCloud UI, update the tables above so the UI
state stays reproducible from version control.

## Related

- `sonar-project.properties` — scanner wiring (root of truth for paths/coverage).
- `.github/workflows/ci.yml` — runs the SonarQube scan (gated on `SONAR_TOKEN`).
- `.golangci.yml` — Go lint gate (hexagonal boundaries, complements Sonar).
- `web/vite.config.ts` / `web/eslint.config.js` — FE coverage + lint gates.
- `docs/runbooks/secret-rotation.md` — `SONAR_TOKEN` handling/rotation.
- `docs/threat-model.md` — reference for triaging security hotspots.
- CAL-120 (agent_plan.md) — security-hotspot clearing tracked against this config.
