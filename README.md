# Vibium

[![npm](https://img.shields.io/npm/v/vibium)](https://www.npmjs.com/package/vibium) [![PyPI](https://img.shields.io/pypi/v/vibium)](https://pypi.org/project/vibium/) [![Maven Central](https://img.shields.io/maven-central/v/com.vibium/vibium)](https://central.sonatype.com/artifact/com.vibium/vibium) [![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)

**The verification layer for coding agents.**

Vibium gives AI agents the tools they need to check their work. Install the `browser` skill and your agent can navigate pages, fill forms, click buttons, and take screenshots — all through simple CLI commands. Also available as an MCP server and as JS/TS, Python, and Java client libraries.

**New here?** Get started in [JavaScript](docs/tutorials/getting-started-js.md), [Python](docs/tutorials/getting-started-python.md), or [Java](docs/tutorials/getting-started-java.md) — zero to hello world in 5 minutes.

## Why Vibium?

- **AI-native.** Install as a skill — your agent learns the full browser automation toolkit instantly.
- **Zero config.** One install, browser downloads automatically, visible by default.
- **Standards-based.** Built on [WebDriver BiDi](docs/explanation/webdriver-bidi.md), not proprietary protocols controlled by large corporations.
- **Lightweight.** Single ~10MB binary. No runtime dependencies.
- **Flexible.** Use as a CLI skill, MCP server, or JS/Python/Java library.

---

## Agent Setup

```bash
npm install -g vibium
npx skills add https://github.com/VibiumDev/vibium
```

Select all Vibium skills in the installer, then choose your agent and installation
scope. Vibium installs Chrome automatically.

- **browser** — explore pages, automate actions, record sessions, and carry out browser tasks from plain-language instructions.
- **check** — independently assess whether a claim holds in the browser or a recording.

Invoke `/browser` or `/check` in agents with slash skills, or `$browser` and
`$check` in Codex. Setup guides: [Codex](docs/how-to-guides/using-vibium-with-codex.md),
[Claude Code](docs/how-to-guides/using-vibium-with-claude-code.md), and
[Claude Desktop Chat via MCP](docs/how-to-guides/using-vibium-with-claude-desktop.md).
Run [`vibium ready`](docs/reference/ready.md) to check setup.

For a walkthrough, follow [Your coding agent’s first Check](docs/tutorials/check-with-a-coding-agent.md).
Run and Check currently require the development build and AI configuration:
run `vibium config init`, fill in the file it writes at `~/.config/vibium/ai.env`,
then `source` it. See [model providers](docs/reference/model-providers.md).

See [Introducing Run and Check](docs/updates/2026-09-07-run-and-check.md) for examples and limitations.

## CLI Quick Reference

```bash
# Give a browser task (requires AI setup)
vibium "open https://var.parts and add one battery pack to the cart" --keep-open

# Or use the explicit run command
vibium run "open https://var.parts and add one battery pack to the cart" --keep-open
# Check the cart
vibium check "the cart contains exactly one battery pack"

# Check and save a recording
vibium check "the cart contains exactly one battery pack" -o cart-check.zip

# Close the browser
vibium stop

# Map & interact (the core workflow)
vibium go https://var.parts           # navigate to URL
vibium map                            # map interactive elements → @e1, @e2, ...
vibium click @e1                      # click using ref
vibium diff map                       # see what changed

# Find elements (semantic — no CSS needed)
vibium find text "Sign In"            # find by visible text
vibium find label "Email"             # find by form label
vibium find placeholder "Search"      # find by placeholder
vibium find role button               # find by ARIA role

# Read & capture
vibium text                           # get all page text
vibium screenshot -o page.png         # capture screenshot
vibium screenshot --annotate -o a.png # annotated with element labels
vibium pdf -o page.pdf                # save page as PDF
vibium eval "document.title"          # run JavaScript

# Wait for things
vibium wait ".modal"                  # wait for element to appear
vibium wait url "/dashboard"          # wait for URL change
vibium wait text "Success"            # wait for text on page

# Record sessions
vibium record start                   # record with screenshots
vibium record stop                    # stop and save to record.zip

# Forms & input
vibium fill @e2 "hello@example.com"   # fill input using ref
vibium select @e3 "US"               # pick dropdown option
vibium set @e4                        # check a checkbox
vibium press Enter                    # press a key
```

Full command list: [SKILL.md](skills/browser/SKILL.md)

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

**Java** (Gradle):
```groovy
implementation 'com.vibium:vibium:26.8.21'
```

**Java** (Maven):
```xml
<dependency>
    <groupId>com.vibium</groupId>
    <artifactId>vibium</artifactId>
    <version>26.8.21</version>
</dependency>
```

This installs the Vibium binary and downloads Chrome automatically. No manual browser setup required.

### JS/TS Client

**Async API:**
```javascript
import { browser } from 'vibium'
import fs from 'node:fs/promises'

const bro = await browser.start()
const vibe = await bro.page()
await vibe.go('https://example.com')

const png = await vibe.screenshot()
await fs.writeFile('screenshot.png', png)

const link = await vibe.find('a')
await link.click()
await bro.stop()
```

**Sync API:**
```javascript
const { browser } = require('vibium/sync')
const fs = require('fs')

const bro = browser.start()
const vibe = bro.page()
vibe.go('https://example.com')

const png = vibe.screenshot()
fs.writeFileSync('screenshot.png', png)

const link = vibe.find('a')
link.click()
bro.stop()
```

### Python Client

```python
# Async
from vibium.async_api import browser

# Sync (default)
from vibium import browser
```

**Async API:**
```python
import asyncio
from vibium.async_api import browser

async def main():
    bro = await browser.start()
    vibe = await bro.page()
    await vibe.go("https://example.com")

    png = await vibe.screenshot()
    with open("screenshot.png", "wb") as f:
        f.write(png)

    link = await vibe.find("a")
    await link.click()
    await bro.stop()

asyncio.run(main())
```

**Sync API:**
```python
from vibium import browser

bro = browser.start()
vibe = bro.page()
vibe.go("https://example.com")

png = vibe.screenshot()
with open("screenshot.png", "wb") as f:
    f.write(png)

link = vibe.find("a")
link.click()
bro.stop()
```

### Java Client

```java
var bro = Vibium.start();
var vibe = bro.page();
vibe.go("https://example.com");

var png = vibe.screenshot();
Files.write(Path.of("screenshot.png"), png);

var link = vibe.find("a");
link.click();
bro.stop();
```

---

## Architecture

```
┌──────────────────────────────────────┐
│             LLM / Agent              │
│  (Claude Code, Codex, Gemini, etc.)  │
└──────────────────────────────────────┘
       ▲                  ▲
       │ CLI (Bash)       │ MCP (stdio)
       ▼                  ▼
┌───────────────────────────────────┐
│          Vibium binary            │
│                                   │
│  ┌──────────────┐ ┌────────────┐  │
│  │ CLI Commands │ │ MCP Server │  │
│  └─────┬────────┘ └──────┬─────┘  │        ┌──────────────────┐
│        └───────▲─────────┘        │        │                  │
│                │                  │        │                  │
│         ┌──────▼───────┐          │  BiDi  │    Web Browser   │
│         │  BiDi Proxy  │          │◄──────►│                  │
│         └──────────────┘          │        │                  │
└───────────────────────────────────┘        └──────────────────┘
          ▲
          │ WebSocket BiDi :9515
          ▼
┌──────────────────────────────────────┐
│          Client Libraries            │
│       (js/ts | python | java)        │
│                                      │
│  ┌─────────────────┐ ┌────────────┐  │
│  │   Async API     │ │  Sync API  │  │
│  │ await vibe.go() │ │  vibe.go() │  │
│  └─────────────────┘ └────────────┘  │
└──────────────────────────────────────┘
```

---

## Platform Support

| Platform | Architecture | Status |
|----------|--------------|--------|
| Linux | x64 | ✅ Supported |
| macOS | x64 (Intel) | ✅ Supported |
| macOS | arm64 (Apple Silicon) | ✅ Supported |
| Windows | x64 | ✅ Supported |

Chrome is the default browser; Firefox is also supported ([how to switch](docs/how-to-guides/using-firefox.md)). Firefox 154+ can [record video of a session](docs/how-to-guides/record-video.md).

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.
Source builds require Go, Node.js/npm, and Java 21 because the default `make`
target builds the Go binary, JS client, and Java client.

---

## Roadmap

See [ROADMAP.md](ROADMAP.md) for current development work and longer-term ideas.

---

## License

Apache 2.0
