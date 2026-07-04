# Caching & read-scaling (CAL-156)

How Caliber keeps hot reads fast as the candidate pool and traffic grow. The
code-side controls below ship in the POC; the "at scale" targets are validated
by the k6 load suite (CAL-142) against a scaled deployment.

## 1. Vector recall — pgvector HNSW

Stage-1 candidate recall (`Recaller.Recall`) is a cosine-distance nearest-neighbour
scan over `talent_profiles.profile_embedding`. It is backed by an **HNSW index**
(`db/migrations/00003_indexes.sql`: `USING hnsw (... vector_cosine_ops)`), so
recall is sub-linear rather than a full-table scan.

**Tuning knob — `hnsw.ef_search`.** HNSW trades recall quality for latency at
query time via the `ef_search` runtime parameter (higher = better recall, more
work). It is exposed as `CALIBER_HNSW_EF_SEARCH` (default `0` = the server
default, currently 40). When set, `newPGPool` installs an `AfterConnect` hook so
every pooled connection runs `SET hnsw.ef_search = N` once — no change to the
recall query itself. Raise it (e.g. 100) if recall quality dips as the pool
grows; measure the latency cost with the k6 suite before committing a value.

Index build parameters (`m`, `ef_construction`) are set at index-creation time in
the migration; retune them there if the index is rebuilt for a much larger pool.

## 2. Hot read-model caching

- **Talent Radar dashboard** (`dashboard.CachedAggregator`, CAL-080) wraps the
  pool / supply-demand / alerts / time-to-shortlist read model in a TTL snapshot
  cache (`CALIBER_DASHBOARD_CACHE_TTL`, default 30s), so the god-view is served
  from memory between refreshes instead of recomputing per request.
- **Shortlists** are computed on demand (recall + scoring) and mutate on refine /
  reject, so they are intentionally *not* cached blind — a stale shortlist would
  mislead a hiring decision. A short-TTL, invalidate-on-mutation cache is the
  natural next step if shortlist reads become a hot path; the dashboard cache is
  the template.

## 3. Read replicas (deployment topology)

Reads (recall, dashboard, listings) vastly outnumber writes, so they scale
horizontally onto Postgres **read replicas**:

- Provision one or more replicas of the primary (Render/managed Postgres).
- Route read-only repositories at a replica pool and keep writes on the primary.
  The seam is `newPGPool` + the `Repositories` wiring: add a read-only DSN
  (`CALIBER_DATABASE_READ_URL`) and a second pool, then point the read-side repos
  (recaller, dashboard aggregator, list queries) at it. This is a deploy-time
  concern — the app is structured for it (all read repos already take a `DBTX`),
  but a live replica is provisioned per environment, not in the POC.
- Mind replica lag: keep read-your-writes paths (e.g. immediately reading back a
  just-created role) on the primary.

## 4. Validating "at scale"

The AC — p95 latency targets met at scale — is a **load + infrastructure**
property, not a unit test. Validate it with the k6 suite (`tests/load/`,
`make test-load`, CAL-142) against a scaled environment: it drives Flow A recall,
the Talent Radar, Flow B streaming, and Flow C, and enforces client-side SLO
thresholds (HTTP error rate < 1%, p95 < 2s). Tune `hnsw.ef_search`, the cache
TTL, and replica count against its output.
