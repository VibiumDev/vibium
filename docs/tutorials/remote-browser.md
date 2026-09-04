# Remote Browser Control

Run Chrome on one machine, control it from another.

---

## Server (the machine with the browser)

Install vibium (this downloads Chrome + chromedriver automatically):

```bash
npm install -g vibium
```

Find the chromedriver path and start it:

```bash
vibium paths
# Chromedriver: /Users/you/.cache/vibium/.../chromedriver

$(vibium paths | grep Chromedriver | cut -d' ' -f2) --port=9515 --allowed-ips=""
```

---

## Client (your dev machine)

Install vibium locally — this gives you both the CLI (`npx vibium`) and the JS library:

```bash
npm install vibium
```

Or install globally for a bare `vibium` command:

```bash
npm install -g vibium
```

### CLI

```bash
# One-liner with env var (simplest)
export VIBIUM_CONNECT_URL=ws://your-server:9515/session
vibium go https://example.com
vibium title        # "Example Domain"
vibium text h1      # "Example Domain"
```

```bash
# Or use the start command with a URL
vibium start ws://your-server:9515/session
vibium go https://example.com
vibium title
vibium stop
```

Both endpoint shapes work: a browser-level URL like `ws://host:9515/session` (vibium creates the session) and a URL for a session that already exists, such as a chromedriver `webSocketUrl` or a Selenium Grid `ws://host:4444/session/<id>/se/bidi` (vibium attaches to it).

`http://` and `https://` URLs are accepted too and rewritten to `ws://` / `wss://`, so a hub URL copied from a cloud provider's docs works as-is.

### MCP Server

The MCP server reads the same env vars, so AI agents can use a remote browser:

```bash
VIBIUM_CONNECT_URL=ws://your-server:9515/session vibium mcp
```

Or in your Claude Desktop / Claude Code config:

```json
{
  "mcpServers": {
    "vibium": {
      "command": "vibium",
      "args": ["mcp"],
      "env": {
        "VIBIUM_CONNECT_URL": "ws://your-server:9515/session"
      }
    }
  }
}
```

### JavaScript

```javascript
import { browser } from 'vibium'

const bro = await browser.start('ws://your-server:9515/session')
const page = await bro.page()

await page.go('https://example.com')
console.log(await page.title())          // "Example Domain"
console.log(await page.find('h1').text()) // "Example Domain"

await bro.stop()
```

Sync API:

```javascript
const { browser } = require('vibium/sync')

const bro = browser.start('ws://your-server:9515/session')
const page = bro.page()

page.go('https://example.com')
console.log(page.title())        // "Example Domain"
console.log(page.find('h1').text())  // "Example Domain"

bro.stop()
```

### Python

```bash
pip install vibium
```

```python
from vibium.async_api import browser

bro = await browser.start("ws://your-server:9515/session")
page = await bro.page()

await page.go("https://example.com")
print(await page.title())          # "Example Domain"
print(await page.find("h1").text())    # "Example Domain"

await bro.stop()
```

Sync API:

```python
from vibium.sync_api import browser

bro = browser.start("ws://your-server:9515/session")
page = bro.page()

page.go("https://example.com")
print(page.title())          # "Example Domain"
print(page.find("h1").text())    # "Example Domain"

bro.stop()
```

---

## With Authentication

If your endpoint requires auth headers (e.g. a cloud browser provider):

**CLI / MCP** — set `VIBIUM_CONNECT_API_KEY` to send a `Bearer` token:

```bash
export VIBIUM_CONNECT_URL=wss://cloud.example.com/session
export VIBIUM_CONNECT_API_KEY=my-token
vibium go https://example.com
```

Or pass headers explicitly with the daemon:

```bash
vibium daemon start --connect wss://cloud.example.com/session \
  --connect-header "Authorization: Bearer my-token"
```

**JavaScript:**

```javascript
const bro = await browser.start('wss://cloud.example.com/bidi', {
  headers: { 'Authorization': 'Bearer my-token' }
})
```

Sync:

```javascript
const bro = browser.start('wss://cloud.example.com/bidi', {
  headers: { 'Authorization': 'Bearer my-token' }
})
```

**Python:**

```python
bro = await browser.start("wss://cloud.example.com/bidi", headers={
    "Authorization": "Bearer my-token",
})
```

Sync:

```python
bro = browser.start("wss://cloud.example.com/bidi", headers={
    "Authorization": "Bearer my-token",
})
```

### Credentials in the URL

Providers that document a hub URL with the credentials embedded — BrowserStack's
`https://username:accesskey@hub-cloud.browserstack.com`, for example — work
without any extra setup. Vibium moves the credentials out of the URL (a
WebSocket URL may not carry them) into an `Authorization: Basic` header:

```javascript
const bro = await browser.start('https://username:accesskey@hub-cloud.browserstack.com')
```

```python
bro = browser.start("https://username:accesskey@hub-cloud.browserstack.com")
```

```bash
vibium start "https://username:accesskey@hub-cloud.browserstack.com"
```

An explicit header wins over a credential in the URL, so `VIBIUM_CONNECT_API_KEY`
and `--connect-header` still override it. Credentials are replaced with
`redacted` wherever vibium prints the endpoint.

Prefer a header or `VIBIUM_CONNECT_API_KEY` where you can — a URL is easier to
leak into shell history and CI logs than an env var.

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `VIBIUM_CONNECT_URL` | Remote BiDi endpoint (e.g. `ws://host:9515/session`); `http://` and `https://` are rewritten to `ws://` / `wss://` |
| `VIBIUM_CONNECT_API_KEY` | Sent as `Authorization: Bearer <key>` |

These work everywhere — CLI commands, daemon auto-start, and the MCP server.

---

## How It Works

```
┌────────── Your Machine ──────────┐              ┌──── Remote Machine ─────┐
│                                  │              │                         │
│  ┌──────────┐    ┌──────────┐    │  WebSocket   │    ┌─────────────┐      │
│  │ your code│◄──►│  vibium  │◄───┼──────────────┼───►│ chromedriver│      │
│  └──────────┘    └──────────┘    │              │    └──────┬──────┘      │
│                                  │              │           │             │
│                                  │              │    ┌──────▼──────┐      │
│                                  │              │    │   Chrome    │      │
│                                  │              │    └─────────────┘      │
└──────────────────────────────────┘              └─────────────────────────┘
```

Your code talks to a local vibium process, which proxies to the remote chromedriver over WebSocket. The transport between your code and vibium depends on the interface: IPC for CLI, stdin/stdout pipes for JS/Python clients.

All vibium features (auto-wait, screenshots, tracing) work over remote connections.
