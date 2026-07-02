# High latency

**Alert:** `CaliberHighHTTPLatency`

## Symptoms

- HTTP/gRPC p95 latency exceeds 2 seconds over 10 minutes.
- Users report slowness; dashboards show latency percentiles climbing.
- `caliber_errors_total` may or may not be rising.

## Impact

- Degraded user experience; timeouts may turn into 5xx errors.
- SLO breach if sustained.

## Triage

1. Open Grafana **Caliber Service Health** dashboard.
2. Confirm p95 latency:

   ```promql
   histogram_quantile(0.95,
     sum(rate(http_server_duration_bucket[10m])) by (le))
   ```

3. Compare gRPC method latency:

   ```promql
   histogram_quantile(0.95,
     sum(rate(rpc_server_duration_bucket[10m])) by (grpc_method, le))
   ```

4. Check downstream dependency latency:

   - Postgres: query latency from application logs or `pg_stat_statements`.
   - LLM: `caliber_ai_latency_seconds`.
   - Redis: `redis_command_duration_seconds` (if exported).

5. Correlate with resource metrics (CPU, memory, connections).

## Mitigation

- If a specific query or RPC is slow, apply a short-term timeout reduction or
  query optimization.
- If downstream dependency latency is the cause, scale or throttle that
  dependency.
- If the API is overloaded, scale pods/instances or enable load shedding.
- If a recent release introduced the regression, consider rolling back.

## Escalation

- Page the on-call backend engineer if p95 latency stays > 5s for more than
  15 minutes.
- Escalate to infrastructure if resource saturation is the root cause.
