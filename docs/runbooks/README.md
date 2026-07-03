# On-call runbooks

This directory contains step-by-step runbooks for the alerts and failure modes
that Caliber operators are most likely to see. Each runbook follows the same
structure:

1. **Symptoms** — what the alert, dashboard, or user report looks like.
2. **Impact** — who or what is affected.
3. **Triage** — PromQL / Loki queries and `trace_id` correlation steps.
4. **Mitigation** — safe actions to take right now.
5. **Escalation** — when to page a second responder or wake the team.

## Quick links (local Docker Compose)

| Tool | URL | Use for |
|---|---|---|
| Grafana | http://localhost:3000 | dashboards, cross-reference metrics/logs |
| Prometheus | http://localhost:9091 | raw metrics, alert state |
| Alertmanager | http://localhost:9093 | alert routing, silences |
| Loki | http://localhost:3100 | log search via Grafana Explore |
| Asynqmon | http://localhost:8080/asynqmon/ | queue depth, retries, scheduled tasks, dead letters |

## Correlation by `trace_id`

Every structured error log contains a `trace_id` when the request is inside an
OTel trace:

```json
{
  "level": "ERROR",
  "msg": "operational_error",
  "error": "...",
  "operation": "/caliber.v1.InterviewService/CompleteInterview",
  "class": "grpc",
  "trace_id": "0123456789abcdef0123456789abcdef"
}
```

Use the `trace_id` to jump from a metric spike to the exact request:

- **Grafana Explore → Loki:** `{service="caliber-api"} | json | trace_id="<id>"`
- **Grafana Explore → Tempo/OTel (if configured):** search the same `trace_id`.
- **Logs by error class:** `{service="caliber-api"} | json | class="llm"`

## Runbook index

| Runbook | Alert / failure mode |
|---|---|
| [5xx-spike.md](./5xx-spike.md) | `CaliberHighHTTPErrorRate`, sudden increase in `caliber_errors_total{class="grpc"}` |
| [high-ai-failure-rate.md](./high-ai-failure-rate.md) | `CaliberHighAIFailureRate` |
| [high-queue-error-rate.md](./high-queue-error-rate.md) | `CaliberHighQueueJobErrorRate` |
| [queue-operations.md](./queue-operations.md) | Asynqmon access, queue inspection, retries, scheduled tasks, archived dead letters |
| [high-latency.md](./high-latency.md) | `CaliberHighHTTPLatency`, gRPC p95 latency elevated |
| [auth-rate-limit-spike.md](./auth-rate-limit-spike.md) | Sudden increase in 401/403/429 responses or `rate_limit_exceeded` logs |
| [secret-rotation.md](./secret-rotation.md) | Secret inventory, rotation policy/procedure, and leaked-secret incident response |
| [backup-restore.md](./backup-restore.md) | Postgres backup/restore, RPO/RTO targets, and the disaster-recovery restore drill |
