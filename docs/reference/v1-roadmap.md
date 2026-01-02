# Vibium V1 Roadmap

> **Archived:** V1 shipped December 2025. This document is preserved as historical reference.

**Target: 2 weeks to MVP**
**Scope: Clicker (Go) + JS/TS Client (async + sync) + MCP Server**
**Pitch: "Browser automation without the drama."**

---

## What's In V1

| Component | Description |
|-----------|-------------|
| Clicker | Go binary: browser launch, BiDi proxy, MCP server |
| JS Client | TypeScript: async API (`await vibe.go()`) + sync API (`vibe.go()`) |
| MCP Server | stdio interface for Claude Code / LLM agents |

## What's NOT In V1

See [V2-ROADMAP.md](V2-ROADMAP.md) for:
- Cortex (memory layer)
- Retina (recording extension)  
- Python / Java clients
- Video recording
- AI-powered locators (`vibe.do()`, `vibe.check()`)

---

## How To Use This Document

1. Give Claude Code one milestone at a time
2. Run the checkpoint test before moving on
3. If checkpoint fails, debug before proceeding
4. Human reviews at ⚠️ markers

---

## ✅ Day 1: Project Bootstrap

### Milestone 1.1: Monorepo Scaffold
```
Create the Vibium monorepo:

vibium/
├── package.json              # npm workspaces
├── .gitignore
├── clicker/
│   ├── go.mod
│   ├── go.sum
│   └── cmd/
│       └── clicker/
│           └── main.go
└── clients/javascript/
    ├── package.json
    └── tsconfig.json
```

**Checkpoint:** 
```bash
cd vibium && npm install
cd clicker && go build ./cmd/clicker
```

### Milestone 1.2: Go Hello World
```
Create minimal Clicker binary:
- --version flag → prints "Clicker v0.1.0"
- --help flag → shows usage
- Use cobra for CLI commands
```

**Checkpoint:**
```bash
cd clicker && go build -o bin/clicker ./cmd/clicker
./bin/clicker --version
./bin/clicker --help
```

### Milestone 1.3: JS Package Stub
```
Create JS client that:
- Exports { browser } with launch() function
- Exports { browserSync } with launch() function  
- Both throw "Not implemented" for now
- Builds with tsup to ESM + CJS
- Has TypeScript declarations
```

**Checkpoint:**
```bash
cd clients/javascript && npm run build
node -e "const { browser } = require('./dist'); console.log(typeof browser.launch)"
```

---

## ✅ Day 2: Browser Detection & Installation

### Milestone 2.1: Platform Paths
```
Implement internal/paths/paths.go:
- GetChromeExecutable(): Chrome path per platform
  - First check Vibium cache: <cache_dir>/chrome-for-testing/<version>/
  - Then system Chrome: /usr/bin/google-chrome, etc.
- GetCacheDir(): platform-specific cache directory
  - Linux: ~/.cache/vibium/
  - macOS: ~/Library/Caches/vibium/
  - Windows: %LOCALAPPDATA%\vibium\
- GetChromedriverPath(): path to cached chromedriver
```

**Checkpoint:**
```bash
./bin/clicker paths
# Prints Chrome path (or "not found") and cache directory
```

### Milestone 2.2: Chrome for Testing Installer
```
Implement internal/browser/installer.go:
- Fetch JSON from https://googlechromelabs.github.io/chrome-for-testing/known-good-versions-with-downloads.json
- Parse to find latest stable Chrome for Testing + chromedriver
- Download correct platform binary (linux64, mac-x64, mac-arm64, win64)
- Extract to <cache_dir>/chrome-for-testing/<version>/
- Make executable (chmod +x on unix)
- Skip if VIBIUM_SKIP_BROWSER_DOWNLOAD=1 is set

CLI command: clicker install
- Downloads Chrome for Testing if not cached
- Downloads matching chromedriver if not cached
- Respects VIBIUM_SKIP_BROWSER_DOWNLOAD=1 (exits early with message)
- Prints paths when done
```

**Checkpoint:**
```bash
./bin/clicker install
# Downloads Chrome for Testing + chromedriver
# Check platform cache (Linux example):
ls ~/.cache/vibium/chrome-for-testing/
# Should show version folder with chrome and chromedriver binaries
```

### Milestone 2.3: Chrome Launcher
```
Implement internal/browser/launcher.go:
- LaunchChrome(headless bool): Launch with BiDi flags
- Prefer cached Chrome for Testing over system Chrome
- Flags: --remote-debugging-port=0 --headless=new (if headless)
- Parse stderr for DevTools WebSocket URL
- Return WebSocket URL string
```

**Checkpoint:**
```bash
./bin/clicker launch-test
# Prints: ws://127.0.0.1:xxxxx/devtools/browser/...
# Uses Chrome for Testing from cache
```

### Milestone 2.4: Process Management  
```
Implement internal/process/process.go:
- Track spawned browser PID
- KillBrowser(): Terminate browser process
- Cleanup on Clicker exit (signal handling)
```

**Checkpoint:**
```bash
./bin/clicker launch-test
# Ctrl+C
ps aux | grep chrome  # Chrome should be gone
```

---

## ✅ Day 3: WebSocket & BiDi Basics

### Milestone 3.1: WebSocket Connection
```
Implement internal/bidi/connection.go:
- Connect(url string): Connect to WebSocket
- Send(msg string): Send text message
- Receive(): Receive text message
- Close(): Close connection

Use gorilla/websocket package.
```

**Checkpoint:**
```bash
./bin/clicker ws-test wss://echo.websocket.org
# Type message, should echo back
```

### Milestone 3.2: BiDi Protocol Types
```
Implement internal/bidi/protocol.go:
- BiDiCommand struct: {ID, Method, Params}
- BiDiResponse struct: {ID, Result, Error}
- BiDiEvent struct: {Method, Params}
- JSON marshaling/unmarshaling
- Command ID generator (atomic incrementing int)
```

### Milestone 3.3: Session Commands
```
Implement internal/bidi/session.go:
- SessionStatus()
- SessionNew()
```

**Checkpoint:**
```bash
./bin/clicker bidi-test
# Launches Chrome, connects, sends session.status, prints response
```

---

## ✅ Day 4: Navigation & Screenshots

### Milestone 4.1: Browsing Context
```
Implement internal/bidi/browsingcontext.go:
- GetTree(): Get current contexts
- Navigate(url string): Go to URL, wait for load
```

**Checkpoint:**
```bash
./bin/clicker navigate https://example.com
# Prints page title or URL
```

### Milestone 4.2: Screenshots
```
Add to browsingcontext.go:
- CaptureScreenshot(): Viewport capture
- Return base64 PNG
- CLI saves to file
```

**Checkpoint:**
```bash
./bin/clicker screenshot https://example.com -o shot.png
# shot.png is valid screenshot
```

### Milestone 4.3: Script Evaluation
```
Implement internal/bidi/script.go:
- Evaluate(expr string): Run JS, return result
- CallFunction(fn string, args): Call function with args
```

**Checkpoint:**
```bash
./bin/clicker eval https://example.com "document.title"
# Prints: Example Domain
```

---

## ✅ Day 5: Element Finding & Input

### Milestone 5.1: Element Location
```
Implement element finding via script:
- Find by CSS selector
- Return element reference (sharedId)
- Get bounding box coordinates
```

**Checkpoint:**
```bash
./bin/clicker find https://example.com "a"
# Prints: tag=A, text="More information...", box={x,y,w,h}
```

### Milestone 5.2: Mouse Input
```
Implement internal/bidi/input.go:
- PerformActions for pointer
- PointerMove to x,y
- PointerDown + PointerUp (click)
```

**Checkpoint:**
```bash
./bin/clicker click https://example.com "a"
# Shows: Current URL: https://www.iana.org/help/example-domains
```

### Milestone 5.3: Keyboard Input
```
Extend input.go:
- Keyboard actions: KeyDown, KeyUp
- TypeText(text string): Sequence of key events
```

**Checkpoint:**
```bash
./bin/clicker type https://the-internet.herokuapp.com/inputs "input" "12345"
# Shows: Typed "12345", value is now: 12345
```

---

## ⚠️ Human Review Checkpoint #1

Verify before proceeding: ✅ 2025-12-16 22:29 CST
- ✅ Chrome launches and exits cleanly
- ✅ No zombie processes after Ctrl+C
- ✅ Screenshots are correct
- ✅ Click navigates correctly
- ✅ Type inputs text correctly
- ✅ Test on 2+ different websites

---

## ✅ Day 6: BiDi Proxy Server

### Milestone 6.1: WebSocket Server
```
Implement internal/proxy/server.go:
- Listen on configurable port (default 9515)
- Accept WebSocket connections
- Log connect/disconnect events
```

**Checkpoint:**
```bash
./bin/clicker serve &
websocat ws://localhost:9515
# Connection accepted, can send/receive
```

### Milestone 6.2: Proxy Router
```
Implement internal/proxy/router.go:
On client connect:
1. Launch browser
2. Connect to browser BiDi WebSocket
3. Route: client → browser, browser → client
On client disconnect:
1. Kill browser
```

**Checkpoint:**
```bash
./bin/clicker serve &
websocat ws://localhost:9515
> {"id":1,"method":"session.status","params":{}}
# Returns session status from Chrome
```

### Milestone 6.3: Session Management
```
Handle:
- Multiple sequential commands per session
- Browser events (async push to client)
- Clean shutdown on disconnect

Use goroutines for concurrent message routing.
```

**Checkpoint:**
```bash
# Script that connects, navigates, screenshots, disconnects
# Verify screenshot returned, Chrome exits after disconnect

# Example flow:
Connected to proxy
> session.status
> browsingContext.getTree
> browsingContext.navigate → https://example.com
> browsingContext.captureScreenshot
< Screenshot received! Base64 length: 20736
Disconnected from proxy
[router] Browser session closed for client 1
```

---

## ✅ Day 7: JS Client - Async API

### Milestone 7.1: Binary Manager
```
Implement clients/javascript/src/clicker/:
- platform.ts: Detect OS (linux/darwin/win32) and arch (x64/arm64)
- binary.ts: Resolve clicker binary path
- process.ts: Spawn "clicker serve", extract port, manage lifecycle
```

**Checkpoint:**
```typescript
import { ClickerProcess } from './clicker/process';
const proc = await ClickerProcess.start();
console.log(proc.port);
await proc.stop();
```

### Milestone 7.2: BiDi Client  
```
Implement clients/javascript/src/bidi/:
- connection.ts: WebSocket client
- types.ts: TypeScript BiDi types
- client.ts: send(method, params) → Promise<result>
```

**Checkpoint:**
```typescript
import { BiDiClient } from './bidi/client';
const client = await BiDiClient.connect('ws://localhost:9515');
const status = await client.send('session.status', {});
console.log(status);
await client.close();
```

### Milestone 7.3: Async Browser API
```
Implement src/browser.ts:
- browser.launch(options?) → Promise<Vibe>
- Options: headless, port, executablePath

Implement src/vibe.ts (async):
- vibe.go(url) → Promise<void>
- vibe.screenshot() → Promise<Buffer>
- vibe.quit() → Promise<void>
```

**Checkpoint:**
```typescript
import { browser } from 'vibium';
const vibe = await browser.launch();
await vibe.go('https://example.com');
const shot = await vibe.screenshot();
require('fs').writeFileSync('test.png', shot);
await vibe.quit();
```

---

## ✅ Day 8: JS Client - Elements & Sync API

### Milestone 8.1: Element Class (Async)
```
Implement src/element.ts:
- element.click() → Promise<void>
- element.type(text) → Promise<void>
- element.text() → Promise<string>
- element.getAttribute(name) → Promise<string|null>
- element.boundingBox() → Promise<{x,y,width,height}>
```

### Milestone 8.2: Vibe.find (Async)
```
Add to src/vibe.ts:
- vibe.find(selector) → Promise<Element>
- CSS selector support
```

**Checkpoint:**
```typescript
const vibe = await browser.launch();
await vibe.go('https://example.com');
const link = await vibe.find('a');
console.log(await link.text()); // "More information..."
await link.click();
await vibe.quit();
```

### Milestone 8.3: Sync API Wrapper
```
Implement src/sync/:
- sync/browser.ts: browserSync.launch() → VibeSync
- sync/vibe.ts: VibeSync with blocking methods
- sync/element.ts: ElementSync with blocking methods

Use a synchronous execution strategy:
- Option A: Worker thread + Atomics.wait
- Option B: deasync package
- Option C: Child process + spawnSync

Recommend Option A for reliability.
```

**Checkpoint:**
```typescript
import { browserSync } from 'vibium';
const vibe = browserSync.launch();
vibe.go('https://example.com');
const link = vibe.find('a');
console.log(link.text()); // "More information..."
link.click();
vibe.quit();
```

---

## ✅ Day 9: Actionability (Playwright-style)

### Actionability Checks (like Playwright)

| Check | Definition | How to verify |
|-------|------------|---------------|
| **Visible** | Non-empty bounding box, no `visibility:hidden` | `getBoundingClientRect()`, `getComputedStyle()` |
| **Stable** | Same bounding box for 2 consecutive checks | Compare box at t and t+50ms |
| **ReceivesEvents** | Element is hit target at action point | `elementFromPoint()` at center |
| **Enabled** | Not `[disabled]` or `[aria-disabled=true]` | Check attributes |
| **Editable** | Enabled + not `[readonly]` or `[aria-readonly=true]` | Check attributes |

**Action requirements:**
- `click()` → Visible, Stable, ReceivesEvents, Enabled
- `type()` → Visible, Stable, ReceivesEvents, Enabled, Editable
- `find()` → Just existence (no actionability checks)

### Milestone 9.1: Actionability Checks (Go)
```
Implement internal/features/actionability.go:

Individual check functions (all via script.callFunction):
- CheckVisible(context, selector) → bool
- CheckStable(context, selector) → bool (compare box at 50ms interval)
- CheckReceivesEvents(context, selector) → bool
- CheckEnabled(context, selector) → bool
- CheckEditable(context, selector) → bool

Each returns true/false. Use JSON string return pattern like element.go.
```

**Checkpoint:**
```bash
# Add a test command that runs all checks on an element
./bin/clicker check-actionable https://example.com "a"
# Output:
# Checking actionability for selector: a
# ✓ Visible: true
# ✓ Stable: true
# ✓ ReceivesEvents: true
# ✓ Enabled: true
# ✗ Editable: false (expected for <a>)
```

### Milestone 9.2: Wait For Actionable (Go)
```
Implement internal/features/autowait.go:

- WaitForSelector(context, selector, timeout) → error
  - Polls until element exists
  - Default: 30s timeout, 100ms interval

- WaitForActionable(context, selector, checks []Check, timeout) → error
  - Polls until ALL specified checks pass
  - checks is a list like: [Visible, Stable, ReceivesEvents, Enabled]
  - Returns nil when all pass, TimeoutError if exceeded

Define check sets for each action:
- ClickChecks = [Visible, Stable, ReceivesEvents, Enabled]
- TypeChecks = [Visible, Stable, ReceivesEvents, Enabled, Editable]
```

### Milestone 9.3: Integrate with Actions (Go)
```
Update input actions to use actionability:

Before Click (in input.go or a new actions.go):
1. WaitForSelector (element exists)
2. WaitForActionable(selector, ClickChecks)
3. Perform click

Before Type:
1. WaitForSelector
2. WaitForActionable(selector, TypeChecks)
3. Perform type

Update CLI commands (click, type) to use these.
```

**Checkpoint:**
```bash
# Test with an element that becomes visible after delay
./bin/clicker click https://example.com "a" --timeout 5s
# Should wait for element to be actionable, then click

# Test actionability failure (overlay blocking element)
# Create a test page with an overlay
./bin/clicker click http://localhost:8080/overlay-test "button"
# Should timeout with "element does not receive events" error
```

### Milestone 9.4: JS Client Timeout Option
```
Minimal JS changes - just pass timeout to Go:

- element.click({timeout?}) passes timeout in BiDi command
- element.type(text, {timeout?}) passes timeout in BiDi command
- vibe.find(selector, {timeout?}) passes timeout in BiDi command

All actionability logic lives in Go. JS client is a thin wrapper.
Default timeout: 30s (set in Go, not JS).
```

**Checkpoint:**
```typescript
const vibe = await browser.launch();
await vibe.go('https://example.com');

// Test with delayed element
await vibe.evaluate(`
  setTimeout(() => {
    document.body.innerHTML += '<button id="delayed">Click me</button>';
  }, 2000);
`);

const btn = await vibe.find('#delayed', { timeout: 5000 });
await btn.click(); // Go handles: visible, stable, receives events, enabled
await vibe.quit();
```

---

## ⚠️ Human Review Checkpoint #2

Verify before proceeding: ✅ 2025-12-20 (automated via `make test`)
- ✅ Async API works end-to-end
- ✅ Sync API works end-to-end
- ✅ Auto-wait correctly waits for elements
- ✅ Timeout errors are clear
- ✅ No zombie processes
- ✅ Both headless and headed modes work

---

## ✅ Day 10: MCP Server

### Milestone 10.1: MCP Protocol Handler
```
Implement internal/mcp/server.go:
- Read JSON-RPC 2.0 from stdin
- Write JSON-RPC 2.0 to stdout  
- Handle: initialize, initialized, tools/list, tools/call
```

**Checkpoint:**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{}}}' | ./bin/clicker mcp
# Returns initialize response
```

### Milestone 10.2: Tool Schemas
```
Implement tools/list with schemas:

browser_launch:
  - headless: boolean (default false)
  
browser_navigate:
  - url: string (required)

browser_click:
  - selector: string (required)

browser_type:
  - selector: string (required)
  - text: string (required)

browser_screenshot:
  - filename: string (optional, save to file if --screenshot-dir is set)
  - Returns base64 image inline, or saves to file if filename provided

browser_find:
  - selector: string (required)
  - Returns: {tag, text, box}

browser_quit:
  - (no params)
```

### Milestone 10.3: Tool Handlers
```
Implement internal/mcp/handlers.go:
- Maintain browser session state
- Each tool calls underlying Clicker functions
- Return results or errors in MCP format
```

**Checkpoint:**
```bash
# Test with MCP inspector or manual stdin:
./bin/clicker mcp
> {"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"browser_launch","arguments":{}}}
> {"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"browser_navigate","arguments":{"url":"https://example.com"}}}
> {"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"browser_screenshot","arguments":{}}}
# Returns base64 screenshot
```

---

## ✅ Day 11: Polish & Error Handling

### Milestone 11.1: Error Types
```
Go errors (internal/errors/errors.go):
- ConnectionError: Can't connect to browser
- TimeoutError: Element not found in time
- ElementNotFoundError: Selector matched nothing
- BrowserCrashedError: Browser process died

JS errors (src/utils/errors.ts):
- Same error types, properly typed
- Include selector/URL context in messages
```

### Milestone 11.2: Logging
```
Go logging (use zerolog or slog):
- JSON structured logs to stderr
- Levels: debug, info, warn, error
- CLI flags: --verbose, --quiet

JS logging:
- debug package or similar
- VIBIUM_DEBUG=1 env var
```

### Milestone 11.3: Graceful Shutdown
```
Ensure cleanup on:
- Normal exit
- SIGINT (Ctrl+C)
- SIGTERM
- Client disconnect
- Panics (recover)

Use context.Context for cancellation propagation.
Test all scenarios leave no zombie Chrome processes.
```

**Checkpoint:**
```bash
# Test each termination scenario:
./bin/clicker serve & 
kill -INT $!
ps aux | grep chrome  # Should be empty

./bin/clicker mcp
# Ctrl+C
ps aux | grep chrome  # Should be empty
```

---

## ⚠️ Human Review Checkpoint #3

Verify before packaging: ✅ 2025-12-21
- ✅ MCP server works with Claude Code
- ✅ Error messages are helpful
- ✅ Logs are useful for debugging
- ✅ All shutdown scenarios clean up properly

---

## ✅ Day 12-13: Packaging & npm Publish

### ✅ Milestone 12.1: Cross-Compile Target
```
Add build-all-platforms target to Makefile:

Uses Go's built-in cross-compilation:
- GOOS=linux GOARCH=amd64 go build ...
- GOOS=linux GOARCH=arm64 go build ...
- GOOS=darwin GOARCH=amd64 go build ...
- GOOS=darwin GOARCH=arm64 go build ...
- GOOS=windows GOARCH=amd64 go build ...

Output to clicker/bin/clicker-{os}-{arch}[.exe]

Use CGO_ENABLED=0 for static binaries.
Use -ldflags="-s -w" for smaller binaries.

Note: Requires platform-specific build files for process management
(process_unix.go, process_windows.go, launcher_unix.go, launcher_windows.go)
```

**Checkpoint:**
```bash
make build-all-platforms
ls -la clicker/bin/
file clicker/bin/clicker-linux-amd64  # Should show static binary
```

### ✅ Milestone 12.2: Platform NPM Packages

> **Detailed Plan:** [docs/plans/V1-MILESTONE-12.2.md](docs/plans/V1-MILESTONE-12.2.md)

```
Create packages/:

packages/
├── linux-x64/                 # npm: @vibium/linux-x64
│   ├── package.json
│   └── bin/clicker
├── linux-arm64/               # npm: @vibium/linux-arm64
│   └── ...
├── darwin-x64/                # npm: @vibium/darwin-x64
│   └── ...
├── darwin-arm64/              # npm: @vibium/darwin-arm64
│   └── ...
└── win32-x64/                 # npm: @vibium/win32-x64
    └── ...

Each package.json:
- name: @vibium/{platform}-{arch}
- os and cpu fields for npm filtering
```

### ✅ Milestone 12.3: Main Package with Postinstall
```
Create packages/vibium/:
- Re-exports clients/javascript
- optionalDependencies for all platform packages
- Exports both async and sync APIs

postinstall.js:
- Find clicker binary from platform package
- Run: clicker install (downloads Chrome for Testing + chromedriver)
- Cache to <cache_dir>/chrome-for-testing/<version>/
- Respects VIBIUM_SKIP_BROWSER_DOWNLOAD=1 (skips download)
- Print success message with installed paths

package.json:
- "scripts": { "postinstall": "node postinstall.js" }
```

### ✅ Milestone 12.4: npx MCP Entry Point
```
Create packages/vibium/bin.js:
- Finds platform-specific clicker binary from optional deps
- Execs: clicker mcp (stdio mode)
- Passes through stdin/stdout/stderr

Update packages/vibium/package.json:
- "bin": { "vibium": "./bin.js" }
```

(Code completed in 12.3. Checkpoint merged into 12.5.)

### ✅ Milestone 12.5: End-to-End Package Test
```
Test the full package works after npm install:
- Clicker binary available
- Chrome for Testing downloaded
- Chromedriver downloaded
- JS API works
- MCP entry point works
```

**Checkpoint:**
```bash
cd packages/vibium && npm pack
mkdir /tmp/test-install && cd /tmp/test-install
npm init -y
npm install /path/to/vibium-0.1.0.tgz

# Verify postinstall downloaded browser (Linux example)
ls ~/.cache/vibium/chrome-for-testing/
# Should show version folder with chrome + chromedriver

# Test JS client API (uses installed Chrome)
node -e "
  const { browser, browserSync } = require('vibium');
  (async () => {
    const vibe = await browser.launch();
    await vibe.go('https://example.com');
    await vibe.quit();
    console.log('async works');
  })();
"

# Test MCP entry point
npx vibium &
# Should start and wait for JSON-RPC input

# Test Claude Code integration
claude mcp add vibium -- npx -y vibium
claude mcp list
# Should show vibium configured
```

---

## ✅ Day 14: Documentation

### ✅ Milestone 13.1: README
```
Update root README.md:
- Installation: npm install vibium
- Quick start (async)
- Quick start (sync)
- MCP setup for Claude Code
- API overview
```

### ✅ Milestone 13.2: Tutorials
```
Create examples/:

examples/
├── async-basic/
│   ├── package.json
│   └── index.ts
├── sync-basic/
│   ├── package.json
│   └── index.ts
└── claude-code-mcp/
    └── README.md (setup instructions)
```

### Milestone 13.3: API Reference (deferred)
```
Create docs/api.md:
- browser.launch(options)
- browserSync.launch(options)
- Vibe class methods
- VibeSync class methods
- Element class methods
- ElementSync class methods
- Error types
- Configuration options
```

---

## Final Checklist

### Functionality
- ✅ browser.launch() works (async)
- ✅ browserSync.launch() works (sync)
- ✅ Navigation works
- ✅ Screenshots captured
- ✅ Element finding works
- ✅ Click works
- ✅ Type works
- ✅ Auto-wait works
- ✅ MCP server responds to all tools
- ✅ Clean shutdown in all scenarios

### Platforms
- ✅ Linux x64
- [ ] Linux arm64
- ✅ macOS x64
- ✅ macOS arm64
- ✅ Windows x64

### Distribution
- ✅ npm install vibium works
- ✅ Binary auto-resolves per platform
- ✅ TypeScript types included
- ✅ ESM and CJS both work

---

## API Summary

### Async API
```typescript
import { browser } from 'vibium';

const vibe = await browser.launch();
await vibe.go('https://example.com');
const el = await vibe.find('button.submit');
await el.click();
await el.type('hello');
console.log(await el.text());
const png = await vibe.screenshot();
await vibe.quit();
```

### Sync API
```typescript
import { browserSync } from 'vibium';

const vibe = browserSync.launch();
vibe.go('https://example.com');
const el = vibe.find('button.submit');
el.click();
el.type('hello');
console.log(el.text());
const png = vibe.screenshot();
vibe.quit();
```

### MCP Tools
```
browser_launch    → Start browser session
browser_navigate  → Go to URL
browser_find      → Find element by selector
browser_click     → Click element
browser_type      → Type into element
browser_screenshot→ Capture viewport
browser_quit      → End session
```
