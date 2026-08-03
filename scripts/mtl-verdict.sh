#!/bin/sh
# Prints 1 if this machine pays the dead-virtual-GPU stall that mtlshim.dylib
# skips, 0 otherwise. The verdict is a property of the machine, so it is cached
# after the first run — the probe itself takes ~15s on an affected guest, which
# is precisely the cost being measured.
#
# Deliberately decides by probe rather than by environment: a VM with working
# GPU passthrough must NOT be shimmed, since the shim hides the GPU rather than
# speeding it up. See docs/how-to-guides/slow-chrome-launch-in-macos-vm.md
#
# Usage: scripts/mtl-verdict.sh <cache-file>

set -u
CACHE="${1:?usage: mtl-verdict.sh <cache-file>}"
SRC="$(dirname "$0")/mtlprobe.m"

if [ -f "$CACHE" ]; then
	cat "$CACHE"
	exit 0
fi

# Anything other than a successful "affected" verdict means leave it alone:
# no compiler, no Metal, a real GPU, or a probe that failed to run.
verdict=0
if [ "$(uname -s)" = "Darwin" ] && [ -f "$SRC" ]; then
	probe="$(mktemp -t vibium-mtlprobe)" || probe=""
	if [ -n "$probe" ]; then
		if clang -fobjc-arc -framework Metal -framework Foundation \
			-o "$probe" "$SRC" >/dev/null 2>&1; then
			if "$probe" >/dev/null 2>&1; then
				verdict=1
			fi
		fi
		rm -f "$probe"
	fi
fi

mkdir -p "$(dirname "$CACHE")" 2>/dev/null
printf '%s' "$verdict" >"$CACHE" 2>/dev/null
printf '%s' "$verdict"
