# Vibium

**Browser automation without the drama.**

Vibium is browser automation infrastructure built for AI agents. A single Go binary handles browser lifecycle, WebDriver BiDi protocol, and exposes an MCP server — so Claude Code (or any MCP client) can drive a browser with zero setup. Works great for AI agents, test automation, and anything else that needs a browser.

---

## Quick Reference

| Component | Purpose | Interface |
|-----------|---------|-----------|
| **Clicker** | Browser automation, BiDi proxy, MCP server | CLI / stdio / WebSocket :9515 |
| **JS Client** | Developer-facing API | npm package |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     LLM / Agent                             │
│              (Claude Code, GPT, Local Models)               │
└─────────────────────────┬───────────────────────────────────┘
                          ▲
                          │ MCP Protocol (stdio)
                          ▼
               ┌─────────────────────┐         ┌─────────────────────┐
               │   Clicker Binary    │         │                     │
               │   (Go, ~10MB)       │         │   Chrome Browser    │
               │                     │         │                     │
               │  ┌───────────────┐  │         │                     │
               │  │  MCP Server   │  │         │                     │
               │  └───────┬───────┘  │         │                     │
               │          │          │  BiDi   │                     │
               │  ┌───────▼───────┐  │◄──────►│                     │
               │  │  BiDi Proxy   │  │WebSocket│                     │
               │  └───────────────┘  │         │                     │
               │                     │         │                     │
               └─────────────────────┘         └─────────────────────┘
                          ▲
                          │ WebSocket BiDi :9515
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                     JS/TS Client                            │
│                    npm install vibium                       │
│                                                             │
│   ┌─────────────┐              ┌─────────────┐              │
│   │ Async API   │              │  Sync API   │              │
│   │ await vibe  │              │ vibe.go()   │              │
│   │   .go()     │              │             │              │
│   └─────────────┘              └─────────────┘              │
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

Two flavors: async (Promise-based) and sync (blocking).

**Async API:**
```typescript
import { browser } from "vibium";

const vibe = await browser.launch();
await vibe.go("https://example.com");

const el = await vibe.find("button.submit");
await el.click();
await el.type("hello");

const png = await vibe.screenshot();
await vibe.quit();
```

**Sync API:**
```typescript
import { browserSync } from "vibium";

const vibe = browserSync.launch();
vibe.go("https://example.com");

const el = vibe.find("button.submit");
el.click();
el.type("hello");

const png = vibe.screenshot();
vibe.quit();
```

---

## For Agents

One command to add browser control to Claude Code:

```bash
claude mcp add vibium -- npx -y vibium
```

That's it. No `npm install` needed. The first run downloads everything automatically.

| Tool | Description |
|------|-------------|
| `browser_launch` | Start browser (headless by default) |
| `browser_navigate` | Go to URL |
| `browser_find` | Find element by CSS selector |
| `browser_click` | Click an element |
| `browser_type` | Type text into an element |
| `browser_screenshot` | Capture viewport as base64 PNG |
| `browser_quit` | Close browser |

---

## For Humans

```bash
npm install vibium
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
| Linux | arm64 | ✅ Supported |
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
- Python and Java clients
- Cortex (memory/navigation layer)
- Retina (recording extension)
- Video recording
- AI-powered locators

---

## Updates

- [2025-12-19: Day 8 - Elements & Sync API](docs/updates/2025-12-19-day8-elements-sync.md)
- [2025-12-17: Halfway There](docs/updates/2025-12-17-halfway-there.md)
- [2025-12-16: Week 1 Progress](docs/updates/2025-12-16-week1-progress.md)
- [2025-12-11: V1 Announcement](docs/updates/2025-12-11-v1-announcement.md)

---

## License

Apache 2.0
