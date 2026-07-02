# Secret management & rotation (CAL-113)

How Project Caliber stores secrets, how to rotate each one, and what to do when
one leaks. This complements the threat model ([threat-model.md](../threat-model.md))
and the data-protection controls ([data-protection.md](../data-protection.md)).

## Principles

- **Secrets live only in the platform secret store, never in the repo.** The
  running services read them from environment variables injected by the host
  (Render for the API/worker; Vercel for the web build). `.env.example` documents
  the variable *names* with non-secret placeholders only.
- **Never logged.** No code logs a secret value. As a defense-in-depth backstop,
  the root logger's redacting handler
  ([internal/platform/logging/redact.go](../../internal/platform/logging/redact.go),
  CAL-117) blanks any attribute whose key names a secret (`*secret*`, `*password*`,
  `*api_key*`, `authorization`, `token`, …) and masks bearer/JWT-shaped values.
- **Scanned on every push.** The `Secrets (gitleaks)` CI job fails the build if a
  secret-shaped string is committed; `.gitleaks.toml` allowlists only documented
  local placeholders and non-secret test fixtures.

## Secret inventory

| Secret | Env var | Used by | Store |
|---|---|---|---|
| JWT signing secret | `CALIBER_JWT_SECRET` | API (access/refresh token signing) | Render secret |
| Postgres DSN (embeds DB password) | `CALIBER_DATABASE_URL` | API, worker, migrate | Render secret |
| Redis URL (may embed password) | `CALIBER_REDIS_URL` | API, worker (Asynq) | Render secret |
| Anthropic API key | `ANTHROPIC_API_KEY` | API, worker (LLM) | Render secret |
| OpenAI API key | `OPENAI_API_KEY` | API, worker (embeddings) | Render secret |
| Loki push URL (may embed basic-auth) | `CALIBER_LOKI_URL` | API, worker (log shipping) | Render secret |

Local development uses `dev-secret-change-me`, the deterministic dev LLM/embedder
(no external keys), and `caliber:caliber@localhost` — all allowlisted, none real.

## Rotation policy

| Secret | Cadence | Trigger to rotate immediately |
|---|---|---|
| `CALIBER_JWT_SECRET` | 90 days | suspected token forgery; staff departure |
| `CALIBER_DATABASE_URL` password | 180 days | DB credential exposure; provider breach |
| `CALIBER_REDIS_URL` password | 180 days | Redis credential exposure |
| `ANTHROPIC_API_KEY` | 90 days | key exposure; unexpected spend (CAL-159 alert) |
| `OPENAI_API_KEY` | 90 days | key exposure; unexpected spend |
| `CALIBER_LOKI_URL` token | 180 days | log-pipeline credential exposure |

## Rotation procedure

General flow (zero-downtime where the secret allows two valid values at once):

1. **Mint** the new secret at its source (provider dashboard for API keys; managed
   Postgres/Redis for DB/cache passwords; a fresh 32+ byte random value for the JWT
   secret, e.g. `openssl rand -base64 48`).
2. **Set** the new value in the Render secret store for **all** services that read
   it (API, worker, migrate as applicable).
3. **Roll** the services (Render redeploys on env change). Verify `/readyz` is
   green and a smoke login + one interview turn succeed.
4. **Revoke** the old secret at its source once the new one is confirmed live.
5. **Record** the rotation date in the ops log.

### Per-secret notes

- **`CALIBER_JWT_SECRET`** — rotating invalidates all existing access **and**
  refresh tokens, so every session must re-authenticate. Access tokens are short
  lived (`CALIBER_ACCESS_TOKEN_TTL`); refresh tokens are single-use and rotated
  (CAL-020), so the blast radius is one forced re-login. Rotate during a low-traffic
  window; there is no dual-secret verification today (a follow-on could accept the
  previous secret for the access-token TTL to avoid the forced re-login).
- **`CALIBER_DATABASE_URL` / `CALIBER_REDIS_URL`** — create the new credential
  before removing the old so both are briefly valid; update the env, roll, then drop
  the old credential.
- **`ANTHROPIC_API_KEY` / `OPENAI_API_KEY`** — create a second key, deploy it, then
  revoke the first. Watch the CAL-159 spend alerts after rotation to confirm traffic
  moved to the new key and no rogue usage remains on the old one.

## Leaked-secret incident response

1. **Contain** — revoke the exposed secret at its source **first** (before cleanup);
   a committed secret is compromised the moment it is pushed, even to a private repo.
2. **Rotate** — follow the procedure above to bring a fresh value online.
3. **Purge** — remove the secret from history if it was committed (`git filter-repo`
   / BFG) and force-push with the team's coordination; the value is already burned,
   so rotation (step 1–2) is what actually protects you.
4. **Assess** — check provider audit logs and the CAL-159 spend alerts / audit trail
   for misuse during the exposure window.
5. **Prevent** — if gitleaks missed it, tighten `.gitleaks.toml`; add a rule for the
   pattern that slipped through.

## Verifying the gate locally

```bash
make scan-go            # govulncheck (dependency CVEs)
gitleaks detect --config .gitleaks.toml --no-banner   # secret scan (as CI runs it)
```

A clean `gitleaks detect` and a green `Secrets (gitleaks)` CI job satisfy the
"secret scan clean" acceptance criterion.
