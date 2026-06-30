# Caliber Demo Runbook & Failure Playbook

> One-page guide for driving the Project Caliber POC demo end-to-end.  
> Audience: any team member asked to run the demo at a rehearsal, stakeholder meeting, or venue.

---

## 1. Demo story at a glance

The demo proves three claims in one continuous narrative:

1. **Flow A** — Hiring manager describes a role in plain language; Caliber returns a structured spec, weighted rubric, and an explainable ranked shortlist in seconds.
2. **Flow B** — A candidate completes an adaptive AI screening interview and receives an evidence-tagged report card (the centrepiece).
3. **Flow C** — The candidate’s autonomous agent "works overnight", submitting only truthful, grounded applications.
4. **Close** — The Talent Radar dashboard shows the live pool, supply/demand gaps, two-way alerts, and the headline time-to-shortlist collapse.

Run-of-show sequence: **Frame → Flow A → Flow B → Flow C → Close on Radar**.

---

## 2. Environment options

| Mode | When to use | Persistence | Seed |
|------|-------------|-------------|------|
| **Local in-memory dev** | Fastest, no Docker, no network needed | In-memory; resets on API restart | Hand-curated demo dataset (default) |
| **Docker Compose full stack** | Staging rehearsal, persistent DB, worker queue | Postgres + Redis | Migrations applied; seed loaded at API boot |

The demo works fully offline when `ANTHROPIC_API_KEY` and `OPENAI_API_KEY` are unset — the deterministic `dev` LLM/embedder returns golden responses.  
Set the keys to demo against the live Claude/OpenAI providers.

---

## 3. Pre-flight checklist

```bash
# 1. Build everything
make build

# 2. Optional: run the full local CI gate
make ci

# 3. Copy environment template if you have not already
cp .env.example .env
```

Verify these env values make sense for your chosen mode (see `.env.example`):

- `CALIBER_ENV=dev`
- `CALIBER_SEED_DEMO=true` (default — loads demo data)
- `CALIBER_SEED_GENERATED=false` (set `true` to use the larger parser-generated dataset; requires API keys + network)
- `CALIBER_INTERVIEW_MAX_QUESTIONS=4` (demo default)
- `CALIBER_DASHBOARD_CACHE_TTL=30s`

Health checks after boot:

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

---

## 4. Start the app

### Option A — Local in-memory dev

```bash
# Terminal 1: API + REST gateway
make run-api

# Terminal 2 (only if CALIBER_REDIS_URL is set): background worker
make run-worker

# Terminal 3: React SPA
cd web && npm run dev
```

Open the UI at **http://localhost:5173**.

### Option B — Docker Compose full stack

```bash
docker compose up --build
```

This brings up Postgres+pgvector, Redis, migrations, API, worker, and the Vite dev server.  
Open **http://localhost:5173**.  Asynqmon queue UI is available at `/asynqmon` on the API port if Redis is wired.

---

## 5. Demo accounts

All seeded demo accounts share the same password:

```
Demo-Caliber-2026
```

### Hand-curated demo pool (default, `CALIBER_SEED_GENERATED=false`)

| Persona | Email | Use in |
|---------|-------|--------|
| Employer — MTN Ghana | `talent@mtn.com.gh` | Flow A, Radar |
| Employer — Hubtel | `talent@hubtel.com` | Flow A, Radar |
| Employer — mPharma | `talent@mpharma.com` | Flow A, Radar |
| Candidate — Ama Mensah | `ama.mensah@example.com` | Flow B / C (hero backend candidate) |
| Candidate — Kofi Asante | `kofi.asante@example.com` | Flow B / C (hero data candidate) |
| Candidate — Esi Owusu | `esi.owusu@example.com` | Flow B / C (hero mobile candidate) |
| Candidate — Yaw Boateng | `yaw.boateng@example.com` | Flow B / C (hero platform candidate) |
| Candidate — Abena Sarpong | `abena.sarpong@example.com` | Flow B / C |
| Candidate — Kwame Boadu | `kwame.boadu@example.com` | Flow B / C |
| Candidate — Adwoa Agyeman | `adwoa.agyeman@example.com` | Flow B / C |
| Candidate — Kojo Antwi | `kojo.antwi@example.com` | Flow B / C |

### Generated demo pool (`CALIBER_SEED_GENERATED=true`)

- ~8 employers, ~12 roles, 55 candidates.
- Candidate emails follow the pattern `{first}.{last}.gen{N}@example.com`.
- Same shared password: `Demo-Caliber-2026`.

---

## 6. Run-of-show

Total target time: **8–12 minutes**.

### Frame (1 min) — Talent Radar

1. Log in as any **employer** (e.g. `talent@mtn.com.gh`).
2. Navigate to **/radar**.
3. Talking points:
   - "This is the live talent pool."
   - "Supply/demand shows where the market is tight."
   - "Time-to-shortlist collapses from weeks to hours — that is the headline."

### Flow A (2–3 min) — Employer intake & explainable shortlisting

1. Stay logged in as the employer.
2. Go to **/roles/new**.
3. Paste a messy hiring sentence, for example:

   > "We need a senior Go backend engineer in Accra to own our matching services — must know Postgres and gRPC, ideally some Kubernetes. GHS 18k–25k, start within a month."

4. Click **Generate spec & rubric**.
5. Expected outputs:
   - Structured role spec (title, location, seniority, salary band, must-haves, nice-to-haves).
   - Weighted, normalized rubric.
   - Alert: "N strong matches already in your pool."
6. Scroll to the shortlist. Expected outputs:
   - Ranked matches with overall score and confidence.
   - Per-competency breakdown with evidence quotes.
   - Plain-English rationale and watch-outs.
   - Hard-filter exclusions with reasons (location, salary, missing must-have).
7. Click **Refine spec & rubric**, change a weight or must-have, save, and watch the shortlist re-rank.
8. Optional: click **Record rejection** on a match to show the human-approval gate — the system never auto-rejects.

### Flow B (3–4 min) — AI screening interview (centrepiece)

1. Copy the **Role ID** from Flow A (shown in the URL or UI).
2. Log out, then log in as a **candidate** who matches that role, e.g.:
   - Backend role → `ama.mensah@example.com`
   - Data role → `kofi.asante@example.com`
   - Mobile/frontend role → `esi.owusu@example.com`
3. Go to **/interview?roleId=<ROLE_ID_FROM_FLOW_A>**.
4. Click **Start interview**.
5. Expected behaviour:
   - Skeleton placeholders appear between turns.
   - Each question targets a different rubric competency.
   - Vague answers trigger an honest-signal follow-up asking for a concrete example.
6. Answer 4 questions (default cap). Use concrete, first-person examples with measurable outcomes for best results.
7. Expected outputs:
   - Final report card with per-competency scores and evidence quotes.
   - Overall verdict (Advance / Hold / Decline), confidence, and recommended next step.
   - A **Contest this assessment** button, showing candidate recourse.

### Flow C (2 min) — Candidate agent & wake-up view

1. Stay logged in as the same candidate.
2. Go to **/agent**.
3. Click **Run overnight**.
4. Expected outputs:
   - Wake-up card: new matches, applications submitted, screenings completed, employers interested.
   - Highlights explaining what the agent did.
   - Applications list with `source: agent` and tailored summaries that trace back to the verified profile.
5. Talking point: "Every claim in that summary is grounded in the profile. If it cannot be grounded, the agent refuses to apply."

### Close (1 min) — Return to Radar

1. Log out and log back in as an **employer**.
2. Return to **/radar** and refresh.
3. Close with the time-to-shortlist metric and the live pool.
4. Final line: "Weeks to hours — with a full evidence trail."

---

## 7. Reset / reseed

There is no single reset CLI command yet (tracked in **CAL-103**). Until then:

### In-memory dev reset

```bash
# Stop the API with Ctrl-C, then restart it
make run-api
```

All in-memory state is rebuilt from the deterministic seed on boot.

### Docker Compose reset

```bash
# Wipe the DB volume and rebuild the stack
docker compose down -v
docker compose up --build
```

### Local Postgres reset (no Docker)

```bash
# 1. Drop and recreate the database
# 2. Run migrations
go run ./cmd/migrate
# 3. Restart the API (seeding happens at boot)
make run-api
```

### Switch seed mode

- Hand-curated (fast, offline, deterministic): `CALIBER_SEED_DEMO=true` and `CALIBER_SEED_GENERATED=false`.
- Parser-generated (larger, needs API keys + network): `CALIBER_SEED_DEMO=true` and `CALIBER_SEED_GENERATED=true`.

---

## 8. Key talking points by flow

| Flow | What to say | Proof point |
|------|-------------|-------------|
| **Frame / Radar** | "The CV is one input, not the verdict." | Live pool, supply/demand, time-to-shortlist metric. |
| **Flow A** | "A messy sentence becomes a structured, editable spec and a bias-safe rubric." | Instant `available_matches`, explainable ranked shortlist, editable weights. |
| **Flow A trust** | "We never auto-reject; every decline needs a human approval and a reason." | Record rejection → audit entry. |
| **Flow B** | "The AI interviews, probes, and scores with evidence." | Adaptive questions, vague-answer pressure, evidence-tagged report card. |
| **Flow B trust** | "Candidates can contest any assessment." | Contest button on report card. |
| **Flow C** | "The agent works while you sleep, honestly." | Wake-up view, agent-sourced applications, grounded summaries. |
| **Close** | "Time-to-shortlist collapses from weeks to hours." | Radar headline metric. |

---

## 9. Expected outputs / acceptance beats

| Step | Expected output | If you do not see this… |
|------|-----------------|-------------------------|
| `/healthz` | `{"status":"ok"}` | API did not boot; check logs. |
| `/readyz` | `{"status":"ready"}` | Postgres/Redis not reachable. |
| Login | Employer or candidate dashboard loads | Wrong role / expired token; re-login. |
| Flow A generate | Spec card + rubric card + available matches alert | Check `CALIBER_SEED_DEMO=true`; verify API logs. |
| Flow A shortlist | Ranked matches, breakdown, rationale, exclusions | In-memory seed missing; restart API. |
| Flow B stream | 4 questions then report card | See Failure Playbook → stream failure. |
| Flow C wake-up | Non-zero submitted applications + highlights | Candidate has no verified profile; check seed. |
| Radar | Pool, supply/demand, time-to-shortlist | Logged in as candidate, not employer. |

---

## 10. Failure playbook

| Failure mode | Trigger / symptom | Fallback |
|--------------|-------------------|----------|
| **Network drop / LLM unavailable** | Flow A/B/C returns 5xx or stalls; logs show Anthropic/OpenAI timeouts. | Stop the API, unset `ANTHROPIC_API_KEY` and `OPENAI_API_KEY`, restart with `make run-api`. The deterministic `dev` provider keeps the demo moving offline. |
| **LLM latency feels slow** | Questions or shortlist take >5 s. | Reduce `CALIBER_INTERVIEW_MAX_QUESTIONS=2` and restart, or switch to the offline `dev` provider by unsetting API keys. |
| **Interview stream failure** | `/interview` stalls, no question arrives, or browser shows stream error. | 1) Refresh the page and re-enter the role ID. 2) Check `/healthz` and `/readyz`. 3) If persistent, restart the API and try again with the `dev` provider. |
| **Worker / queue not processing** | Flow C "Run overnight" returns but applications never update; Redis is configured. | Ensure `make run-worker` or the `worker` Docker service is running. Inspect `/asynqmon` for dead-lettered tasks. |
| **Frontend dev server crash** | `npm run dev` fails or 404 on `/v1/*`. | Run `cd web && npm run build` to surface type errors. Check `VITE_API_URL` / Vite proxy target points to `http://localhost:8080`. |
| **Seed data missing / wrong** | Radar empty, no demo accounts login, shortlist empty. | Verify `CALIBER_SEED_DEMO=true`. Check API boot logs for `loaded demo dataset` and `demo_login_password`. Restart the API (in-memory) or `docker compose down -v && up --build`. |
| **401/403 during demo** | "Session expired" or permission denied. | Re-login with the correct role account. Employer/recruiter for Flow A and Radar; candidate for Flow B/C. |
| **Unexpected / fabricated AI output** | A match or summary cites evidence not in the profile. | Pause and use it as a teaching moment: show the no-fabrication guardrail. If reproducible, switch to the deterministic `dev` provider for the rest of the demo. |
| **Venue loses all connectivity** | Cannot reach localhost or Docker. | Use a pre-recorded backup capture (tracked in **CAL-106**) or an offline standalone build (tracked in **CAL-107**). |

---

## 11. Quick reference

```bash
# Build
make build

# Local API
make run-api

# Local worker (if Redis configured)
make run-worker

# Local web
cd web && npm run dev

# Docker full stack
docker compose up --build

# Reset stack
docker compose down -v && docker compose up --build

# Migrations (Postgres path)
go run ./cmd/migrate

# Health
open http://localhost:8080/healthz
open http://localhost:8080/readyz

# Routes
#   /           landing
#   /login      sign in
#   /app        candidate/employer dashboard
#   /roles/new  Flow A
#   /interview  Flow B
#   /agent      Flow C
#   /radar      Talent Radar
```

---

## 12. Notes & known limitations

- This runbook is written against the existing flows; **CAL-105** (run-of-show wiring) and **CAL-103** (one-shot reset command) are still TODO.
- The Radar time-to-shortlist metric is a demo signal; real per-role timing computation is tracked separately.
- Voice interview mode is post-win only (EPIC-22).
