# Are We WebDriver Yet?

Vibium's coverage of the WebDriver spec (W3C + BiDi). Maps every WebDriver command to its Vibium equivalent.

**Legend:** ✅ Done · 🟡 Partial · ⬜ Not started

---

## Session (5 commands)

| WebDriver | Vibium | Status |
|-----------|--------|--------|
| New Session | `browser.launch(caps?)` | ⬜ |
| Delete Session | `browser.close()` | ⬜ |
| Status | `browser.status()` | ⬜ |
| Get Timeouts | `browser.timeouts()` | ⬜ |
| Set Timeouts | `browser.setTimeouts(t)` | ⬜ |

## Navigation (5 commands)

| WebDriver | Vibium | Status |
|-----------|--------|--------|
| Navigate To | `page.go(url)` | ⬜ |
| Get Current URL | `page.url()` | ⬜ |
| Back | `page.back()` | ⬜ |
| Forward | `page.forward()` | ⬜ |
| Refresh | `page.reload()` | ⬜ |
| Get Title | `page.title()` | ⬜ |

## Window / Context (10 commands)

| WebDriver | Vibium | Status |
|-----------|--------|--------|
| Get Window Handle | `page.id` | ⬜ |
| Get Window Handles | `browser.pages()` | ⬜ |
| Close Window | `page.close()` | ⬜ |
| Switch To Window | N/A (not needed) | ⬜ |
| New Window | `browser.newPage()` | ⬜ |
| Get Window Rect | `page.viewport()` | ⬜ |
| Set Window Rect | `page.setViewport()` | ⬜ |
| Maximize | `page.maximize()` | ⬜ |
| Minimize | `page.minimize()` | ⬜ |
| Fullscreen | `page.fullscreen()` | ⬜ |

## Frame (2 commands)

| WebDriver | Vibium | Status |
|-----------|--------|--------|
| Switch To Frame | `page.frame(ref)` | ⬜ |
| Switch To Parent | `frame.parent()` | ⬜ |

## Element Finding (5 commands)

| WebDriver | Vibium | Status |
|-----------|--------|--------|
| Find Element | `page.find(sel)` | ⬜ |
| Find Elements | `page.findAll(sel)` | ⬜ |
| Find From Element | `el.find(sel)` | ⬜ |
| Find All From Element | `el.findAll(sel)` | ⬜ |
| Get Active Element | `page.activeElement()` | ⬜ |

## Element Interaction (3 commands)

| WebDriver | Vibium | Status |
|-----------|--------|--------|
| Element Click | `el.click()` | ⬜ |
| Element Clear | `el.clear()` | ⬜ |
| Element Send Keys | `el.type(text)` | ⬜ |

## Element State (10 commands)

| WebDriver | Vibium | Status |
|-----------|--------|--------|
| Is Element Selected | `el.isChecked()` | ⬜ |
| Get Attribute | `el.attr(name)` | ⬜ |
| Get Property | `el.eval(fn)` | ⬜ |
| Get CSS Value | `el.eval(fn)` | ⬜ |
| Get Text | `el.text()` | ⬜ |
| Get Tag Name | `el.eval(fn)` | ⬜ |
| Get Rect | `el.bounds()` | ⬜ |
| Is Enabled | `el.isEnabled()` | ⬜ |
| Computed Role | `el.role()` | ⬜ |
| Computed Label | `el.label()` | ⬜ |

## Document (3 commands)

| WebDriver | Vibium | Status |
|-----------|--------|--------|
| Get Page Source | `page.eval(fn)` | ⬜ |
| Execute Script | `page.eval(expr)` | ⬜ |
| Execute Async Script | `page.eval(asyncExpr)` | ⬜ |

## Cookies (5 commands)

| WebDriver | Vibium | Status |
|-----------|--------|--------|
| Get All Cookies | `context.cookies()` | ⬜ |
| Get Named Cookie | `context.cookies({name})` | ⬜ |
| Add Cookie | `context.setCookies([c])` | ⬜ |
| Delete Cookie | `context.clearCookies({name})` | ⬜ |
| Delete All Cookies | `context.clearCookies()` | ⬜ |

## User Prompts (4 commands)

| WebDriver | Vibium | Status |
|-----------|--------|--------|
| Dismiss Alert | `dialog.dismiss()` | ⬜ |
| Accept Alert | `dialog.accept()` | ⬜ |
| Get Alert Text | `dialog.message()` | ⬜ |
| Send Alert Text | `dialog.accept(text)` | ⬜ |

## Screen Capture (2 commands)

| WebDriver | Vibium | Status |
|-----------|--------|--------|
| Take Screenshot | `page.screenshot()` | ⬜ |
| Element Screenshot | `el.screenshot()` | ⬜ |

## Print (1 command)

| WebDriver | Vibium | Status |
|-----------|--------|--------|
| Print Page | `page.pdf()` | ⬜ |

## Actions (2 commands)

| WebDriver | Vibium | Status |
|-----------|--------|--------|
| Perform Actions | `page.keyboard.* / page.mouse.*` | ⬜ |
| Release Actions | (automatic) | ⬜ |
