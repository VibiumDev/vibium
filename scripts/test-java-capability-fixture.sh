#!/bin/sh
# Assert the Java capability marker validator rejects the synthetic fixtures.
set -eu
cd "$(dirname "$0")/../clients/java"

expect_failure() {
    fixture=$1
    pattern=$2
    if output=$(./gradlew -q validateCapabilityMarkers \
        -PcapabilityMarkerRoot="../../tests/capability-fixtures/java/$fixture" 2>&1); then
        echo "validateCapabilityMarkers accepted $fixture:" >&2
        echo "$output" >&2
        exit 1
    fi
    case $output in
        *"$pattern"*) ;;
        *)
            echo "unexpected failure output for $fixture:" >&2
            echo "$output" >&2
            exit 1
            ;;
    esac
}

expect_failure missing-class-marker "missing class-level @RequiresCapability"
expect_failure unknown-capability "unknown capability nonexistent"
echo "Java capability validator fixtures ok"
