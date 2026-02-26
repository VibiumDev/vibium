# Vibium

Browser automation for AI agents and humans.

## Installation

```bash
npm install vibium
```

This automatically downloads Chrome for Testing on first install.

## Quick Start

### Async API

```javascript
import { browser } from 'vibium'
import { writeFile } from 'fs/promises'

const bro = await browser.launch()
const vibe = await bro.page()
await vibe.go('https://example.com')

const link = await vibe.find('a')
console.log(await link.text())
await link.click()

const screenshot = await vibe.screenshot()
await writeFile('screenshot.png', screenshot)

await bro.close()
```

### Sync API

```javascript
const { browser } = require('vibium/sync')
const { writeFileSync } = require('fs')

const bro = browser.launch()
const vibe = bro.page()
vibe.go('https://example.com')

const link = vibe.find('a')
console.log(link.text())
link.click()

const screenshot = vibe.screenshot()
writeFileSync('screenshot.png', screenshot)

bro.close()
```

## API Reference

### browser.launch(options?)

Launch a new browser session. Returns a `Browser`.

```javascript
const bro = await browser.launch({ headless: true })
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `headless` | boolean | `false` | Run without visible window |

### bro.page()

Get the default page (tab).

```javascript
const vibe = await bro.page()
```

### vibe.go(url)

Navigate to a URL.

```javascript
await vibe.go('https://example.com')
```

### vibe.find(selector, options?)

Find an element by CSS selector.

```javascript
const button = await vibe.find('button.submit')
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `timeout` | number | `30000` | Max wait time in ms |

### vibe.screenshot()

Capture a screenshot. Returns a `Buffer` (PNG).

```javascript
const png = await vibe.screenshot()
```

### bro.close()

Close the browser session.

```javascript
await bro.close()
```

### element.click(options?)

Click the element. Waits for element to be visible, stable, and enabled.

```javascript
await element.click()
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `timeout` | number | `30000` | Max wait time in ms |

### element.type(text, options?)

Type text into the element. Waits for element to be visible, stable, enabled, and editable.

```javascript
await element.type('hello@example.com')
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `timeout` | number | `30000` | Max wait time in ms |

### element.text()

Get the element's text content.

```javascript
const text = await element.text()
```

### element.getAttribute(name)

Get an attribute value.

```javascript
const testId = await element.getAttribute('data-testid')
```

### element.boundingBox()

Get the element's position and size. Returns `{x, y, width, height}`.

```javascript
const box = await element.boundingBox()
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `VIBIUM_SKIP_BROWSER_DOWNLOAD` | Set to `1` to skip Chrome download on install |

## Claude Code / MCP

Add Vibium to Claude Code:

```bash
claude mcp add vibium -- npx vibium
```

### MCP Tools

#### browser_launch

Start a browser session.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `headless` | boolean | `false` | Run without visible window |

#### browser_navigate

Navigate to a URL.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `url` | string | yes | The URL to navigate to |

#### browser_click

Click an element. Waits for element to be visible, stable, and enabled.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `selector` | string | yes | CSS selector |

#### browser_type

Type text into an element. Waits for element to be visible, stable, enabled, and editable.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `selector` | string | yes | CSS selector |
| `text` | string | yes | Text to type |

#### browser_screenshot

Capture a screenshot.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `filename` | string | no | Save to file (e.g., screenshot.png) |

#### browser_find

Find an element and return its info (tag, text, bounding box).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `selector` | string | yes | CSS selector |

#### browser_quit

Close the browser session.

## Links

- [GitHub / Documentation](https://github.com/VibiumDev/vibium)
- [Website](https://vibium.com)

## License

Apache-2.0
