# Self-hosted cloud browser on Fly.io

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

## Teardown

```bash
fly apps destroy vibium-browsers
```

Stopped machines only cost rootfs storage, so teardown is optional
between runs — destroy when you're done with the app entirely.

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

## Machines vs. Sprites

Fly's other product, [Sprites](https://sprites.dev), is a persistent
agent sandbox rather than a deployed image:

- **No image to build.** A sprite is a ready Linux sandbox you install
  into over its CLI — closer to exe.dev's model than to `fly deploy`.
- **Pause, not stop.** After an idle window ("about 30 seconds today",
  per their docs) a sprite suspends with processes intact — Chrome
  included — and wakes warm in well under a second, billing ~nothing
  while paused. The catch for split mode is categorical, not
  occasional: their docs state open TCP connections **do not survive a
  pause, warm or cold** — so any tunneled browser session dies the
  moment a quiet gap outlasts the window. Their Tasks API can hold a
  sprite awake until a run finishes; or run the whole loop inside the
  sprite, where the processes talking over loopback pause and wake
  together.
- **Single public HTTPS port** (WebSocket behavior undocumented); other
  ports go through the sprite CLI's proxy. No per-sprite private-DNS
  fleet story like Machines have.

Rule of thumb: Machines for steady split-mode use and fleets; Sprites
when agent, vibium, and Chrome all live inside the sandbox.

To be concrete about fleets: BiDi through `sprite proxy` works exactly
like BiDi through `fly proxy` — the protocol doesn't care. But a
20–50 browser burst wants Machines, because one WireGuard tunnel
addresses the whole fleet by per-machine DNS, while sprites need one
proxy process per sprite and sever every open connection on pause. A
sprite *fleet* makes sense in the other shape: N everything-inside
workers, each running its own agent + vibium + Chrome loop with no
tunnels at all, idling to ~$0 between runs.

### Sprites recipe

```bash
curl -fsSL https://sprites.dev/install.sh | sh   # sprite CLI
sprite org auth
sprite create vibium-sprite --skip-console

# Install Chrome + chromedriver. No --service: sprites replace systemd
# with their own service runtime (next step).
sprite exec -s vibium-sprite \
  --file deploy/self-hosted/install-chrome.sh:/tmp/install-chrome.sh \
  -- sudo bash /tmp/install-chrome.sh

# Register chromedriver with that runtime: starts on boot, survives
# warm wakes mid-process, auto-restarts on crash.
sprite exec -s vibium-sprite -- \
  sprite-env services create chromedriver \
  --cmd /usr/local/bin/chromedriver --args "--port=9515"

sprite proxy -s vibium-sprite 9515 &     # the tunnel
vibium start http://127.0.0.1:9515
vibium go https://example.com
vibium stop

sprite destroy vibium-sprite             # teardown
```

If a long idle pause drops the tunneled session, run `vibium start`
again — the service comes back with the wake.

