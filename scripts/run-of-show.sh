#!/usr/bin/env bash
# Caliber run-of-show demo driver (CAL-105).
#
# Sequences the whole demo narrative end-to-end via the REST gateway:
#   Frame → Flow A → Flow B → Flow C → Close on Talent Radar.
#
# Defaults run against the offline, deterministic dev stack so the demo works
# without Docker, Postgres, Redis, or API keys. Set ANTHROPIC_API_KEY /
# OPENAI_API_KEY and CALIBER_DATABASE_URL to exercise live providers / Postgres.
#
# Usage:
#   bin/run-of-show.sh              # run the sequence and stop the API
#   bin/run-of-show.sh --keep-alive # keep API running after the sequence

set -uo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
API_BASE="${CALIBER_RUNOFSHOW_API_BASE:-http://localhost:8080}"
WEB_BASE="${CALIBER_RUNOFSHOW_WEB_BASE:-http://localhost:5173}"
SEED_GENERATED="${CALIBER_SEED_GENERATED:-true}"
MAX_WAIT="${CALIBER_RUNOFSHOW_MAX_WAIT:-90}"

if [[ "$SEED_GENERATED" == "true" ]]; then
  EMPLOYER_EMAIL="${CALIBER_RUNOFSHOW_EMPLOYER_EMAIL:-talent@mtn.com.gh}"
  CANDIDATE_EMAIL="${CALIBER_RUNOFSHOW_CANDIDATE_EMAIL:-ama.mensah.hero@example.com}"
else
  EMPLOYER_EMAIL="${CALIBER_RUNOFSHOW_EMPLOYER_EMAIL:-talent@mtn.com.gh}"
  CANDIDATE_EMAIL="${CALIBER_RUNOFSHOW_CANDIDATE_EMAIL:-ama.mensah@example.com}"
fi
PASSWORD="${CALIBER_RUNOFSHOW_PASSWORD:-Demo-Caliber-2026}"
KEEP_ALIVE=false

for arg in "$@"; do
  case "$arg" in
    --keep-alive) KEEP_ALIVE=true ;;
    -h|--help)
      sed -n '2,14p' "$0"
      exit 0
      ;;
    *) echo "Unknown argument: $arg" >&2; exit 1 ;;
  esac
done

# ---------------------------------------------------------------------------
# Colours
# ---------------------------------------------------------------------------
BOLD='\033[1m'
BLUE='\033[34m'
GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
RESET='\033[0m'

section() { echo -e "\n${BOLD}${BLUE}▶ $1${RESET}"; }
success() { echo -e "${GREEN}✓${RESET} $1"; }
warn()  { echo -e "${YELLOW}⚠${RESET} $1"; }
error() { echo -e "${RED}✗${RESET} $1"; }
link()  { echo -e "  ${BLUE}$1${RESET}"; }

# ---------------------------------------------------------------------------
# Dependencies
# ---------------------------------------------------------------------------
if ! command -v curl >/dev/null 2>&1; then
  error "curl is required"; exit 1
fi
if ! command -v jq >/dev/null 2>&1; then
  error "jq is required"; exit 1
fi

# ---------------------------------------------------------------------------
# API lifecycle
# ---------------------------------------------------------------------------
API_PID=""
API_WAS_RUNNING=false

cleanup_api() {
  if [[ "$KEEP_ALIVE" == "true" ]] || [[ "$API_WAS_RUNNING" == "true" ]]; then
    return
  fi
  if [[ -n "$API_PID" ]] && kill -0 "$API_PID" >/dev/null 2>&1; then
    kill "$API_PID" >/dev/null 2>&1
    wait "$API_PID" >/dev/null 2>&1 || true
  fi
}
trap cleanup_api EXIT INT TERM

api_healthy() {
  curl -sf "$API_BASE/healthz" >/dev/null 2>&1
}

wait_for_api() {
  local i
  for ((i = 0; i < MAX_WAIT; i++)); do
    api_healthy && return 0
    sleep 1
  done
  return 1
}

if api_healthy; then
  API_WAS_RUNNING=true
  warn "API already running at $API_BASE — reusing it (state may not be fresh)."
else
  section "Booting Caliber API (seed_generated=$SEED_GENERATED)…"
  if [[ -n "${CALIBER_DATABASE_URL:-}" ]]; then
    section "Reseeding Postgres via CAL-103 reseed command…"
    go run ./cmd/reseed >/tmp/caliber-reseed.log 2>&1 || {
      error "reseed failed"; cat /tmp/caliber-reseed.log >&2; exit 1
    }
  fi
  CALIBER_ENV=dev CALIBER_SEED_GENERATED="$SEED_GENERATED" go run ./cmd/api >/tmp/caliber-api.log 2>&1 &
  API_PID=$!
  if ! wait_for_api; then
    error "API failed to become healthy within ${MAX_WAIT}s"
    cat /tmp/caliber-api.log >&2
    exit 1
  fi
  success "API healthy at $API_BASE (pid $API_PID)"
fi

# ---------------------------------------------------------------------------
# HTTP helpers
# ---------------------------------------------------------------------------
http_post() {
  local url="$1" token="${2:-}" body="${3:-}"
  local -a cmd=(curl -sS -w "\n%{http_code}" -X POST "$url" -H 'Content-Type: application/json')
  [[ -n "$token" ]] && cmd+=(-H "Authorization: Bearer $token")
  [[ -n "$body" ]] && cmd+=(-d "$body")
  "${cmd[@]}"
}

http_get() {
  local url="$1" token="${2:-}"
  local -a cmd=(curl -sS -w "\n%{http_code}")
  [[ -n "$token" ]] && cmd+=(-H "Authorization: Bearer $token")
  cmd+=("$url")
  "${cmd[@]}"
}

# Extracts JSON body and HTTP code from a curl response, exits on non-2xx.
expect_2xx() {
  local raw="$1" label="$2"
  local code body
  code=$(echo "$raw" | tail -n1)
  body=$(echo "$raw" | sed '$d')
  if [[ ! "$code" =~ ^2 ]]; then
    error "$label failed (HTTP $code)"
    echo "$body" | jq . 2>/dev/null || echo "$body"
    exit 1
  fi
  echo "$body"
}

# ---------------------------------------------------------------------------
# Auth
# ---------------------------------------------------------------------------
login() {
  local email="$1"
  local resp
  resp=$(http_post "$API_BASE/v1/auth/login" "" "{\"email\":\"$email\",\"password\":\"$PASSWORD\"}")
  expect_2xx "$resp" "login ($email)"
}

section "Authenticating demo accounts…"
EMP_JSON=$(login "$EMPLOYER_EMAIL") || exit 1
EMP_TOKEN=$(echo "$EMP_JSON" | jq -r '.tokens.accessToken')
EMP_ID=$(echo "$EMP_JSON" | jq -r '.user.id')
success "Employer: $(echo "$EMP_JSON" | jq -r '.user.name') <$EMPLOYER_EMAIL>"

CAND_JSON=$(login "$CANDIDATE_EMAIL") || exit 1
CAND_TOKEN=$(echo "$CAND_JSON" | jq -r '.tokens.accessToken')
CAND_ID=$(echo "$CAND_JSON" | jq -r '.user.id')
success "Candidate: $(echo "$CAND_JSON" | jq -r '.user.name') <$CANDIDATE_EMAIL>"

# ---------------------------------------------------------------------------
# Frame — Talent Radar
# ---------------------------------------------------------------------------
section "Frame — Talent Radar"
link "$WEB_BASE/radar"

# Aggregate all pool pages so shortlist names can be resolved.
pages=()
page=1
while true; do
  body=$(expect_2xx "$(http_get "$API_BASE/v1/radar/pool?page.page=$page&page.page_size=20" "$EMP_TOKEN")" "radar pool page $page") || exit 1
  cnt=$(echo "$body" | jq '[.candidates[]] | length')
  pages+=("$body")
  total_pages=$(echo "$body" | jq -r '.page.totalPages')
  if [[ "$cnt" -eq 0 ]] || [[ "$page" -ge "$total_pages" ]]; then
    break
  fi
  page=$((page + 1))
done
POOL_JSON=$(printf '%s\n' "${pages[@]}" | jq -s '{candidates: [.[].candidates[]]}')

SUPPLY_JSON=$(expect_2xx "$(http_get "$API_BASE/v1/radar/supply-demand" "$EMP_TOKEN")" "radar supply/demand") || exit 1
TTS_JSON=$(expect_2xx "$(http_get "$API_BASE/v1/radar/time-to-shortlist" "$EMP_TOKEN")" "radar time-to-shortlist") || exit 1

POOL_COUNT=$(echo "$POOL_JSON" | jq '[.candidates[]] | length')
echo "  Pool candidates: $POOL_COUNT"
echo "  Supply/demand families: $(echo "$SUPPLY_JSON" | jq '[.items[]] | length')"
echo "  Time-to-shortlist: $(echo "$TTS_JSON" | jq -r '.metric.baselineHours')h → $(echo "$TTS_JSON" | jq -r '.metric.currentHours')h ($(echo "$TTS_JSON" | jq -r '.metric.improvementFactor')× faster)"

# Candidate name lookup table for nicer shortlist output.
CANDIDATE_MAP=$(echo "$POOL_JSON" | jq '[.candidates[] | {key: .candidateId, value: .name}] | from_entries')

# ---------------------------------------------------------------------------
# Flow A — Employer intake & explainable shortlisting
# ---------------------------------------------------------------------------
section "Flow A — Employer intake & explainable shortlisting"
link "$WEB_BASE/roles/new"

FREE_TEXT="We need a senior Go backend engineer in Accra to own our matching services — must know Postgres and gRPC, ideally some Kubernetes. GHS 18k–25k, start within a month."
ROLE_RESP=$(expect_2xx "$(http_post "$API_BASE/v1/roles:generate" "$EMP_TOKEN" "{\"employer_id\":\"$EMP_ID\",\"free_text\":\"$FREE_TEXT\"}")" "generate role spec") || exit 1
ROLE_ID=$(echo "$ROLE_RESP" | jq -r '.role.id')
ROLE_TITLE=$(echo "$ROLE_RESP" | jq -r '.role.spec.title')
AVAILABLE=$(echo "$ROLE_RESP" | jq -r '.availableMatches')

echo "  Generated role: $ROLE_TITLE"
echo "  Location: $(echo "$ROLE_RESP" | jq -r '.role.spec.location')"
echo "  Seniority: $(echo "$ROLE_RESP" | jq -r '.role.spec.seniority')"
echo "  Must-haves: $(echo "$ROLE_RESP" | jq -r '[.role.spec.mustHaves[]] | join(", ")')"
echo "  Rubric:"
echo "$ROLE_RESP" | jq -r '.role.rubric.competencies[] | "    - \(.name) (weight \(.weight), must-have: \(.mustHave))"'
echo "  Available matches: $AVAILABLE"

SHORTLIST_RESP=$(expect_2xx "$(http_get "$API_BASE/v1/roles/$ROLE_ID/shortlist?page.page=1&page.page_size=5" "$EMP_TOKEN")" "generate shortlist") || exit 1
POOL_DEPTH=$(echo "$SHORTLIST_RESP" | jq -r '.shortlist.poolDepth')
MATCH_COUNT=$(echo "$SHORTLIST_RESP" | jq '[.shortlist.matches[]] | length')
EXCLUSION_COUNT=$(echo "$SHORTLIST_RESP" | jq '[.shortlist.exclusions[]] | length')

echo "  Shortlist pool depth: $POOL_DEPTH"
echo "  Matches on this page: $MATCH_COUNT"
if [[ "$MATCH_COUNT" -gt 0 ]]; then
  echo "  Top matches:"
  echo "$SHORTLIST_RESP" | jq -r --argjson cmap "$CANDIDATE_MAP" '.shortlist.matches[] | "    - " + ($cmap[.candidateId] // .candidateId) + " | score: " + (.overallScore|tostring) + " | confidence: " + .confidence'
fi
if [[ "$EXCLUSION_COUNT" -gt 0 ]]; then
  echo "  Surfaced exclusions: $EXCLUSION_COUNT (first: $(echo "$SHORTLIST_RESP" | jq -r '.shortlist.exclusions[0].reason'))"
fi

# ---------------------------------------------------------------------------
# Flow B — AI screening interview (centrepiece)
# ---------------------------------------------------------------------------
section "Flow B — AI screening interview"
link "$WEB_BASE/interview?roleId=$ROLE_ID"

FIFO=/tmp/caliber-interview-$$
rm -f "$FIFO"
mkfifo "$FIFO"

curl -sS -N -X POST "$API_BASE/v1/interviews:start" \
  -H "Authorization: Bearer $CAND_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"roleId\":\"$ROLE_ID\",\"candidateId\":\"$CAND_ID\",\"mode\":\"INTERVIEW_MODE_TEXT\"}" \
  > "$FIFO" 2>/tmp/caliber-interview-stream.log &
STREAM_PID=$!
exec 3<>"$FIFO"
rm -f "$FIFO"

read_event() {
  local line
  if IFS= read -r -t 20 line <&3; then
    echo "$line"
  else
    echo ""
  fi
}

# Consume the opening status event.
EVENT=$(read_event)
if [[ -z "$EVENT" ]]; then
  error "Interview stream timed out before first event"
  cat /tmp/caliber-interview-stream.log >&2
  exit 1
fi

INTERVIEW_ID=""
Q_ORD=0
for ((i = 1; i <= 6; i++)); do
  EVENT=$(read_event)
  if [[ -z "$EVENT" ]]; then
    error "Interview stream closed unexpectedly"
    exit 1
  fi

  if echo "$EVENT" | jq -e '.result.reportCard' >/dev/null 2>&1; then
    REPORT_CARD="$EVENT"
    break
  fi

  if echo "$EVENT" | jq -e '.result.question' >/dev/null 2>&1; then
    INTERVIEW_ID=$(echo "$EVENT" | jq -r '.result.question.interviewId')
    Q_ORD=$(echo "$EVENT" | jq -r '.result.question.ordinal')
    Q_TEXT=$(echo "$EVENT" | jq -r '.result.question.text')
    Q_TAG=$(echo "$EVENT" | jq -r '.result.question.competencyTag')
    echo "  Q$Q_ORD [$Q_TAG]: $Q_TEXT"

    # Demo answer: concrete, first-person, measurable.
    ANSWER="I led a team of four to rebuild the payments gateway in Go, cut p99 latency from 400ms to 80ms, and rolled it out to production over six weeks."
    ANSWER_RESP=$(http_post "$API_BASE/v1/interviews/$INTERVIEW_ID/answers" "$CAND_TOKEN" "{\"answer\":\"$ANSWER\"}")
    expect_2xx "$ANSWER_RESP" "submit answer $Q_ORD" >/dev/null
    continue
  fi

  if echo "$EVENT" | jq -e '.result.turn' >/dev/null 2>&1; then
    echo "  Turn $(echo "$EVENT" | jq -r '.result.turn.ordinal') recorded"
    continue
  fi

  warn "Unhandled interview event: $EVENT"
done

exec 3<&-
kill "$STREAM_PID" >/dev/null 2>&1 || true
wait "$STREAM_PID" >/dev/null 2>&1 || true

if [[ -z "${REPORT_CARD:-}" ]]; then
  error "Interview did not produce a report card"
  exit 1
fi

VERDICT=$(echo "$REPORT_CARD" | jq -r '.result.reportCard.verdict')
CONFIDENCE=$(echo "$REPORT_CARD" | jq -r '.result.reportCard.confidence')
NEXT_STEP=$(echo "$REPORT_CARD" | jq -r '.result.reportCard.recommendedNextStep')
success "Report card — verdict: $VERDICT, confidence: $CONFIDENCE, next step: $NEXT_STEP"
echo "$REPORT_CARD" | jq -r '.result.reportCard.scores[] | "    - \(.competency): \(.score) — evidence: \(.evidence)"'

# ---------------------------------------------------------------------------
# Flow C — Candidate agent & wake-up view
# ---------------------------------------------------------------------------
section "Flow C — Candidate agent"
link "$WEB_BASE/agent"

FLOWC_RESP=$(expect_2xx "$(http_post "$API_BASE/v1/candidates/$CAND_ID/agent:timeAdvance" "$CAND_TOKEN" '{}')" "time advance") || exit 1
WAKE=$(echo "$FLOWC_RESP" | jq '.wakeUp')
echo "  New matches: $(echo "$WAKE" | jq -r '.newMatches')"
echo "  Applications submitted: $(echo "$WAKE" | jq -r '.applicationsSubmitted')"
echo "  Screenings completed: $(echo "$WAKE" | jq -r '.screeningsCompleted')"
echo "  Employers interested: $(echo "$WAKE" | jq -r '.employersInterested')"
echo "  Highlights:"
echo "$WAKE" | jq -r '.highlights[] | "    - " + .' | head -n 5

# ---------------------------------------------------------------------------
# Close — return to Radar
# ---------------------------------------------------------------------------
section "Close — Talent Radar"
link "$WEB_BASE/radar"

TTS_CLOSE=$(expect_2xx "$(http_get "$API_BASE/v1/radar/time-to-shortlist" "$EMP_TOKEN")" "radar time-to-shortlist") || exit 1
echo "  Time-to-shortlist: $(echo "$TTS_CLOSE" | jq -r '.metric.baselineHours')h → $(echo "$TTS_CLOSE" | jq -r '.metric.currentHours')h"
success "Run-of-show complete. Weeks to hours — with a full evidence trail."

if [[ "$KEEP_ALIVE" == "true" ]] || [[ "$API_WAS_RUNNING" == "true" ]]; then
  echo
  warn "API left running at $API_BASE. Press Ctrl-C or run 'kill $API_PID' to stop."
  if [[ "$KEEP_ALIVE" == "true" ]] && [[ -n "$API_PID" ]]; then
    wait "$API_PID" >/dev/null 2>&1 || true
  fi
fi
