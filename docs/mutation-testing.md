# Mutation testing & flake control (CAL-144)

Line coverage (the ≥80% gate, CAL-139) proves code *ran* during tests; it does not
prove the tests would *fail* if the code were wrong. Mutation testing closes that
gap: it introduces small faults into the source (flip a conditional, tweak
arithmetic, drop a statement) and reruns the tests. A mutant the tests still pass
("LIVED") is a spot where the code could break silently — a missing assertion, not
a missing line. We run it on the **domain** (the pure business logic), where a
silent regression is most costly.

Tooling: [gremlins](https://gremlins.dev), configured in [`.gremlins.yaml`](../.gremlins.yaml).

## Running it

```bash
go install github.com/go-gremlins/gremlins/cmd/gremlins@latest
make mutation                                   # core domain packages
make mutation MUTATION_PKGS="matching guard"    # a subset
gremlins unleash ./internal/domain/matching     # one package, verbose
```

It is slow — the tests rerun once per mutant — so it is **not** a per-PR merge
gate. CI runs it on a weekly schedule ([.github/workflows/mutation.yml](../.github/workflows/mutation.yml))
and on demand. `make mutation` fails if a package's kill rate drops below the
floors in `.gremlins.yaml`, so a real weakening of the tests is caught.

## Baseline (set 2026-07-03)

Test efficacy = mutants killed / mutants covered. Recorded on the core domain:

| Domain package | Test efficacy | Mutator coverage |
|---|---|---|
| `matching` (explainable scoring engine) | 89.7% | 87.9% |
| `guard` (prompt-injection sanitiser) | 80.0% | 62.5% |
| `salary` | 25.0% | 100% |

`make mutation` runs `matching guard salary` — the pure-domain packages with
substantial **in-package** test suites, where per-package mutation testing is
meaningful. `matching` and `guard` — the highest-stakes logic (scoring, security) —
are strongly tested. `salary` is a tiny arithmetic helper whose surviving mutants
are boundary tweaks worth follow-up assertions — exactly the kind of gap mutation
testing surfaces that line coverage misses. The `.gremlins.yaml` floors (efficacy
15%, mutant-coverage 40%) sit below the baseline so the run passes today;
**ratchet them up** as tests gain assertions.

Other domain packages (`role`, `interview`, `candidateagent`, `kernel`) are
exercised mostly through the **app-layer** tests, not in-package, so gremlins run
against the domain package alone reports near-zero mutant coverage. Mutation-testing
those behaviours means running it against the app-layer suites that drive them — a
worthwhile follow-up, tracked as a docs note rather than forced into a misleading
per-package score. Override with `make mutation MUTATION_PKGS="…"` to explore any
package.

## Reading the output

- **KILLED** — a test failed on the mutant. Good: the behaviour is pinned.
- **LIVED** — all tests passed on the mutant. A gap: add an assertion that would
  have caught it (gremlins prints the file:line and mutator, e.g.
  `LIVED ARITHMETIC_BASE at market.go:84:52`).
- **NOT COVERED** — no test exercised the line; fix with a test (also a coverage gap).

## Flake control

Mutation testing amplifies flakiness: a non-deterministic test makes a mutant's
result unstable, polluting the score. Keep the suite deterministic and quarantine
flakes:

- **Detect** — `go test -race -count=3 ./...` surfaces order/timing-dependent
  tests; a test that isn't reliably green in three back-to-back runs is flaky.
- **Quarantine** — tag a confirmed flaky test `t.Skip("flaky: <link>")` with a
  tracking note so it neither blocks CI nor silently rots, and fix the root cause
  (shared state, real clocks, real randomness — inject them instead) before
  un-skipping.
- **Prevent** — the domain already injects clocks (`func() time.Time`) and avoids
  wall-clock/`rand` in logic, which is what keeps both the race suite and the
  mutation run stable.
