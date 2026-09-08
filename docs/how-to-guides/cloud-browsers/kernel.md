# Connect vibium to Kernel

Kernel browsers return a WebDriver BiDi URL directly, so vibium
connects with no extra configuration — the auth token rides inside
the URL. Kernel documents this integration first-party at
[kernel.sh/docs/integrations/vibium](https://www.kernel.sh/docs/integrations/vibium);
this page is the vibium-side summary.

Verified live: every command below, verbatim, including teardown.
The full lifecycle (create, connect, navigate, title, screenshot,
close) runs in about 2 seconds — browser create is a few hundred
milliseconds, roughly an order of magnitude faster than the classic
WebDriver grids.

## Prereqs (human)

1. Account at dashboard.onkernel.com
2. An API key from the dashboard, exported as `KERNEL_API_KEY`
3. `jq`, or the Kernel CLI (`npm install -g @onkernel/cli`) if you
   prefer it over curl

## Create a browser and connect

```bash
browser_json=$(curl -s -X POST https://api.onkernel.com/browsers \
  -H "Authorization: Bearer $KERNEL_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"headless": true, "timeout_seconds": 300}')

vibium start "$(echo "$browser_json" | jq -r .webdriver_ws_url)"
vibium go https://example.com
vibium title
```

Notes on the create call: `headless` defaults to false (a headful
session) — set it explicitly to what you want. `timeout_seconds`
defaults to 60, counted once the browser goes idle; raise it for
sessions with quiet gaps.

## MCP server

The MCP server reads the same connection from the environment:

```bash
VIBIUM_CONNECT_URL="$(echo "$browser_json" | jq -r .webdriver_ws_url)" vibium mcp
```

## Teardown

```bash
vibium stop
curl -s -X DELETE "https://api.onkernel.com/browsers/$(echo "$browser_json" | jq -r .session_id)" \
  -H "Authorization: Bearer $KERNEL_API_KEY"
```

`vibium stop` closes the connection; deleting the Kernel session
releases the browser on their side rather than waiting out the
timeout.
