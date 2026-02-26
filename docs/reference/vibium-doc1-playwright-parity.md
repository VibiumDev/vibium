# Doc #1: Vibium Playwright-Equivalent Functionality

**Tracker:** docs/trackers/api.md
**Goal:** Build Vibium's native API to Playwright-level DX, powered by WebDriver BiDi with JS injection fallback where BiDi doesn't yet cover a feature.

This is the primary design document. Doc #2 (WebDriver spec parity) and Doc #3 (Selenium compat layer) build on top of this work.

---

## Handoff Notes (for Claude Code)

**What already exists:** A Go binary ("clicker engine") that handles browser lifecycle, WebDriver BiDi protocol over WebSocket, and exposes an MCP server. The clients (JS/TS, Python, CLI) are thin wrappers — the heavy lifting is in Go. A sync JS mechanism is already in place.

**What this doc covers:** The target API surface — naming, object model, command signatures, implementation approach per command (BiDi vs JS injection), and priority tiers. This doc does NOT redefine architecture — the codebase is the source of truth for how things are built. This doc defines what the public API should look like.

**What to do:** Implement the API surface described below, tier by tier, across the six targets. Read the existing code to understand the Go engine's capabilities and work within that architecture.

**Breaking changes from current codebase:**
- **Python: sync is now the default.** `from vibium import browser` gives you the sync API. The current async-first behavior moves to `from vibium.async_api import browser`. Existing Python code that uses `await` with the bare import will need to switch to the async import. This aligns with Python ecosystem conventions — `requests`, Selenium, and most Python libraries are sync by default.

**Critical architecture principle: thin clients, fat engine.** The Go-based clicker binary should do as much work as possible. The language-specific libraries (JS/TS, Python) should be thin proxies that serialize commands, send them to the clicker engine, and deserialize responses. Auto-waiting logic, element resolution, selector strategy matching, retry behavior, screenshot capture, BiDi protocol handling — all of this lives in Go. If you're tempted to put logic in a language client, put it in the clicker engine instead. This keeps clients trivial to write and ensures consistent behavior across all six targets. The reason we can ship Java, C#, Ruby, Kotlin, and Nim bindings later without heroic effort is because the clients are thin.

---

## Object Model

```
browser.launch()        → Browser   (the browser process)
browser.newContext()    → Context   (isolated cookie jar / storage)
browser.newPage()       → Page      (default context + newPage)
context.newPage()       → Page      (page in a specific context)
page.find(sel)          → Element
page.keyboard           → Keyboard
page.mouse              → Mouse
page.check(claim)       → CheckResult   ← AI-native verification
page.do(action)         → ActionResult  ← AI-native action
```

Three levels exist, but most users only see two (`browser` + `page`). The default context is implicit.

### Core Pattern

```javascript
import { browser } from 'vibium'

const bro = await browser.launch()
const vibe = await bro.newPage()     // uses default context internally
await vibe.go('https://example.com')
await vibe.check('the page loaded')
await bro.close()
```

### Multi-Page

```javascript
const bro = await browser.launch()
const vibe1 = await bro.newPage()
const vibe2 = await bro.newPage()

await vibe1.go('https://app.com/dashboard')
await vibe2.go('https://app.com/settings')

await vibe1.check('dashboard has 3 widgets')
await vibe2.check('dark mode is enabled')
```

### Isolated Contexts (multi-user, test isolation)

```javascript
const bro = await browser.launch()

// Each context has its own cookies, localStorage, state
const alice = await bro.newContext()
const bob = await bro.newContext()

const aliceVibe = await alice.newPage()
const bobVibe = await bob.newPage()

await aliceVibe.go('https://chat.app/login')
await bobVibe.go('https://chat.app/login')
// Alice and Bob have completely separate login state
```

```javascript
// Test isolation — fresh state per test, one browser process
const bro = await browser.launch()

test('adds to cart', async () => {
  const ctx = await bro.newContext()
  const vibe = await ctx.newPage()
  await vibe.go('https://store.example.com')
  await vibe.check('cart is empty')
  await ctx.close()  // clean up
})

test('checkout flow', async () => {
  const ctx = await bro.newContext()  // isolated from previous test
  const vibe = await ctx.newPage()
  // ...
})
```

### Python

```python
from vibium import browser

# Sync — the default, no ceremony
bro = browser.launch()
vibe = bro.new_page()
vibe.go("https://example.com")
vibe.check("the page loaded")
bro.close()

# Isolated contexts
alice = bro.new_context()
bob = bro.new_context()
alice_vibe = alice.new_page()
bob_vibe = bob.new_page()

# Async — opt-in
from vibium.async_api import browser

bro = await browser.launch()
vibe = await bro.new_page()
await vibe.go("https://example.com")
```

---

## Naming Principles

- **Vibium's own idiom** — not a Playwright clone, not a Selenium clone.
- **Short names:** `go()` not `goto()`, `el.text()` not `textContent()`, `find()` not `querySelector()`.
- **One find method, two signatures:** `find('css')` for CSS selectors (80% of cases), `find({role, text, label, ...})` for semantic strategies. Combinable: `find({role: 'button', text: 'Submit'})`. In Python: `find(role='button', text='Submit')`.
- **No switching:** pages and frames are independently addressable by reference.
- **Events:** `page.onDialog(fn)`, `page.onConsole(fn)` — discoverable method style, not EventEmitter.

---

## Feature Categories

### 1. Navigation (9 commands)

| Vibium | Playwright equiv | Implementation | Notes |
|--------|-----------------|----------------|-------|
| `page.go(url)` | `page.goto(url)` | BiDi: `browsingContext.navigate` | Returns when load event fires |
| `page.back()` | `page.goBack()` | BiDi: `browsingContext.traverseHistory(-1)` | |
| `page.forward()` | `page.goForward()` | BiDi: `browsingContext.traverseHistory(1)` | |
| `page.reload()` | `page.reload()` | BiDi: `browsingContext.reload` | |
| `page.url()` | `page.url()` | Client-side (tracked from navigate responses) | |
| `page.title()` | `page.title()` | JS: `document.title` via `script.evaluate` | |
| `page.content()` | `page.content()` | JS: `document.documentElement.outerHTML` | Full page HTML |
| `page.waitForURL(pattern)` | `page.waitForURL()` | BiDi: poll `browsingContext.navigate` events | |
| `page.waitForLoad(state?)` | `page.waitForLoadState()` | BiDi: `browsingContext.load` / `DOMContentLoaded` events | |

### 2. Pages, Sessions & Contexts (11 commands)

| Vibium | Playwright equiv | Implementation | Notes |
|--------|-----------------|----------------|-------|
| `browser.newPage()` | `browser.newPage()` | BiDi: `browsingContext.create` (default context) | Returns a Page |
| `browser.newContext()` | `browser.newContext()` | BiDi: `browser.createUserContext` | Isolated cookie jar / storage |
| `context.newPage()` | `context.newPage()` | BiDi: `browsingContext.create` (in user context) | Returns a Page |
| `browser.pages()` | `browser.contexts()` → pages | BiDi: `browsingContext.getTree` | Returns Page[] |
| `context.close()` | `context.close()` | BiDi: close all pages in context | |
| `browser.close()` | `browser.close()` | BiDi: close all contexts + end | |
| `browser.onPage(fn)` | `context.on('page')` | BiDi: `browsingContext.contextCreated` | New tab/window |
| `browser.onPopup(fn)` | `page.on('popup')` | BiDi: `browsingContext.contextCreated` (opener) | New popup window |
| `browser.removeAllListeners(event?)` | `browser.removeAllListeners()` | Client-side: clear callback arrays | Clears 'page' and/or 'popup' listeners |
| `page.bringToFront()` | `page.bringToFront()` | BiDi: `browsingContext.activate` | Focus this tab |
| `page.close()` | `page.close()` | BiDi: `browsingContext.close` | Close single page |

### 3. Element Finding (5 core commands + selector strategies)

**The big idea:** `find()` takes either a CSS string or structured options. CSS for the common case (terse), options for semantic strategies (autocomplete, combinable, type-safe).

```javascript
// CSS — just a string (80% of cases)
vibe.find('.btn-primary')
vibe.find('#email')
vibe.find('input[name="q"]')

// Semantic — object with full autocomplete
vibe.find({ role: 'button' })
vibe.find({ text: 'Sign In' })
vibe.find({ label: 'Email' })

// Combos — the killer feature (Playwright requires chaining for this)
vibe.find({ role: 'button', text: 'Submit' })
vibe.find({ role: 'textbox', label: 'Email' })
vibe.find({ role: 'link', text: 'Learn more', near: '.hero' })
```

```python
# Python — kwargs are perfect
vibe.find('.btn-primary')
vibe.find(role='button')
vibe.find(role='button', text='Submit')
vibe.find(label='Email')
vibe.find(placeholder='Search...')
```

| Vibium | Playwright equiv | Implementation | Notes |
|--------|-----------------|----------------|-------|
| `page.find('css')` | `page.locator('css')` | BiDi: `browsingContext.locateNodes` (CSS) | String = CSS selector. Returns Element, auto-waits. |
| `page.find({...})` | `getByRole/Text/Label/...` | BiDi + JS depending on strategy | Object = semantic. Full autocomplete. Combinable. |
| `page.findAll('css')` | `page.locator().all()` | BiDi: `browsingContext.locateNodes` | Returns Element[], immediate |
| `page.findAll({...})` | N/A (Playwright chains) | BiDi + JS | Semantic findAll |
| `el.find('css')` | `locator.locator()` | BiDi: scoped `locateNodes` | Scoped to parent element |
| `el.find({...})` | `locator.getByRole/...` | BiDi + JS, scoped | Scoped semantic find |

**Selector strategies (all usable as object keys):**

| Key | Playwright equiv | Implementation | Notes |
|-----|-----------------|----------------|-------|
| `role` | `getByRole()` | JS: `computedRole` matching | ARIA roles |
| `text` | `getByText()` | JS: `innerText` matching | String or regex |
| `label` | `getByLabel()` | JS: `aria-label` + `<label>` association | Computed accessible name |
| `placeholder` | `getByPlaceholder()` | JS: `[placeholder]` matching | |
| `alt` | `getByAltText()` | JS: `[alt]` attribute matching | Images, areas |
| `title` | `getByTitle()` | JS: `[title]` attribute matching | |
| `testid` | `getByTestId()` | BiDi: CSS `[data-testid="..."]` | Configurable attribute name |
| `xpath` | `locator('xpath=...')` | BiDi: `locateNodes` xpath strategy | For complex DOM traversal |
| `near` | N/A | BiDi + JS: scope to CSS parent | Narrow search region |

**Why this is better than Playwright:** Playwright has 7 separate find methods (`locator`, `getByRole`, `getByText`, `getByLabel`, `getByPlaceholder`, `getByAltText`, `getByTitle`, `getByTestId`). Vibium has one: `find()`. And combining strategies that requires chaining in Playwright (`getByRole('button').filter({hasText: 'Submit'})`) is just `find({ role: 'button', text: 'Submit' })` in Vibium.

### 4. Locator Chaining & Filtering (8 commands)

| Vibium | Playwright equiv | Implementation | Notes |
|--------|-----------------|----------------|-------|
| `el.first()` | `locator.first()` | Client-side: index 0 of matches | |
| `el.last()` | `locator.last()` | Client-side: last index of matches | |
| `el.nth(index)` | `locator.nth(index)` | Client-side: index into matches | 0-based |
| `el.count()` | `locator.count()` | BiDi: `locateNodes` + count | Returns number |
| `el.filter({hasText})` | `locator.filter({hasText})` | JS: filter by text content | String or regex |
| `el.filter({has})` | `locator.filter({has})` | JS: filter by child locator | Composable |
| `el.or(other)` | `locator.or(other)` | Client-side: union of matches | First match wins |
| `el.and(other)` | `locator.and(other)` | Client-side: intersection of matches | Both must match |

### 5. Element Interaction (16 commands)

| Vibium | Playwright equiv | Implementation | Notes |
|--------|-----------------|----------------|-------|
| `el.click()` | `locator.click()` | BiDi: `input.performActions` (pointer) | Scrolls into view, waits for actionable |
| `el.dblclick()` | `locator.dblclick()` | BiDi: `input.performActions` (2x click) | |
| `el.fill(value)` | `locator.fill()` | BiDi: focus + clear + `input.setFiles` or key actions | Clears first |
| `el.type(text)` | `locator.pressSequentially()` | BiDi: `input.performActions` (key sequence) | Char by char, no clear |
| `el.press(key)` | `locator.press()` | BiDi: `input.performActions` (key down+up) | Single key combo |
| `el.clear()` | `locator.clear()` | JS: select all + delete | |
| `el.check()` | `locator.check()` | JS: click if not already checked | |
| `el.uncheck()` | `locator.uncheck()` | JS: click if currently checked | |
| `el.selectOption(val)` | `locator.selectOption()` | JS: set `<select>` value, dispatch change | No BiDi `<select>` command |
| `el.setFiles(paths)` | `locator.setInputFiles()` | JS: FileList manipulation or CDP fallback | |
| `el.hover()` | `locator.hover()` | BiDi: `input.performActions` (pointer move) | |
| `el.focus()` | `locator.focus()` | JS: `element.focus()` | |
| `el.dragTo(target)` | `locator.dragTo()` | BiDi: `input.performActions` (pointer sequence) | |
| `el.tap()` | `locator.tap()` | BiDi: `input.performActions` (touch) | |
| `el.scrollIntoView()` | `locator.scrollIntoViewIfNeeded()` | JS: `element.scrollIntoViewIfNeeded()` | Explicit scroll (auto-scroll is separate behavior) |
| `el.dispatchEvent(type)` | `locator.dispatchEvent()` | JS: `element.dispatchEvent(new Event(type))` | Fire custom DOM events |

### 6. Element State (14 commands)

| Vibium | Playwright equiv | Implementation | Notes |
|--------|-----------------|----------------|-------|
| `el.text()` | `locator.textContent()` | JS: `element.textContent` via `script.evaluate` | |
| `el.innerText()` | `locator.innerText()` | JS: `element.innerText` | Rendered text only |
| `el.html()` | `locator.innerHTML()` | JS: `element.innerHTML` | |
| `el.value()` | `locator.inputValue()` | JS: `element.value` | For inputs |
| `el.attr(name)` | `locator.getAttribute()` | JS: `element.getAttribute()` | |
| `el.bounds()` | `locator.boundingBox()` | JS: `element.getBoundingClientRect()` | Returns {x,y,w,h} |
| `el.isVisible()` | `locator.isVisible()` | JS: computed style + intersection check | |
| `el.isHidden()` | `locator.isHidden()` | JS: inverse of isVisible | |
| `el.isEnabled()` | `locator.isEnabled()` | JS: `!element.disabled` | |
| `el.isChecked()` | `locator.isChecked()` | JS: `element.checked` | |
| `el.isEditable()` | `locator.isEditable()` | JS: not disabled, not readonly | |
| `el.eval(fn)` | `locator.evaluate()` | BiDi: `script.callFunction` with element ref | |
| `el.screenshot()` | `locator.screenshot()` | BiDi: `browsingContext.captureScreenshot` with clip | |
| `el.waitFor({state})` | `locator.waitFor()` | BiDi: poll locateNodes + state check | Wait for visible/hidden/attached/detached |

### 7. Keyboard & Mouse (10 commands)

| Vibium | Playwright equiv | Implementation | Notes |
|--------|-----------------|----------------|-------|
| `page.keyboard.press(key)` | `keyboard.press()` | BiDi: `input.performActions` key | |
| `page.keyboard.down(key)` | `keyboard.down()` | BiDi: `input.performActions` keyDown | |
| `page.keyboard.up(key)` | `keyboard.up()` | BiDi: `input.performActions` keyUp | |
| `page.keyboard.type(text)` | `keyboard.type()` | BiDi: `input.performActions` key sequence | |
| `page.mouse.click(x, y)` | `mouse.click()` | BiDi: `input.performActions` pointer | |
| `page.mouse.move(x, y)` | `mouse.move()` | BiDi: `input.performActions` pointerMove | |
| `page.mouse.down()` | `mouse.down()` | BiDi: `input.performActions` pointerDown | |
| `page.mouse.up()` | `mouse.up()` | BiDi: `input.performActions` pointerUp | |
| `page.mouse.wheel(dx, dy)` | `mouse.wheel()` | BiDi: `input.performActions` scroll | |
| `page.touch.tap(x, y)` | `touchscreen.tap()` | BiDi: `input.performActions` touch | |

### 8. Network Interception (12 commands)

| Vibium | Playwright equiv | Implementation | Notes |
|--------|-----------------|----------------|-------|
| `page.route(pattern, handler)` | `page.route()` | BiDi: `network.addIntercept` | Major DX win over Selenium |
| `route.fulfill(response)` | `route.fulfill()` | BiDi: `network.provideResponse` | |
| `route.continue(overrides?)` | `route.continue()` | BiDi: `network.continueRequest` | |
| `route.abort(reason?)` | `route.abort()` | BiDi: `network.failRequest` | |
| `page.onRequest(fn)` | `page.on('request')` | BiDi: `network.beforeRequestSent` | |
| `page.onResponse(fn)` | `page.on('response')` | BiDi: `network.responseCompleted` | |
| `page.setHeaders(headers)` | `page.setExtraHTTPHeaders()` | BiDi: `network.addIntercept` + modify | |
| `page.waitForRequest(pattern)` | `page.waitForRequest()` | BiDi: subscribe + filter | |
| `page.waitForResponse(pattern)` | `page.waitForResponse()` | BiDi: subscribe + filter | |
| `page.removeAllListeners(event?)` | `page.removeAllListeners()` | Client-side: clear callback arrays + teardown data collector | Clears 'request', 'response', and/or 'dialog' listeners |
| `page.routeWebSocket(pattern)` | `page.routeWebSocket()` | BiDi: intercept WebSocket frames | Mock WebSocket connections |
| `page.onWebSocket(fn)` | `page.on('websocket')` | BiDi: WebSocket events | |

### 9. Request & Response Objects (8 commands)

| Vibium | Playwright equiv | Implementation | Notes |
|--------|-----------------|----------------|-------|
| `request.url()` | `request.url()` | BiDi: from `network.beforeRequestSent` data | |
| `request.method()` | `request.method()` | BiDi: from request data | GET, POST, etc. |
| `request.headers()` | `request.allHeaders()` | BiDi: from request data | |
| `request.postData()` | `request.postData()` | BiDi: `network.getData` with `network.addDataCollector` | Requires data collector; returns null if no body or unsupported |
| `response.status()` | `response.status()` | BiDi: from `network.responseCompleted` data | HTTP status code |
| `response.headers()` | `response.allHeaders()` | BiDi: from response data | |
| `response.body()` | `response.body()` | BiDi: from response body | Returns Buffer |
| `response.json()` | `response.json()` | BiDi: parse body as JSON | Convenience method |

### 10. Dialogs (5 commands)

| Vibium | Playwright equiv | Implementation | Notes |
|--------|-----------------|----------------|-------|
| `page.onDialog(fn)` | `page.on('dialog')` | BiDi: `browsingContext.userPromptOpened` | |
| `dialog.accept(text?)` | `dialog.accept()` | BiDi: `browsingContext.handleUserPrompt(accept: true)` | |
| `dialog.dismiss()` | `dialog.dismiss()` | BiDi: `browsingContext.handleUserPrompt(accept: false)` | |
| `dialog.message()` | `dialog.message()` | From `userPromptOpened` event data | |
| `dialog.type()` | `dialog.type()` | From event: alert, confirm, prompt, beforeunload | |

### 11. Screenshots & PDF (4 commands)

| Vibium | Playwright equiv | Implementation | Notes |
|--------|-----------------|----------------|-------|
| `page.screenshot()` | `page.screenshot()` | BiDi: `browsingContext.captureScreenshot` | |
| `page.screenshot({ fullPage: true })` | `page.screenshot({ fullPage })` | BiDi: captureScreenshot with full document origin | |
| `page.screenshot({ clip: rect })` | `page.screenshot({ clip })` | BiDi: captureScreenshot with clip rect | |
| `page.pdf()` | `page.pdf()` | BiDi: `browsingContext.print` | Chromium only in Playwright too |

### 12. Cookies & Storage (5 commands)

| Vibium | Playwright equiv | Implementation | Notes |
|--------|-----------------|----------------|-------|
| `context.cookies(urls?)` | `context.cookies()` | BiDi: `storage.getCookies` | Context-level |
| `context.setCookies(cookies)` | `context.addCookies()` | BiDi: `storage.setCookie` | |
| `context.clearCookies()` | `context.clearCookies()` | BiDi: `storage.deleteCookies` | |
| `context.storageState()` | `context.storageState()` | JS: read localStorage/sessionStorage + cookies | Serialize full state |
| `context.addInitScript(script)` | `context.addInitScript()` | BiDi: `script.addPreloadScript` | Runs before page scripts |

### 13. Emulation (6 commands)

| Vibium | Playwright equiv | Implementation | Notes |
|--------|-----------------|----------------|-------|
| `page.setViewport(size)` | `page.setViewportSize()` | BiDi: `browsingContext.setViewport` | |
| `page.viewport()` | `page.viewportSize()` | Client-side (tracked from setViewport) | |
| `page.emulateMedia(opts)` | `page.emulateMedia()` | JS/CDP: override media features | No BiDi command yet |
| `page.setContent(html)` | `page.setContent()` | JS: `document.open/write/close` or navigate to data URL | |
| `page.setGeolocation(coords)` | `context.setGeolocation()` | JS/CDP: override geolocation API | No BiDi command yet |
| `page.grantPermissions(perms)` | `context.grantPermissions()` | BiDi: `permissions.setPermission` | |

### 14. Frames (4 commands)

| Vibium | Playwright equiv | Implementation | Notes |
|--------|-----------------|----------------|-------|
| `page.frame(nameOrUrl)` | `page.frame()` | BiDi: `browsingContext.getTree` + filter | Returns Page-like object |
| `page.frames()` | `page.frames()` | BiDi: `browsingContext.getTree` children | |
| `page.mainFrame()` | `page.mainFrame()` | The top-level browsing context | |
| Frame has full Vibe API | Frame has full Page API | Frames ARE browsing contexts in BiDi | No switching needed |

### 15. Accessibility (4 commands)

| Vibium | Playwright equiv | Implementation | Notes |
|--------|-----------------|----------------|-------|
| `page.a11yTree()` | `page.accessibility.snapshot()` | JS: walk DOM, compute ARIA roles/names/states | BiDi has no a11y module yet |
| `el.role()` | `locator.role` (via getByRole) | JS: `element.computedRole` | |
| `el.label()` | `locator.label` (via getByLabel) | JS: `element.computedLabel` | |

### 16. Console, Errors & Workers (3 commands)

| Vibium | Playwright equiv | Implementation | Notes |
|--------|-----------------|----------------|-------|
| `page.onConsole(fn)` | `page.on('console')` | BiDi: `log.entryAdded` | |
| `page.onError(fn)` | `page.on('pageerror')` | BiDi: `log.entryAdded` (level: error) | |
| `page.workers()` | `page.workers()` | BiDi: dedicated worker contexts | Returns Worker[] |

### 17. Waiting (5 commands)

| Vibium | Playwright equiv | Implementation | Notes |
|--------|-----------------|----------------|-------|
| `page.waitFor(selector)` | `page.waitForSelector()` | BiDi: poll `locateNodes` | Returns Element when found |
| `page.wait(ms)` | `page.waitForTimeout()` | Client-side setTimeout | Discouraged but useful |
| `page.waitForFunction(fn)` | `page.waitForFunction()` | BiDi: poll `script.evaluate` | |
| `page.waitForEvent(name)` | `page.waitForEvent()` | BiDi: subscribe + resolve on match | |
| `page.pause()` | `page.pause()` | Client-side: debugger breakpoint | Opens inspector when headed |

### 18. Downloads & Files (3 commands)

| Vibium | Playwright equiv | Implementation | Notes |
|--------|-----------------|----------------|-------|
| `page.onDownload(fn)` | `page.on('download')` | BiDi: detect navigation to download | |
| `download.saveAs(path)` | `download.saveAs()` | Client-side: file management | |
| `page.onFileChooser(fn)` | `page.on('filechooser')` | BiDi/JS: detect file input activation | |

### 19. Clock (8 commands)

| Vibium | Playwright equiv | Implementation | Notes |
|--------|-----------------|----------------|-------|
| `page.clock.install(opts?)` | `page.clock.install()` | JS: override Date, setTimeout, setInterval, requestAnimationFrame, performance.now | Options: `time` (epoch ms), `timezone` (IANA ID). Preload script survives navigation. |
| `page.clock.fastForward(ms)` | `page.clock.fastForward()` | JS: advance fake timers | Fire each due timer at most once |
| `page.clock.runFor(ms)` | `page.clock.runFor()` | JS: advance time systematically | Fires all callbacks, reschedules intervals |
| `page.clock.pauseAt(time)` | `page.clock.pauseAt()` | JS: jump to time and pause | No timers fire until resumed/advanced |
| `page.clock.resume()` | `page.clock.resume()` | JS: resume real-time progression | Starts real-time sync loop from current fake time |
| `page.clock.setFixedTime(time)` | `page.clock.setFixedTime()` | JS: freeze Date.now() permanently | Timers still run |
| `page.clock.setSystemTime(time)` | `page.clock.setSystemTime()` | JS: set Date.now() without firing timers | Relocate clock without side effects |
| `page.clock.setTimezone(tz)` | N/A (context option) | BiDi: `emulation.setTimezoneOverride` | IANA timezone ID; empty string resets to default |

### 20. Tracing (6 commands)

| Vibium | Playwright equiv | Implementation | Notes |
|--------|-----------------|----------------|-------|
| `context.tracing.start(opts)` | `tracing.start()` | BiDi: record events + periodic screenshots | Options: name, screenshots, snapshots, sources, title |
| `context.tracing.stop(opts)` | `tracing.stop()` | Package trace into Playwright-compatible zip | Option: path |
| `context.tracing.startChunk(opts)` | `tracing.startChunk()` | Reset event buffer, increment chunk index | Options: name, title |
| `context.tracing.stopChunk(opts)` | `tracing.stopChunk()` | Package current chunk into zip | Option: path |
| `context.tracing.startGroup(name)` | `tracing.group()` | Add group-start marker to trace | Renamed for start/stop consistency |
| `context.tracing.stopGroup()` | `tracing.groupEnd()` | Add group-end marker to trace | Renamed for start/stop consistency |

### 21. Evaluation (5 commands)

| Vibium | Playwright equiv | Implementation | Notes |
|--------|-----------------|----------------|-------|
| `page.eval(expression)` | `page.evaluate()` | BiDi: `script.evaluate` | Returns serialized result |
| `page.evalHandle(expression)` | `page.evaluateHandle()` | BiDi: `script.evaluate` (return remote ref) | Returns handle, not value |
| `page.addScript(url_or_content)` | `page.addScriptTag()` | JS: create `<script>` element | |
| `page.addStyle(url_or_content)` | `page.addStyleTag()` | JS: create `<link>` or `<style>` element | |
| `page.expose(name, fn)` | `page.exposeFunction()` | BiDi: `script.addPreloadScript` + `script.callFunction` callback | Bridge page→Node |

### 22. AI-Native Methods (Vibium-only)

These are Vibium's signature differentiators. No Playwright or Selenium equivalent exists.

#### `page.check(claim, options?)` — AI-Powered Verification

```javascript
// Plain English assertions — the thing nobody else can do
await vibe.check('the shopping cart icon shows 0 items')
await vibe.check('user is logged in')
await vibe.check('prices are sorted low to high')
await vibe.check('the form shows a validation error for email')
await vibe.check('dark mode is active')

// With selector hint to narrow scope
await vibe.check('shows 3 results', { near: '.search-results' })
await vibe.check('the total is correct', { near: '#checkout-summary' })

// Structured result
const result = await vibe.check('the dashboard loaded successfully')
// {
//   passed: true,
//   reason: "Dashboard shows welcome message and 3 widget panels",
//   screenshot: Buffer,
//   confidence: 0.95
// }

// Use in test assertions
const { passed } = await vibe.check('no error messages visible')
assert(passed)
```

**Implementation:** screenshot → multimodal LLM → structured response. Optionally augmented with DOM snapshot / a11y tree for precision.

**Options:**
- `near` — CSS selector to constrain the visual/DOM region
- `timeout` — max wait time (default: 5s, retries until claim passes or timeout)
- `screenshot` — include screenshot in result (default: true)
- `model` — override default AI model

#### `page.do(action, options?)` — AI-Powered Action

```javascript
// Natural language actions when you don't know the exact selectors
await vibe.do('log in with username "admin" and password "secret"')
await vibe.do('add the first item to cart')
await vibe.do('close the cookie consent banner')
await vibe.do('navigate to the settings page')

// With constraints
await vibe.do('fill out the shipping form', {
  data: { name: 'Jane Doe', address: '123 Main St', zip: '60601' }
})

// Structured result
const result = await vibe.do('click the submit button')
// {
//   done: true,
//   steps: ['Found submit button with text "Submit Order"', 'Clicked button'],
//   screenshot: Buffer
// }
```

**Implementation:** screenshot + DOM snapshot → LLM plans actions → executes via Vibium's own API (find, click, fill, etc.) → verifies result.

**The key insight:** `page.do()` uses Vibium's own deterministic API under the hood. It's AI planning, not AI puppeteering. The actions it takes are the same `find()`, `click()`, `fill()` commands a human would write.

#### Philosophy

Traditional test:
```javascript
await vibe.find('testid=cart-count').text() // "0"
```

Vibium test:
```javascript
await vibe.check('cart is empty')
```

The deterministic API is for when you know what you're looking for. The AI methods are for when you want to describe intent. Both coexist — `page.check()` doesn't replace `el.text()`, it complements it.

---

## Implementation Targets

Each command is tracked across six delivery surfaces:

| Target | Description | Notes |
|--------|-------------|-------|
| **JS/TS async** | JavaScript / TypeScript async API | Primary implementation. All BiDi protocol work happens here first. |
| **JS/TS sync** | JavaScript / TypeScript sync wrapper | Sync convenience layer over the async API. |
| **Python async** | Python asyncio API | `async`/`await` style. Wraps same BiDi connection. |
| **Python sync** | Python sync API | No-await API like Playwright's `sync_playwright`. The default for most Python users. |
| **MCP** | MCP Server | Exposes Vibium commands as MCP tools for LLM agents. Callback-based APIs (events, routing) marked N/A. |
| **CLI** | CLI / Agent Skill | Command-line interface and agent-callable skill. Stateless per-invocation. Event-based APIs marked N/A. |

**N/A rules:** Callback/event-based APIs (`onDialog`, `onConsole`, `route`, etc.) are N/A for MCP and CLI since those surfaces are request/response, not long-lived. Low-level primitives (`mouse.down`, `keyboard.up`) are also N/A for MCP/CLI — agents use higher-level commands.

### Part 2: Additional Language Bindings

After the core six targets ship, extend to:

| Language | Notes |
|----------|-------|
| **Java** | Largest existing Selenium user base. Critical for enterprise migration. |
| **C#** | .NET ecosystem. Second-largest Selenium user base. |
| **Ruby** | Strong in Rails/testing community. Selenium has solid Ruby bindings. |
| **Kotlin** | Growing Android/server-side adoption. JVM but distinct enough to warrant first-class support. |
| **Nim** | Lean, compiled, fast. Ideal for embedded/edge automation scenarios. |

These all wrap the same BiDi protocol layer — the per-language work is API surface and packaging, not protocol reimplementation. The JS async library IS the protocol layer; everything else is a client that speaks to it or wraps it.

---

## Implementation Tiers

### Tier 1 — Core (ship first)

Page object model (`browser.newPage()`, `page.close()`), navigation (`go`, `back`, `forward`, `reload`, `url`, `title`), element finding (`find`, `findAll`, CSS/semantic selectors), basic interaction (`click`, `fill`, `type`, `press`), element state (`text`, `html`, `value`, `attr`, `isVisible`), evaluation (`eval`), screenshots, basic waiting (`waitFor`, `waitForLoad`).

~13 commands. Enough to be useful.

### Tier 2 — Multi-page & Interaction

Keyboard/mouse APIs, advanced interaction (`check`, `uncheck`, `selectOption`, `hover`, `dragTo`), frames, dialogs, cookies/storage, console/error events, `text=`/`role=`/`label=` selector engines.

~9 command groups. Full single-page automation.

### Tier 3 — Network & Advanced

Network interception (`route`, `fulfill`, `abort`), `addInitScript`, accessibility tree, emulation (`setViewport`, `emulateMedia`), downloads, `waitForFunction`, `waitForEvent`.

~8 command groups. Feature parity with Playwright's power-user features.

### Tier 4 — Extras

Clock mocking, tracing, storage state serialization, touch input, `evalHandle`, `expose`.

~5 command groups. Nice-to-have, not blockers.

### AI-Native (parallel track)

`page.check()` and `page.do()` can ship at any tier — they sit on TOP of the deterministic API. Could soft-launch with Tier 1 (check uses screenshot + LLM, do uses find/click/fill).

---

## Key Design Decisions

1. **Types are formal, variables are fun.** The API uses `Browser`, `Context`, `Page` — standard, unsurprising, self-documenting. But the convention in examples is `bro` and `vibe` — short, memorable, and distinctly Vibium. Agents and IDEs see `browser.newPage()` → `Page`. Humans see `const vibe = await bro.newPage()`.

2. **Three levels exist, most users see two.** `Browser` → `Context` → `Page`. But `browser.newPage()` skips the context layer by using a default context internally. Only call `browser.newContext()` when you need isolation (multi-user, test-per-context).

3. **One find, two signatures.** `find('.css')` for CSS (terse, 80% of cases). `find({role: 'button', text: 'Submit'})` for semantic (autocomplete, type-safe, combinable). In Python: `find(role='button', text='Submit')`. Playwright needs 8 separate methods and chaining to do what Vibium does with one method and two signatures.

4. **Events via `.on*()` methods.** `page.onDialog(fn)` not `page.on('dialog', fn)`. More discoverable, better autocomplete.

5. **`findAll()` returns immediately.** Empty array if nothing matches. Use `waitFor()` if you need to wait.

6. **Frames get full Page API.** In BiDi, frames ARE browsing contexts. `page.frame('name')` returns an object with the same interface as a page.

7. **AI methods are first-class.** `page.check()` and `page.do()` aren't afterthoughts — they're the reason Vibium exists. They use the deterministic API under the hood.
