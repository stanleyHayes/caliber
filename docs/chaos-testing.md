# Chaos and resilience testing

CAL-143 is covered with deterministic fault-injection tests instead of live
process killing in CI. The goal is the same: downstream failures must surface as
clean typed or wrapped errors, must not panic, and must not write fabricated or
partial AI outputs.

## Coverage matrix

| Failure | Coverage | Expected behavior |
| --- | --- | --- |
| LLM unavailable while starting an interview | `internal/app/interview/interviewer_resilience_test.go` | No interview row is created; the caller receives an error. |
| LLM unavailable while scoring an interview | `internal/app/interview/interviewer_resilience_test.go` | No report card is fabricated; the interview is not closed. |
| Malformed LLM scoring output | `internal/app/interview/interviewer_resilience_test.go` | Retries are exhausted, `KindInvalid` is returned, and no report card is attached. |
| LLM budget or rate limit exhausted | `internal/app/interview/interviewer_resilience_test.go`, `internal/app/roles/generate_resilience_test.go` | The operation fails fast with a clean error before additional writes. |
| DB or persistence port unavailable | `internal/app/interview/interviewer_resilience_test.go`, `internal/app/roles/generate_resilience_test.go` | The error is returned cleanly; generated artifacts are not exposed as durable success. |
| Embeddings provider unavailable | `internal/app/roles/generate_resilience_test.go` | Role creation aborts before persistence, so roles are not stored without vectors. |
| Redis unavailable for task enqueue | `internal/adapters/outbound/queue/asynq_test.go` | The dispatcher returns a wrapped enqueue error; the caller can retry or fall back. |
| Redis down at the gRPC boundary (enqueue fails) | `internal/adapters/inbound/grpc/resilience_test.go` | `RunAgent`/`TimeAdvance` return a typed gRPC error and no bogus job id; raw redis detail is not leaked to the client. |
| DB down at the gRPC boundary (repo read errors) | `internal/adapters/inbound/grpc/resilience_test.go` | The handler returns an opaque gRPC `Internal`; it never panics. |
| Error contract (every `kernel.Kind` + raw infra errors) | `internal/adapters/inbound/grpc/resilience_test.go` (`TestErrToStatusMapsEveryKind`) | Full domain-kind → gRPC-code table pinned; infra errors collapse to opaque `Internal`. |
| Real Postgres pool closed mid-operation | `internal/adapters/outbound/postgres/pool_chaos_test.go` | The repo read/write errors cleanly, never panics, and a connection loss is not misreported as `NotFound`. |
| Redis readiness failure | `internal/platform/readiness/readiness_test.go` | `/readyz` dependency checks fail closed instead of reporting ready. |
| Poison queue task | `internal/adapters/inbound/jobs/archive_test.go` | Retries exhaust and the task is archived with a structured dead-letter log. |

## The error contract (what a client sees when a dependency is down)

`errToStatus` (`internal/adapters/inbound/grpc/mapping.go`) is the single mapping
point. The kernel deliberately has **no `Unavailable` kind**: an infrastructure
fault (dead DB, dead Redis, LLM transport error) arrives as an *unclassified*
error and maps to gRPC **`codes.Internal`** with an opaque `"internal error"`
message — the raw pgx/redis text is logged server-side but never returned to the
client (CWE-209). Author-controlled kinds keep their explanatory messages:

| `kernel.Kind` | gRPC code |
| --- | --- |
| `KindInvalid` | `InvalidArgument` |
| `KindNotFound` | `NotFound` |
| `KindConflict` | `AlreadyExists` |
| `KindUnauthorized` | `Unauthenticated` |
| `KindForbidden` | `PermissionDenied` |
| `KindTooManyRequests` | `ResourceExhausted` |
| unclassified / `KindInternal` (DB/Redis/LLM down) | `Internal` (opaque) |

The tests assert this **real** behavior rather than an aspirational `Unavailable`
(see `GAP-2`).

## Known gap

`GAP-1`: `Interviewer.Answer` records the answer in memory, then asks the next
LLM question or scores the transcript before calling `InterviewRepository.Update`.
If the downstream LLM call fails at that point, the just-submitted answer is not
durably persisted and the candidate must submit it again. The executable test
`TestAnswerLLMFailureOnNextQuestionDropsInFlightAnswer` pins this current
behavior so a future persist-before-generate fallback can change the expectation
intentionally.

`GAP-2`: DB/Redis/LLM outages surface as gRPC `Internal`, not `Unavailable` (see
the error contract above). This is a deliberate, safe default — opaque, with no
raw detail leaked — and is fine for the POC, but a client cannot today distinguish
"retry me, I'm down" from "I broke". Adding a `KindUnavailable` in
`internal/domain/kernel/errors.go` plus a mapper arm to `codes.Unavailable` is a
small, isolated follow-up if that distinction becomes valuable.

The production runbook fallback for provider or venue-network failure remains:
switch to the deterministic `dev` LLM/embedder path by unsetting provider API
keys and restart the API/worker stack, as documented in `docs/demo-runbook.md`.

## No-fabrication on the failure path

When the scoring LLM call fails or returns unparseable output, the interview
use-case returns a typed error and attaches **no** report card, and the interview
is **not** advanced to `closed` — a verdict is never invented. This is enforced
twice: the use-case does not build a card on error, and the domain's
`ReportCard.Validate()` rejects any card with zero scores or a score lacking
evidence. The deterministic dev provider (`llm.Dev`, the offline fallback used
without an API key) is likewise grounded — `devReport` quotes the candidate's own
answer as evidence and emits `"no concrete example was provided"` when there is
none, so the degraded path never fabricates skills either.

Note: `ModeText` is the interview's reliable default *modality*
(`interviewModeFromProto` falls back to text), not an LLM-outage fallback — it does
not substitute for the model being down.

## Running the tests

```bash
# Unit-level resilience tests (no Docker) — part of the standard fast suite:
go test -short ./...

# Just the resilience suites:
go test -short ./internal/app/interview/... ./internal/app/roles/... \
                ./internal/adapters/inbound/grpc/...

# The container-backed DB chaos test (needs Docker; skipped by -short and
# skipped fast when Docker is down):
go test -run TestRoleRepoDegradesWhenPoolClosed ./internal/adapters/outbound/postgres/
```

The unit-level tests run on every push inside the ≥80%-coverage suite. The
container-backed test runs in CI (which provides Docker) and on demand locally; it
self-skips cleanly when Docker is unavailable, so it never breaks a Docker-less
`make test`.
