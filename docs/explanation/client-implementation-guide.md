# Client Implementation Guide

> **Draft:** This is a work-in-progress draft that may be used to generate client libraries for additional languages in the future.

Reference for implementing Vibium clients in new languages (Java, C#, Ruby, Kotlin, Swift, Rust, Go, Nim, etc.).

Use the **JS client** (`clients/javascript/`) and **Python client** (`clients/python/`) as reference implementations.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Which Transport](#which-transport)
3. [Drive It By Hand First](#drive-it-by-hand-first)
4. [Class Hierarchy](#class-hierarchy)
5. [Command Reference](#command-reference) — see [API Reference](../reference/api.md) for full tables
6. [Naming Conventions](#naming-conventions)
7. [Error Types](#error-types)
8. [Async / Sync Patterns](#async--sync-patterns)
9. [Reserved Keyword Handling](#reserved-keyword-handling)
10. [Aliases](#aliases)
11. [Key Design Decisions](#key-design-decisions)
12. [Binary Discovery](#binary-discovery)
13. [Testing Checklist](#testing-checklist)

---

## Architecture Overview

```
┌────────────────┐  stdin/stdout  ┌─────────────┐    BiDi/WS     ┌─────────┐
│ Client (JS,    │◄──────────────►│   vibium    │◄──────────────►│ Chrome  │
│ Python, etc.)  │ ndjson (pipes) │   binary    │ WebDriver BiDi │ browser │
└────────────────┘                └─────────────┘                └─────────┘
```

1. Client spawns the `vibium pipe` command as a subprocess
2. Client communicates via newline-delimited JSON over stdin/stdout
3. The binary sends a `vibium:lifecycle.ready` signal on stdout once the browser is launched
4. `vibium:` extension commands are handled by the binary; standard BiDi commands are forwarded to Chrome

### Message Format

**Request** (client → vibium):
```json
{"id": 1, "method": "vibium:page.navigate", "params": {"context": "ctx-1", "url": "https://example.com"}}
```

**Success response** (vibium → client):
```json
{"id": 1, "type": "success", "result": {}}
```

**Error response** (vibium → client):
```json
{"id": 1, "type": "error", "error": "timeout", "message": "Timeout after 30000ms waiting for '#btn'"}
```

**Event** (vibium → client, no `id`):
```json
{"method": "browsingContext.load", "params": {"context": "ctx-1", "url": "https://example.com"}}
```

---

## Which Transport

vibium exposes the same capabilities over four surfaces. Two engines, two
transports each. Knowing which one you are on matters, because a change to one
engine does not reach the other.

| Surface | Engine | Transport | Method names | Used by |
|---|---|---|---|---|
| `vibium pipe` | `api.Router` | stdin/stdout | `vibium:page.screenshot` | JS, Python, Java clients |
| `vibium serve` | `api.Router` | WebSocket | `vibium:page.screenshot` | third-party BiDi clients |
| `vibium daemon` | `agent.Handlers` | Unix socket | `browser_screenshot` | the `vibium` CLI |
| `vibium mcp` | `agent.Handlers` | stdin/stdout | `browser_screenshot` | AI clients over MCP |

**Write your client against `vibium pipe`.** It is what every shipped client
uses, so it is the best tested path. There is no port to allocate, no
authentication question, and the browser's lifetime is your process's lifetime,
so a crashed client cannot leak a browser.

`vibium serve` speaks the identical protocol over a WebSocket. It exists for
clients that already speak WebDriver BiDi and should not have to learn to spawn
our binary — see [use vibium as a BiDi endpoint](../how-to-guides/use-vibium-as-a-bidi-endpoint.md).
If you are writing a vibium client, you do not want it.

The daemon and MCP surfaces run a different engine with different method names.
They are not alternative transports for a client library.

## Drive It By Hand First

Before writing any code, talk to the binary from a shell. Everything a client
must handle is visible in the output.

```bash
{ echo '{"id":1,"method":"vibium:browser.page","params":{}}'; cat; } | vibium pipe --headless
```

`cat` holds stdin open so you can keep typing JSON lines; Ctrl-C when done.

```json
{"method":"browsingContext.contextCreated","params":{...},"type":"event"}
{"method":"vibium:lifecycle.ready","params":{"version":"26.5.31"}}
{"id":1,"type":"success","result":{"context":"B030B20D...","userContext":"default"}}
```

Three things that will otherwise cost you an afternoon:

**Wait for `vibium:lifecycle.ready` before sending.** Launching Chrome takes
seconds. Commands sent before the ready signal are answered after it, or not at
all if stdin closes first.

**Keep stdin open.** A bare `echo` closes it immediately, and the command comes
back `{"type":"error","message":"connection closed"}` because the browser was
still launching. That is what `cat` is for above. Do not swap in a fixed
`sleep`: launch is around a second normally but can stretch to tens of seconds
on a cold cache, a loaded CI box, or a VM with no GPU. Wait for the ready
signal, never a timer. Your client has to do the same.

**Events arrive before and between responses.** `contextCreated` shows up before
anything was sent. Events have no `id`; responses always do. Route on that.

### Drain stderr

The binary writes diagnostics to stderr. If your client does not read that pipe,
the OS buffer fills and **vibium blocks forever** on its next write. This is the
most common way a new client hangs with no error.

```js
proc.stderr.on('data', () => {});   // discard is fine; not reading is not
```

Use stderr for diagnostics only. Protocol messages are on stdout.

## Class Hierarchy

All clients must implement these classes:

```
Browser                  ← manages browser lifecycle
├── .context             ← default BrowserContext (property)
├── .keyboard            ← (accessed via Page)
├── .mouse               ← (accessed via Page)
└── .touch               ← (accessed via Page)

BrowserContext            ← cookie/storage isolation boundary
├── .recording           ← Recording (property)
└── newPage()            ← creates Page

Page                      ← a browser tab
├── .keyboard            ← Keyboard (property)
├── .mouse               ← Mouse (property)
├── .touch               ← Touch (property)
├── .clock               ← Clock (property)
├── .context             ← back-reference to BrowserContext
├── find() / findAll()   ← returns Element(s)
├── route()              ← creates Route via callback
├── onDialog()           ← creates Dialog via callback
├── onConsole()          ← creates ConsoleMessage via callback
├── onDownload()         ← creates Download via callback
├── onRequest()          ← creates Request via callback
├── onResponse()         ← creates Response via callback
└── onWebSocket()        ← creates WebSocketInfo via callback

Element                   ← a resolved DOM element
├── click/fill/type/...  ← interaction methods
├── text/html/value/...  ← state query methods
└── find() / findAll()   ← scoped element search

Keyboard                  ← page-level keyboard input
Mouse                     ← page-level mouse input
Touch                     ← page-level touch input
Clock                     ← fake timer control
Recording                 ← trace recording control
Route                     ← network interception handler
  └── .request           ← Request (property)
Dialog                    ← browser dialog (alert/confirm/prompt)
Request                   ← network request info
Response                  ← network response info
Download                  ← file download handle
ConsoleMessage            ← console.log() message
WebSocketInfo             ← WebSocket connection info
```

### Data Types

These should be structured types (interfaces/structs), not raw dicts:

| Type | Fields |
|---|---|
| `Cookie` | `name`, `value`, `domain`, `path`, `size`, `httpOnly`, `secure`, `sameSite`, `expiry?` |
| `SetCookieParam` | `name`, `value`, `domain?`, `url?`, `path?`, `httpOnly?`, `secure?`, `sameSite?`, `expiry?` |
| `StorageState` | `cookies: Cookie[]`, `origins: OriginState[]` |
| `OriginState` | `origin`, `localStorage: {name, value}[]`, `sessionStorage: {name, value}[]` |
| `BoundingBox` | `x`, `y`, `width`, `height` |
| `ElementInfo` | `tag`, `text`, `box: BoundingBox` |
| `A11yNode` | `role`, `name?`, `value?`, `description?`, `disabled?`, `expanded?`, `focused?`, `checked?`, `pressed?`, `selected?`, `level?`, `multiselectable?`, `children?: A11yNode[]` |
| `ScreenshotOptions` | `fullPage?`, `clip?: {x, y, width, height}` |
| `FindOptions` | `timeout?` |

---

## Command Reference

For the full command reference with wire commands, JS/Python signatures, MCP tools, and CLI commands, see **[Vibium API Reference](../reference/api.md)**.

All extension commands use the `vibium:` prefix. Standard WebDriver BiDi commands (e.g., `browsingContext.getTree`, `session.subscribe`) are forwarded directly to Chrome.

### Request / Response / ConsoleMessage / WebSocketInfo

These are lightweight data classes constructed from events. See the JS or Python source for their exact fields.

---

## Naming Conventions

### Method Names

| Convention | JS | Python | Java/Kotlin | C# | Ruby | Rust | Go | Nim |
|---|---|---|---|---|---|---|---|---|
| Multi-word methods | `camelCase` | `snake_case` | `camelCase` | `PascalCase` | `snake_case` | `snake_case` | `PascalCase` | `camelCase` |
| Boolean queries | `isVisible()` | `is_visible()` | `isVisible()` | `IsVisible()` | `visible?` | `is_visible()` | `IsVisible()` | `isVisible()` |
| Setters | `setViewport()` | `set_viewport()` | `setViewport()` | `SetViewport()` | `set_viewport` / `viewport=` | `set_viewport()` | `SetViewport()` | `setViewport()` |
| Event handlers | `onDialog(fn)` | `on_dialog(fn)` | `onDialog(fn)` | `OnDialog(fn)` | `on_dialog(&block)` | `on_dialog(fn)` | `OnDialog(fn)` | `onDialog(fn)` |

### Wire → Client Mapping

The wire protocol uses `camelCase`. Each language converts to its idiomatic style:

```
vibium:page.setViewport  →  JS: setViewport()   Python: set_viewport()   Ruby: set_viewport   Nim: setViewport()
vibium:element.isVisible →  JS: isVisible()     Python: is_visible()     Ruby: visible?        Nim: isVisible()
vibium:page.a11yTree     →  JS: a11yTree()      Python: a11y_tree()      Ruby: a11y_tree       Nim: a11yTree()
```

### Parameter Names

Wire parameters are `camelCase`. Convert to language idioms:

```
Wire: {"colorScheme": "dark", "reducedMotion": "reduce"}
JS:   colorScheme: "dark", reducedMotion: "reduce"     (same as wire)
Py:   color_scheme="dark", reduced_motion="reduce"     (snake_case)
Ruby: color_scheme: "dark", reduced_motion: "reduce"   (snake_case)
Nim:  colorScheme = "dark", reducedMotion = "reduce"   (same as wire)
```

**Important:** Always convert at the client boundary. Never leak wire-protocol casing to users (see [#91](https://github.com/VibiumDev/vibium/issues/91)).

---

## Error Types

Every client must define these error types:

| Error | When Thrown |
|---|---|
| `ConnectionError` | Failed to start or connect to the vibium binary |
| `TimeoutError` | Element wait or `waitForFunction` timed out |
| `ElementNotFoundError` | Selector matched no elements |
| `BrowserCrashedError` | Browser process died unexpectedly |

### Wire Error Detection

The wire protocol returns errors in this format:

```json
{"id": 1, "type": "error", "error": "timeout", "message": "Timeout after 30000ms waiting for '#btn'"}
```

Map the `error` field to structured error types:
- `"timeout"` → `TimeoutError`
- Messages containing `"not found"` or `"no elements"` → `ElementNotFoundError`
- The binary exits with no response → `BrowserCrashedError`
- The binary cannot be started or its pipes close → `ConnectionError`

### Language-Specific Names

Some languages have built-in `TimeoutError` or `ConnectionError`. Use prefixed names to avoid conflicts:

| Language | Timeout | Connection |
|---|---|---|
| JS/TS | `TimeoutError` | `ConnectionError` |
| Python | `VibiumTimeoutError` | `VibiumConnectionError` |
| Java | `VibiumTimeoutException` | `VibiumConnectionException` |
| C# | `VibiumTimeoutException` | `VibiumConnectionException` |
| Ruby | `TimeoutError` | `ConnectionError` (namespaced under `Vibium::`) |
| Rust | `Error::Timeout` | `Error::Connection` (enum variants) |
| Go | `ErrTimeout` | `ErrConnection` (sentinel errors) |
| Nim | `TimeoutError` | `ConnectionError` (namespaced under `vibium` module) |

---

## Async / Sync Patterns

### Every client must have an async API

The wire protocol is inherently async: responses and events arrive interleaved on stdout, and events have no `id`. The primary API should be async.

### Sync wrappers are optional but recommended

For scripting and REPL use, a sync wrapper dramatically improves the getting-started experience.

| Language | Async Pattern | Sync Pattern |
|---|---|---|
| JS/TS | `async/await` (native) | Separate `*Sync` classes |
| Python | `async/await` | Separate `sync_api/` module (blocks on event loop) |
| Java | `CompletableFuture<T>` | Blocking `.get()` wrappers |
| Kotlin | `suspend fun` (coroutines) | `runBlocking { }` wrappers |
| C# | `Task<T>` / `async` | `.GetAwaiter().GetResult()` wrappers |
| Ruby | Not needed (GIL) | Primary API is sync; use threads for events |
| Rust | `async fn` (tokio/async-std) | `block_on()` wrappers |
| Go | Goroutines (inherently concurrent) | Primary API is sync with channels for events |
| Swift | `async/await` (structured concurrency) | Sync wrappers with `DispatchSemaphore` |
| Nim | `async/await` (`asyncdispatch`) | `waitFor()` wrappers |

### Event Handling

Events (`onDialog`, `onRequest`, etc.) are received as WebSocket messages with no `id`. The client must:

1. Parse incoming messages
2. If `type` is `"success"` or `"error"` → match to pending request by `id`
3. If `method` is present (event) → dispatch to registered listeners

### Dialog Policy

The engine dismisses every dialog itself while no dialog handler is registered — clients do not implement that default. The policy is per browsing context: when a page's first dialog handler registers, send `vibium:dialog.setPolicy` with `{"context": <the page's context>, "policy": "manual"}`; when its last one deregisters, send the same with `"policy": "dismiss"`. Issue the command through the same ordered channel as regular commands and ahead of any command that could trigger a dialog (the engine handles it in message order), so a dialog can never open under the policy the page just left. One-shot captures need no policy flip: a pending `vibium:page.captureEvent` with kind `dialog` holds off the auto-dismiss default by itself.

### One-Shot Captures

`capture.request` / `capture.response` send `vibium:page.captureRequest` / `vibium:page.captureResponse` with `{context, pattern, timeout}`; `capture.navigation`, `capture.dialog`, and `capture.event("console" | "error")` send `vibium:page.captureEvent` with `{context, kind, timeout}`. The engine registers the capture in client message order, waits for the first matching event, and answers with its raw params (`{"event": {...}}`) — clients keep no listener or timeout machinery, they build the language object from the returned params. Start the capture command on the wire before running the trigger action, and run the action so that its blocking cannot block the capture's return (a trigger stuck inside `evaluate("alert(...)")` resolves only after the captured dialog is handled). `capture.download` stays a local listener for now: the captured Download must be the same object the `downloadEnd` event completes.

---

## Reserved Keyword Handling

Some method names conflict with language reserved words. Here's how to handle them:

| Wire Method | Conflict | Resolution |
|---|---|---|
| `vibium:network.continue` | `continue` is reserved in most languages | Python: `continue_()`, Java: `doContinue()`, Ruby: `continue_request`, C#: `Continue()` (C# allows PascalCase), Rust: `r#continue()` or `continue_()`, Go: `Continue()`, Nim: `continueRequest()` |

### General Rules

1. **Append underscore** (Python, Ruby): `continue_()`, `import_()`
2. **Prefix with `do`** (Java, Kotlin): `doContinue()`
3. **Raw identifier** (Rust): `r#continue()`
4. **PascalCase avoids most conflicts** (C#, Go)
5. **Rename** (Nim): `continueRequest()` — avoids backtick stropping for a cleaner API

---

## Aliases

The JS client provides some aliases for Playwright compatibility and discoverability. New clients should include these:

| Primary | Alias | Reason |
|---|---|---|
| `attr(name)` | `getAttribute(name)` | Playwright compat |
| `bounds()` | `boundingBox()` | Playwright compat |
| `go(url)` | — | Short and memorable; `navigate` is the wire name |
| `waitUntil(state)` | — | Maps to `vibium:element.waitFor` on wire |

### Which to Include

- **Always include the primary name** (shorter, Vibium-native)
- **Include Playwright aliases** for `getAttribute` and `boundingBox` — many users come from Playwright
- **Do not** alias everything — keep the API surface small

---

## Key Design Decisions

1. **Types are formal, variables are fun.** The API uses `Browser`, `Context`, `Page` — standard, unsurprising, self-documenting. But the convention in examples is `bro` and `vibe` — short, memorable, and distinctly Vibium. Agents and IDEs see `browser.newPage()` → `Page`. Humans see `const vibe = await bro.newPage()`.

2. **Three levels exist, most users see two.** `Browser` → `Context` → `Page`. But `browser.newPage()` skips the context layer by using a default context internally. Only call `browser.newContext()` when you need isolation (multi-user, test-per-context).

3. **One find, two signatures.** `find('.css')` for CSS (terse, 80% of cases). `find({role: 'button', text: 'Submit'})` for semantic (autocomplete, type-safe, combinable). In Python: `find(role='button', text='Submit')`. Playwright needs 8 separate methods and chaining to do what Vibium does with one method and two signatures.

4. **Events via `.on*()` methods.** `page.onDialog(fn)` not `page.on('dialog', fn)`. More discoverable, better autocomplete.

5. **`findAll()` returns immediately.** Empty array if nothing matches. Use `waitFor()` if you need to wait.

6. **Frames get full Page API.** In BiDi, frames ARE browsing contexts. `page.frame('name')` returns an object with the same interface as a page.

7. **AI methods are first-class.** `page.check()` and `page.do()` aren't afterthoughts — they're the reason Vibium exists. They use the deterministic API under the hood.

8. **Logic lives in the binary, clients stay thin.** The default implementation of any method is: send the wire command, return the result. Auto-waiting, actionability, selector semantics, dialog handling, geolocation persistence — all of it runs inside the vibium binary so that every client gets identical behavior for free and none of it is written once per language. Client-side logic is reserved for what genuinely cannot live in the binary: the transport, language-idiomatic types, and delivering events to user callbacks. If a port finds itself implementing behavior, stop and move that behavior into the binary first. This is what keeps N clients maintainable and is enforced socially in review; the API drift checker keeps the surfaces aligned, this rule keeps the semantics aligned.

---

## Binary Discovery

Each client needs to find and launch the `vibium` binary. The resolution order:

1. **Environment variable** `VIBIUM_BIN_PATH` — highest priority
2. **PATH lookup** — `which vibium` / `where vibium`
3. **npm-installed binary** — check `node_modules/.bin/vibium`
4. **Known install locations** — platform-specific defaults

### Reference

- JS: `clients/javascript/src/clicker/binary.ts` → `getVibiumBinPath()`
- Python: `clients/python/src/vibium/binary.py` → `find_vibium_bin()`

---

## Testing Checklist

Before releasing a new client, verify:

- [ ] `browser.start()` launches a visible browser
- [ ] `browser.start(headless=True)` launches headless
- [ ] `page.go(url)` navigates and waits for load
- [ ] `page.find("selector")` returns an Element
- [ ] `element.click()` performs a click
- [ ] `element.fill("text")` fills an input
- [ ] `page.screenshot()` returns image bytes
- [ ] `page.evaluate("1 + 1")` returns `2`
- [ ] `context.cookies()` / `setCookies()` round-trips
- [ ] `page.route()` intercepts and can fulfill requests
- [ ] `page.onDialog()` handles alert/confirm/prompt
- [ ] Error types are raised (timeout, element not found)
- [ ] `browser.stop()` cleanly shuts down
- [ ] Binary discovery works via `VIBIUM_BIN_PATH` and PATH
- [ ] Sync wrapper works (if provided)

Run the existing test suite against your client:

```bash
make test  # runs CLI + JS + MCP + Python tests
```
