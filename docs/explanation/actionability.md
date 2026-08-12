# Actionability: How Vibium Waits for Elements

When you tell Vibium to click a button, you expect it to actually click the button. But web pages are dynamic—elements load asynchronously, animations play, overlays appear and disappear. A naive automation tool might try to click before the button exists, or click coordinates where a loading spinner is covering the target.

Vibium solves this with **actionability checks**: a set of conditions that must all be true before an action is performed. This concept comes from [Playwright](https://playwright.dev/docs/actionability), and Vibium implements it server-side in the Go binary so that client libraries don't need to worry about timing issues.

## The Five Checks

Before performing an action, Vibium verifies a subset of these conditions:

| Check | What it means | Why it matters |
|-------|--------------|----------------|
| **Visible** | Element has non-zero size and isn't hidden by CSS (`display: none`, `visibility: hidden`) | Can't interact with invisible things |
| **Stable** | Element's bounding box hasn't changed over 50ms | Clicking a moving target misses |
| **ReceivesEvents** | `elementFromPoint()` at the element's **in-view center** returns the element itself (or a descendant) | Clicks would go to the covering element instead |
| **Enabled** | Element isn't `disabled`, `aria-disabled="true"`, or inside a disabled `<fieldset>` | Disabled controls don't respond to input |
| **Editable** | Element accepts text input (text-type `<input>`, `<textarea>`, or `contentEditable`), and isn't `readOnly` or `aria-readonly` | Only checked for fill actions |

### ReceivesEvents and the in-view center point

The probe point is the WebDriver **in-view center point**: the center of the
element's bounding rect *clipped to the viewport*, floored to whole pixels.

Clipping matters for anything larger than the window. `document.elementFromPoint()`
returns `null` outside the viewport, so probing the raw bounding-rect center made
every element more than twice the viewport tall or wide look obscured — a
884 × 5314 scrim in a 901 × 981 viewport was probed at y ≈ 2657, off-screen, and
blamed on an overlay that did not exist (#340). Scrolling could not help: an
element taller than the viewport is always already in view. The flooring keeps
the check and the action on the same pixel, since BiDi input coordinates are
integers — the script reports the point it probed and the action dispatches there
(`(*ElementInfo).InputPoint()`). Only checks that include ReceivesEvents report a
point; the others still use the bounding-box center.

Failures say which of three things happened, rather than calling everything
obscured: `element has no visible area in the viewport` (nothing on screen),
`no element at in-view center (x, y)` (nothing painted there), or
`element is obscured by <div#overlay> at in-view center (x, y)`.

### ReceivesEvents and shadow DOM

`document.elementFromPoint()` returns the shadow **host**, not what is rendered
inside it. Taken literally that makes every element in a shadow root look
covered by its own component, so nothing inside a web component could be
clicked.

The check therefore descends: after the first hit, it calls
`elementFromPoint()` on that element's shadow root at the same coordinates, and
repeats until the answer stops changing. What comes back is the element actually
on top, wherever it lives.

This only affects the hit test. Everything else about the check is unchanged —
an element genuinely behind an overlay still fails, shadow root or not.

## Which Actions Run Which Checks

Different actions require different check sets. These are defined as Go slices in `actionability.go`:

| Check set | Checks | Used by |
|-----------|--------|---------|
| **ClickChecks** | Visible + Stable + ReceivesEvents + Enabled | `Click`, `DblClick`, `Tap`, `Check`, `Uncheck`, `TypeInto`, `PressOn` |
| **HoverChecks** | Visible + Stable + ReceivesEvents | `Hover`, `DragTo` (both source and target) |
| **FillChecks** | Visible + Enabled + Editable | `Fill` (clear uses Fill with empty string) |
| **SelectChecks** | Visible + Enabled | `SelectOption` |
| **ScrollChecks** | Stable | `ScrollIntoView` |

`TypeInto` and `PressOn` use `ClickChecks` (they click to focus first, then type). `Fill` uses its own set because it sets the value via JavaScript rather than simulating keystrokes—it doesn't need the element to receive pointer events, but it does need the element to be editable.

## How It Works

### Single-Script Architecture

Vibium runs all JS-side checks in **one BiDi round-trip** per poll attempt. The function `buildActionableScript()` produces a single JavaScript function with boolean flags controlling which checks to run:

```go
// actionability.go — check set definitions
var (
    ClickChecks  = []ActionCheck{CheckVisible, CheckStable, CheckReceivesEvents, CheckEnabled}
    HoverChecks  = []ActionCheck{CheckVisible, CheckStable, CheckReceivesEvents}
    FillChecks   = []ActionCheck{CheckVisible, CheckEnabled, CheckEditable}
    SelectChecks = []ActionCheck{CheckVisible, CheckEnabled}
    ScrollChecks = []ActionCheck{CheckStable}
)
```

The generated JS function receives flags like `chkVisible`, `chkEvents`, `chkEnabled`, `chkEditable` and conditionally runs each check inline. The core check body looks like this:

```javascript
// From actionabilityCheckBody() — shared across CSS and semantic scripts
if (chkVisible) {
    if (rect.width === 0 || rect.height === 0)
        return JSON.stringify({status:'failed', check:'visible', reason:'zero size'});
    const style = window.getComputedStyle(el);
    if (style.visibility === 'hidden')
        return JSON.stringify({status:'failed', check:'visible', reason:'visibility: hidden'});
    if (style.display === 'none')
        return JSON.stringify({status:'failed', check:'visible', reason:'display: none'});
}
if (chkEnabled) {
    if (el.disabled === true)
        return JSON.stringify({status:'failed', check:'enabled', reason:'disabled attribute'});
    if (el.getAttribute('aria-disabled') === 'true')
        return JSON.stringify({status:'failed', check:'enabled', reason:'aria-disabled'});
    const fs = el.closest('fieldset[disabled]');
    if (fs) {
        const legend = fs.querySelector('legend');
        if (!legend || !legend.contains(el))
            return JSON.stringify({status:'failed', check:'enabled', reason:'inside disabled fieldset'});
    }
}
if (chkEditable) { /* readOnly, aria-readonly, input type checks */ }
if (chkEvents) {
    // inViewCenter() clips the rect to the viewport and floors the result;
    // null means no part of the element is on screen.
    if (!inView)
        return JSON.stringify({status:'failed', check:'receivesEvents', reason:'element has no visible area in the viewport'});
    const cx = inView.x, cy = inView.y;
    let hit = document.elementFromPoint(cx, cy);   // then descends shadow roots
    if (!hit)
        return JSON.stringify({status:'failed', check:'receivesEvents', reason:'no element' + at});
    if (el !== hit && !el.contains(hit)) {
        const blocker = hit.tagName.toLowerCase() + (hit.id ? '#' + hit.id : '');
        return JSON.stringify({status:'failed', check:'receivesEvents', reason:'element is obscured by <' + blocker + '>' + at});
    }
}
```

If all checks pass, the script returns `{status: "ok"}` along with the element's
tag, text, bounding box, and — when ReceivesEvents ran — the in-view `point` it
probed, which is the coordinate the action then dispatches at.

### Stability: Go-Side Check

Stability is the one check that *doesn't* run in JavaScript. Because it requires a time delay (comparing two bounding boxes 50ms apart), running it in JS would require `awaitPromise: true` and a `setTimeout`. Instead, `WaitForActionable()` handles it on the Go side:

1. Run the actionability script (all JS-side checks pass, returns bbox)
2. Sleep 50ms
3. Run the same script again
4. Compare bounding boxes — if they match, the element is stable

### The Polling Loop

`WaitForActionable()` polls until all checks pass or the timeout is reached:

```
deadline = now + timeout (default 30s)

loop:
    run actionability script
    if failed or not found:
        if past deadline: return TimeoutError
        sleep 100ms
        continue

    if stability check needed:
        sleep 50ms
        run script again
        if bboxes differ:
            if past deadline: return TimeoutError
            sleep 100ms
            continue

    return element info (tag, text, box)
```

This means your code doesn't need retry logic. When you write:

```javascript
await page.find('#submit').click();
```

Vibium will automatically wait up to 30 seconds for the element to become clickable. You can customize the timeout per-action:

```javascript
await page.find('#submit').click({ timeout: 5000 }); // 5 seconds
```

## Element Finding

Before actionability checks can run, Vibium needs to locate the element. It supports two strategies.

### CSS Selectors

The simplest case — pass a CSS selector string:

```javascript
await page.find('button.submit').click();
await page.find('#login-form input[type="email"]').fill('user@example.com');
```

Internally this uses `document.querySelector()` (or `querySelectorAll()` with an `index` param).

### Semantic Selectors

Vibium also supports Playwright-style semantic selectors that match elements by their accessible role, text content, labels, and other attributes:

| Parameter | Matches against |
|-----------|----------------|
| `role` | ARIA role (explicit or implicit from tag, e.g. `<button>` → `"button"`) |
| `text` | `textContent` (substring match) |
| `label` | `aria-label`, `aria-labelledby`, or associated `<label>` element |
| `placeholder` | `placeholder` attribute |
| `alt` | `alt` attribute |
| `title` | `title` attribute |
| `testid` | `data-testid` attribute |
| `xpath` | XPath expression |

Semantic selectors can be combined with each other and with a CSS selector for precise targeting:

```javascript
await page.find({ role: 'button', text: 'Submit' }).click();
await page.find({ label: 'Email' }).fill('user@example.com');
await page.find({ selector: 'nav', role: 'link', text: 'Home' }).click();
```

### The `pickBest()` Heuristic

When a `text` param is provided and multiple elements match, Vibium picks the element with the **shortest `textContent`**. This prefers a `<button>Submit</button>` over a `<div>` that happens to contain "Submit" buried in a paragraph. If an explicit `index` param is provided, it uses that instead of the heuristic.

### Scope

The `scope` parameter restricts element finding to descendants of a container element (matched by CSS selector). This is useful for pages with repeated structures like card grids or table rows.

## Scroll Into View

Before running any checks, the actionability script automatically scrolls the element into the viewport:

```javascript
if (el.scrollIntoViewIfNeeded) {
    el.scrollIntoViewIfNeeded(true);
} else {
    el.scrollIntoView({ block: 'center', inline: 'nearest' });
}
```

This uses Chrome's non-standard `scrollIntoViewIfNeeded` when available (which only scrolls if the element isn't already visible), falling back to `scrollIntoView`. This happens before visibility and hit-testing checks, ensuring that off-screen elements get a chance to pass.

## Why Server-Side?

Vibium implements actionability in Go rather than in client libraries because:

1. **Single implementation**: The logic is written once, not duplicated across JavaScript, Python, and future clients.
2. **Reduced latency**: Polling happens over a local WebSocket (proxy to browser), not client → proxy → browser round trips.
3. **Simpler clients**: Client libraries just send a command and wait for success/error.
4. **Consistent behavior**: All clients get identical timing behavior.

The client code becomes trivial — send a command, get back success or a timeout error. All the complexity lives in the vibium binary where it can be tested once and shared everywhere.

## Code References

The actionability implementation lives in `clicker/internal/api/`:

| File | What's there |
|------|-------------|
| `actionability.go` | Check definitions, `buildActionableScript()`, `actionabilityCheckBody()`, `WaitForActionable()`, `resolveWithActionability()` |
| `handlers_interaction.go` | Exported action functions (`Click`, `Hover`, `Fill`, `TypeInto`, `SelectOption`, `DragTo`, `Tap`, `ScrollIntoView`, etc.) and their API command handlers |
| `handlers_elements.go` | Element finding (`buildFindScript`, `semanticMatchesHelper`, `pickBest`, CSS and semantic find scripts) |
| `helpers.go` | `ElementParams` struct, `ExtractElementParams()` |
| `router.go` | `DefaultTimeout` (30s), command routing |
