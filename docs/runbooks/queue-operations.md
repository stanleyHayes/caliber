# Queue operations

## Purpose

Use this runbook to inspect Asynq queue depth, retries, scheduled work, and archived
dead-letter tasks for Caliber background jobs.

## Access

- Local/API URL: `http://localhost:8080/asynqmon/`
- Docker Compose API URL: `http://localhost:8080/asynqmon/`
- Staging/production: the same API host under `/asynqmon/`

The dashboard is mounted only when `CALIBER_REDIS_URL` is configured. It is
protected by the API bearer-token verifier and only employer/recruiter principals
can access it. Use an operator API client or browser profile that can attach:

```text
Authorization: Bearer <access-token>
```

Do not make Asynqmon public or bypass the token guard.

## What To Check

1. **Pending**: confirms workers are receiving tasks but have not started them.
2. **Active**: shows currently executing work.
3. **Scheduled**: delayed work created with `ProcessIn` or `ProcessAt`.
4. **Retry**: transient failures waiting for the retry policy/backoff.
5. **Archived**: dead-lettered tasks that exhausted `MaxRetry`.

## Normal Operation

- `candidate_agent:run` and `interview:score` use the default queue unless the
  caller overrides it.
- `matching:rematch` can be delayed for batch re-ranking and defaults to a lower
  retry ceiling than user-visible work.
- `privacy:data_retention` is scheduled by the worker when
  `CALIBER_RETENTION_WINDOW > 0`.

## Triage

1. Open `/asynqmon/` with an employer/recruiter bearer token.
2. Check the queue with rising depth, then filter by task type.
3. For retrying or archived tasks, correlate the task type and task ID with
   worker logs:

   ```logql
   {service="caliber-worker"}
   | json
   | task_type="<task-type>"
   ```

4. For dead-letter alerts, search:

   ```logql
   {service="caliber-worker"}
   | json
   | alert="dead_letter"
   ```

5. Cross-check Prometheus queue metrics in Grafana's **Caliber Queue Health**
   dashboard.

## Mitigation

- If tasks are **pending** and not active, verify the worker process is running
  and connected to the same Redis instance as the API.
- If tasks are **retrying**, inspect the last error first. Do not increase retry
  counts until the dependency failure or poison payload is understood.
- If tasks are **scheduled** longer than expected, verify system time and the
  `ProcessIn`/`ProcessAt` caller path.
- If tasks are **archived**, leave them in the archive until the payload is
  captured for a bug. Requeue only after the underlying handler fix is deployed.

## Verification

- `go test ./internal/adapters/inbound/jobs`
- `go test ./internal/adapters/outbound/queue`
- `go test ./internal/platform/server ./internal/platform/wiring`
