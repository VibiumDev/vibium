#!/usr/bin/env bash
# vibe-diff — diff two vibe-inventory snapshots. No browser, no daemon.
# Usage: diff.sh <old-run-dir> <new-run-dir> [--out DIFF.md]

set -uo pipefail

OLD="${1:?old-run-dir required}"
NEW="${2:?new-run-dir required}"
OUT="$NEW/DIFF.md"

shift 2
while [ $# -gt 0 ]; do
  case "$1" in
    --out) OUT="$2"; shift 2 ;;
    *) echo "Unknown arg: $1" >&2; exit 1 ;;
  esac
done

for f in api-endpoints.txt frontend-route-patterns.txt; do
  [ -f "$OLD/$f" ] || { echo "ERROR: $OLD/$f not found (run vibe-inventory first)" >&2; exit 1; }
  [ -f "$NEW/$f" ] || { echo "ERROR: $NEW/$f not found (run vibe-inventory first)" >&2; exit 1; }
done

mktmp() { mktemp /tmp/vibe-diff.XXXXXX; }

API_NEW=$(mktmp); API_REM=$(mktmp)
ROUTE_NEW=$(mktmp); ROUTE_REM=$(mktmp)
PERM_NEW=$(mktmp); PERM_REM=$(mktmp)

comm -13 <(sort -u "$OLD/api-endpoints.txt") <(sort -u "$NEW/api-endpoints.txt") > "$API_NEW"
comm -23 <(sort -u "$OLD/api-endpoints.txt") <(sort -u "$NEW/api-endpoints.txt") > "$API_REM"
comm -13 <(sort -u "$OLD/frontend-route-patterns.txt") <(sort -u "$NEW/frontend-route-patterns.txt") > "$ROUTE_NEW"
comm -23 <(sort -u "$OLD/frontend-route-patterns.txt") <(sort -u "$NEW/frontend-route-patterns.txt") > "$ROUTE_REM"

if [ -f "$OLD/permissions.txt" ] && [ -f "$NEW/permissions.txt" ]; then
  comm -13 <(sort -u "$OLD/permissions.txt") <(sort -u "$NEW/permissions.txt") > "$PERM_NEW"
  comm -23 <(sort -u "$OLD/permissions.txt") <(sort -u "$NEW/permissions.txt") > "$PERM_REM"
fi

OLD_VER=$(grep -oE '"v[0-9]+\.[0-9]+\.[0-9]+"' "$OLD/app.bundle.js" 2>/dev/null | sort -u | head -1 | tr -d '"')
NEW_VER=$(grep -oE '"v[0-9]+\.[0-9]+\.[0-9]+"' "$NEW/app.bundle.js" 2>/dev/null | sort -u | head -1 | tr -d '"')
OLD_SIZE=$(wc -c < "$OLD/app.bundle.js" 2>/dev/null || echo 0)
NEW_SIZE=$(wc -c < "$NEW/app.bundle.js" 2>/dev/null || echo 0)
SIZE_DELTA=$((NEW_SIZE - OLD_SIZE))

NEW_API_COUNT=$(wc -l < "$API_NEW")
REM_API_COUNT=$(wc -l < "$API_REM")
NEW_ROUTE_COUNT=$(wc -l < "$ROUTE_NEW")
REM_ROUTE_COUNT=$(wc -l < "$ROUTE_REM")

{
  echo "# DIFF — $OLD → $NEW"
  echo ""
  echo "**Summary**: +${NEW_API_COUNT} endpoints / -${REM_API_COUNT} endpoints, +${NEW_ROUTE_COUNT} routes / -${REM_ROUTE_COUNT} routes, version ${OLD_VER:-?} → ${NEW_VER:-?}, bundle delta ${SIZE_DELTA} bytes"
  echo ""

  if [ "$NEW_API_COUNT" -gt 0 ]; then
    echo "## New API endpoints (${NEW_API_COUNT})"
    echo ""
    awk -F'/' '{print "/" $3}' "$API_NEW" | sort | uniq -c | sort -rn | while read -r count domain; do
      echo "### $domain ($count new)"
      grep "^$domain" "$API_NEW" | sed 's/^/- `/;s/$/`/'
      echo ""
    done
  fi

  if [ "$REM_API_COUNT" -gt 0 ]; then
    echo "## Removed API endpoints (${REM_API_COUNT})"
    echo ""
    awk -F'/' '{print "/" $3}' "$API_REM" | sort | uniq -c | sort -rn | while read -r count domain; do
      echo "### $domain ($count removed)"
      grep "^$domain" "$API_REM" | sed 's/^/- `/;s/$/`/'
      echo ""
    done
  fi

  if [ "$NEW_ROUTE_COUNT" -gt 0 ]; then
    echo "## New frontend routes"
    echo ""
    sed 's/^/- `/;s/$/`/' "$ROUTE_NEW"
    echo ""
  fi
  if [ "$REM_ROUTE_COUNT" -gt 0 ]; then
    echo "## Removed frontend routes"
    echo ""
    sed 's/^/- `/;s/$/`/' "$ROUTE_REM"
    echo ""
  fi

  if [ -s "$PERM_NEW" ]; then
    echo "## New permission tokens"
    echo ""
    sed 's/^/- /' "$PERM_NEW"
    echo ""
  fi
  if [ -s "$PERM_REM" ]; then
    echo "## Removed permission tokens"
    echo ""
    sed 's/^/- /' "$PERM_REM"
    echo ""
  fi

  echo "## Version + bundle"
  echo ""
  echo "| Metric | Old | New | Delta |"
  echo "|---|---|---|---|"
  echo "| Version | ${OLD_VER:-?} | ${NEW_VER:-?} | $([ "$OLD_VER" != "$NEW_VER" ] && echo CHANGED || echo same) |"
  echo "| Bundle size | $OLD_SIZE | $NEW_SIZE | $SIZE_DELTA bytes |"
  echo ""
} > "$OUT"

rm -f "$API_NEW" "$API_REM" "$ROUTE_NEW" "$ROUTE_REM" "$PERM_NEW" "$PERM_REM"

echo "Wrote $OUT"
