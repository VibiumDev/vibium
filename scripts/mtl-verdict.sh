#!/bin/sh
# Prints 1 if this machine pays the dead-virtual-GPU stall that mtlshim.dylib
# skips, 0 otherwise.
#
# Only a 1 is cached. The verdict is not stable on a UTM guest — the virtual
# GPU flips between working and dead across host sleep, VM restarts, and
# graphics-stack hiccups — so caching a 0 pins the shim off and every Chrome
# launch pays the full stall again, which is the bug the shim exists to fix.
#
# Caching asymmetrically is safe because the probe is only expensive in the
# case it caches: ~15s when affected (measured once, then remembered), ~0.01s
# when healthy (re-measured every run, too cheap to matter).
#
# Deliberately decides by probe rather than by environment: a VM with working
# GPU passthrough must NOT be shimmed, since the shim hides the GPU rather than
# speeding it up. See docs/how-to-guides/slow-chrome-launch-in-macos-vm.md
#
# Usage: scripts/mtl-verdict.sh <cache-file>

set -u
CACHE="${1:?usage: mtl-verdict.sh <cache-file>}"
SRC="$(dirname "$0")/mtlprobe.m"

# A cached 1 stands; a 0 is never written, so absence means "probe again".
if [ -f "$CACHE" ] && [ "$(cat "$CACHE")" = "1" ]; then
	printf '1'
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

if [ "$verdict" = "1" ]; then
	mkdir -p "$(dirname "$CACHE")" 2>/dev/null
	printf '1' >"$CACHE" 2>/dev/null
else
	rm -f "$CACHE" 2>/dev/null
fi
printf '%s' "$verdict"
