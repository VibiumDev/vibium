#!/usr/bin/env bash
# vibe-screenshot-all — one PNG per route + contact-sheet HTML.
# Usage: screenshot-all.sh <run-dir> <url> [--max-routes N] [--auth-required]

set -uo pipefail

RUN_DIR="${1:?run-dir required}"
BASE_URL="${2:?url required}"
shift 2

MAX_ROUTES=50
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

mkdir -p "$RUN_DIR/screens"
LOG="$RUN_DIR/walk.log"
: > "$LOG"
log() { printf '[%s] %s\n' "$(date +%H:%M:%S)" "$*" | tee -a "$LOG"; }

log "vibe-screenshot-all $BASE_URL  (max_routes=$MAX_ROUTES)"

$VIBIUM daemon start >/dev/null 2>&1 || true
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

LANDED_URL=$($VIBIUM eval 'location.href' 2>/dev/null | tr -d '"')
LANDED_ORIGIN=$(printf '%s' "$LANDED_URL" | awk -F/ '{print $1"//"$3}')

# Pull bundle + extract routes
log "extracting routes from bundle"
INDEX=$(curl -sS -L --max-time 15 "${LANDED_ORIGIN}/" || true)
BUNDLE_PATH=$(echo "$INDEX" | grep -oE '/assets/[a-zA-Z0-9._-]+\.js("|[^a-zA-Z])' | grep -v '\.json' | head -1 | sed 's/.$//')
[ -z "$BUNDLE_PATH" ] && BUNDLE_PATH=$(echo "$INDEX" | grep -oE 'src="[^"]+\.js"' | head -1 | sed 's/src="//;s/"$//')

if [ -n "$BUNDLE_PATH" ]; then
  BUNDLE_URL="${LANDED_ORIGIN}${BUNDLE_PATH}"
  [[ "$BUNDLE_PATH" =~ ^https?:// ]] && BUNDLE_URL="$BUNDLE_PATH"
  curl -sS --max-time 30 "$BUNDLE_URL" | grep -oE 'path:"[^"]{1,120}"' | sort -u | sed 's/^path:"//;s/"$//' | grep -v '^/api' | grep -v ':' | grep -v '*' > "$RUN_DIR/route-list.txt"
fi

# Always include home
{ echo "/"; cat "$RUN_DIR/route-list.txt" 2>/dev/null; } | sort -u | head -$MAX_ROUTES > "$RUN_DIR/route-list.txt.tmp"
mv "$RUN_DIR/route-list.txt.tmp" "$RUN_DIR/route-list.txt"

ROUTE_COUNT=$(wc -l < "$RUN_DIR/route-list.txt")
log "  walking $ROUTE_COUNT routes"

# Walk + screenshot
declare -a SHOTS=()
while read -r P; do
  [ -z "$P" ] && continue
  slug=$(echo "$P" | sed 's|^/||;s|/|-|g;s|[^a-zA-Z0-9-]|_|g')
  [ -z "$slug" ] && slug="home"
  log "→ $slug $P"
  $VIBIUM go "${LANDED_ORIGIN}${P}" >/dev/null 2>&1
  sleep 3
  $VIBIUM screenshot -o "${slug}.png" >/dev/null 2>&1 || true
  if [ -f "$HOME/Pictures/Vibium/${slug}.png" ]; then
    mv "$HOME/Pictures/Vibium/${slug}.png" "$RUN_DIR/screens/${slug}.png"
    LANDED=$($VIBIUM eval 'location.href' 2>/dev/null | tr -d '"')
    log "    landed=$LANDED  saved=screens/${slug}.png"
    SHOTS+=("$slug|$P|$LANDED")
  else
    log "    screenshot failed"
  fi
done < "$RUN_DIR/route-list.txt"

# Contact sheet
log "writing contact-sheet.html"
{
  cat <<'HEAD'
<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>vibe-screenshot-all contact sheet</title>
<style>
body{font-family:system-ui,sans-serif;background:#111;color:#eee;margin:0;padding:1rem}
h1{font-size:1rem;margin:0 0 1rem}
.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(320px,1fr));gap:1rem}
.card{background:#222;border-radius:6px;overflow:hidden}
.card img{width:100%;height:auto;display:block}
.meta{padding:.5rem .75rem;font-size:.8rem;font-family:monospace}
.meta .slug{color:#fa0;font-weight:600}
.meta .url{color:#7df;word-break:break-all}
.meta .landed{color:#888;font-size:.7rem;margin-top:.25rem}
</style></head><body>
HEAD
  echo "<h1>$BASE_URL — ${#SHOTS[@]} routes</h1>"
  echo "<div class=grid>"
  for entry in "${SHOTS[@]}"; do
    IFS='|' read -r slug path landed <<< "$entry"
    printf '<div class=card><img src="screens/%s.png"><div class=meta><div class=slug>%s</div><div class=url>%s</div><div class=landed>landed: %s</div></div></div>\n' "$slug" "$slug" "$path" "$landed"
  done
  echo "</div></body></html>"
} > "$RUN_DIR/contact-sheet.html"

log "done. open $RUN_DIR/contact-sheet.html"
