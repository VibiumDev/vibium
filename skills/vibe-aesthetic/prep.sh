#!/usr/bin/env bash
# Ensure the vibium daemon is in a clean state, running headless.
# Required for non-interactive sessions (SSH, CI) where DISPLAY is empty —
# without --headless the daemon defaults to a visible browser and every
# capture silently produces 0-byte output.
#
# Usage: skills/vibe-aesthetic/prep.sh
# Exit codes: 0 on success, 1 if daemon won't start, 2 if browser won't launch.

set -euo pipefail
IFS=$'\n\t'

# Binary resolution (same convention as the vibe-* skill family).
resolve_vibium() {
  if command -v vibium >/dev/null 2>&1; then echo "vibium"; return; fi
  if [[ -x "./clicker/bin/vibium" ]]; then echo "./clicker/bin/vibium"; return; fi
  if [[ -x "./node_modules/.bin/vibium" ]]; then echo "./node_modules/.bin/vibium"; return; fi
  echo "ERROR: vibium binary not found" >&2
  exit 127
}
VIBIUM="$(resolve_vibium)"

# Bring down anything that might be stale.
"$VIBIUM" daemon stop >/dev/null 2>&1 || true
pkill -9 chromedriver 2>/dev/null || true
pkill -9 -f "chrome-for-testing" 2>/dev/null || true
sleep 1
rm -f "$HOME/.cache/vibium/vibium.sock" "$HOME/.cache/vibium/vibium.pid" 2>/dev/null || true
sleep 1

# Start with --headless inherited by all subsequent commands.
"$VIBIUM" --headless daemon start >/dev/null 2>&1
sleep 3

if ! "$VIBIUM" daemon status 2>&1 | grep -q "running"; then
  echo "ERROR: vibium daemon failed to start" >&2
  exit 1
fi

# Smoke-test browser session creation with a no-op JS expression.
if ! "$VIBIUM" eval "1+1" 2>&1 | grep -q "2"; then
  echo "ERROR: vibium browser session failed to launch" >&2
  echo "Check: ~/.cache/vibium/chrome-for-testing/ has chrome + chromedriver" >&2
  exit 2
fi

echo "vibium daemon ready (headless mode)"
