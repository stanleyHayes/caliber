#!/usr/bin/env bash
# Offline/standby demo launcher (CAL-107).
#
# Boots the self-contained Docker Compose stack with local/dev AI providers so the
# Caliber POC survives a venue network drop. All images must already be built
# locally (run `make offline-build` while you still have network).
#
# Usage:
#   scripts/offline-demo.sh          # start the stack
#   scripts/offline-demo.sh --stop   # stop the stack
#   scripts/offline-demo.sh --check  # verify images and compose config

set -uo pipefail

COMPOSE_FILE="docker-compose.offline.yml"
API_BASE="http://localhost:8080"
WEB_BASE="http://localhost:5173"
MAX_WAIT="${CALIBER_OFFLINE_MAX_WAIT:-90}"

BOLD='\033[1m'
BLUE='\033[34m'
GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
RESET='\033[0m'

section() { echo -e "\n${BOLD}${BLUE}▶ $1${RESET}"; }
success() { echo -e "${GREEN}✓${RESET} $1"; }
warn()    { echo -e "${YELLOW}⚠${RESET} $1"; }
error()   { echo -e "${RED}✗${RESET} $1"; }

if ! command -v docker >/dev/null 2>&1; then
  error "docker is required"; exit 1
fi

usage() {
  sed -n '2,14p' "$0"
}

ACTION=up
for arg in "$@"; do
  case "$arg" in
    --stop)   ACTION=down ;;
    --check)  ACTION=check ;;
    -h|--help) usage; exit 0 ;;
    *) error "Unknown argument: $arg"; usage; exit 1 ;;
  esac
done

if [[ "$ACTION" == "down" ]]; then
  section "Stopping offline demo stack…"
  docker compose -f "$COMPOSE_FILE" down
  success "Offline stack stopped."
  exit 0
fi

if [[ "$ACTION" == "check" ]]; then
  section "Checking offline demo readiness…"
  docker compose -f "$COMPOSE_FILE" config >/dev/null || { error "compose config invalid"; exit 1; }
  success "Compose config valid"

  missing=()
  for img in pgvector/pgvector:pg17 redis:7-alpine node:24-alpine nginx:1.27-alpine golang:1.26.4 gcr.io/distroless/static-debian12:nonroot; do
    if ! docker image inspect "$img" >/dev/null 2>&1; then
      missing+=("$img")
    fi
  done
  if [[ ${#missing[@]} -gt 0 ]]; then
    warn "Base images not present locally: ${missing[*]}"
    warn "Run 'make offline-pull' (needs network) to pull them, or 'make offline-build' to build all images."
    exit 1
  fi
  success "Required base images are present locally"
  success "Offline demo is ready to start."
  exit 0
fi

section "Starting Caliber offline demo stack…"
section "Checking compose configuration…"
docker compose -f "$COMPOSE_FILE" config >/dev/null || { error "compose config invalid"; exit 1; }
success "Compose config valid"

docker compose -f "$COMPOSE_FILE" up -d --build || { error "failed to start stack"; exit 1; }

section "Waiting for API health…"
for ((i = 0; i < MAX_WAIT; i++)); do
  if curl -sf "$API_BASE/healthz" >/dev/null 2>&1; then
    success "API is healthy"
    break
  fi
  sleep 1
done

if ! curl -sf "$API_BASE/healthz" >/dev/null 2>&1; then
  error "API did not become healthy within ${MAX_WAIT}s"
  docker compose -f "$COMPOSE_FILE" logs api --tail 50
  exit 1
fi

section "Offline demo is live"
echo "  Web UI:   $WEB_BASE"
echo "  API:      $API_BASE"
echo "  Health:   $API_BASE/healthz"
echo "  Ready:    $API_BASE/readyz"
echo ""
echo "Demo accounts share password: Demo-Caliber-2026"
echo "  Employer: talent@mtn.com.gh"
echo "  Candidate: ama.mensah@example.com"
echo ""
echo "Stop with: make offline-stop"
