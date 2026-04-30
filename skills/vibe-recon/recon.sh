#!/usr/bin/env bash
# vibe-recon — map a web app's auth wall + edge surface. No login.
# Usage: recon.sh <run-dir> <url>

set -uo pipefail

RUN_DIR="${1:?run-dir required}"
BASE_URL="${2:?url required}"

if command -v vibium >/dev/null 2>&1; then
  VIBIUM=vibium
elif [ -x ./clicker/bin/vibium ]; then
  VIBIUM=./clicker/bin/vibium
elif [ -x ./node_modules/.bin/vibium ]; then
  VIBIUM=./node_modules/.bin/vibium
else
  echo "ERROR: cannot find vibium binary" >&2; exit 1
fi

mkdir -p "$RUN_DIR"
HERE="$(cd "$(dirname "$0")" && pwd)"
LOG="$RUN_DIR/recon.log"
: > "$LOG"
log() { printf '[%s] %s\n' "$(date +%H:%M:%S)" "$*" | tee -a "$LOG"; }

log "vibe-recon $BASE_URL"

# 1. Edge curl
log "edge curl"
curl -sSI -L --max-time 15 "$BASE_URL" > "$RUN_DIR/edge-headers.txt" 2>&1
log "  redirect chain: $(grep -c '^HTTP/' "$RUN_DIR/edge-headers.txt") response(s)"

# 2. Daemon
$VIBIUM daemon start >/dev/null 2>&1 || true

# 3. Navigate
log "navigate"
$VIBIUM go "$BASE_URL" >/dev/null 2>&1 || { log "ERROR: navigate failed"; exit 1; }
sleep 3

# 4. Probe
log "DOM probe"
$VIBIUM eval --stdin < "$HERE/probe.js" > "$RUN_DIR/landing.json" 2>/dev/null
LANDED_URL=$($VIBIUM eval 'location.href' 2>/dev/null | tr -d '"')
log "  landed at: $LANDED_URL"

# 5. Bundle grep against the LANDED origin (post-redirect)
LANDED_ORIGIN=$(printf '%s' "$LANDED_URL" | awk -F/ '{print $1"//"$3}')
log "bundle grep against $LANDED_ORIGIN"
"$HERE/bundle-grep.sh" "$RUN_DIR" "$LANDED_ORIGIN" 2>&1 | tee -a "$LOG"

# 6. Screenshot
log "screenshot"
$VIBIUM screenshot -o 01-landing.png >/dev/null 2>&1 || true
[ -f "$HOME/Pictures/Vibium/01-landing.png" ] && mv "$HOME/Pictures/Vibium/01-landing.png" "$RUN_DIR/01-landing.png"

log "done. artifacts in $RUN_DIR"
log "next: read $RUN_DIR/landing.json + edge-headers.txt + api-endpoints.txt and write RECON.md"
