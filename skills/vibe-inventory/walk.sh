#!/usr/bin/env bash
# Walk a list of routes, JSON probe + PNG screenshot per route.
# Args: <run-dir> <base-url>
# stdin: lines of "slug:path"

set -uo pipefail
RUN_DIR="${1:?run dir}"
BASE="${2:?base url}"
BASE="${BASE%/}"

if command -v vibium >/dev/null 2>&1; then VIBIUM=vibium
elif [ -x ./clicker/bin/vibium ]; then VIBIUM=./clicker/bin/vibium
elif [ -x ./node_modules/.bin/vibium ]; then VIBIUM=./node_modules/.bin/vibium
else echo "ERROR: cannot find vibium binary" >&2; exit 1; fi

HERE="$(cd "$(dirname "$0")" && pwd)"
PROBE="$HERE/probe.js"
mkdir -p "$RUN_DIR/routes"
LOG="$RUN_DIR/walk.log"
: > "$LOG"

while IFS=: read -r SLUG P; do
  [ -z "$SLUG" ] && continue
  echo "→ $SLUG $P" | tee -a "$LOG"
  $VIBIUM go "${BASE}${P}" >/dev/null 2>&1
  sleep 3
  $VIBIUM eval --stdin < "$PROBE" > "$RUN_DIR/routes/${SLUG}.json" 2>/dev/null || echo "  eval-failed" >> "$LOG"
  $VIBIUM screenshot -o "${SLUG}.png" >/dev/null 2>&1 || true
  if [ -f "$HOME/Pictures/Vibium/${SLUG}.png" ]; then
    mv "$HOME/Pictures/Vibium/${SLUG}.png" "$RUN_DIR/routes/${SLUG}.png"
  fi
  URL=$(jq -r '.url // "?"' "$RUN_DIR/routes/${SLUG}.json" 2>/dev/null)
  ERR=$(jq -r '.isErrPage // false' "$RUN_DIR/routes/${SLUG}.json" 2>/dev/null)
  LEN=$(jq -r '.bodyTextLen // 0' "$RUN_DIR/routes/${SLUG}.json" 2>/dev/null)
  echo "    landed=$URL err=$ERR bodyLen=$LEN" | tee -a "$LOG"
done
