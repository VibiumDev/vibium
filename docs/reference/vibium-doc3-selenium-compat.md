# Doc #3: Vibium Selenium Compatibility Layer

**Future site:** areweseleniumyet.com
**Goal:** Drop-in Selenium WebDriver replacement backed by Vibium's BiDi engine.
**Depends on:** Doc #1 (Playwright parity, Tier 1 minimum) and Doc #2 (WebDriver spec coverage).

---

## What This Is

A shim that exposes the Selenium WebDriver API but routes everything through Vibium's BiDi connection. Selenium users can migrate by changing one import:

```python
# Before
from selenium import webdriver
driver = webdriver.Chrome()

# After
from vibium.compat.selenium import webdriver
driver = webdriver.Chrome()  # ← no chromedriver binary needed
```

Everything else stays the same. Existing Selenium scripts work unchanged.

---

## What This Is NOT

- Not a new API. Vibium's native API is `bro`/`vibe` (Doc #1).
- Not the recommended way to use Vibium for new projects.
- Not 100% bug-for-bug compatible — edge cases around implicit waits and stale element refs will differ.

This is a **migration bridge**. Use it to get running, then incrementally adopt `bro`/`vibe` for new code.

---

## Architecture

```
┌──────────────────────────────┐
│  Existing Selenium Script    │
│  driver.find_element(...)    │
│  driver.get(url)             │
└──────────┬───────────────────┘
           │  same API
┌──────────▼───────────────────┐
│  vibium.compat.selenium      │
│  Translates Selenium calls   │
│  to Vibium native API        │
└──────────┬───────────────────┘
           │  calls
┌──────────▼───────────────────┐
│  Vibium Core                 │
│  browser = browser process   │
│  context = isolated state    │
│  page = page                 │
└──────────┬───────────────────┘
           │  WebDriver BiDi
┌──────────▼───────────────────┐
│  Browser (Chrome, Firefox)   │
└──────────────────────────────┘
```

The compat layer is thin: it maps Selenium's `driver.method()` calls to Vibium's `page.method()` calls. No protocol translation — Vibium already speaks BiDi.

---

## Internal Mapping

The Selenium `driver` object wraps a Vibium `browser` + default context + default `page`:

```python
class WebDriver:
    def __init__(self):
        self._bro = browser.launch()             # browser process
        self._ctx = self._bro.new_context()      # default context (cookie jar)
        self._page = self._ctx.new_page()        # default page
        self._current_page = self._page           # for window switching

    def get(self, url):
        self._current_page.go(url)

    def find_element(self, by, value):
        selector = self._translate_by(by, value)
        return CompatElement(self._current_page.find(selector))

    def switch_to_window(self, handle):
        self._current_page = self._handle_map[handle]
```

### The Window Switching Problem

Selenium uses `driver.switch_to.window(handle)` to change which tab is active. Vibium doesn't need this — pages are independently addressable. The compat layer maintains a handle→page mapping and routes calls to the "current" page:

```python
class SwitchTo:
    def window(self, handle):
        self._driver._current_page = self._driver._handle_map[handle]

    def frame(self, ref):
        frame = self._driver._current_page.frame(ref)
        self._driver._current_page = frame  # frame has Page API

    def default_content(self):
        self._driver._current_page = self._driver._page
```

---

## Supported Selenium API Surface

### Priority 1 — Core (covers 90% of scripts)

| Selenium | Vibium Mapping | Complexity |
|----------|---------------|------------|
| `driver.get(url)` | `page.go(url)` | Trivial |
| `driver.current_url` | `page.url()` | Trivial |
| `driver.title` | `page.title()` | Trivial |
| `driver.back()` | `page.back()` | Trivial |
| `driver.forward()` | `page.forward()` | Trivial |
| `driver.refresh()` | `page.reload()` | Trivial |
| `driver.find_element(By.CSS, sel)` | `page.find(sel)` | Low |
| `driver.find_elements(By.CSS, sel)` | `page.findAll(sel)` | Low |
| `element.click()` | `el.click()` | Low |
| `element.send_keys(text)` | `el.type(text)` | Low |
| `element.clear()` | `el.clear()` | Low |
| `element.text` | `el.text()` | Trivial |
| `element.get_attribute(name)` | `el.attr(name)` | Trivial |
| `element.is_displayed()` | `el.isVisible()` | Low |
| `element.is_enabled()` | `el.isEnabled()` | Low |
| `element.is_selected()` | `el.isChecked()` | Low |
| `driver.quit()` | `browser.close()` | Low |
| `driver.close()` | `page.close()` | Low |

### Priority 2 — Common Operations

| Selenium | Vibium Mapping | Complexity |
|----------|---------------|------------|
| `driver.execute_script(js)` | `page.eval(js)` | Low |
| `driver.execute_async_script(js)` | `page.eval(asyncJs)` | Medium |
| `driver.get_screenshot_as_png()` | `page.screenshot()` | Low |
| `driver.page_source` | `page.eval(() => document.documentElement.outerHTML)` | Low |
| `driver.get_cookies()` | `context.cookies()` | Low |
| `driver.add_cookie(cookie)` | `context.setCookies([cookie])` | Low |
| `driver.delete_all_cookies()` | `context.clearCookies()` | Low |
| `Select(el).select_by_value(v)` | `el.selectOption(v)` | Medium |
| `ActionChains(driver).move_to(el)` | `el.hover()` | Medium |
| `ActionChains(driver).drag_and_drop()` | `el.dragTo(target)` | Medium |

### Priority 3 — Window/Frame Management

| Selenium | Vibium Mapping | Complexity |
|----------|---------------|------------|
| `driver.window_handles` | `browser.pages()` → handle map | Medium |
| `driver.current_window_handle` | `currentPage.id` | Low |
| `driver.switch_to.window(h)` | Update current vibe | Medium |
| `driver.switch_to.frame(ref)` | `page.frame(ref)` | Medium |
| `driver.switch_to.default_content()` | Reset to main vibe | Low |
| `driver.switch_to.alert` | `page.onDialog()` wrapper | Medium |

### Priority 4 — Advanced / Edge Cases

| Selenium | Vibium Mapping | Complexity |
|----------|---------------|------------|
| Implicit waits | Timeout wrapper around find | High |
| Explicit waits (`WebDriverWait`) | `page.waitFor()` / `waitForFunction()` | Medium |
| Expected Conditions | Map to Vibium wait predicates | Medium |
| `element.screenshot()` | `el.screenshot()` | Low |
| `element.location` / `.size` | `el.bounds()` decomposed | Low |
| `element.tag_name` | `el.eval(e => e.tagName)` | Low |
| `element.value_of_css_property()` | `el.eval(fn)` | Low |
| Proxy configuration | Launch options | Medium |
| Desired capabilities | Launch options mapping | High |

### Not Supported / Out of Scope

| Selenium Feature | Reason |
|-----------------|--------|
| `chromedriver` / `geckodriver` binaries | Vibium connects directly via BiDi |
| Selenium Grid / Remote WebDriver | Different architecture (future: Vibium cloud) |
| `driver.service` | No driver service to manage |
| Legacy browser support (IE) | BiDi requires modern browsers |

---

## By Locator Strategy

| Selenium `By` | Vibium Selector | Notes |
|---------------|-----------------|-------|
| `By.CSS_SELECTOR` | `find('.class')` | Default, no prefix |
| `By.XPATH` | `find('xpath=//div')` | |
| `By.ID` | `find('#id')` | CSS shorthand |
| `By.CLASS_NAME` | `find('.className')` | CSS shorthand |
| `By.TAG_NAME` | `find('tagName')` | CSS covers it |
| `By.NAME` | `find('[name="val"]')` | CSS attribute selector |
| `By.LINK_TEXT` | `find('text=Click Here')` | |
| `By.PARTIAL_LINK_TEXT` | `find('text=Click')` | Partial match |

---

## Migration Path

### Phase 1: Drop-in replacement
Change the import, run existing tests. Fix any failures from timing differences.

### Phase 2: Hybrid
New tests use `bro`/`vibe` native API. Old tests stay on compat layer.

### Phase 3: Full migration
Rewrite old tests to native API. Remove compat layer dependency.

### The Carrot

Once on the native API, you get:
- **`page.check('...')`** — AI-powered assertions in plain English
- **`page.do('...')`** — AI-powered actions
- **`page.route()`** — network interception (impossible in Selenium)
- **Multi-page without switching** — no `switch_to.window()` ever again
- **No driver binaries** — BiDi is built into the browser

---

## Implementation Notes

### Stale Element References

Selenium throws `StaleElementReferenceException` when an element's DOM node is garbage collected. Vibium elements are based on BiDi remote object references, which have different lifecycle semantics. The compat layer should:

1. Catch Vibium's equivalent error
2. Wrap it in `StaleElementReferenceException` for Selenium compatibility
3. Document that timing may differ slightly

### Implicit Waits

Selenium's implicit wait is global and applies to every `find_element` call. Vibium's `find()` has its own timeout behavior. The compat layer needs to respect `driver.implicitly_wait(seconds)` by configuring Vibium's find timeout.

### ActionChains

Selenium's `ActionChains` builder pattern maps to Vibium's `keyboard` and `mouse` APIs:

```python
# Selenium
ActionChains(driver).move_to_element(el).click().perform()

# Vibium (under the hood)
await el.hover()
await el.click()
```

The compat layer builds up actions and executes them on `.perform()`.
