# Observability stack (CAL-130 / CAL-131 / CAL-132 / CAL-133 / CAL-135)

Project Caliber exposes OpenTelemetry traces and Prometheus metrics, ships logs
to Loki, and provides Grafana dashboards out of the box.

## Local stack

Run the full stack with Docker Compose:

```bash
docker compose up --build
```

| Service         | URL                     | Purpose                                  |
|-----------------|-------------------------|------------------------------------------|
| API             | http://localhost:8080   | gRPC + REST gateway                      |
| Worker          | http://localhost:8081   | Prometheus metrics scrape endpoint       |
| Prometheus      | http://localhost:9090   | Metric scraper for API + worker          |
| Alertmanager    | http://localhost:9093   | Alert routing                            |
| Alert receiver  | http://localhost:8082   | Webhook echo receiver (logs alerts)      |
| Loki            | http://localhost:3100   | Log backend                              |
| Grafana         | http://localhost:3000   | Dashboards (login `admin` / `admin`)     |

## Metrics

The API serves Prometheus exposition format at `/metrics`. The worker exposes the
same endpoint on `CALIBER_WORKER_METRICS_ADDR` (default `:8081`).

### Custom metric families

- `caliber_ai_*` — AI call volume, failures, JSON failures, refusals, guardrail
  trips, input/output character counts, and latency (CAL-131).
- `caliber_queue_*` — task enqueue rate, job processing rate by status, and job
  processing duration (CAL-133).
- `caliber_errors_total` — structured operational errors grouped by `operation`
  and `class` (`grpc`, `llm`, etc.). Useful for triaging spikes and correlating
  with logs via `trace_id` (CAL-135).

### Instrumentation metrics

HTTP requests to the REST gateway are instrumented by `otelhttp`, and gRPC
requests are instrumented by `otelgrpc`. The dashboard queries use the metric
names produced by the current OTel dependency set; if those dependencies change,
the dashboard panels may need to be updated.

## Logs

When `CALIBER_LOKI_URL` is set, the same redacted JSON log stream that is written
to stdout is also batched to Loki. Logs include `request_id` and `trace_id` so
requests can be correlated across traces, metrics, and logs.

## Dashboards

Grafana is provisioned with three dashboards under `deploy/grafana/dashboards/`:

1. **Caliber Service Health** — HTTP/gRPC RED metrics and target uptime.
2. **Caliber AI Usage** — AI call rates, failure/refusal rates, character rates,
   and latency.
3. **Caliber Queue Health** — enqueue rate, job processing rate, error rate,
   job duration, and failed job logs from Loki.

Dashboard JSON is stored in version control, so changes follow the normal PR
workflow.

## SLOs and alerts (CAL-134)

| SLO | Threshold | Alert |
|---|---|---|
| Availability | 99.9% over 5 minutes | `CaliberAPIDown`, `CaliberWorkerDown` |
| HTTP error rate | <1% | `CaliberHighHTTPErrorRate` |
| HTTP latency | p95 <2s | `CaliberHighHTTPLatency` |
| AI failure rate | <5% | `CaliberHighAIFailureRate` |
| Queue job error rate | <5% | `CaliberHighQueueJobErrorRate` |

Alert rules live in `deploy/prometheus/alerts.yml` and route through Alertmanager
to the local webhook echo receiver. In Docker Compose you can watch received
alerts with:

```bash
docker compose logs -f alertreceiver
```

Trigger an alert manually by stopping the API service:

```bash
docker compose stop api
```

Alertmanager itself is available at http://localhost:9093.

## Error tracking (CAL-135)

Code paths record structured errors with `errortracking.Record(ctx, err,
operation, class)`. Each record increments `caliber_errors_total` and emits a
redacted log line containing the error message, operation, class, and `trace_id`.
No payloads or PII are logged.

Typical error classes:

| Class | Source |
|---|---|
| `grpc` | gRPC unary and streaming handler errors |
| `llm` | LLM completion / warm-up failures |

Search recent errors in Loki:

```logql
{service="caliber-api"} | json | msg="operational_error"
```

Filter by class:

```logql
{service="caliber-api"} | json | msg="operational_error" | class="llm"
```

## Runbooks

Operational runbooks for every alert live in `docs/runbooks/`. The runbooks
include PromQL / Loki queries and `trace_id` correlation steps for the most
common failure modes.

> **Note:** The webhook receiver is a local demonstration target. Production
> deployments should replace it with a real receiver (Slack, PagerDuty, Opsgenie,
> etc.) in `deploy/alertmanager/alertmanager.yml`.
