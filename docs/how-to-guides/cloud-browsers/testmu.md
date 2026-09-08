# Connect vibium to TestMu (LambdaTest)

TestMu exposes a classic WebDriver endpoint at
`hub.lambdatest.com/wd/hub`. vibium creates the session there with
`webSocketUrl: true` and attaches to the BiDi socket the grid returns
— credentials ride in the URL and vibium folds them into Basic auth.

Verified live against their hub: full lifecycle (session create,
navigate, title, screenshot, close) with session create taking
10–13 seconds and everything after it sub-second.

## Prereqs (human)

1. Account at lambdatest.com
2. Username and access key from the dashboard, exported as
   `LT_USERNAME` and `LT_ACCESS_KEY`

## Connect

```bash
export VIBIUM_CONNECT_CAPS='{"browserName":"chrome","browserVersion":"latest","LT:Options":{"name":"vibium","idleTimeout":300}}'
vibium start "https://$LT_USERNAME:$LT_ACCESS_KEY@hub.lambdatest.com/wd/hub"
vibium go https://example.com
vibium title
```

Notes on the capabilities: `name` labels the session in their
dashboard. `idleTimeout` (seconds) ends an abandoned session on their
side. See their capability docs for the full `LT:Options` surface
(OS, resolution, video, and more).

## MCP server

The MCP server reads the same connection from the environment — with
`VIBIUM_CONNECT_CAPS` still exported from above:

```bash
VIBIUM_CONNECT_URL="https://$LT_USERNAME:$LT_ACCESS_KEY@hub.lambdatest.com/wd/hub" vibium mcp
```

## Teardown

```bash
vibium stop
```

`vibium stop` deletes the WebDriver session, which releases the grid
slot immediately — no vendor-side cleanup call needed.

## Private sites

Their browsers can only reach public URLs. For an app on localhost or
a private network, their tunnel product covers reachability — see
[testing private sites](../testing-private-sites.md) for how tunnels
fit, and keep benchmarking on public targets so you measure the grid,
not the tunnel.
