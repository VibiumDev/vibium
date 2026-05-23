#!/usr/bin/env bash
# Section-aware aesthetic walker — captures probe + per-section screenshots
# at the requested viewport(s).
#
# Auto-discovers section anchors from the DOM (every `<section id="...">`
# with non-zero rendered height). Falls back to viewport-height stepping
# when no anchors are present.
#
# Usage: walk.sh <run-dir> <base-url> [--viewport mobile|desktop|both] [--quick]
# Output: <run-dir>/sections/s<NN>_<id>__<viewport>.png
#         <run-dir>/probes/tokens_<viewport>.json
#         <run-dir>/walk.log

set -euo pipefail
IFS=$'\n\t'

RUN_DIR="${1:?run dir required}"
BASE="${2:?base url required}"
shift 2

VIEWPORT="both"
QUICK=0
while (( $# )); do
  case "$1" in
    --viewport) VIEWPORT="$2"; shift 2;;
    --quick) QUICK=1; shift;;
    *) shift;;
  esac
done

BASE="${BASE%/}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROBE="$SCRIPT_DIR/probe.js"

mkdir -p "$RUN_DIR/sections" "$RUN_DIR/probes"
LOG="$RUN_DIR/walk.log"
: > "$LOG"

resolve_vibium() {
  if command -v vibium >/dev/null 2>&1; then echo "vibium"; return; fi
  if [[ -x "./clicker/bin/vibium" ]]; then echo "./clicker/bin/vibium"; return; fi
  if [[ -x "./node_modules/.bin/vibium" ]]; then echo "./node_modules/.bin/vibium"; return; fi
  echo "ERROR: vibium binary not found" >&2
  exit 127
}
VIBIUM="$(resolve_vibium)"

set_viewport() {
  case "$1" in
    mobile)  "$VIBIUM" viewport 390 844 --dpr 3 >/dev/null 2>&1 || true;;
    desktop) "$VIBIUM" viewport 1280 800 --dpr 1 >/dev/null 2>&1 || true;;
  esac
}

# Move screenshot from vibium's default save location into the run dir.
# Older vibium builds save to ~/Pictures/Vibium/ regardless of -o; newer
# builds (post-#119) honor the -o path. We handle both.
relocate_shot() {
  local out_name="$1"
  local target="$RUN_DIR/sections/$out_name"
  if [[ -f "$target" ]]; then return 0; fi
  if [[ -f "$HOME/Pictures/Vibium/$out_name" ]]; then
    mv "$HOME/Pictures/Vibium/$out_name" "$target"
  fi
}

capture_at() {
  local vp="$1" Y="$2" id="$3"
  local out="${id}__${vp}.png"
  "$VIBIUM" eval "window.scrollTo({top: $Y, behavior: 'smooth'}); '$id'" >/dev/null 2>&1 || true
  sleep 3
  "$VIBIUM" screenshot -o "$RUN_DIR/sections/$out" >/dev/null 2>&1 || true
  relocate_shot "$out"
  echo "  $id @ $vp y=$Y" >> "$LOG"
}

# Discover anchored sections in DOM order.
# Returns a JSON array of {id, top, height}.
discover_sections() {
  "$VIBIUM" eval --stdin <<'EOF' 2>/dev/null | tail -1
const sections = Array.from(document.querySelectorAll('section[id], main section[id], [data-section]'));
const seen = new Set();
const out = [];
for (const el of sections) {
  const r = el.getBoundingClientRect();
  if (r.height < 50) continue;
  const id = el.id || el.getAttribute('data-section') || '';
  if (!id || seen.has(id)) continue;
  seen.add(id);
  out.push({ id, top: Math.round(r.top + window.scrollY), height: Math.round(r.height) });
}
JSON.stringify(out);
EOF
}

probe_one_viewport() {
  local vp="$1"
  echo "→ $vp" | tee -a "$LOG"
  set_viewport "$vp"
  "$VIBIUM" go "$BASE" >/dev/null 2>&1 || true
  sleep 5

  # Top-of-page probe → tokens JSON.
  if ! "$VIBIUM" eval --stdin < "$PROBE" > "$RUN_DIR/probes/tokens_${vp}.json" 2>/dev/null; then
    echo "  probe-failed" >> "$LOG"
  fi

  # Always capture top.
  capture_at "$vp" 0 "s00_top"

  if (( QUICK )); then
    # Quick mode: one mid shot + footer, done.
    capture_at "$vp" 1200 "s01_mid"
    local Y_BOT
    Y_BOT=$("$VIBIUM" eval "document.documentElement.scrollHeight - window.innerHeight" 2>/dev/null | tail -1 | tr -d '"')
    capture_at "$vp" "${Y_BOT:-4000}" "s99_bottom"
    return
  fi

  # Discover section anchors.
  local sections_json
  sections_json="$(discover_sections || echo '[]')"
  echo "  sections: $sections_json" >> "$LOG"

  # Iterate sections by index.
  local n
  n=$(echo "$sections_json" | python3 -c "import json,sys; d=json.load(sys.stdin); print(len(d))" 2>/dev/null || echo 0)

  if (( n > 0 )); then
    local i=0
    while (( i < n )); do
      local id_top
      id_top=$(echo "$sections_json" | python3 -c "
import json,sys
d=json.load(sys.stdin)
e=d[$i]
print(f\"{e['id']}|{e['top']}\")" 2>/dev/null)
      local sec_id="${id_top%%|*}"
      local sec_y="${id_top##*|}"
      local idx
      idx=$(printf "s%02d_%s" $((i + 1)) "$sec_id")
      capture_at "$vp" "$sec_y" "$idx"
      i=$((i + 1))
    done
  else
    # Fallback: viewport-height stepping.
    local doc_h
    doc_h=$("$VIBIUM" eval "document.documentElement.scrollHeight" 2>/dev/null | tail -1 | tr -d '"')
    doc_h="${doc_h:-4000}"
    local step=${VP_H:-800}
    local y=$step
    local i=1
    while (( y < doc_h - step )); do
      local idx
      idx=$(printf "s%02d_step" $i)
      capture_at "$vp" "$y" "$idx"
      y=$((y + step))
      i=$((i + 1))
      (( i > 8 )) && break  # safety cap
    done
  fi

  # Always capture bottom.
  local Y_BOT
  Y_BOT=$("$VIBIUM" eval "document.documentElement.scrollHeight - window.innerHeight" 2>/dev/null | tail -1 | tr -d '"')
  capture_at "$vp" "${Y_BOT:-4000}" "s99_bottom"
}

VP_H=844
case "$VIEWPORT" in
  mobile)  VP_H=844; probe_one_viewport "mobile";;
  desktop) VP_H=800; probe_one_viewport "desktop";;
  both)
    VP_H=844; probe_one_viewport "mobile"
    VP_H=800; probe_one_viewport "desktop"
    ;;
  *) echo "ERROR: --viewport must be mobile|desktop|both" >&2; exit 64;;
esac

shot_count=$(find "$RUN_DIR/sections" -name "*.png" 2>/dev/null | wc -l)
probe_count=$(find "$RUN_DIR/probes" -name "*.json" 2>/dev/null | wc -l)
echo "DONE. $shot_count section shots, $probe_count probes."
