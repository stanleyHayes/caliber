# 5xx spike

**Alerts:** `CaliberHighHTTPErrorRate`, `CaliberAPIDown`

## Symptoms

- Prometheus alert fires for HTTP 5xx rate > 1% over 5 minutes.
- `caliber_errors_total{class="grpc"}` is rising.
- Users report failed requests or gateway errors.

## Impact

- API availability drops below the 99.9% SLO.
- Affected endpoints may span identity, matching, interviews, or contests.

## Triage

1. Open Grafana **Caliber Service Health** dashboard.
2. Confirm the spike is real:

   ```promql
   sum(rate(http_server_duration_count{status_code=~"5.."}[5m]))
   /
   sum(rate(http_server_duration_count[5m]))
   ```

3. Find which gRPC methods are failing:

   ```promql
   sum by (operation) (rate(caliber_errors_total{class="grpc"}[5m]))
   ```

4. Get recent error logs with trace IDs:

   ```logql
   {service="caliber-api"}
   | json
   | msg="operational_error"
   | class="grpc"
   ```

5. Pick a `trace_id` and search traces or related logs to identify the root
   cause (database timeout, panic, downstream LLM failure, etc.).

## Mitigation

- If a specific deployment triggered it, consider a rollback.
- If a downstream service (LLM, Postgres, Redis) is degraded, follow the
  relevant runbook and/or apply circuit-breaker / fallback behavior if available.
- If a single pod/instance is unhealthy, restart or drain it.
- For runaway requests causing OOM/panics, apply a temporary rate limit or
  feature flag.

## Escalation

- Page the on-call backend engineer if 5xx rate stays > 5% for more than
  10 minutes.
- Page the SRE/infra lead if the issue correlates with infrastructure metrics
  (CPU, memory, disk, network).
