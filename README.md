# Vibium

**Browser automation for AI agents and humans.**

Vibium gives AI agents a browser. Install the `vibium` skill and your agent can navigate pages, fill forms, click buttons, and take screenshots — all through simple CLI commands. Also available as an MCP server and as JS/TS and Python client libraries.

**New here?** [Getting Started Tutorial](docs/tutorials/getting-started-js.md) — zero to hello world in 5 minutes.

## Why Vibium?

- **AI-native.** Install as a skill — your agent learns 81 browser automation tools instantly.
- **Zero config.** One install, browser downloads automatically, visible by default.
- **Standards-based.** Built on [WebDriver BiDi](docs/explanation/webdriver-bidi.md), not proprietary protocols controlled by large corporations.
- **Lightweight.** Single ~10MB binary. No runtime dependencies.
- **Flexible.** Use as a CLI skill, MCP server, or JS/Python library.

---

## Architecture

```
┌──────────────────────────────────────┐
│             LLM / Agent              │
│  (Claude Code, Codex, Gemini, etc.)  │
└──────────────────────────────────────┘
          ▲                  ▲
          │ MCP (stdio)      │ CLI (Bash)
          ▼                  ▼
┌──────────────────────────────────┐
│         Vibium binary            │
│        (vibium CLI)              │
│                                  │
│  ┌───────────┐ ┌──────────────┐  │
│  │ MCP Server│ │ CLI Commands │  │
│  └─────┬─────┘ └──────┬───────┘  │        ┌──────────────────┐
│        └──────▲───────┘          │        │                  │
│               │                  │        │                  │
│        ┌──────▼───────┐          │  BiDi  │  Chrome Browser  │
│        │  BiDi Proxy  │          │◄──────►│                  │
│        └──────────────┘          │        │                  │
└──────────────────────────────────┘        └──────────────────┘
          ▲
          │ WebSocket BiDi :9515
          ▼
┌──────────────────────────────────────┐
│          Client Libraries            │
│          (js/ts | python)            │
│                                      │
│  ┌─────────────────┐ ┌────────────┐  │
│  │   Async API     │ │  Sync API  │  │
│  │ await vibe.go() │ │  vibe.go() │  │
│  └─────────────────┘ └────────────┘  │
└──────────────────────────────────────┘
```

See [internals](docs/explanation/internals.md) for component details.

---

## Agent Setup

```bash
npm install -g vibium
npx skills add https://github.com/VibiumDev/vibium --skill vibe-check
```

The first command installs Vibium and the `vibium` binary, and downloads Chrome. The second installs the skill to `{project}/.agents/skills/vibium`.

### CLI Quick Reference

```bash
vibium go https://example.com          # go to a page
vibium text                            # get page text
vibium click "a"                       # click an element
vibium type "input" "hello"            # type into a field
vibium screenshot -o page.png          # capture screenshot
vibium eval "document.title"           # run JavaScript
```

Full command list: [SKILL.md](skills/vibe-check/SKILL.md)

**Alternative: MCP server** (for structured tool use instead of CLI):

```bash
claude mcp add vibium -- npx -y vibium mcp    # Claude Code
gemini mcp add vibium npx -y vibium mcp       # Gemini CLI
```

See [MCP setup guide](docs/tutorials/getting-started-mcp.md) for options and troubleshooting.

---

## Language APIs

```bash
npm install vibium   # JavaScript/TypeScript
pip install vibium   # Python
```

This automatically:
1. Installs the Vibium binary for your platform
2. Downloads Chrome for Testing + chromedriver to platform cache:
   - Linux: `~/.cache/vibium/`
   - macOS: `~/Library/Caches/vibium/`
   - Windows: `%LOCALAPPDATA%\vibium\`

No manual browser setup required.

**Skip browser download** (if you manage browsers separately):
```bash
VIBIUM_SKIP_BROWSER_DOWNLOAD=1 npm install vibium
```

### JS/TS Client

```javascript
// Sync (require-friendly)
const { browser } = require('vibium/sync')

// Async (import)
import { browser } from 'vibium'
```

**Sync API:**
```javascript
const fs = require('fs')
const { browser } = require('vibium/sync')

const bro = browser.launch()
const vibe = bro.page()
vibe.go('https://example.com')

const png = vibe.screenshot()
fs.writeFileSync('screenshot.png', png)

const link = vibe.find('a')
link.click()
bro.close()
```

**Async API:**
```javascript
import { browser } from 'vibium'

const bro = await browser.launch()
const vibe = await bro.page()
await vibe.go('https://example.com')

const png = await vibe.screenshot()
await fs.writeFile('screenshot.png', png)

const link = await vibe.find('a')
await link.click()
await bro.close()
```

### Python Client

```python
# Sync (default)
from vibium import browser

# Async
from vibium.async_api import browser
```

**Sync API:**
```python
from vibium import browser

bro = browser.launch()
vibe = bro.page()
vibe.go("https://example.com")

png = vibe.screenshot()
with open("screenshot.png", "wb") as f:
    f.write(png)

link = vibe.find("a")
link.click()
bro.close()
```

**Async API:**
```python
import asyncio
from vibium.async_api import browser

async def main():
    bro = await browser.launch()
    vibe = await bro.page()
    await vibe.go("https://example.com")

    png = await vibe.screenshot()
    with open("screenshot.png", "wb") as f:
        f.write(png)

    link = await vibe.find("a")
    await link.click()
    await bro.close()

asyncio.run(main())
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

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

---

## Roadmap

V1 focuses on the core loop: browser control via CLI, MCP, and client libraries.

See [ROADMAP.md](ROADMAP.md) for planned features:
- Java client
- Cortex (memory/navigation layer)
- Retina (recording extension)
- Video recording
- AI-powered locators

---

## License

Apache 2.0
