#!/usr/bin/env bash
set -euo pipefail

ITERATIONS="${1:-10}"

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

for i in $(seq 1 "$ITERATIONS"); do
  echo "=== Vibium Java integration soak: run $i/$ITERATIONS ==="
  mvn -q verify -DskipITs=false
done

echo "OK: $ITERATIONS integration runs passed"

