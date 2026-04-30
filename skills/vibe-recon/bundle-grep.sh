#!/usr/bin/env bash
# Pull the SPA's main JS bundle, grep React Router paths + /api/* endpoints.
# Usage: bundle-grep.sh <run-dir> <base-url>

set -uo pipefail
RUN_DIR="${1:?run dir}"
BASE="${2:?base url}"
BASE="${BASE%/}"

INDEX=$(curl -sS -L --max-time 15 "${BASE}/" || true)
# Match /assets/<name>.js but require a non-word boundary after .js so we don't
# accidentally truncate /assets/manifest.json into /assets/manifest.js.
BUNDLE_PATH=$(echo "$INDEX" | grep -oE '/assets/[a-zA-Z0-9._-]+\.js("|[^a-zA-Z])' | grep -v '\.json' | head -1 | sed 's/.$//')
if [ -z "$BUNDLE_PATH" ]; then
  BUNDLE_PATH=$(echo "$INDEX" | grep -oE 'src="[^"]+\.js"' | head -1 | sed 's/src="//;s/"$//')
fi
if [ -z "$BUNDLE_PATH" ]; then
  echo "no JS bundle found in ${BASE}/" > "$RUN_DIR/bundle-grep.log"
  : > "$RUN_DIR/api-endpoints.txt"
  : > "$RUN_DIR/frontend-route-patterns.txt"
  : > "$RUN_DIR/frontend-route-literals.txt"
  echo "no SPA bundle"
  exit 0
fi

BUNDLE_URL="${BASE}${BUNDLE_PATH}"
[[ "$BUNDLE_PATH" =~ ^https?:// ]] && BUNDLE_URL="$BUNDLE_PATH"

curl -sS --max-time 30 "$BUNDLE_URL" -o "$RUN_DIR/app.bundle.js"
BUNDLE_SIZE=$(wc -c < "$RUN_DIR/app.bundle.js")
echo "Bundle: $BUNDLE_URL (${BUNDLE_SIZE} bytes)"

grep -oE 'path:"[^"]{1,120}"' "$RUN_DIR/app.bundle.js" | sort -u | sed 's/^path:"//;s/"$//' | grep '^/api' > "$RUN_DIR/api-endpoints.txt" 2>/dev/null || true
grep -oE 'path:"[^"]{1,120}"' "$RUN_DIR/app.bundle.js" | sort -u | sed 's/^path:"//;s/"$//' | grep -v '^/api' > "$RUN_DIR/frontend-route-patterns.txt" 2>/dev/null || true
echo "API endpoints: $(wc -l < "$RUN_DIR/api-endpoints.txt")"
echo "Frontend route patterns: $(wc -l < "$RUN_DIR/frontend-route-patterns.txt")"

awk -F'/' '{print $3}' "$RUN_DIR/api-endpoints.txt" | sort | uniq -c | sort -rn > "$RUN_DIR/api-domains.txt"
echo "Distinct API domains: $(wc -l < "$RUN_DIR/api-domains.txt")"
