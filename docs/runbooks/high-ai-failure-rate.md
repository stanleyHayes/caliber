# High AI failure rate

**Alert:** `CaliberHighAIFailureRate`

## Symptoms

- AI failure rate is > 5% over 10 minutes.
- `caliber_errors_total{class="llm"}` is rising.
- `/debug/ai-quality` shows elevated failures, JSON failures, or refusals.

## Impact

- AI-powered features (interview generation, matching, candidate summaries) may
  return errors or degraded results.
- User experience degrades; background jobs may retry and queue backlog grows.

## Triage

1. Open Grafana **Caliber AI Usage** dashboard.
2. Confirm failure rate:

   ```promql
   sum(rate(caliber_ai_calls_total{status="failed"}[10m]))
   /
   sum(rate(caliber_ai_calls_total[10m]))
   ```

3. Find which operation is failing:

   ```promql
   sum by (operation) (rate(caliber_errors_total{class="llm"}[10m]))
   ```

4. Inspect LLM error logs:

   ```logql
   {service="caliber-api"}
   | json
   | msg="operational_error"
   | class="llm"
   ```

5. Look for provider-level errors (rate limits, timeouts, malformed payloads)
   using the `trace_id`.

## Mitigation

- If a single provider (Anthropic, OpenAI) is failing, fail over to the other
  provider if configured.
- If rate limits are the cause, reduce non-essential warm-up/background calls
  and increase retries/backoff.
- If JSON failures spike, check the prompt version and structured-output
  instructions for recent changes.
- If refusals spike, review guardrails and prompt safety settings.

## Escalation

- Page the ML/AI feature owner if failure rate stays > 10% for more than
  15 minutes.
- Escalate to the provider support channel if the issue is upstream and not
  resolved by failover.
