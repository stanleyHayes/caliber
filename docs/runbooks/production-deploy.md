# Production deploy with approval gate (CAL-148)

<!--
UNVERIFIED IN SANDBOX: the GitHub Environment protection rules, the Actions
workflow, and the Render deploy hook described here cannot be exercised in this
repo checkout. Validate every setting against GitHub's current Environments
documentation (https://docs.github.com/actions/deployment/targeting-different-environments)
and Render's current deploy-hook docs before relying on this runbook.
-->

How a build reaches **production** for Project Caliber: a gated, audited
promotion of an already-built, staging-verified `main` commit, with
auto-generated release notes. This complements the environment topology
([../environments.md](../environments.md)), secret handling
([secret-rotation.md](./secret-rotation.md)), and backups
([backup-restore.md](./backup-restore.md)). Zero-downtime rollout and automatic
rollback are a separate story (CAL-149).

## Overview

Production is promoted by the [`Deploy production`](../../.github/workflows/deploy-production.yml)
GitHub Actions workflow. It:

1. **Resolves** the exact commit to promote and refuses anything not merged to
   `origin/main`.
2. **Pauses at the QA gate** — a job bound to the GitHub **`production`
   Environment**, which is configured to require a reviewer. The run does not
   proceed until an authorized reviewer approves.
3. **Deploys** to Render production by calling the `RENDER_PROD_DEPLOY_HOOK_URL`
   deploy hook (redeploys the API/worker/migrate services built from
   `deploy/Dockerfile.{api,worker,migrate}`).
4. **Generates release notes** from the commit range since the last production
   tag, uploads them as an artifact, and tags/publishes a GitHub Release.

The same environment-agnostic binary runs in every environment; promotion changes
only configuration (see [../environments.md](../environments.md)), so this is a
deploy of an existing image, not a rebuild.

## The approval gate is configured in repo settings (one-time)

The workflow **cannot** create its own protection rules — they live in repo
settings and must be configured once by a repo admin. In
**Settings → Environments → `production`**:

- **Required reviewers** — add the QA approver(s) / release manager. This is the
  gate: the `deploy` job stays *Waiting* until one of them approves in the run's
  UI. GitHub records who approved and when (see the audit trail below).
- **Deployment branches** — restrict to `main` (and/or `prod-*` tags) so only
  protected refs can target the environment.
- **Environment secret `RENDER_PROD_DEPLOY_HOOK_URL`** — add the Render
  production deploy-hook URL here (Environment-scoped, not a plain repo secret),
  so it is only readable from an approved production run. It is a secret: store
  it only here, never in the repo ([secret-rotation.md](./secret-rotation.md)).

> Optional hardening: enable a **wait timer** and **"prevent self-review"** so
> the person who triggered the deploy cannot also approve it (separation of duties).

## How to promote to production

1. Confirm the target commit is green in CI and verified in **staging** (staging
   always reflects `main` — CAL-147).
2. In GitHub → **Actions → Deploy production → Run workflow**. Optionally set
   `git_sha` to promote an exact revision; otherwise the dispatched ref is used.
   (Publishing a GitHub Release also triggers the workflow; the gate still applies.)
3. The run pauses at the `deploy` job pending review. The QA approver reviews the
   resolved SHA and **Approves** (or **Rejects**) in the run UI.
4. On approval, the Render deploy hook fires. Watch the Render dashboard and
   `/readyz` until the new revision is healthy (see verification below).
5. The `release-notes` job publishes a `prod-<timestamp>` tag + GitHub Release and
   uploads `release-notes.md` as a build artifact.

## Audit trail (what is recorded)

The promotion is auditable end to end:

- **Approval** — the GitHub deployment/run timeline records the reviewer, the
  decision (approved/rejected), and the timestamp for the `production`
  Environment. This is the durable record that a human gated the deploy.
- **What shipped** — the `prod-<timestamp>` tag + GitHub Release pin the exact
  commit, and `release-notes.md` (90-day artifact) lists the changes since the
  previous production tag.
- **Who triggered it** — the workflow run records the actor and the input `git_sha`.

Application-level audit events (candidate/role/interview mutations) remain in the
DB audit trail (CAL exportable audit log); this runbook covers the *deploy*
audit, not request-level audit.

## Verify the deploy

After the hook fires:

```bash
curl -fsS https://projectcaliber.app/readyz     # readiness (deps wired) — expect 200
```

Then smoke-test a login + one shortlist + one interview turn, and confirm
`CALIBER_SERVICE_VERSION` on the running service matches the promoted SHA.

## Rollback (interim, until CAL-149)

Automatic health-gated rollback is CAL-149. Until then, roll back manually:

1. Re-run **Deploy production** with `git_sha` set to the **last-known-good**
   commit (its `prod-*` tag / Release identifies it), and approve the gate.
2. If a schema change is implicated, follow the expand/contract and restore
   guidance in [backup-restore.md](./backup-restore.md); goose migrations are
   applied by the migrate service, so a rollback of application code must be
   paired with a compatible schema (CAL-149 formalizes this).
3. Confirm `/readyz` is green and smoke-test as above.

## Notes / caveats

- This workflow triggers a deploy; it does **not** yet block on a post-deploy
  health check or auto-revert (CAL-149).
- The Render deploy-hook `?ref=<sha>` pin and the `gh release create` flags are
  **unverified here** — confirm them against the providers' current docs before
  first use.
