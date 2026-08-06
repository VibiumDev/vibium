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

### Classic WebDriver endpoints (Selenium Grid, cloud grids)

An `http://` or `https://` URL is treated as a classic WebDriver endpoint.
vibium creates a session there with `webSocketUrl: true` and connects to the
BiDi URL the endpoint returns — one URL drives a local Selenium Grid or a
hosted grid cloud:

```bash
# Self-hosted Selenium Grid (or docker selenium standalone-chrome).
# Verified against selenium-server 4.46.0 standalone: vibium creates the
# session, attaches to the returned /se/bidi socket, and the slot is
# released on stop.
vibium start http://localhost:4444

# Cloud grid: credentials go in the URL (sent as Basic auth),
# vendor options in VIBIUM_CONNECT_CAPS
export VIBIUM_CONNECT_CAPS='{"vendor:options":{"someOption":"value"}}'
vibium start https://USER:KEY@grid.example.com/wd/hub
vibium go https://example.com
vibium stop   # sends DELETE /session/<id> — releases the cloud slot
```

Extra `alwaysMatch` capabilities come from `VIBIUM_CONNECT_CAPS` (a JSON
object), the `--connect-caps` flag on `vibium daemon start` / `vibium pipe`,
or the `caps` option in the JS/Python/Java clients. `webSocketUrl: true` is
always added for you. If the endpoint does not return a `webSocketUrl`, the
remote end does not support WebDriver BiDi, and vibium says so instead of
leaving the session running.

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

### Java

```groovy
implementation 'com.vibium:vibium:26.5.31'
```

```java
import com.vibium.Vibium;
import com.vibium.Browser;
import com.vibium.Page;
import com.vibium.types.StartOptions;

Browser bro = Vibium.start(new StartOptions()
    .connectURL("ws://your-server:9515/session"));
Page page = bro.page();

page.go("https://example.com");
System.out.println(page.title());    // "Example Domain"

bro.stop();
```

The Java client takes the connect URL through `StartOptions` — it does not
read the `VIBIUM_CONNECT_URL` environment variable.

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

**Java:**

```java
Map<String, String> headers = new HashMap<>();
headers.put("Authorization", "Bearer my-token");

Browser bro = Vibium.start(new StartOptions()
    .connectURL("wss://cloud.example.com/bidi")
    .connectHeaders(headers));
```

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `VIBIUM_CONNECT_URL` | Remote BiDi WebSocket endpoint (e.g. `ws://host:9515/session`) |
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
