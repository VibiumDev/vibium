# Actionability: How Vibium Waits for Elements

When you tell Vibium to click a button, you expect it to actually click the button. But web pages are dynamic—elements load asynchronously, animations play, overlays appear and disappear. A naive automation tool might try to click before the button exists, or click coordinates where a loading spinner is covering the target.

Vibium solves this with **actionability checks**: a set of conditions that must all be true before an action is performed. This concept comes from Playwright, and Vibium implements it server-side in the Go vibium binary so that client libraries don't need to worry about timing issues.

## The Five Checks

Before performing an action, Vibium verifies these conditions:

| Check | What it means | Why it matters |
|-------|--------------|----------------|
| **Visible** | Element has size and isn't hidden | Can't click invisible things |
| **Stable** | Element isn't moving | Clicking a moving target misses |
| **ReceivesEvents** | Element isn't covered | Clicks go to the covering element |
| **Enabled** | Element isn't disabled | Disabled buttons don't respond |
| **Editable** | Element accepts text input | Only checked for `type()` |

Different actions require different checks:

- **Click**: Visible + Stable + ReceivesEvents + Enabled
- **Type**: Visible + Stable + ReceivesEvents + Enabled + Editable

## How Each Check Works

Every check runs JavaScript in the browser via WebDriver BiDi's `script.callFunction`. Here's the actual code.

### Visible

An element is visible if it has non-zero dimensions and isn't hidden by CSS:

```javascript
(selector) => {
  const el = document.querySelector(selector);
  if (!el) return JSON.stringify({ error: 'not found' });

  const rect = el.getBoundingClientRect();
  if (rect.width === 0 || rect.height === 0) {
    return JSON.stringify({ visible: false, reason: 'zero size' });
  }

  const style = window.getComputedStyle(el);
  if (style.visibility === 'hidden') {
    return JSON.stringify({ visible: false, reason: 'visibility hidden' });
  }
  if (style.display === 'none') {
    return JSON.stringify({ visible: false, reason: 'display none' });
  }

  return JSON.stringify({ visible: true });
}
```
<sub>[`actionability.go#L26`](https://github.com/VibiumDev/vibium/blob/66b5bc3/clicker/internal/features/actionability.go#L26)</sub>

### Stable

An element is stable if its position hasn't changed over 50ms. This catches CSS animations and transitions:

```javascript
// At time T:
(selector) => {
  const el = document.querySelector(selector);
  const rect = el.getBoundingClientRect();
  return JSON.stringify({
    x: rect.x, y: rect.y,
    width: rect.width, height: rect.height
  });
}

// Wait 50ms, then check again
// Stable = (position at T) equals (position at T+50ms)
```
<sub>[`CheckStable#L70`](https://github.com/VibiumDev/vibium/blob/66b5bc3/clicker/internal/features/actionability.go#L70) · [`getBoundingBox#L366`](https://github.com/VibiumDev/vibium/blob/66b5bc3/clicker/internal/features/actionability.go#L366)</sub>

The Go code runs `getBoundingClientRect()` twice with a 50ms gap and compares the results.

### ReceivesEvents

This is the most subtle check. An element might be visible but covered by another element (a modal, tooltip, or sticky header). Vibium uses `elementFromPoint()` at the element's center:

```javascript
(selector) => {
  const el = document.querySelector(selector);
  if (!el) return JSON.stringify({ error: 'not found' });

  const rect = el.getBoundingClientRect();
  const centerX = rect.x + rect.width / 2;
  const centerY = rect.y + rect.height / 2;

  // What element is actually at this point?
  const hitTarget = document.elementFromPoint(centerX, centerY);
  if (!hitTarget) {
    return JSON.stringify({ receivesEvents: false, reason: 'no element at point' });
  }

  // Is it our element, or a child of our element?
  if (el === hitTarget || el.contains(hitTarget)) {
    return JSON.stringify({ receivesEvents: true });
  }

  // Something else is covering it
  return JSON.stringify({
    receivesEvents: false,
    reason: 'obscured by ' + hitTarget.tagName.toLowerCase()
  });
}
```
<sub>[`actionability.go#L98`](https://github.com/VibiumDev/vibium/blob/66b5bc3/clicker/internal/features/actionability.go#L98)</sub>

### Enabled

An element is disabled if it has the `disabled` attribute, `aria-disabled="true"`, or is inside a disabled `<fieldset>`:

```javascript
(selector) => {
  const el = document.querySelector(selector);
  if (!el) return JSON.stringify({ error: 'not found' });

  if (el.disabled === true) {
    return JSON.stringify({ enabled: false, reason: 'disabled attribute' });
  }

  if (el.getAttribute('aria-disabled') === 'true') {
    return JSON.stringify({ enabled: false, reason: 'aria-disabled' });
  }

  // Check if inside disabled fieldset
  const fieldset = el.closest('fieldset[disabled]');
  if (fieldset) {
    // Exception: elements in the first legend are not disabled
    const legend = fieldset.querySelector('legend');
    if (!legend || !legend.contains(el)) {
      return JSON.stringify({ enabled: false, reason: 'inside disabled fieldset' });
    }
  }

  return JSON.stringify({ enabled: true });
}
```
<sub>[`actionability.go#L152`](https://github.com/VibiumDev/vibium/blob/66b5bc3/clicker/internal/features/actionability.go#L152)</sub>

### Editable

For typing, the element must also accept text input:

```javascript
(selector) => {
  const el = document.querySelector(selector);
  if (!el) return JSON.stringify({ error: 'not found' });

  if (el.readOnly === true) {
    return JSON.stringify({ editable: false, reason: 'readonly attribute' });
  }

  if (el.getAttribute('aria-readonly') === 'true') {
    return JSON.stringify({ editable: false, reason: 'aria-readonly' });
  }

  // For <input>, check if it's a type that accepts text
  const tag = el.tagName.toLowerCase();
  if (tag === 'input') {
    const type = (el.type || 'text').toLowerCase();
    const textTypes = ['text', 'password', 'email', 'number', 'search', 'tel', 'url'];
    if (!textTypes.includes(type)) {
      return JSON.stringify({ editable: false, reason: 'input type ' + type + ' not editable' });
    }
  }

  if (el.isContentEditable) {
    return JSON.stringify({ editable: true });
  }

  if (tag === 'input' || tag === 'textarea') {
    return JSON.stringify({ editable: true });
  }

  return JSON.stringify({ editable: false, reason: 'not a form element or contenteditable' });
}
```
<sub>[`actionability.go#L217`](https://github.com/VibiumDev/vibium/blob/66b5bc3/clicker/internal/features/actionability.go#L217)</sub>

## The Autowait Loop

Vibium doesn't just check once—it polls repeatedly until all checks pass or timeout is reached:

```
deadline = now + timeout (default 30s)

loop:
    for each check in required_checks:
        if check fails:
            if now > deadline:
                return TimeoutError
            sleep 100ms
            continue loop

    // All checks passed
    perform action
```
<sub>[`autowait.go#L126`](https://github.com/VibiumDev/vibium/blob/66b5bc3/clicker/internal/features/autowait.go#L126)</sub>

This means your code doesn't need retry logic. When you write:

```javascript
await element.click();
```

Vibium will automatically wait up to 30 seconds for the element to become clickable. You can customize this per-action:

```javascript
await element.click({ timeout: 5000 }); // 5 seconds
```

## WebDriver BiDi Extension Commands

The WebDriver BiDi specification allows implementations to define extension modules. From the spec:

> An implementation may define extension modules. These must have a module name that contains a single colon ":" character.

Vibium defines three extension commands:

| Command | Parameters | Description |
|---------|------------|-------------|
| `vibium:find` | `context`, `selector`, `timeout` | Wait for element to exist |
| `vibium:click` | `context`, `selector`, `timeout` | Wait for actionable, then click |
| `vibium:type` | `context`, `selector`, `text`, `timeout` | Wait for actionable, then type |

These commands are handled by the vibium proxy, not forwarded to the browser. When the proxy receives a `vibium:click` command:

1. Parse selector and timeout from params
2. Poll until element exists (`vibium:find` behavior)
3. Poll until all click checks pass
4. Get element's bounding box
5. Calculate center coordinates
6. Send `input.performActions` to browser with pointer move + click

The response follows standard BiDi format:

```json
{
  "id": 1,
  "type": "success",
  "result": { "clicked": true }
}
```

Or on timeout:

```json
{
  "id": 1,
  "type": "error",
  "error": {
    "error": "timeout",
    "message": "timeout after 30s waiting for 'button.submit': check 'ReceivesEvents' failed"
  }
}
```

## Code References

If you want to implement your own extension commands, here's where to look:

**Client side** (sending the command):
- [`clients/javascript/src/vibe.ts#L69`](https://github.com/VibiumDev/vibium/blob/66b5bc3/clients/javascript/src/vibe.ts#L69) — calls `client.send('vibium:find', { ... })`

**Server side** (handling the command):
- [`clicker/internal/proxy/router.go#L150`](https://github.com/VibiumDev/vibium/blob/66b5bc3/clicker/internal/proxy/router.go#L150) — switch case routes `vibium:find` to handler
- [`clicker/internal/proxy/router.go#L303`](https://github.com/VibiumDev/vibium/blob/66b5bc3/clicker/internal/proxy/router.go#L303) — `handleVibiumFind()` implements the logic

The pattern is:
1. Client sends a JSON message with `method: "yourprefix:commandname"`
2. Router's `OnClientMessage` parses and dispatches to your handler
3. Handler does work (possibly sending internal BiDi commands to browser)
4. Handler calls `sendSuccess()` or `sendError()` to respond to client

## Why Server-Side?

Vibium implements actionability in Go rather than in client libraries because:

1. **Single implementation**: The logic is written once, not duplicated across JavaScript, Python, Ruby, etc.
2. **Reduced latency**: Polling happens over a local WebSocket, not client→proxy→browser round trips.
3. **Simpler clients**: Client libraries just send a command and wait for success/error.
4. **Consistent behavior**: All clients get identical timing behavior.

The client code becomes trivial:

```javascript
async click(options?: { timeout?: number }): Promise<void> {
  await this.client.send('vibium:click', {
    context: this.context,
    selector: this.selector,
    timeout: options?.timeout,
  });
}
```

All the complexity lives in the vibium binary where it can be tested once and shared everywhere.
