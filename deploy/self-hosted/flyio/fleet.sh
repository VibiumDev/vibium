#!/usr/bin/env bash
# Fleet control for self-hosted Fly.io browsers: N in parallel on demand.
#
#   ./fleet.sh up 10       # start/clone machines until 10 are running
#   ./fleet.sh urls        # one connect URL per running machine
#   ./fleet.sh stop        # stop all (≈$0 while stopped, sub-second restart)
#   ./fleet.sh destroy     # delete all machines
#   ./fleet.sh status
#
# URLs use Fly's per-machine private DNS (<machine-id>.vm.<app>.internal),
# reachable once a WireGuard tunnel is up:
#   fly wireguard create   # import the .conf into any WireGuard client
# One tunnel reaches the whole fleet concurrently.
set -euo pipefail

APP=${FLY_APP:-vibium-browsers}
CMD=${1:-status}
N=${2:-1}

machines() { fly machine list -a "$APP" --json; }
started_count() { machines | jq '[.[] | select(.state=="started")] | length'; }

case $CMD in
up)
    # Wake stopped machines first (sub-second), clone only for the shortfall.
    for id in $(machines | jq -r '.[] | select(.state=="stopped") | .id'); do
        [ "$(started_count)" -ge "$N" ] && break
        fly machine start "$id" -a "$APP"
    done
    template=$(machines | jq -r '.[0].id // empty')
    if [ -z "$template" ]; then
        echo "error: no machines in app '$APP' — run 'fly deploy -c flyio/fly.toml' first" >&2
        exit 1
    fi
    while [ "$(started_count)" -lt "$N" ]; do
        fly machine clone "$template" -a "$APP"
    done
    echo "$(started_count) machine(s) running"
    ;;
urls)
    machines | jq -r --arg app "$APP" \
        '.[] | select(.state=="started") | "http://\(.id).vm.\($app).internal:9515"'
    ;;
stop)
    machines | jq -r '.[] | select(.state=="started") | .id' \
        | xargs -r -n1 -I{} fly machine stop {} -a "$APP"
    ;;
destroy)
    machines | jq -r '.[].id' \
        | xargs -r -n1 -I{} fly machine destroy --force {} -a "$APP"
    ;;
status)
    machines | jq -r '.[] | "\(.id)\t\(.state)\t\(.region)"'
    ;;
*)
    echo "usage: fleet.sh up <n> | urls | stop | destroy | status" >&2
    exit 1
    ;;
esac
