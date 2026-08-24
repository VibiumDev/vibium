#!/usr/bin/env bash
# vibe-inventory — full feature map of a SPA. Walks declared routes, grep bundle.
# Usage: inventory.sh <run-dir> <url> [--max-routes N] [--auth-required]

set -uo pipefail

RUN_DIR="${1:?run-dir required}"
BASE_URL="${2:?url required}"
shift 2

MAX_ROUTES=30
AUTH_REQUIRED=false

while [ $# -gt 0 ]; do
  case "$1" in
    --max-routes) MAX_ROUTES="$2"; shift 2 ;;
    --auth-required) AUTH_REQUIRED=true; shift ;;
    *) echo "Unknown arg: $1" >&2; exit 1 ;;
  esac
done

if command -v vibium >/dev/null 2>&1; then VIBIUM=vibium
elif [ -x ./clicker/bin/vibium ]; then VIBIUM=./clicker/bin/vibium
elif [ -x ./node_modules/.bin/vibium ]; then VIBIUM=./node_modules/.bin/vibium
else echo "ERROR: cannot find vibium binary" >&2; exit 1; fi

if ! command -v jq >/dev/null 2>&1; then
  echo "ERROR: jq is required" >&2; exit 1
fi

mkdir -p "$RUN_DIR"
HERE="$(cd "$(dirname "$0")" && pwd)"
LOG="$RUN_DIR/inventory.log"
: > "$LOG"
log() { printf '[%s] %s\n' "$(date +%H:%M:%S)" "$*" | tee -a "$LOG"; }

log "vibe-inventory $BASE_URL  (max_routes=$MAX_ROUTES)"

$VIBIUM daemon start >/dev/null 2>&1 || true

log "navigate"
$VIBIUM go "$BASE_URL" >/dev/null 2>&1 || { log "ERROR: navigate failed"; exit 1; }
sleep 3

if [ "$AUTH_REQUIRED" = true ]; then
  log "auth_required=true; pausing for operator login"
  echo ""
  echo "==> Sign in via the headed Chrome window. Press Enter when past the wall."
  read -r _
  $VIBIUM go "$BASE_URL" >/dev/null 2>&1
  sleep 2
fi

log "bundle grep"
LANDED_URL=$($VIBIUM eval 'location.href' 2>/dev/null | tr -d '"')
LANDED_ORIGIN=$(printf '%s' "$LANDED_URL" | awk -F/ '{print $1"//"$3}')
"$HERE/bundle-grep.sh" "$RUN_DIR" "$LANDED_ORIGIN" 2>&1 | tee -a "$LOG"

# Build route list from bundle + always include home
log "building route candidate list"
{
  echo "home:/"
  if [ -s "$RUN_DIR/frontend-route-patterns.txt" ]; then
    grep -v ':' "$RUN_DIR/frontend-route-patterns.txt" | grep -v '*' | head -$((MAX_ROUTES - 1)) | while read -r p; do
      [ -z "$p" ] && continue
      slug=$(echo "$p" | sed 's|^/||;s|/|-|g;s|[^a-zA-Z0-9-]|_|g')
      [ -z "$slug" ] && slug="root"
      echo "${slug}:${p}"
    done
  fi
} | head -$MAX_ROUTES > "$RUN_DIR/route-list.txt"

ROUTE_COUNT=$(wc -l < "$RUN_DIR/route-list.txt")
log "  walking $ROUTE_COUNT routes"

"$HERE/walk.sh" "$RUN_DIR" "$LANDED_ORIGIN" < "$RUN_DIR/route-list.txt" 2>&1 | tee -a "$LOG"

log "done. artifacts in $RUN_DIR"
log "next: read $RUN_DIR/walk.log + routes/*.json + api-endpoints.txt and write INVENTORY.md"
