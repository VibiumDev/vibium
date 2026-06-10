# Contributing to Vibium

## Development Environment

We recommend developing inside a VM to limit the blast radius of AI-assisted tools like Claude Code. Check the [system requirements](docs/reference/mac-system-requirements.md) for hardware specs, then see the setup guide for your platform:
- [macOS](docs/how-to-guides/local-dev-setup-mac.md)
- [Linux x86](docs/how-to-guides/local-dev-setup-x86-linux.md)
- [Windows x86](docs/how-to-guides/local-dev-setup-x86-windows.md)

If you prefer to develop directly on your host machine, follow the steps below.

---

## Prerequisites

- Go 1.21+
- Node.js 18+
- Python 3.9+ (for Python client development)
- Java 21+ (required for the default `make` target and Java client development)
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
make test
```

`make` installs npm dependencies and builds the vibium binary, JS client, and Java client. On a fresh checkout, it needs network access for `npm install` and for the Gradle wrapper to download Gradle.

`make test` downloads Chrome for Testing (if needed) and runs the full test suite. If you are only working on the Go binary or JS client, use `make build-go` or `make build-js` to avoid the Java build.

---

## Available Make Targets

### Build

```bash
make                       # Build everything (default)
make build-go              # Build vibium binary
make build-js              # Build JS client
make build-java            # Build Java client JAR
make build-go-all          # Cross-compile vibium for all platforms
```

### Package

```bash
make package               # Build all packages (npm + Python)
make package-js            # Build npm packages only
make package-python        # Build Python wheels only
make package-java          # Build Java JAR with native binaries
```

### Test

```bash
make test                  # Run all tests (auto-installs Chrome for Testing)
make test-cli              # Run CLI tests only
make test-js               # Run JS library tests only
make test-mcp              # Run MCP server tests only
make test-python           # Run Python client tests
make test-java             # Run Java client tests
make test-daemon           # Run daemon lifecycle tests
```

### Other

```bash
make install-browser       # Install Chrome for Testing
make deps                  # Install npm dependencies
make serve                 # Start proxy server on :9515
make double-tap            # Kill zombie Chrome/chromedriver processes
make get-version           # Show current version
make set-version VERSION=x.x.x  # Set version across all packages
```

### Clean

```bash
make clean                 # Clean binaries and JS dist
make clean-go              # Clean vibium binaries
make clean-js              # Clean JS client dist
make clean-java            # Clean Java build artifacts
make clean-npm-packages    # Clean built npm packages
make clean-python-packages # Clean Python packages
make clean-packages        # Clean all packages (npm + Python)
make clean-cache           # Clean cached Chrome for Testing
make clean-all             # Clean everything
```

---

## Using the JS Client

After building, you can test the JS client in a Node REPL:

```bash
cd clients/javascript && node
```

```javascript
// Option 1: require sync API (REPL-friendly)
const { browser } = require('./dist/sync')

// Option 2: dynamic import async API
const { browser } = await import('./dist/index.mjs')

// Option 3: static import async API (in .mjs files)
import { browser } from './dist/index.mjs'
```

Sync example:

```javascript
const { browser } = require('./dist/sync')
const bro = browser.start()
const vibe = bro.page()
vibe.go('https://example.com')

const el = vibe.find('h1')
console.log(el.text())

// Execute JavaScript
const title = vibe.evaluate('document.title')
console.log('Page title:', title)

const shot = vibe.screenshot()
require('fs').writeFileSync('test.png', shot)
bro.stop()
```

Async example:

```javascript
const { browser } = await import('./dist/index.mjs')
const bro = await browser.start()
const vibe = await bro.page()
await vibe.go('https://example.com')

const el = await vibe.find('h1')
console.log(await el.text())

// Execute JavaScript
const title = await vibe.evaluate('document.title')
console.log('Page title:', title)

const shot = await vibe.screenshot()
require('fs').writeFileSync('test.png', shot)
await bro.stop()
```

---

## Using the Python Client

The Python client provides both sync and async APIs.

### Setup

For local development, build the Go binary first (if you haven't already), then set up a virtual environment:

```bash
# From the repo root
make build-go

# Set up the Python client
cd clients/python
python -m venv .venv
source .venv/bin/activate  # On Windows: .venv\Scripts\activate
pip install -e .           # Editable install - code changes take effect immediately

# Point the client to the locally-built binary
export VIBIUM_BIN_PATH=../../clicker/bin/vibium
```

Or install from PyPI (binary is bundled automatically):

```bash
pip install vibium
```

### Sync Example

```python
from vibium import browser

bro = browser.start()
vibe = bro.page()
vibe.go("https://example.com")

el = vibe.find("h1")
print(el.text())

# Execute JavaScript
title = vibe.evaluate("document.title")
print(f"Page title: {title}")

with open("screenshot.png", "wb") as f:
    f.write(vibe.screenshot())

bro.stop()
```

### Async Example

```python
import asyncio
from vibium.async_api import browser

async def main():
    bro = await browser.start()
    vibe = await bro.page()
    await vibe.go("https://example.com")

    el = await vibe.find("h1")
    print(await el.text())

    # Execute JavaScript
    title = await vibe.evaluate("document.title")
    print(f"Page title: {title}")

    with open("screenshot.png", "wb") as f:
        f.write(await vibe.screenshot())

    await bro.stop()

asyncio.run(main())
```

---

## Using the Java Client

The Java client provides a synchronous API for browser automation.

### Setup

```bash
# From the repo root — builds Go binary + Java JAR
make build-java

# Point the client to the locally-built binary
export VIBIUM_BIN_PATH=./clicker/bin/vibium
```

Or install from Maven Central (binary is bundled in the JAR):

```xml
<dependency>
    <groupId>com.vibium</groupId>
    <artifactId>vibium</artifactId>
    <version>26.3.18</version>
</dependency>
```

### Interactive REPL (JShell)

After building, you can test the Java client interactively with JShell:

```bash
make jshell
```

```java
import com.vibium.*;
import com.vibium.types.*;

var bro = Vibium.start();
var vibe = bro.page();
vibe.go("https://example.com")

var el = vibe.find("h1")
el.text()

// Execute JavaScript
vibe.evaluate("document.title")

// Screenshot
var shot = vibe.screenshot()
java.nio.file.Files.write(java.nio.file.Path.of("screenshot.png"), shot)

bro.stop()
```

### File Example

Save this as `Example.java`:

```java
import com.vibium.Vibium;
import com.vibium.Browser;
import com.vibium.Page;
import com.vibium.Element;

public class Example {
    public static void main(String[] args) throws Exception {
        Browser bro = Vibium.start();
        Page vibe = bro.page();
        vibe.go("https://example.com");

        Element el = vibe.find("h1");
        System.out.println(el.text());

        byte[] shot = vibe.screenshot();
        java.nio.file.Files.write(java.nio.file.Path.of("screenshot.png"), shot);

        bro.stop();
    }
}
```

Compile and run (from the repo root):

```bash
javac -cp "clients/java/build/libs/*:clients/java/build/dependencies/*" Example.java
java -cp ".:clients/java/build/libs/*:clients/java/build/dependencies/*" Example
```

---

## Using the Vibium Binary

The vibium binary is the Go binary at the heart of Vibium. It handles browser lifecycle, WebDriver BiDi protocol, and exposes an MCP server for AI agents.

Long-term, vibium runs silently in the background — called by client libraries (JS/TS, Python, etc.). Most users won't interact with it directly.

For now, the CLI is a development and testing aid. It lets you verify browser automation works before the client libraries are built on top.

After building, the binary is at `./clicker/bin/vibium`.

### Setup

```bash
cd clicker/bin
./vibium install   # Download Chrome for Testing + chromedriver
./vibium paths     # Show browser and cache paths
./vibium version   # Show version
```

### Browser Commands

By default, vibium runs in **daemon mode** — the browser stays open between commands:

```bash
cd clicker/bin

# Navigate to a URL
./vibium go https://example.com

# Interact with the current page (no URL needed)
./vibium find "h1"
./vibium click "a"
./vibium type "input" "hello"
./vibium eval "document.title"
./vibium screenshot -o shot.png

# You can also provide a URL to navigate first
./vibium find https://example.com "a"
./vibium screenshot https://example.com -o shot.png
```

### Useful Flags

```bash
--headless        # Hide the browser window (visible by default)
--json             # Output results as JSON
-v, --verbose     # Enable debug logging
```

### Daemon Management

```bash
cd clicker/bin
./vibium daemon start    # Start daemon in foreground
./vibium daemon start    # Start daemon in background
./vibium daemon status   # Show daemon status
./vibium daemon stop     # Stop the daemon
```

The daemon auto-starts on the first command, so you rarely need to manage it manually.

---

## Using the MCP Server

The vibium binary includes an MCP (Model Context Protocol) server for AI agent integration. For end-user setup instructions and the full list of tools, see [Getting Started with MCP](docs/tutorials/getting-started-mcp.md).

### Running the MCP Server

```bash
cd clicker/bin

# Run directly (for testing)
./vibium mcp

# With custom screenshot directory
./vibium mcp --screenshot-dir ./screenshots

# Disable screenshot file saving (inline base64 only)
./vibium mcp --screenshot-dir ""
```

### Configuring with Claude Code

```bash
claude mcp add vibium -- vibium mcp
```

### Testing with JSON-RPC

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{}}}' | clicker/bin/vibium mcp
```

---

## Debugging

For low-level debugging tools and troubleshooting tips, see [docs/how-to-guides/debugging.md](docs/how-to-guides/debugging.md).

---

## Submitting Changes

- **Team members**: push directly to `VibiumDev/vibium`
- **External contributors**: fork the repo, push to your fork, then open a PR to `VibiumDev/vibium`

See [docs/how-to-guides/local-dev-setup-mac.md](docs/how-to-guides/local-dev-setup-mac.md) for details on the fork-based workflow.
