#!/usr/bin/env bash
# Per-package Go coverage gate for Project Caliber (CAL-139).
# Fails if any non-excluded package with tests is below the threshold.
set -euo pipefail

threshold="${1:-80}"

exclude_re='node_modules|internal/gen/|internal/mocks/|internal/platform/migrate|internal/adapters/outbound/postgres|^github\.com/xcreativs/caliber/(cmd|web)|internal/adapters/inbound/httpserver|internal/adapters/inbound/jobs|internal/platform/wiring'

# Run the tests first, capturing output AND exit status. A package that fails to
# compile or whose tests fail emits no "ok" line, so parsing only "ok" lines
# would silently skip it; and because the loop below reads from a pipe, the
# go-test exit code would otherwise be discarded. Capturing both here means a
# broken build or a failing test fails the gate instead of passing it (CAL-139).
if ! test_output="$(go test -cover ./... 2>&1)"; then
  printf '%s\n' "$test_output" >&2
  echo "go test failed; coverage gate cannot pass" >&2
  exit 1
fi

fail=0
while IFS=$'\t' read -r _ pkg _ cov_field; do
  if echo "$pkg" | grep -Eq "$exclude_re"; then
    continue
  fi
  cov=$(echo "$cov_field" | sed -E 's/.*coverage: ([0-9.]+)% of statements/\1/')
  if [[ -z "$cov" ]] || ! [[ "$cov" =~ ^[0-9]+(\.[0-9]+)?$ ]]; then
    continue
  fi
  printf "%s\t%s\n" "$cov%" "$pkg"
  if awk -v c="$cov" -v t="$threshold" 'BEGIN { exit (c+0 >= t+0) ? 0 : 1 }'; then
    :
  else
    fail=1
    echo "FAIL: $pkg coverage $cov% is below ${threshold}%" >&2
  fi
done < <(printf '%s\n' "$test_output" | grep -E '^ok\s+')

exit "$fail"
