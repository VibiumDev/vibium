---
name: vibe-check
description: Browser automation for AI agents. Use when the user needs to navigate websites, read page content, fill forms, click elements, take screenshots, or manage browser tabs.
---

# Vibium Browser Automation — CLI Reference

The `vibium` CLI automates Chrome via the command line. The browser auto-launches on first use (daemon mode keeps it running between commands).

```
vibium go <url> && vibium map && vibium click @e1 && vibium map
```

## Core Workflow

Every browser automation follows this pattern:

1. **Navigate**: `vibium go <url>`
2. **Map**: `vibium map` (get element refs like `@e1`, `@e2`)
3. **Interact**: Use refs to click, fill, select — e.g. `vibium click @e1`
4. **Re-map**: After navigation or DOM changes, get fresh refs with `vibium map`

## Binary Resolution

Before running any commands, resolve the `vibium` binary path once:

1. Try `vibium` directly (works if globally installed via `npm install -g vibium`)
2. Fall back to `./clicker/bin/vibium` (dev environment, in project root)
3. Fall back to `./node_modules/.bin/vibium` (local npm install)

Run `vibium --help` (or the resolved path) to confirm. Use the resolved path for all subsequent commands.

**Windows note:** Use forward slashes in paths (e.g. `./clicker/bin/vibium.exe`) and quote paths containing spaces.

## Command Chaining

Chain commands with `&&` to run them sequentially. The chain stops on first error:

```sh
vibium go https://example.com && vibium map && vibium click @e3 && vibium diff map
```

**When to chain:** Use `&&` for sequences that should happen back-to-back (navigate → interact → verify). Run commands separately when you need to inspect output between steps.

**When NOT to chain:** Don't chain commands that depend on parsing the previous output (e.g. reading map output to decide what to click). Run those separately so you can analyze the result first.

## Commands

### Discovery
- `vibium map` — map interactive elements with @refs (recommended before interacting)
- `vibium map --selector "nav"` — scope map to elements within a CSS subtree
- `vibium diff map` — compare current vs last map (see what changed)

### Navigation
- `vibium go <url>` — go to a page
- `vibium back` — go back in history
- `vibium forward` — go forward in history
- `vibium reload` — reload the current page
- `vibium url` — print current URL
- `vibium title` — print page title

### Reading Content
- `vibium text` — get all page text
- `vibium text "<selector>"` — get text of a specific element
- `vibium html` — get page HTML (use `--outer` for outerHTML)
- `vibium find "<selector>"` — find element, return `@e1` ref (clickable with `vibium click @e1`)
- `vibium find --text "Sign In"` — find element by text content → `@e1`
- `vibium find --label "Email"` — find input by label → `@e1`
- `vibium find --placeholder "Search"` — find by placeholder → `@e1`
- `vibium find --testid "submit-btn"` — find by data-testid → `@e1`
- `vibium find --xpath "//div[@class]"` — find by XPath → `@e1`
- `vibium find --alt "Logo"` — find by alt attribute → `@e1`
- `vibium find --title "Settings"` — find by title attribute → `@e1`
- `vibium find-all "<selector>"` — find all matching elements → `@e1`, `@e2`, ... (`--limit N`)
- `vibium find --role <role>` — find element by ARIA role → `@e1` (combine with `--text`, `--label`, etc.)
- `vibium eval "<js>"` — run JavaScript and print result (`--stdin` to read from stdin)
- `vibium count "<selector>"` — count matching elements
- `vibium screenshot -o file.png` — capture screenshot (`--full-page`, `--annotate`)
- `vibium a11y-tree` — accessibility tree (`--everything` for all nodes)

### Interaction
- `vibium click "<selector>"` — click an element (also accepts `@ref` from map)
- `vibium dblclick "<selector>"` — double-click an element
- `vibium type "<selector>" "<text>"` — type into an input (appends to existing value)
- `vibium fill "<selector>" "<text>"` — clear field and type new text (replaces value)
- `vibium press <key> [selector]` — press a key on element or focused element
- `vibium focus "<selector>"` — focus an element
- `vibium hover "<selector>"` — hover over an element
- `vibium scroll [direction]` — scroll page (`--amount N`, `--selector`)
- `vibium scroll-into-view "<selector>"` — scroll element into view (centered)
- `vibium keys "<combo>"` — press keys (Enter, Control+a, Shift+Tab)
- `vibium select "<selector>" "<value>"` — pick a dropdown option
- `vibium check "<selector>"` — check a checkbox/radio (idempotent)
- `vibium uncheck "<selector>"` — uncheck a checkbox (idempotent)

### Mouse Primitives
- `vibium mouse-click [x] [y]` — click at coordinates or current position (`--button 0|1|2`)
- `vibium mouse-move <x> <y>` — move mouse to coordinates
- `vibium mouse-down` — press mouse button (`--button 0|1|2`)
- `vibium mouse-up` — release mouse button (`--button 0|1|2`)
- `vibium drag "<source>" "<target>"` — drag from one element to another

### Element State
- `vibium value "<selector>"` — get input/textarea/select value
- `vibium attr "<selector>" "<attribute>"` — get HTML attribute value
- `vibium is-visible "<selector>"` — check if element is visible (true/false)
- `vibium is-enabled "<selector>"` — check if element is enabled (true/false)
- `vibium is-checked "<selector>"` — check if checkbox/radio is checked (true/false)

### Waiting
- `vibium wait "<selector>"` — wait for element (`--state visible|hidden|attached`, `--timeout ms`)
- `vibium wait-for-url "<pattern>"` — wait until URL contains substring (`--timeout ms`)
- `vibium wait-for-load` — wait until page is fully loaded (`--timeout ms`)
- `vibium wait-for-text "<text>"` — wait until text appears on page (`--timeout ms`)
- `vibium wait-for-fn "<expression>"` — wait until JS expression returns truthy (`--timeout ms`)
- `vibium sleep <ms>` — pause execution (max 30000ms)

### Capture
- `vibium screenshot -o file.png` — capture screenshot (`--full-page`, `--annotate`)
- `vibium pdf -o file.pdf` — save page as PDF

### Dialogs
- `vibium dialog accept [text]` — accept dialog (optionally with prompt text)
- `vibium dialog dismiss` — dismiss dialog

### Emulation
- `vibium set-viewport <width> <height>` — set viewport size (`--dpr` for device pixel ratio)
- `vibium viewport` — get current viewport dimensions
- `vibium window` — get OS browser window dimensions and state
- `vibium set-window <width> <height> [x] [y]` — set window size and position (`--state`)
- `vibium emulate-media` — override CSS media features (`--color-scheme`, `--reduced-motion`, `--forced-colors`, `--contrast`, `--media`)
- `vibium set-geolocation <lat> <lng>` — override geolocation (`--accuracy`)
- `vibium set-content "<html>"` — replace page HTML (`--stdin` to read from stdin)

### Frames
- `vibium frames` — list all iframes on the page
- `vibium frame "<nameOrUrl>"` — find a frame by name or URL substring

### File Upload
- `vibium upload "<selector>" <files...>` — set files on input[type=file]

### Tracing
- `vibium trace start` — start recording (`--screenshots`, `--snapshots`, `--name`)
- `vibium trace stop` — stop recording and save ZIP (`-o path`)

### Cookies
- `vibium cookies` — list all cookies
- `vibium cookies set <name> <value>` — set a cookie
- `vibium cookies clear` — clear all cookies

### Storage State
- `vibium storage-state` — export cookies + localStorage + sessionStorage (`-o state.json`)
- `vibium restore-storage <path>` — restore state from JSON file

### Downloads
- `vibium download set-dir <path>` — set download directory

### Tabs
- `vibium tabs` — list open tabs
- `vibium tab-new [url]` — open new tab
- `vibium tab-switch <index|url>` — switch tab
- `vibium tab-close [index]` — close tab

### Debug
- `vibium highlight "<selector>"` — highlight element visually (3 seconds)

### Session
- `vibium quit` — close the browser (daemon keeps running)
- `vibium close` — alias for quit
- `vibium daemon start` — start background browser
- `vibium daemon status` — check if running
- `vibium daemon stop` — stop daemon

## Common Patterns

### Ref-based workflow (recommended for AI)
```sh
vibium go https://example.com
vibium map
vibium click @e1
vibium map  # re-map after interaction
```

### Verify action worked
```sh
vibium map
vibium click @e3
vibium diff map  # see what changed
```

### Read a page
```sh
vibium go https://example.com && vibium text
```

### Fill a form (end-to-end)
```sh
vibium go https://example.com/login
vibium map
# Look at map output to identify form fields
vibium fill @e1 "user@example.com"
vibium fill @e2 "secret"
vibium click @e3
vibium wait-for-url "/dashboard"
vibium screenshot -o after-login.png
```

### Scoped map (large pages)
```sh
vibium map --selector "nav"        # Only map elements in <nav>
vibium map --selector "#sidebar"   # Only map elements in #sidebar
vibium map --selector "form"       # Only map form controls
```

### Semantic find (no CSS selectors needed)
```sh
vibium find --text "Sign In"           # → @e1 [button] "Sign In"
vibium find --label "Email"            # → @e1 [input] placeholder="Email"
vibium click @e1                       # Click the found element
vibium find --placeholder "Search..."  # → @e1 [input] placeholder="Search..."
vibium find --testid "submit-btn"      # → @e1 [button] "Submit"
vibium find --alt "Company logo"       # → @e1 [img] alt="Company logo"
vibium find --title "Close"            # → @e1 [button] title="Close"
vibium find --xpath "//a[@href='/about']"  # → @e1 [a] "About"
```

### Authentication with state persistence
```sh
# Log in once and save state
vibium go https://app.example.com/login
vibium fill "input[name=email]" "user@example.com"
vibium fill "input[name=password]" "secret"
vibium click "button[type=submit]"
vibium wait-for-url "/dashboard"
vibium storage-state -o auth.json

# Restore in a later session (skips login)
vibium restore-storage auth.json
vibium go https://app.example.com/dashboard
```

### Extract structured data
```sh
vibium go https://example.com
vibium eval "JSON.stringify([...document.querySelectorAll('a')].map(a => ({text: a.textContent.trim(), href: a.href})))"
```

### Check page structure without rendering
```sh
vibium go https://example.com && vibium a11y-tree
```

### Multi-tab workflow
```sh
vibium tab-new https://docs.example.com
vibium text "h1"
vibium tab-switch 0
```

### Annotated screenshot
```sh
vibium screenshot -o annotated.png --annotate
```

### Inspect an element
```sh
vibium attr "a" "href"
vibium value "input[name=email]"
vibium is-visible ".modal"
```

### Save as PDF
```sh
vibium go https://example.com && vibium pdf -o page.pdf
```

## Eval / JavaScript

`vibium eval` is the escape hatch for any DOM query or mutation the CLI doesn't cover directly.

**Simple expressions** — use single quotes:
```sh
vibium eval 'document.title'
vibium eval 'document.querySelectorAll("li").length'
```

**Complex scripts** — use `--stdin` with a heredoc:
```sh
vibium eval --stdin <<'EOF'
const rows = [...document.querySelectorAll('table tbody tr')];
JSON.stringify(rows.map(r => {
  const cells = r.querySelectorAll('td');
  return { name: cells[0].textContent.trim(), price: cells[1].textContent.trim() };
}));
EOF
```

**JSON output** — use `--json` to get machine-readable output:
```sh
vibium eval --json 'JSON.stringify({url: location.href, title: document.title})'
```

**Important:** `eval` returns the expression result. If your script doesn't return a value, you'll get `null`. Always make sure the last expression evaluates to the data you want.

## Timeouts and Waiting

All interaction commands (`click`, `fill`, `type`, etc.) auto-wait for the target element to be actionable. You usually don't need explicit waits.

Use explicit waits when:
- **Waiting for navigation:** `vibium wait-for-url "/dashboard"` — after clicking a link that navigates
- **Waiting for content:** `vibium wait-for-text "Success"` — after form submission, wait for confirmation
- **Waiting for element:** `vibium wait ".modal"` — wait for a modal to appear
- **Waiting for page load:** `vibium wait-for-load` — after navigation to a slow page
- **Waiting for JS condition:** `vibium wait-for-fn "window.appReady === true"` — wait for app initialization
- **Fixed delay (last resort):** `vibium sleep 2000` — only when no better signal exists (max 30s)

All wait commands accept `--timeout <ms>` (default varies by command).

## Ref Lifecycle

Refs (`@e1`, `@e2`) are invalidated when the page changes. Always re-map after:
- Clicking links or buttons that navigate
- Form submissions
- Dynamic content loading (dropdowns, modals)

## Global Flags

| Flag | Description |
|------|-------------|
| `--headless` | Hide browser window |
| `--json` | Output as JSON |
| `--oneshot` | One-shot mode (no daemon) |
| `-v, --verbose` | Debug logging |
| `--wait-open N` | Wait N seconds after navigation |
| `--wait-close N` | Keep browser open N seconds before closing |

## Daemon vs Oneshot

By default, commands connect to a **daemon** — a background process that keeps the browser alive between commands. This is fast and lets you chain commands against the same page.

Use `--oneshot` (or `VIBIUM_ONESHOT=1`) to launch a fresh browser for each command, then tear it down. Useful for CI or one-off scripts.

## Tips

- All click/type/hover/fill actions auto-wait for the element to be actionable
- All selector arguments also accept `@ref` from `vibium map`
- Use `vibium map` before interacting to discover interactive elements
- Use `vibium map --selector` to reduce noise on large pages
- Use `vibium fill` to replace a field's value, `vibium type` to append to it
- Use `vibium find --text` / `--label` / `--testid` for semantic element lookup (more reliable than CSS selectors)
- Use `vibium find --role` for ARIA-role-based lookup
- Use `vibium a11y-tree` to understand page structure without visual rendering
- Use `vibium text "<selector>"` to read specific sections
- Use `vibium diff map` after interactions to see what changed
- `vibium eval` is the escape hatch for complex DOM queries
- `vibium check`/`vibium uncheck` are idempotent — safe to call without checking state first
- Screenshots save to the current directory by default (`-o` to change)
- Use `vibium storage-state` / `vibium restore-storage` to persist auth across sessions
