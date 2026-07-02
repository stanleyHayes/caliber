# High queue job error rate

**Alert:** `CaliberHighQueueJobErrorRate`

## Symptoms

- Asynq job error rate is > 5% over 10 minutes.
- Dead-letter queue or retry counts grow.
- Background tasks (email, scoring, exports) are delayed.

## Impact

- User-facing async workflows are stuck or retried repeatedly.
- Downstream services may receive duplicate work on retry.

## Triage

1. Open Grafana **Caliber Queue Health** dashboard.
2. Confirm error rate:

   ```promql
   sum(rate(caliber_queue_jobs_total{status="failed"}[10m]))
   /
   sum(rate(caliber_queue_jobs_total[10m]))
   ```

3. Break down by task type:

   ```promql
   sum by (task_type) (rate(caliber_queue_jobs_total{status="failed"}[10m]))
   ```

4. Search worker logs for failed task traces:

   ```logql
   {service="caliber-worker"}
   | json
   | level="ERROR"
   ```

5. Check Redis/Asynq inspect UI or CLI for retry counts and dead tasks.

## Mitigation

- If a specific task type is failing, pause or drain that queue if the worker
  supports queue-level toggles.
- If the failure is downstream (DB, LLM, external API), fix or throttle the
  dependency.
- For poison messages causing repeated failures, move them to the dead-letter
  queue manually and open a bug.
- Scale workers if the backlog is caused by insufficient concurrency.

## Escalation

- Page the backend on-call if the error rate stays > 10% or the queue backlog
  is growing for more than 20 minutes.
- Escalate to data/platform if Redis or the worker pool itself appears unhealthy.
