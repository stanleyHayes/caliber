#!/usr/bin/env bash
# Load-test orchestration script (CAL-142).
# Starts a Docker Compose stack with the load-test overlay, reseeds the demo
# dataset, and runs k6. Tears the stack down unless --keep-alive is passed.
set -euo pipefail

PROJECT="caliber-load"
COMPOSE_FILES="-f docker-compose.yml -f docker-compose.load.yml"
SCENARIO="demo-traffic.js"
KEEP_ALIVE=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --smoke)
      SCENARIO="smoke.js"
      shift
      ;;
    --keep-alive)
      KEEP_ALIVE=true
      shift
      ;;
    *)
      echo "Usage: $0 [--smoke] [--keep-alive]"
      exit 1
      ;;
  esac
done

cd "$(dirname "$0")/.."

mkdir -p tests/load/results

echo "==> Starting load-test stack (project: $PROJECT)..."
# Only start the services the load test needs. Avoids host-port conflicts with
# Prometheus/Grafana/web that are not exercised by the harness.
docker compose -p "$PROJECT" $COMPOSE_FILES up --build -d postgres redis migrate api worker

cleanup() {
  if [[ "$KEEP_ALIVE" == "true" ]]; then
    echo "==> Keeping stack alive. Stop with: docker compose -p $PROJECT $COMPOSE_FILES down -v"
    return
  fi
  echo "==> Tearing down load-test stack..."
  docker compose -p "$PROJECT" $COMPOSE_FILES down -v || true
}
trap cleanup EXIT

echo "==> Waiting for API health..."
for i in {1..60}; do
  if curl -sf http://localhost:8080/healthz >/dev/null; then
    break
  fi
  sleep 2
done
if ! curl -sf http://localhost:8080/healthz >/dev/null; then
  echo "ERROR: API did not become healthy"
  exit 1
fi

echo "==> Seeding demo dataset..."
CALIBER_DATABASE_URL="postgres://caliber:caliber@localhost:5432/caliber?sslmode=disable" \
  go run ./cmd/reseed

echo "==> Resolving API container network..."
api_container=$(docker compose -p "$PROJECT" $COMPOSE_FILES ps -q api | head -n1)
if [[ -z "$api_container" ]]; then
  echo "ERROR: API container not found"
  exit 1
fi
network=$(docker inspect --format '{{range $k,$v := .NetworkSettings.Networks}}{{$k}}{{end}}' "$api_container")
if [[ -z "$network" ]]; then
  echo "ERROR: API container has no network"
  exit 1
fi
echo "    network: $network"

echo "==> Running k6 scenario: $SCENARIO ..."
k6_exit=0
docker run --rm \
  --network "$network" \
  -v "$(pwd)/tests/load:/tests:ro" \
  -e REST_ADDR="http://api:8080" \
  -e GRPC_ADDR="api:9090" \
  -e EMP_EMAIL="talent@mtn.com.gh" \
  -e CAND_EMAIL="ama.mensah@example.com" \
  -e PASSWORD="Demo-Caliber-2026" \
  -e K6_NO_USAGE_REPORT="true" \
  grafana/k6:latest run "/tests/scenarios/$SCENARIO" \
  | tee "tests/load/results/k6-summary.txt" \
  || k6_exit=${PIPESTATUS[0]}

if [[ "$k6_exit" -ne 0 ]]; then
  echo "ERROR: k6 exited with code $k6_exit"
  exit "$k6_exit"
fi

echo "==> Load test passed."
