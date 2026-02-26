# Doc #2: Vibium WebDriver Spec Parity

**Tracker:** arewebidiyet.md
**Goal:** Track coverage of the W3C WebDriver specification — both Classic and BiDi.
**Depends on:** Doc #1 (Playwright parity) — most Tier 1 work overlaps.

---

## Relationship to Doc #1

The WebDriver spec and Playwright's API cover much of the same ground. The key difference is framing:

- **Doc #1** asks: *Can a Playwright user do everything they're used to?* (DX-first)
- **Doc #2** asks: *Does Vibium implement the W3C WebDriver commands?* (Spec-first)

In practice, building Doc #1's Tier 1 and Tier 2 gets us ~80% of WebDriver spec coverage. This document tracks the remaining gaps and maps WebDriver command names to Vibium's API.

---

## Object Model Mapping

| WebDriver Concept | Vibium | Notes |
|-------------------|--------|-------|
| Session | `browser` (Browser) | `browser.launch()` creates browser process |
| User Context | `context` (Context) | `browser.newContext()` — isolated cookies/storage |
| Browsing Context | `page` (Page) | `browser.newPage()` or `context.newPage()` |
| Element | `Element` | `page.find()` returns Element |
| Window / Tab | `page` | No switching — pages are addressable |

```javascript
import { browser } from 'vibium'

const bro = await browser.launch()     // ≈ New Session (browser process)
const ctx = await bro.newContext()     // ≈ User Context (isolated state)
const vibe = await ctx.newPage()      // ≈ browsing context
await vibe.go('https://example.com')   // ≈ Navigate To
await vibe.find('#btn').click()        // ≈ Find Element + Element Click
await bro.close()                      // ≈ Delete Session

// Shorthand — most users skip the context layer
const bro = await browser.launch()
const vibe = await bro.newPage()       // default context, new page
```

---

## WebDriver Classic Commands

### Session

| WebDriver Command | Vibium API | Status | Notes |
|-------------------|-----------|--------|-------|
| New Session | `browser.launch(caps?)` | ⬜ | BiDi: session.new |
| Delete Session | `browser.close()` | ⬜ | BiDi: session.end |
| Status | `browser.status()` | ⬜ | Server readiness check |
| Get Timeouts | `browser.timeouts()` | ⬜ | |
| Set Timeouts | `browser.setTimeouts(t)` | ⬜ | Implicit/page load/script |

### Navigation

| WebDriver Command | Vibium API | Status | Notes |
|-------------------|-----------|--------|-------|
| Navigate To | `page.go(url)` | ⬜ | BiDi: browsingContext.navigate |
| Get Current URL | `page.url()` | ⬜ | Client-side |
| Back | `page.back()` | ⬜ | BiDi: traverseHistory(-1) |
| Forward | `page.forward()` | ⬜ | BiDi: traverseHistory(1) |
| Refresh | `page.reload()` | ⬜ | BiDi: browsingContext.reload |
| Get Title | `page.title()` | ⬜ | JS: document.title |

### Window / Browsing Context

| WebDriver Command | Vibium API | Status | Notes |
|-------------------|-----------|--------|-------|
| Get Window Handle | `page.id` | ⬜ | Context ID |
| Get Window Handles | `browser.pages()` → map to IDs | ⬜ | browsingContext.getTree |
| Close Window | `page.close()` | ⬜ | browsingContext.close |
| Switch To Window | N/A | ⬜ | Not needed — pages addressable |
| New Window | `browser.newPage()` | ⬜ | browsingContext.create |
| Get Window Rect | `page.viewport()` | ⬜ | |
| Set Window Rect | `page.setViewport(size)` | ⬜ | browsingContext.setViewport |
| Maximize Window | `page.maximize()` | ⬜ | |
| Minimize Window | `page.minimize()` | ⬜ | |
| Fullscreen Window | `page.fullscreen()` | ⬜ | |

### Frame

| WebDriver Command | Vibium API | Status | Notes |
|-------------------|-----------|--------|-------|
| Switch To Frame | `page.frame(ref)` | ⬜ | Returns frame with Page API |
| Switch To Parent Frame | `frame.parent()` | ⬜ | Return parent context |
| Find Frame | `page.frames()` | ⬜ | browsingContext.getTree |

### Element Finding

| WebDriver Command | Vibium API | Status | Notes |
|-------------------|-----------|--------|-------|
| Find Element | `page.find(sel)` | ⬜ | locateNodes (single) |
| Find Elements | `page.findAll(sel)` | ⬜ | locateNodes (all) |
| Find Element From Element | `el.find(sel)` | ⬜ | Scoped locateNodes |
| Find Elements From Element | `el.findAll(sel)` | ⬜ | Scoped locateNodes |
| Get Active Element | `page.activeElement()` | ⬜ | JS: document.activeElement |

#### Locator Strategies

| WebDriver Strategy | Vibium Selector | Notes |
|-------------------|-----------------|-------|
| css selector | `find('.class')` | Default, no prefix needed |
| xpath | `find('xpath=//div')` | |
| link text | `find('text=Click Here')` | Exact match |
| partial link text | `find('text=Click')` | Partial match |
| tag name | `find('div')` | CSS covers this |

### Element Interaction

| WebDriver Command | Vibium API | Status | Notes |
|-------------------|-----------|--------|-------|
| Element Click | `el.click()` | ⬜ | input.performActions |
| Element Clear | `el.clear()` | ⬜ | JS |
| Element Send Keys | `el.type(text)` | ⬜ | input.performActions |

### Element State

| WebDriver Command | Vibium API | Status | Notes |
|-------------------|-----------|--------|-------|
| Is Element Selected | `el.isChecked()` | ⬜ | JS |
| Get Element Attribute | `el.attr(name)` | ⬜ | JS |
| Get Element Property | `el.eval(e => e[prop])` | ⬜ | script.callFunction |
| Get Element CSS Value | `el.eval(e => getComputedStyle(e)[p])` | ⬜ | JS |
| Get Element Text | `el.text()` | ⬜ | JS |
| Get Element Tag Name | `el.eval(e => e.tagName)` | ⬜ | JS |
| Get Element Rect | `el.bounds()` | ⬜ | JS |
| Is Element Enabled | `el.isEnabled()` | ⬜ | JS |
| Get Computed Role | `el.role()` | ⬜ | JS |
| Get Computed Label | `el.label()` | ⬜ | JS |

### Document

| WebDriver Command | Vibium API | Status | Notes |
|-------------------|-----------|--------|-------|
| Get Page Source | `page.eval(() => document.documentElement.outerHTML)` | ⬜ | script.evaluate |
| Execute Script | `page.eval(expr)` | ⬜ | script.evaluate |
| Execute Async Script | `page.eval(asyncExpr)` | ⬜ | script.evaluate with await |

### Cookies

| WebDriver Command | Vibium API | Status | Notes |
|-------------------|-----------|--------|-------|
| Get All Cookies | `context.cookies()` | ⬜ | storage.getCookies |
| Get Named Cookie | `context.cookies({name})` | ⬜ | storage.getCookies filtered |
| Add Cookie | `context.setCookies([c])` | ⬜ | storage.setCookie |
| Delete Cookie | `context.clearCookies({name})` | ⬜ | storage.deleteCookies |
| Delete All Cookies | `context.clearCookies()` | ⬜ | storage.deleteCookies |

### User Prompts (Alerts/Dialogs)

| WebDriver Command | Vibium API | Status | Notes |
|-------------------|-----------|--------|-------|
| Dismiss Alert | `dialog.dismiss()` | ⬜ | handleUserPrompt(accept:false) |
| Accept Alert | `dialog.accept()` | ⬜ | handleUserPrompt(accept:true) |
| Get Alert Text | `dialog.message()` | ⬜ | From event data |
| Send Alert Text | `dialog.accept(text)` | ⬜ | handleUserPrompt with text |

### Screen Capture

| WebDriver Command | Vibium API | Status | Notes |
|-------------------|-----------|--------|-------|
| Take Screenshot | `page.screenshot()` | ⬜ | captureScreenshot |
| Take Element Screenshot | `el.screenshot()` | ⬜ | captureScreenshot with clip |

### Print

| WebDriver Command | Vibium API | Status | Notes |
|-------------------|-----------|--------|-------|
| Print Page | `page.pdf()` | ⬜ | browsingContext.print |

### Actions

| WebDriver Command | Vibium API | Status | Notes |
|-------------------|-----------|--------|-------|
| Perform Actions | `page.keyboard.*`, `page.mouse.*` | ⬜ | input.performActions |
| Release Actions | (automatic) | ⬜ | input.releaseActions |

---

## WebDriver BiDi Modules

| BiDi Module | Vibium Coverage | Notes |
|-------------|----------------|-------|
| session | `browser.launch()`, `browser.close()` | Core |
| browser | `browser.newContext()` (createUserContext) | Core |
| browsingContext | Pages, navigation, frames, screenshots | Core |
| script | `page.eval()`, `expose()`, `addInitScript()` | Core |
| input | keyboard, mouse, touch APIs | Core |
| network | `page.route()`, events | Tier 3 |
| storage | `context.cookies()` | Tier 2 |
| log | `page.onConsole()`, `page.onError()` | Tier 2 |
| permissions | `page.grantPermissions()` | Tier 3 |

---

## Coverage Summary

| Category | Total Commands | Implemented | Coverage |
|----------|---------------|-------------|----------|
| Session | 5 | 0 | 0% |
| Navigation | 6 | 0 | 0% |
| Window | 9 | 0 | 0% |
| Frame | 3 | 0 | 0% |
| Element Finding | 5 | 0 | 0% |
| Element Interaction | 3 | 0 | 0% |
| Element State | 10 | 0 | 0% |
| Document | 3 | 0 | 0% |
| Cookies | 5 | 0 | 0% |
| User Prompts | 4 | 0 | 0% |
| Screen Capture | 2 | 0 | 0% |
| Print | 1 | 0 | 0% |
| Actions | 2 | 0 | 0% |
| **Total** | **58** | **0** | **0%** |

These will fill in as Doc #1 tiers ship.
