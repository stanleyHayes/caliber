# Async scale-out & idempotency at volume (CAL-157)

Background work (candidate-agent runs, interview scoring, batch re-match,
retention sweeps) runs on Asynq workers. This describes how that tier scales out
horizontally while preserving **exactly-once effects**.

## Worker autoscaling

Workers are stateless — they pull tasks from Redis and call use-cases. So they
scale by **adding replicas** (Render service instances / k8s pods); no work is
pinned to a specific worker.

- Per-worker parallelism: `CALIBER_WORKER_CONCURRENCY` (asynq `Concurrency`).
- Fleet parallelism: the replica count. Scale replicas on queue depth /
  processing latency (Asynqmon and the queue metrics expose both).
- Total in-flight ≈ `replicas × CALIBER_WORKER_CONCURRENCY`. Size it against the
  downstream limits (the LLM concurrency/rate guard, the Postgres pool).

## Queue partitioning

Tasks are routed to weighted priority queues (`internal/app/queue`):
`critical` > `default` > `low` (`queue.Priorities()`), so latency-sensitive work
(e.g. interview scoring) is not starved by bulk work (e.g. batch re-match).
Callers pick a queue with the `queue.Queue(name)` dispatch option. Because the
weights are shared config, every replica drains the same priority mix, so adding
replicas scales each class proportionally.

## Exactly-once effects at volume

At-least-once delivery (retries, at-most-once-per-visibility-timeout redelivery,
and — once there are many replicas — the same task reaching two workers) means a
handler can run more than once. The handler framework guards effects with an
`IdempotencyStore`: **Claim** before doing work, **Complete** on success,
**Release** on failure (so a genuine retry can re-run).

- `MemoryIdempotencyStore` dedupes only within one process — fine for a single
  worker, unsafe across replicas.
- **`RedisIdempotencyStore` (CAL-157)** makes the claim atomic across the whole
  fleet with `SET NX`: the first replica to claim a key wins; a duplicate
  delivery or a second replica gets `claimed=false` and skips. `Complete`
  refreshes the key with a 24h suppression TTL; `Release` drops only *in-progress*
  claims (a completed key is retained, via an atomic GET-then-DEL Lua script, so
  it keeps rejecting late re-deliveries). A 15-minute claim TTL bounds recovery
  if a worker dies mid-task — the claim expires and another replica retries. The
  worker wires this store by default (`cmd/worker`), so exactly-once holds the
  moment you add a second replica. Tested against miniredis (claim/duplicate/
  release-retry/complete-suppresses/empty-key).

## Sustaining target throughput

The AC — sustains target job throughput — is validated by load, not a unit test:
drive the queues with the k6 suite (CAL-142) against a scaled worker fleet and
watch queue depth stay bounded and processing-latency SLOs hold, tuning replica
count + `CALIBER_WORKER_CONCURRENCY` against the downstream (LLM/DB) limits.
