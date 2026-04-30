#!/usr/bin/env bash
# vibe-explore — clicks every safe interactive element on a page, classifies what each does,
# and produces a ranked "what you can do here" report.
#
# Usage: explore.sh <run-dir> <url> [--max N] [--include-destructive] [--auth-required]

set -uo pipefail

RUN_DIR="${1:?run-dir required}"
BASE_URL="${2:?url required}"
shift 2

MAX_CLICKS=30
INCLUDE_DESTRUCTIVE=false
AUTH_REQUIRED=false

while [ $# -gt 0 ]; do
  case "$1" in
    --max) MAX_CLICKS="$2"; shift 2 ;;
    --include-destructive) INCLUDE_DESTRUCTIVE=true; shift ;;
    --auth-required) AUTH_REQUIRED=true; shift ;;
    *) echo "Unknown arg: $1" >&2; exit 1 ;;
  esac
done

# Locate the vibium binary (mirrors vibe-check resolution)
if command -v vibium >/dev/null 2>&1; then
  VIBIUM=vibium
elif [ -x ./clicker/bin/vibium ]; then
  VIBIUM=./clicker/bin/vibium
elif [ -x ./node_modules/.bin/vibium ]; then
  VIBIUM=./node_modules/.bin/vibium
else
  echo "ERROR: cannot find vibium binary on PATH or in repo dev locations" >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "ERROR: jq is required (sudo apt install jq / brew install jq)" >&2
  exit 1
fi

mkdir -p "$RUN_DIR/routes"
HERE="$(cd "$(dirname "$0")" && pwd)"
LOG="$RUN_DIR/explore.log"
RESULTS="$RUN_DIR/results.jsonl"
: > "$LOG"
: > "$RESULTS"

log() { printf '[%s] %s\n' "$(date +%H:%M:%S)" "$*" | tee -a "$LOG"; }

log "vibe-explore starting"
log "  base_url=$BASE_URL"
log "  max_clicks=$MAX_CLICKS"
log "  include_destructive=$INCLUDE_DESTRUCTIVE"
log "  auth_required=$AUTH_REQUIRED"

# 1. Daemon (idempotent)
$VIBIUM daemon start >/dev/null 2>&1 || true

# 2. Navigate
log "navigating to $BASE_URL"
$VIBIUM go "$BASE_URL" >/dev/null 2>&1 || { log "ERROR: navigate failed"; exit 1; }
sleep 3

# 3. Auth gate
if [ "$AUTH_REQUIRED" = true ]; then
  log "auth_required=true; pausing for operator login"
  echo ""
  echo "==> Sign in via the headed Chrome window. Press Enter when you're past the wall."
  read -r _
  log "operator confirmed login"
  $VIBIUM go "$BASE_URL" >/dev/null 2>&1
  sleep 2
fi

# 4. Dismiss consent banner
log "attempting to dismiss consent banner"
$VIBIUM eval --stdin < "$HERE/dismiss-consent.js" > "$RUN_DIR/consent.json" 2>/dev/null
DISMISSED=$(jq -r '.dismissed' "$RUN_DIR/consent.json" 2>/dev/null)
VENDOR=$(jq -r '.vendor // "none"' "$RUN_DIR/consent.json" 2>/dev/null)
log "  consent dismissed=$DISMISSED  vendor=$VENDOR"
sleep 1

# 5. Capture baseline
log "capturing baseline"
$VIBIUM eval --stdin < "$HERE/probe-state.js" > "$RUN_DIR/baseline.json" 2>/dev/null
$VIBIUM screenshot -o before.png >/dev/null 2>&1 || true
[ -f "$HOME/Pictures/Vibium/before.png" ] && mv "$HOME/Pictures/Vibium/before.png" "$RUN_DIR/before.png"
BASELINE_URL=$(jq -r '.url // empty' "$RUN_DIR/baseline.json")
log "  baseline_url=$BASELINE_URL"

# 6. Enumerate clickables
log "enumerating clickables"
ENUM_SCRIPT="$RUN_DIR/enumerate.js"
cp "$HERE/enumerate-clickables.js" "$ENUM_SCRIPT"
[ "$INCLUDE_DESTRUCTIVE" = true ] && sed -i.bak 's/INCLUDE_DESTRUCTIVE = false/INCLUDE_DESTRUCTIVE = true/' "$ENUM_SCRIPT" && rm -f "$ENUM_SCRIPT.bak"

$VIBIUM eval --stdin < "$ENUM_SCRIPT" > "$RUN_DIR/clickables.json" 2>/dev/null
TOTAL=$(jq -r '.summary.total' "$RUN_DIR/clickables.json")
SAFE=$(jq -r '.summary.safe' "$RUN_DIR/clickables.json")
log "  found total=$TOTAL safe=$SAFE"

# 7. Click loop
log "click loop starting (cap=$MAX_CLICKS)"
CLICKED=0
jq -c '.clickables[] | select(.safe == true)' "$RUN_DIR/clickables.json" | while read -r row; do
  if [ "$CLICKED" -ge "$MAX_CLICKS" ]; then
    log "reached --max=$MAX_CLICKS, stopping"
    break
  fi
  ORD=$(echo "$row" | jq -r '.ord')
  LABEL=$(echo "$row" | jq -r '.label')
  SELECTOR=$(echo "$row" | jq -r '.selector')

  SLUG=$(printf '%s' "$LABEL" | tr 'A-Z' 'a-z' | sed 's/[^a-z0-9]\+/-/g; s/^-//; s/-$//' | cut -c1-40)
  [ -z "$SLUG" ] && SLUG="ord$ORD"
  PREFIX=$(printf '%03d-%s' "$ORD" "$SLUG")

  log "[$ORD] click \"$LABEL\""

  $VIBIUM screenshot -o "${PREFIX}-before.png" >/dev/null 2>&1
  [ -f "$HOME/Pictures/Vibium/${PREFIX}-before.png" ] && mv "$HOME/Pictures/Vibium/${PREFIX}-before.png" "$RUN_DIR/routes/${PREFIX}-before.png"
  $VIBIUM eval --stdin < "$HERE/probe-state.js" > "$RUN_DIR/routes/${PREFIX}-before.json" 2>/dev/null

  CLICK_OUT=$($VIBIUM click "$SELECTOR" 2>&1)
  CLICK_OK=$?
  if [ $CLICK_OK -ne 0 ]; then
    log "  click failed: $CLICK_OUT"
    echo "{\"ord\":$ORD,\"label\":$(jq -Rs <<<"$LABEL"),\"outcome\":\"click_failed\",\"detail\":$(jq -Rs <<<"$CLICK_OUT")}" >> "$RESULTS"
    CLICKED=$((CLICKED + 1))
    continue
  fi
  sleep 1.5

  $VIBIUM eval --stdin < "$HERE/probe-state.js" > "$RUN_DIR/routes/${PREFIX}-after.json" 2>/dev/null
  $VIBIUM screenshot -o "${PREFIX}-after.png" >/dev/null 2>&1
  [ -f "$HOME/Pictures/Vibium/${PREFIX}-after.png" ] && mv "$HOME/Pictures/Vibium/${PREFIX}-after.png" "$RUN_DIR/routes/${PREFIX}-after.png"

  BEFORE_URL=$(jq -r '.url' "$RUN_DIR/routes/${PREFIX}-before.json")
  AFTER_URL=$(jq -r '.url' "$RUN_DIR/routes/${PREFIX}-after.json")
  BEFORE_HASH=$(jq -r '.body_text_hash' "$RUN_DIR/routes/${PREFIX}-before.json")
  AFTER_HASH=$(jq -r '.body_text_hash' "$RUN_DIR/routes/${PREFIX}-after.json")
  BEFORE_MOD=$(jq -r '.modal_count' "$RUN_DIR/routes/${PREFIX}-before.json")
  AFTER_MOD=$(jq -r '.modal_count' "$RUN_DIR/routes/${PREFIX}-after.json")
  AFTER_ERR=$(jq -r '.error_visible' "$RUN_DIR/routes/${PREFIX}-after.json")

  OUTCOME=unknown
  if [ "$AFTER_ERR" = "true" ]; then
    OUTCOME=route-error
  elif [ "$AFTER_URL" != "$BEFORE_URL" ]; then
    BEFORE_ORIGIN=$(printf '%s' "$BEFORE_URL" | awk -F/ '{print $1"//"$3}')
    AFTER_ORIGIN=$(printf '%s' "$AFTER_URL"  | awk -F/ '{print $1"//"$3}')
    [ "$BEFORE_ORIGIN" != "$AFTER_ORIGIN" ] && OUTCOME=external || OUTCOME=navigation
  elif [ "$AFTER_MOD" -gt "$BEFORE_MOD" ]; then
    OUTCOME=modal
  elif [ "$AFTER_HASH" != "$BEFORE_HASH" ]; then
    OUTCOME=inline-disclosure
  else
    OUTCOME=noop
  fi

  log "  outcome=$OUTCOME"
  echo "{\"ord\":$ORD,\"label\":$(jq -Rs <<<"$LABEL"),\"selector\":$(jq -Rs <<<"$SELECTOR"),\"outcome\":\"$OUTCOME\",\"before_url\":$(jq -Rs <<<"$BEFORE_URL"),\"after_url\":$(jq -Rs <<<"$AFTER_URL"),\"before_screenshot\":\"routes/${PREFIX}-before.png\",\"after_screenshot\":\"routes/${PREFIX}-after.png\"}" >> "$RESULTS"

  case "$OUTCOME" in
    navigation|route-error|external|click_failed)
      $VIBIUM go "$BASELINE_URL" >/dev/null 2>&1
      sleep 2
      ;;
    modal)
      $VIBIUM keys Escape >/dev/null 2>&1
      sleep 1
      AFTER_MOD2=$($VIBIUM eval --stdin < "$HERE/probe-state.js" 2>/dev/null | jq -r '.modal_count // 0')
      if [ "${AFTER_MOD2:-0}" -gt "$BEFORE_MOD" ]; then
        $VIBIUM go "$BASELINE_URL" >/dev/null 2>&1
        sleep 2
      fi
      ;;
    inline-disclosure)
      $VIBIUM click "$SELECTOR" >/dev/null 2>&1 || $VIBIUM go "$BASELINE_URL" >/dev/null 2>&1
      sleep 1
      ;;
    *)
      :
      ;;
  esac

  CLICKED=$((CLICKED + 1))
done

log "click loop done. clicked=$CLICKED  results in $RESULTS"
log "next step: read $RESULTS + screenshots and write EXPLORE.md"
