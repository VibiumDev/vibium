# Contributing to Vibium

## Development Environment

We recommend developing inside a VM to limit the blast radius of AI-assisted tools like Claude Code. See the setup guide for your platform:
- [macOS](docs/local-dev-setup-mac.md)
- [Linux x86](docs/local-dev-setup-x86-linux.md)
- [Windows x86](docs/local-dev-setup-x86-windows.md)

If you prefer to develop directly on your host machine, follow the steps below.

---

## Prerequisites

- Go 1.21+
- Node.js 18+
- Python 3.9+ (for Python client development)
- GitHub CLI (optional, for managing issues/PRs from terminal):
  - macOS: `brew install gh`
  - Linux: `sudo apt install gh` or `sudo dnf install gh`
  - Windows: `winget install GitHub.cli`

---

## Clone and Build

```bash
git clone https://github.com/VibiumDev/vibium.git
cd vibium
make
make install-browser
```

This installs npm dependencies, builds clicker and the JS client, and downloads Chrome for Testing.

---

## Available Make Targets

### Build

```bash
make                    # Build everything (default)
make build-go           # Build clicker binary
make build-js           # Build JS client
make build-all-platforms # Cross-compile clicker for all platforms
```

### Package (npm)

```bash
make package            # Build all npm packages
make package-platforms  # Build platform packages only
make package-main       # Build main package only
```

### Package (Python)

```bash
make package-python           # Build all Python wheels
make package-python-platforms # Copy binaries to Python packages
```

### Test

```bash
make test               # Run all tests (CLI + JS + MCP)
make test-cli           # Run CLI tests only
make test-js            # Run JS library tests only
make test-mcp           # Run MCP server tests only
make test-python        # Run Python client tests
```

### Other

```bash
make install-browser    # Install Chrome for Testing
make deps               # Install npm dependencies
make serve              # Start proxy server on :9515
make double-tap         # Kill zombie Chrome/chromedriver processes
```

### Clean

```bash
make clean              # Clean binaries and JS dist
make clean-packages     # Clean built npm packages
make clean-python-packages # Clean built Python packages
make clean-cache        # Clean cached Chrome for Testing
make clean-all          # Clean everything
```

---

## Using the JS Client

After building, you can test the JS client in a Node REPL:

```bash
cd clients/javascript && node --experimental-repl-await
```

```javascript
// Option 1: require (REPL-friendly)
const { browserSync } = require('./dist')

// Option 2: dynamic import (REPL with --experimental-repl-await)
const { browser } = await import('./dist/index.mjs')

// Option 3: static import (in .mjs files)
import { browser } from './dist/index.mjs'
```

Sync example:

```javascript
const { browserSync } = require('./dist')
const vibe = browserSync.launch()
vibe.go('https://example.com')

const el = vibe.find('h1')
console.log(el.text())

const shot = vibe.screenshot()
require('fs').writeFileSync('test.png', shot)
vibe.quit()
```

Async example:

```javascript
const { browser } = await import('./dist/index.mjs')
const vibe = await browser.launch()
await vibe.go('https://example.com')

const el = await vibe.find('h1')
console.log(await el.text())

const shot = await vibe.screenshot()
require('fs').writeFileSync('test.png', shot)
await vibe.quit()
```

---

## Using the Python Client

The Python client provides both sync and async APIs.

### Setup

For local development, use a virtual environment:

```bash
cd clients/python
python -m venv .venv
source .venv/bin/activate  # On Windows: .venv\Scripts\activate
pip install -e .           # Editable install - code changes take effect immediately
```

Or install from PyPI:

```bash
pip install vibium
```

### Sync Example

```python
from vibium import browser_sync

vibe = browser_sync.launch()
vibe.go("https://example.com")

el = vibe.find("h1")
print(el.text())

with open("screenshot.png", "wb") as f:
    f.write(vibe.screenshot())

vibe.quit()
```

### Async Example

```python
import asyncio
from vibium import browser

async def main():
    vibe = await browser.launch()
    await vibe.go("https://example.com")

    el = await vibe.find("h1")
    print(await el.text())

    with open("screenshot.png", "wb") as f:
        f.write(await vibe.screenshot())

    await vibe.quit()

asyncio.run(main())
```

---

## Using Clicker

Clicker is the Go binary at the heart of Vibium. It handles browser lifecycle, WebDriver BiDi protocol, and exposes an MCP server for AI agents.

Long-term, clicker runs silently in the background — called by client libraries (JS/TS, Python, etc.). Most users won't interact with it directly.

For now, the CLI is a development and testing aid. It lets you verify browser automation works before the client libraries are built on top.

After building, the binary is at `./clicker/bin/clicker`.

### Setup

```bash
./clicker/bin/clicker install   # Download Chrome for Testing + chromedriver
./clicker/bin/clicker paths     # Show browser and cache paths
./clicker/bin/clicker version   # Show version
```

### Browser Commands

```bash
# Navigate to a URL
./clicker/bin/clicker navigate https://example.com

# Take a screenshot
./clicker/bin/clicker screenshot https://example.com -o shot.png

# Evaluate JavaScript
./clicker/bin/clicker eval https://example.com "document.title"

# Find an element
./clicker/bin/clicker find https://example.com "a"

# Click an element
./clicker/bin/clicker click https://example.com "a"

# Type into an element
./clicker/bin/clicker type https://the-internet.herokuapp.com/inputs "input" "12345"
```

### Useful Flags

```bash
--headless        # Hide the browser window (visible by default)
--wait-open 5     # Wait 5 seconds after navigation for page to load
--wait-close 3    # Keep browser open 3 seconds before closing
```

Example:

```bash
./clicker/bin/clicker screenshot https://example.com --wait-close 5 -o shot.png
```

---

## Using the MCP Server

Clicker includes an MCP (Model Context Protocol) server for AI agent integration.

### Available Tools

| Tool | Description |
|------|-------------|
| `browser_launch` | Start a browser session |
| `browser_navigate` | Go to a URL |
| `browser_click` | Click an element by CSS selector |
| `browser_type` | Type into an element |
| `browser_screenshot` | Capture the page |
| `browser_find` | Find element info |
| `browser_quit` | Close the browser |

### Running the MCP Server

```bash
# Run directly (for testing)
./clicker/bin/clicker mcp

# With custom screenshot directory
./clicker/bin/clicker mcp --screenshot-dir ./screenshots

# Disable screenshot file saving (inline base64 only)
./clicker/bin/clicker mcp --screenshot-dir ""
```

### Configuring with Claude Code

```bash
claude mcp add vibium -- clicker mcp
```

### Testing with JSON-RPC

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{}}}' | ./clicker/bin/clicker mcp
```

---

## Debugging

For low-level debugging tools and troubleshooting tips, see [docs/how-to-guides/debugging.md](docs/how-to-guides/debugging.md).

---

## Submitting Changes

- **Team members**: push directly to `VibiumDev/vibium`
- **External contributors**: fork the repo, push to your fork, then open a PR to `VibiumDev/vibium`

See [docs/local-dev-setup-mac.md](docs/local-dev-setup-mac.md) for details on the fork-based workflow.
