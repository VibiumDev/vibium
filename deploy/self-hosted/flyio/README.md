# Fly.io self-hosted browser

A Chrome + chromedriver machine that vibium drives over Fly's WireGuard
tunnel. Per-second billing; a stopped machine costs only rootfs storage.

## Prereqs (human)

1. `brew install flyctl` (or the platform equivalent)
2. `fly auth signup` — trial gives ~2 hours of machine runtime; paid needs a card

## Deploy

```bash
cd deploy/self-hosted
fly launch --no-deploy --name vibium-browsers -c flyio/fly.toml   # first time
fly deploy -c flyio/fly.toml
```

## Connect

```bash
fly proxy 9515:9515 -a vibium-browsers   # keep running; it is the tunnel
vibium start http://127.0.0.1:9515       # classic endpoint → BiDi
vibium go https://example.com
vibium title
vibium stop
```

The app exposes no public ports — `fly proxy` is the only path in, so the
unauthenticated chromedriver is reachable only by your Fly account.

## Cost control

```bash
fly machine stop -a vibium-browsers     # ~$0 while stopped (rootfs only)
fly machine start -a vibium-browsers    # sub-second restart
```

Machines bill per-second, only while running.

## Fleet mode: N parallel browsers

`fly proxy` reaches one machine; for a fleet, use the WireGuard tunnel and
Fly's per-machine private DNS (`<machine-id>.vm.<app>.internal`) — one
tunnel addresses every machine concurrently:

```bash
fly wireguard create        # once; import the .conf into a WireGuard client
./fleet.sh up 10            # wake/clone until 10 machines run
./fleet.sh urls             # → http://<id>.vm.vibium-browsers.internal:9515 ×10
# hand one URL per parallel vibium session/client
./fleet.sh stop             # ≈$0 until next time; 'up' wakes them in <1s
```

A stopped fleet costs ~nothing; you pay per-second only for machines
that are running. (Brand-new Fly organizations may have soft
machine-count quotas; ask support to raise them once.)

## Everything-cloud variant

Instead of tunneling, install vibium *on* a Fly machine next to Chrome and
run your agent there too (`fly ssh console`). No tunnel, no network hop
between client and browser.
