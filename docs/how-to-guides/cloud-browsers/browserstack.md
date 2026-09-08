# Connect vibium to BrowserStack

BrowserStack exposes a classic WebDriver endpoint at
`hub-cloud.browserstack.com/wd/hub`. vibium creates the session there
with `webSocketUrl: true` and attaches to the BiDi socket the grid
returns — credentials ride in the URL and vibium folds them into
Basic auth.

Verified live against their hub: session create (~12–13s), BiDi
attach, navigate, title, and slot release on stop — with one
exception, screenshots, covered below.

## Prereqs (human)

1. Account at browserstack.com (Automate)
2. Username and access key from the dashboard, exported as
   `BROWSERSTACK_USERNAME` and `BROWSERSTACK_ACCESS_KEY`

## Connect

```bash
export VIBIUM_CONNECT_CAPS='{"browserName":"chrome","bstack:options":{"seleniumVersion":"4.20.0","seleniumBidi":"true","sessionName":"vibium","idleTimeout":300}}'
vibium start "https://$BROWSERSTACK_USERNAME:$BROWSERSTACK_ACCESS_KEY@hub-cloud.browserstack.com/wd/hub"
vibium go https://example.com
vibium title
```

Two capabilities are load-bearing: `seleniumBidi: "true"` turns BiDi
on, and `seleniumVersion` must be 4.20.0 or above — omit it and the
hub refuses the session with
`BROWSERSTACK_BIDI_UNSUPPORTED_JAR_VERSION`. `sessionName` labels the
session in their dashboard; `idleTimeout` (seconds) ends an abandoned
session on their side.

## Screenshots don't work (their side)

Their BiDi proxy answers `browsingContext.captureScreenshot` with an
empty payload instead of image data — verified on seleniumVersion
4.20.0 and 4.31.0 as of August 2026. Their BiDi docs list Log and
Network as the supported modules; screenshots aren't among them.
Everything else vibium does (navigate, read, click, evaluate) works.
If your flow needs screenshot evidence, use a different vendor or ask
BrowserStack support about `captureScreenshot` on their BiDi
implementation.

## MCP server

The MCP server reads the same connection from the environment — with
`VIBIUM_CONNECT_CAPS` still exported from above:

```bash
VIBIUM_CONNECT_URL="https://$BROWSERSTACK_USERNAME:$BROWSERSTACK_ACCESS_KEY@hub-cloud.browserstack.com/wd/hub" vibium mcp
```

## Teardown

```bash
vibium stop
```

`vibium stop` deletes the WebDriver session, which releases the slot
and stops the Automate-minutes meter.

## Private sites

Their browsers can only reach public URLs. For an app on localhost or
a private network, their tunnel product (BrowserStack Local) covers
reachability — see [testing private sites](../testing-private-sites.md)
for how tunnels fit, and keep benchmarking on public targets so you
measure the grid, not the tunnel.
