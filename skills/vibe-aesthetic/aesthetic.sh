#!/usr/bin/env bash
# vibe-aesthetic — design evaluation capture.
#
# Orchestrates: prep daemon → walk page (desktop + mobile) → render
# evaluator prompt. The host LLM reads the rendered prompt + attached
# screenshots and writes AESTHETIC.md.
#
# Usage: aesthetic.sh <run-dir> <url> [--viewport mobile|desktop|both]
#                                      [--quick] [--brief "..."]
#                                      [--brief-from <path>]
#
# Output:
#   <run-dir>/probes/tokens_<viewport>.json
#   <run-dir>/sections/s<NN>_<id>__<viewport>.png
#   <run-dir>/walk.log
#   <run-dir>/PROMPT.filled.md
#   <run-dir>/RUN.md

set -euo pipefail
IFS=$'\n\t'

RUN_DIR="${1:?run dir required}"
URL="${2:?url required}"
shift 2

VIEWPORT="both"
QUICK=0
BRIEF=""
BRIEF_FROM=""
while (( $# )); do
  case "$1" in
    --viewport)   VIEWPORT="$2"; shift 2;;
    --quick)      QUICK=1; shift;;
    --brief)      BRIEF="$2"; shift 2;;
    --brief-from) BRIEF_FROM="$2"; shift 2;;
    *) echo "WARNING: unknown arg $1" >&2; shift;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROMPT_TPL="$SCRIPT_DIR/PROMPT.md"

mkdir -p "$RUN_DIR"
TS_START="$(date -u +%FT%TZ)"

echo "vibe-aesthetic → $URL"
echo "  run-dir: $RUN_DIR"
echo "  viewport: $VIEWPORT $([[ $QUICK == 1 ]] && echo '(quick)')"

# 1. Prep daemon.
echo "→ prepping daemon"
"$SCRIPT_DIR/prep.sh"

# 2. Walk.
echo "→ walking page"
walk_args=("$RUN_DIR" "$URL" --viewport "$VIEWPORT")
(( QUICK )) && walk_args+=(--quick)
"$SCRIPT_DIR/walk.sh" "${walk_args[@]}"

# 3. Render the prompt.
echo "→ rendering evaluator prompt"

# Resolve brief: --brief wins, else --brief-from, else meta description from probe.
RESOLVED_BRIEF=""
if [[ -n "$BRIEF" ]]; then
  RESOLVED_BRIEF="$BRIEF"
elif [[ -n "$BRIEF_FROM" && -f "$BRIEF_FROM" ]]; then
  RESOLVED_BRIEF="$(cat "$BRIEF_FROM")"
else
  # Try to pull meta description from the desktop probe.
  if [[ -f "$RUN_DIR/probes/tokens_desktop.json" ]]; then
    RESOLVED_BRIEF="$(python3 -c "
import json, sys
try:
  d = json.load(open('$RUN_DIR/probes/tokens_desktop.json'))
  print(d.get('meta', {}).get('description') or '')
except Exception:
  pass
" 2>/dev/null || echo "")"
  fi
fi
[[ -z "$RESOLVED_BRIEF" ]] && RESOLVED_BRIEF="(no brief provided — infer from page content)"

# List section screenshots in scroll order, per viewport.
section_list_for() {
  local vp="$1"
  find "$RUN_DIR/sections" -name "*__${vp}.png" 2>/dev/null | sort | while read -r p; do
    echo "  - ${p#$RUN_DIR/}"
  done
}

# Compose the filled prompt by inlining placeholders.
FILLED="$RUN_DIR/PROMPT.filled.md"
{
  if [[ -f "$PROMPT_TPL" ]]; then
    sed -e "s|{{URL}}|$URL|g" \
        -e "s|{{TIMESTAMP}}|$TS_START|g" \
        -e "s|{{VIEWPORT}}|$VIEWPORT|g" \
        "$PROMPT_TPL"
  fi
  printf "\n## Brief\n\n%s\n" "$RESOLVED_BRIEF"

  printf "\n## Captured screenshots\n"
  for vp in desktop mobile; do
    if find "$RUN_DIR/sections" -name "*__${vp}.png" 2>/dev/null | grep -q .; then
      printf "\n### %s\n" "$vp"
      section_list_for "$vp"
    fi
  done

  printf "\n## Design-token probes\n"
  for vp in desktop mobile; do
    if [[ -f "$RUN_DIR/probes/tokens_${vp}.json" ]]; then
      printf "\n### %s\n\n\`\`\`json\n" "$vp"
      cat "$RUN_DIR/probes/tokens_${vp}.json"
      printf "\n\`\`\`\n"
    fi
  done
} > "$FILLED"

# 4. RUN.md summary.
TS_END="$(date -u +%FT%TZ)"
shot_count=$(find "$RUN_DIR/sections" -name "*.png" 2>/dev/null | wc -l)
probe_count=$(find "$RUN_DIR/probes" -name "*.json" 2>/dev/null | wc -l)

cat > "$RUN_DIR/RUN.md" <<EOF
# vibe-aesthetic run

- URL: $URL
- Started: $TS_START
- Ended: $TS_END
- Viewport: $VIEWPORT$([[ $QUICK == 1 ]] && echo " (quick)")
- Brief: $RESOLVED_BRIEF
- Sections captured: $shot_count
- Probes captured: $probe_count

## Next step

The capture phase is complete. Hand the file \`PROMPT.filled.md\` to your
host LLM (Claude, GPT, etc.) along with the screenshots in \`sections/\`.
The LLM writes \`AESTHETIC.md\` in this run dir per the structure described
in the prompt.
EOF

echo ""
echo "DONE."
echo "  $shot_count screenshots in $RUN_DIR/sections/"
echo "  $probe_count probes in $RUN_DIR/probes/"
echo "  Prompt ready: $FILLED"
echo "  Run summary: $RUN_DIR/RUN.md"
