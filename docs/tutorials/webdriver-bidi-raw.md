# WebDriver BiDi: Raw Protocol Tutorial

Learn WebDriver BiDi by talking directly to Firefox with nothing but a terminal and websocat. No libraries, no abstractions — just raw JSON over WebSocket.

By the end, you'll understand exactly what Vibium does under the hood.

## Prerequisites

- Firefox (any recent version)
- [websocat](https://github.com/vi/websocat) — WebSocket CLI tool
- A terminal

### Installing websocat

**macOS:**
```bash
brew install websocat
```

**Linux:**
```bash
# Debian/Ubuntu
sudo apt install websocat

# Or via Cargo (any platform)
cargo install websocat
```

**Windows:**
```powershell
# Via Scoop
scoop install websocat

# Or download from GitHub releases
```

### Opening a Terminal

**macOS:** Press `Cmd+Space`, type "Terminal", press Enter

**Linux:** Press `Ctrl+Alt+T` or find Terminal in your applications

**Windows:** Press `Win+R`, type "cmd" or "powershell", press Enter

## Step 1: Launch Firefox with BiDi Enabled

Firefox has native WebDriver BiDi support. Launch it with remote debugging enabled.

**macOS:**
```bash
/Applications/Firefox.app/Contents/MacOS/firefox --remote-debugging-port=9222
```

**Linux:**
```bash
firefox --remote-debugging-port=9222
```

**Windows (Command Prompt):**
```cmd
"C:\Program Files\Mozilla Firefox\firefox.exe" --remote-debugging-port=9222
```

**Windows (PowerShell):**
```powershell
& "C:\Program Files\Mozilla Firefox\firefox.exe" --remote-debugging-port=9222
```

Firefox starts and listens for BiDi connections on port 9222. Leave this terminal open.

## Step 2: Connect with websocat

In a new terminal, connect to Firefox:

```bash
websocat ws://localhost:9222/session
```

You're now talking directly to the browser over WebSocket. Every message you send and receive is JSON.

## Step 3: Start a Session

Send the `session.new` command to create a BiDi session:

```json
{"id": 1, "method": "session.new", "params": {"capabilities": {}}}
```

Firefox responds with session info:

```json
{
  "type": "success",
  "id": 1,
  "result": {
    "sessionId": "...",
    "capabilities": { ... }
  }
}
```

## Step 4: Get the Browsing Context

Firefox usually opens with a blank tab. Get its context ID:

```json
{"id": 2, "method": "browsingContext.getTree", "params": {}}
```

Response:

```json
{
  "type": "success",
  "id": 2,
  "result": {
    "contexts": [
      { "context": "abc123", "url": "about:blank", ... }
    ]
  }
}
```

Note the `"context"` value — you'll need it for all subsequent commands.

## Step 5: Navigate to a Page

Go to example.com (replace `abc123` with your actual context ID):

```json
{"id": 3, "method": "browsingContext.navigate", "params": {"context": "abc123", "url": "https://example.com", "wait": "complete"}}
```

Response confirms navigation:

```json
{
  "type": "success",
  "id": 3,
  "result": {
    "navigation": "...",
    "url": "https://example.com/"
  }
}
```

## Step 6: Get the Document Title

Use `script.evaluate` to run JavaScript in the page:

```json
{"id": 4, "method": "script.evaluate", "params": {"expression": "document.title", "target": {"context": "abc123"}, "awaitPromise": false}}
```

Response:

```json
{
  "type": "success",
  "id": 4,
  "result": {
    "type": "success",
    "realm": "...",
    "result": { "type": "string", "value": "Example Domain" }
  }
}
```

> **What's `realm`?** A realm is BiDi's term for a JavaScript execution environment — roughly one per page or iframe. The browser returns which realm the script ran in so you can track results across multiple contexts.

## Step 7: Find and Click a Link

First, get the link's position. Note: `getBoundingClientRect()` returns a `DOMRect` whose properties live on the prototype, so BiDi won't serialize them directly. Extract them into a plain object:

```json
{"id": 5, "method": "script.evaluate", "params": {"expression": "(() => { const r = document.querySelector('a').getBoundingClientRect(); return { x: r.x, y: r.y, width: r.width, height: r.height }; })()", "target": {"context": "abc123"}, "awaitPromise": false}}
```

Response gives you the bounding box:

```json
{
  "type": "success",
  "id": 5,
  "result": {
    "type": "success",
    "realm": "...",
    "result": {
      "type": "object",
      "value": [
        ["x", { "type": "number", "value": 216 }],
        ["y", { "type": "number", "value": 203 }],
        ["width", { "type": "number", "value": 82 }],
        ["height", { "type": "number", "value": 20 }]
      ]
    }
  }
}
```

Now click at the center of that element using `input.performActions` (calculate center from the coordinates above):

```json
{"id": 6, "method": "input.performActions", "params": {"context": "abc123", "actions": [{"type": "pointer", "id": "mouse", "actions": [{"type": "pointerMove", "x": 257, "y": 213}, {"type": "pointerDown", "button": 0}, {"type": "pointerUp", "button": 0}]}]}}
```

The browser clicks and navigates to the link target.

## Step 8: Close the Browser

End the session:

```json
{"id": 7, "method": "browser.close", "params": {}}
```

Firefox closes.

---

## What You Just Did

You automated a browser using nothing but:
- WebSocket connection
- JSON messages
- The WebDriver BiDi protocol

This is how browser automation works at the protocol level — JSON commands over WebSocket. Vibium is built on BiDi from the start. Selenium has adopted it too. Playwright is investigating BiDi but still relies on CDP and custom protocols.

| Browser | Vibium | Playwright | Selenium |
|---------|--------|------------|----------|
| Chrome | WebDriver BiDi | CDP | WebDriver BiDi |
| Firefox | Planned | Custom protocol (patched browser) | WebDriver BiDi |
| Safari | — | Custom protocol (patched WebKit) | WebDriver (classic) |

Vibium currently supports Chrome via BiDi. Firefox has native BiDi support (as you just saw in this tutorial) — Vibium support is planned. Safari BiDi is still in development by Apple.

Playwright requires patched browser builds maintained by Microsoft. Vibium uses stock browsers from each vendor via the W3C standard protocol.

---

## What Vibium Adds

Raw BiDi is powerful but tedious. Here's what Vibium handles for you:

### 1. Connection Management

You had to manually:
- Launch Firefox with the right flags
- Connect to the WebSocket
- Track context IDs

Vibium does this automatically. `browser.launch()` handles everything.

### 2. Actionability Checks

When you clicked, you:
- Found the element's bounding box
- Calculated the center coordinates
- Sent pointer actions

But what if the element wasn't visible? Or was covered by an overlay? Or still animating?

Vibium waits for elements to be **actionable** before interacting:

| Check | What it means |
|-------|---------------|
| **Visible** | Has non-zero size, not `visibility: hidden` |
| **Stable** | Same position for 2 consecutive checks (not animating) |
| **Receives Events** | Not covered by another element |
| **Enabled** | Not `disabled` or `aria-disabled` |
| **Editable** | For typing: not `readonly` (in addition to above) |

When you call `element.click()`, Vibium polls these checks until they all pass (or timeout). You never send a click to a hidden button.

### 3. Auto-Waiting

You had to manually wait for navigation to complete (`"wait": "complete"`).

Vibium's `find()` automatically waits for elements to appear. No explicit waits, no sleep statements.

### 4. Error Messages

Raw BiDi errors are cryptic:

```json
{"id": 5, "error": "no such element", "message": "Unable to locate element"}
```

Vibium gives you context:

```
TimeoutError: Element not found: button.submit
  Waited 30s for element to appear
  Page: https://example.com/checkout
```

---

## Summary

| Layer | What it does |
|-------|--------------|
| **WebDriver BiDi** | The protocol: JSON over WebSocket |
| **Browser** | Executes commands, returns results |
| **Vibium** | Connection management, actionability, auto-waiting, errors |
| **Your code** | `vibe.find('a').click()` |

The raw protocol is simple. Everything else is saving you from writing it by hand.

---

## Further Reading

- [WebDriver BiDi Explainer](../explanation/webdriver-bidi.md) — history and why it matters
- [WebDriver BiDi Spec](https://w3c.github.io/webdriver-bidi/) — the full specification
