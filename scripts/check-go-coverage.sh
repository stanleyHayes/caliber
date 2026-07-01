#!/usr/bin/env bash
# Per-package Go coverage gate for Project Caliber (CAL-139).
# Fails if any non-excluded package with tests is below the threshold.
set -euo pipefail

threshold="${1:-80}"

exclude_re='node_modules|internal/gen/|internal/mocks/|internal/platform/migrate|internal/adapters/outbound/postgres|^github\.com/xcreativs/caliber/(cmd|web)|internal/adapters/inbound/httpserver|internal/adapters/inbound/jobs|internal/platform/wiring'

fail=0
while IFS=$'\t' read -r _ pkg _ cov_field; do
  if echo "$pkg" | grep -Eq "$exclude_re"; then
    continue
  fi
  cov=$(echo "$cov_field" | sed -E 's/.*coverage: ([0-9.]+)% of statements/\1/')
  if [[ -z "$cov" ]]; then
    continue
  fi
  printf "%s\t%s\n" "$cov%" "$pkg"
  if awk -v c="$cov" -v t="$threshold" 'BEGIN { exit (c+0 >= t+0) ? 0 : 1 }'; then
    :
  else
    fail=1
    echo "FAIL: $pkg coverage $cov% is below ${threshold}%" >&2
  fi
done < <(go test -cover ./... 2>&1 | grep -E '^ok\s+')

exit "$fail"
