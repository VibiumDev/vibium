# Vibium

**Browser automation without the drama.**

Vibium is browser automation infrastructure built for AI agents. A single binary handles browser lifecycle, [WebDriver BiDi](docs/explanation/webdriver-bidi.md) protocol, and exposes an MCP server — so Claude Code (or any MCP client) can drive a browser with zero setup. Works great for AI agents, test automation, and anything else that needs a browser.

**New here?** [Getting Started Tutorial](docs/tutorials/getting-started.md) — zero to hello world in 5 minutes.

---

## Why Vibium?

**Browser automation for AI agents and humans.**

- **AI-native.** MCP server built-in. Claude Code can drive a browser out of the box.
- **Zero config.** One install, browser downloads automatically, visible by default.
- **Standards-based.** Built on [WebDriver BiDi](docs/explanation/webdriver-bidi.md), not proprietary protocols controlled by large corporations.
- **Lightweight.** Single ~10MB binary. No runtime dependencies.

---

## Quick Reference

| Component | Purpose | Interface |
|-----------|---------|-----------|
| **Clicker** | Browser automation, BiDi proxy, MCP server | CLI / stdio / WebSocket :9515 |
| **JS Client** | Developer-facing API | npm package |
| **Python Client** | Developer-facing API | pip package |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         LLM / Agent                         │
│          (Claude Code, Codex, Gemini, Local Models)         │
└─────────────────────────────────────────────────────────────┘
                      ▲
                      │ MCP Protocol (stdio)
                      ▼
           ┌─────────────────────┐         
           │   Vibium Clicker    │
           │                     │
           │  ┌───────────────┐  │
           │  │  MCP Server   │  │
           │  └───────▲───────┘  │         ┌──────────────────┐
           │          │          │         │                  │
           │  ┌───────▼───────┐  │WebSocket│                  │
           │  │  BiDi Proxy   │  │◄───────►│  Chrome Browser  │
           │  └───────────────┘  │  BiDi   │                  │
           │                     │         │                  │
           └─────────────────────┘         └──────────────────┘
                      ▲
                      │ WebSocket BiDi :9515
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                     Client Libraries                        │
│         npm install vibium  ·  pip install vibium           │
│                                                             │
│    ┌─────────────────┐               ┌─────────────────┐    │
│    │ Async API       │               │    Sync API     │    │
│    │ await vibe.go() │               │    vibe.go()    │    │
│    │                 │               │                 │    │
│    └─────────────────┘               └─────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

---

## Components

### Clicker

A single Go binary (~10MB) that does everything:

- **Browser Management:** Detects/launches Chrome with BiDi enabled
- **BiDi Proxy:** WebSocket server that routes commands to browser
- **MCP Server:** stdio interface for LLM agents
- **Auto-Wait:** Polls for elements before interacting
- **Screenshots:** Viewport capture as PNG

**Design goal:** The binary is invisible. JS developers just `npm install vibium` and it works.

### JS/TS Client

```javascript
// Option 1: require (REPL-friendly)
const { browserSync } = require('vibium')

// Option 2: dynamic import (REPL with --experimental-repl-await)
const { browser } = await import('vibium')

// Option 3: static import (in .mjs or .ts files)
import { browser, browserSync } from 'vibium'
```

**Sync API:**
```javascript
const fs = require('fs')
const { browserSync } = require('vibium')

const vibe = browserSync.launch()
vibe.go('https://example.com')

const png = vibe.screenshot()
fs.writeFileSync('screenshot.png', png)

const link = vibe.find('a')
link.click()
vibe.quit()
```

**Async API:**
```javascript
const fs = await import('fs/promises')
const { browser } = await import('vibium')

const vibe = await browser.launch()
await vibe.go('https://example.com')

const png = await vibe.screenshot()
await fs.writeFile('screenshot.png', png)

const link = await vibe.find('a')
await link.click()
await vibe.quit()
```

### Python Client

```python
from vibium import browser, browser_sync
```

**Sync API:**
```python
from vibium import browser_sync as browser

vibe = browser.launch()
vibe.go("https://example.com")

png = vibe.screenshot()
with open("screenshot.png", "wb") as f:
    f.write(png)

link = vibe.find("a")
link.click()
vibe.quit()
```

**Async API:**
```python
import asyncio
from vibium import browser

async def main():
    vibe = await browser.launch()
    await vibe.go("https://example.com")

    png = await vibe.screenshot()
    with open("screenshot.png", "wb") as f:
        f.write(png)

    link = await vibe.find("a")
    await link.click()
    await vibe.quit()

asyncio.run(main())
```

---

## For Agents

One command to add browser control to your AI coding assistant:

**Claude Code:**
```bash
claude mcp add vibium -- npx -y vibium
```

**Gemini CLI:**
```bash
gemini mcp add vibium npx -y vibium
```

That's it. Chrome downloads automatically on first use.

See detailed setup guides: [Claude Code](docs/tutorials/claude-code-mcp-setup.md) | [Gemini CLI](docs/tutorials/gemini-cli-mcp-setup.md)

| Tool | Description |
|------|-------------|
| `browser_launch` | Start browser (visible by default) |
| `browser_navigate` | Go to URL |
| `browser_find` | Find element by CSS selector |
| `browser_click` | Click an element |
| `browser_type` | Type text into an element |
| `browser_screenshot` | Capture viewport (base64 or save to file with `--screenshot-dir`) |
| `browser_quit` | Close browser |

---

## For Humans

```bash
npm install vibium   # JavaScript/TypeScript
pip install vibium   # Python
```

This automatically:
1. Installs the Clicker binary for your platform
2. Downloads Chrome for Testing + chromedriver to platform cache:
   - Linux: `~/.cache/vibium/`
   - macOS: `~/Library/Caches/vibium/`
   - Windows: `%LOCALAPPDATA%\vibium\`

No manual browser setup required.

**Skip browser download** (if you manage browsers separately):
```bash
VIBIUM_SKIP_BROWSER_DOWNLOAD=1 npm install vibium
```

---

## Platform Support

| Platform | Architecture | Status |
|----------|--------------|--------|
| Linux | x64 | ✅ Supported |
| macOS | x64 (Intel) | ✅ Supported |
| macOS | arm64 (Apple Silicon) | ✅ Supported |
| Windows | x64 | ✅ Supported |

---

## Quick Start

**As a library:**
```typescript
import { browser } from "vibium";

const vibe = await browser.launch();
await vibe.go("https://example.com");
const el = await vibe.find("a");
await el.click();
await vibe.quit();
```

**With Claude Code:**

Once installed via `claude mcp add`, just ask Claude to browse:

> "Go to example.com and click the first link"

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

---

## Roadmap

V1 focuses on the core loop: browser control via MCP and JS client.

See [V2-ROADMAP.md](V2-ROADMAP.md) for planned features:
- Java client (WIP: `clients/java`)
- Cortex (memory/navigation layer)
- Retina (recording extension)
- Video recording
- AI-powered locators

---

## Updates

- [2025-12-31: Python Client](docs/updates/2025-12-31-python-client.md)
- [2025-12-22: Day 12 - Published to npm](docs/updates/2025-12-22-day12-npm-publish.md)
- [2025-12-21: Day 11 - Polish & Error Handling](docs/updates/2025-12-21-day11-polish.md)
- [2025-12-20: Day 10 - MCP Server](docs/updates/2025-12-20-day10-mcp.md)
- [2025-12-19: Day 9 - Actionability](docs/updates/2025-12-19-day9-actionability.md)
- [2025-12-19: Day 8 - Elements & Sync API](docs/updates/2025-12-19-day8-elements-sync.md)
- [2025-12-17: Halfway There](docs/updates/2025-12-17-halfway-there.md)
- [2025-12-16: Week 1 Progress](docs/updates/2025-12-16-week1-progress.md)
- [2025-12-11: V1 Announcement](docs/updates/2025-12-11-v1-announcement.md)

---

## License

Apache 2.0
