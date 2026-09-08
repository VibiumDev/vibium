# Connect vibium to TestingBot

TestingBot exposes a classic WebDriver endpoint at
`hub.testingbot.com/wd/hub`. vibium creates the session there with
`webSocketUrl: true` and attaches to the BiDi socket the grid returns.
Credentials ride in the URL — the key goes in the username slot and
the secret in the password slot — and vibium folds them into Basic
auth.

Verified live against their hub: full lifecycle (session create,
navigate, title, screenshot, close) with session create taking about
12 seconds and screenshots returning real image data. No
selenium-version pin was needed — the defaults speak BiDi as-is.

## Prereqs (human)

1. Account at testingbot.com
2. Key and secret from the account page, exported as
   `TESTINGBOT_KEY` and `TESTINGBOT_SECRET`

## Connect

```bash
export VIBIUM_CONNECT_CAPS='{"browserName":"chrome","tb:options":{"name":"vibium"}}'
vibium start "https://$TESTINGBOT_KEY:$TESTINGBOT_SECRET@hub.testingbot.com/wd/hub"
vibium go https://example.com
vibium title
```

In the capabilities, `name` labels the session in their dashboard;
see their capability docs for the full `tb:options` surface (OS,
screen resolution, video, timeouts).

## MCP server

The MCP server reads the same connection from the environment — with
`VIBIUM_CONNECT_CAPS` still exported from above:

```bash
VIBIUM_CONNECT_URL="https://$TESTINGBOT_KEY:$TESTINGBOT_SECRET@hub.testingbot.com/wd/hub" vibium mcp
```

## Teardown

```bash
vibium stop
```

`vibium stop` deletes the WebDriver session, which releases the grid
slot immediately.

## Private sites

Their browsers can only reach public URLs. For an app on localhost or
a private network, their tunnel product covers reachability — see
[testing private sites](../testing-private-sites.md) for how tunnels
fit, and keep benchmarking on public targets so you measure the grid,
not the tunnel.
