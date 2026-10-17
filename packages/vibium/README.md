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

const bro = await browser.start()
const vibe = await bro.page()
await vibe.go('https://example.com')

const link = await vibe.find('a')
console.log(await link.text())
await link.click()

const screenshot = await vibe.screenshot()
await writeFile('screenshot.png', screenshot)

await bro.stop()
```

### Sync API

```javascript
const { browser } = require('vibium/sync')
const { writeFileSync } = require('fs')

const bro = browser.start()
const vibe = bro.page()
vibe.go('https://example.com')

const link = vibe.find('a')
console.log(link.text())
link.click()

const screenshot = vibe.screenshot()
writeFileSync('screenshot.png', screenshot)

bro.stop()
```

## CLI

The same package ships a `vibium` command-line tool. It is a native binary, so
once installed it drives the browser from a shell or a non-Node harness (Make,
Python, bash) with no Node runtime involved.

```bash
npx vibium go https://example.com   # or node_modules/.bin/vibium
npx vibium map                      # list interactive elements as @refs
npx vibium find "a"                 # find an element
npx vibium click "a"
npx vibium screenshot -o shot.png
npx vibium --help                   # full command list
```

Give it a plain-language goal (needs AI configured), or use the explicit
commands above:

```bash
npx vibium "open example.com and screenshot the pricing page"
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `VIBIUM_SKIP_BROWSER_DOWNLOAD` | Set to `1` to skip Chrome download on install |

## Requirements

- Node.js 18+ for the JavaScript API. The `vibium` CLI is a native binary and
  needs no Node runtime to run.

## Links

- [GitHub / Documentation](https://github.com/VibiumDev/vibium)
- [Website](https://vibium.com)

## License

Apache-2.0
