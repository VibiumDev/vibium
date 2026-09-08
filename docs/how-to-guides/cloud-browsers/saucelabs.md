# Connect vibium to Sauce Labs

Sauce Labs exposes a classic WebDriver endpoint per data center at
`ondemand.<region>.saucelabs.com/wd/hub`. vibium creates the session
there with `webSocketUrl: true` and attaches to the BiDi socket the
grid returns — credentials ride in the URL and vibium folds them into
Basic auth.

Verified live against their us-west-1 hub: full lifecycle (session
create, navigate, title, screenshot, close) with session create
taking about 15 seconds and close about 4–5 seconds; everything in
between is sub-second.

## Prereqs (human)

1. Account at saucelabs.com
2. Username and access key from the dashboard, exported as
   `SAUCE_USERNAME` and `SAUCE_ACCESS_KEY` — note credentials are per
   data center, so they must match the region you connect to

## Connect

```bash
export VIBIUM_CONNECT_CAPS='{"browserName":"chrome","sauce:options":{"name":"vibium","idleTimeout":300}}'
vibium start "https://$SAUCE_USERNAME:$SAUCE_ACCESS_KEY@ondemand.us-west-1.saucelabs.com/wd/hub"
vibium go https://example.com
vibium title
```

Swap `us-west-1` for your data center — their docs list US West and
EU Central for BiDi sessions. In the capabilities, `name` labels the
session in their dashboard and `idleTimeout` (seconds) ends an
abandoned session on their side; see their capability docs for the
full `sauce:options` surface.

Two limits from their docs: BiDi sessions are capped at 10 minutes —
plan work in sessions shorter than that rather than one long-lived
browser — and BiDi runs on chromium-based browsers only (Chrome,
Edge), so no Firefox here.

## MCP server

The MCP server reads the same connection from the environment — with
`VIBIUM_CONNECT_CAPS` still exported from above:

```bash
VIBIUM_CONNECT_URL="https://$SAUCE_USERNAME:$SAUCE_ACCESS_KEY@ondemand.us-west-1.saucelabs.com/wd/hub" vibium mcp
```

## Teardown

```bash
vibium stop
```

`vibium stop` deletes the WebDriver session, which releases the grid
slot — expect the delete to take a few seconds on their side.

## Private sites

Their browsers can only reach public URLs. For an app on localhost or
a private network, their tunnel product (Sauce Connect) covers
reachability — see [testing private sites](../testing-private-sites.md)
for how tunnels fit, and keep benchmarking on public targets so you
measure the grid, not the tunnel.
