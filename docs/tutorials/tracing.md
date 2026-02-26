# Tracing Browser Sessions

Record a timeline of screenshots, network requests, DOM snapshots, and action groups — then view it in Vibium Trace.

---

## What You'll Learn

How to capture a trace of a browser session and view it as an interactive timeline.

---

## Quick Start

The fastest way to trace a session — use `page.context` to access tracing without creating an explicit context:

```javascript
const { browser } = require('vibium')

async function main() {
  const bro = await browser.launch()
  const vibe = await bro.page()

  await vibe.context.tracing.start({ screenshots: true })

  await vibe.go('https://example.com')
  await vibe.find('a').click()

  await vibe.context.tracing.stop({ path: 'trace.zip' })
  await bro.close()
}

main()
```

Open `trace.zip` in [Vibium Trace](https://trace.vibium.dev) to see a timeline of screenshots and actions.

---

## Basic Tracing

Tracing lives on `BrowserContext`, not `Page`. The Quick Start above uses `page.context` as a shortcut — under the hood, every page belongs to a context, and `page.context` gives you direct access to it. This is equivalent to creating an explicit context:

```javascript
const { browser } = require('vibium')

async function main() {
  const bro = await browser.launch()
  const ctx = await bro.newContext()
  const vibe = await ctx.newPage()

  await ctx.tracing.start({ name: 'my-session' })

  await vibe.go('https://example.com')
  await vibe.find('a').click()

  const zip = await ctx.tracing.stop()
  require('fs').writeFileSync('trace.zip', zip)

  await bro.close()
}

main()
```

Use an explicit context when you need multiple pages in the same trace, or when you want to configure context options (viewport, locale, etc.). Use `page.context` when you just want to trace a single page quickly.

`stop()` returns a `Buffer` containing the trace zip. You can also pass a `path` to write the file directly:

```javascript
await ctx.tracing.stop({ path: 'trace.zip' })
```

Enable `screenshots` and `snapshots` for a more complete trace:

```javascript
await ctx.tracing.start({ screenshots: true, snapshots: true })
```

- **screenshots** — captures the page periodically (~100ms), creating a visual filmstrip. Identical frames are deduplicated.
- **snapshots** — captures the full HTML when the trace stops, so you can inspect the DOM in the viewer.

---

## Actions

Every vibium command (`click`, `fill`, `navigate`, etc.) is automatically recorded in the trace as an action marker. You don't need to wrap commands in groups to see them — they show up individually in the timeline.

```javascript
await ctx.tracing.start({ screenshots: true })

await vibe.go('https://example.com')       // recorded as Page.navigate
await vibe.find('#btn').click()             // recorded as Element.click
await vibe.find('#input').fill('hello')     // recorded as Element.fill

await ctx.tracing.stop({ path: 'trace.zip' })
```

To also record the raw BiDi protocol commands sent to the browser (e.g. `input.performActions`, `script.callFunction`), enable `bidi`:

```javascript
await ctx.tracing.start({ screenshots: true, bidi: true })
```

This is useful for debugging low-level protocol issues but makes traces larger.

---

## Action Groups

Use `startGroup()` and `stopGroup()` to label sections of your trace. Groups show up as named spans in the timeline.

```javascript
await ctx.tracing.start({ screenshots: true })
await vibe.go('https://example.com')

await ctx.tracing.startGroup('fill login form')
await vibe.find('#username').fill('alice')
await vibe.find('#password').fill('secret')
await ctx.tracing.stopGroup()

await ctx.tracing.startGroup('submit')
await vibe.find('button[type="submit"]').click()
await ctx.tracing.stopGroup()

await ctx.tracing.stop({ path: 'trace.zip' })
```

Groups can be nested:

```javascript
await ctx.tracing.startGroup('checkout flow')

  await ctx.tracing.startGroup('shipping')
  // ... fill shipping form
  await ctx.tracing.stopGroup()

  await ctx.tracing.startGroup('payment')
  // ... fill payment form
  await ctx.tracing.stopGroup()

await ctx.tracing.stopGroup()
```

---

## Chunks

Chunks split a long trace into segments without stopping the recording. Each chunk produces its own zip.

```javascript
await ctx.tracing.start({ screenshots: true })

// First chunk: login
await vibe.go('https://example.com/login')
await vibe.find('#username').fill('alice')
const loginZip = await ctx.tracing.stopChunk({ path: 'login.zip' })

// Second chunk: dashboard
await ctx.tracing.startChunk({ name: 'dashboard' })
await vibe.go('https://example.com/dashboard')
const dashboardZip = await ctx.tracing.stopChunk({ path: 'dashboard.zip' })

// Final stop
await ctx.tracing.stop()
```

---

## Viewing Traces

Open a trace in [Vibium Trace](https://trace.vibium.dev):

1. Go to [trace.vibium.dev](https://trace.vibium.dev)
2. Drop your `trace.zip` file onto the page

The viewer shows:
- **Timeline** — scrub through screenshots frame by frame
- **Actions** — see group markers from `startGroup()`/`stopGroup()`
- **Network** — waterfall of all HTTP requests
- **Snapshots** — inspect the DOM at capture time

---

## CLI Usage

In daemon mode, you can start and stop traces from the command line:

```bash
# Start tracing with screenshots
vibium trace start --screenshots --snapshots --name my-session

# Do some work (navigate, click, etc.)
vibium go https://example.com
vibium click '#btn'

# Stop and save the trace
vibium trace stop -o trace.zip

# View it at https://trace.vibium.dev
```

CLI tracing requires the daemon to be running (the default mode). It is not available with `--oneshot`.

---

## Reference

### start() Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `name` | string | `"trace"` | Name for the trace |
| `title` | string | — | Title shown in Vibium Trace |
| `screenshots` | boolean | `false` | Capture screenshots (~100ms interval) |
| `snapshots` | boolean | `false` | Capture DOM snapshots on stop |
| `sources` | boolean | `false` | Reserved for future use |
| `bidi` | boolean | `false` | Record raw BiDi commands in the trace |

### stop() / stopChunk() Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `path` | string | — | File path to write the zip to |

When `path` is omitted, the zip data is returned as a `Buffer`.

### CLI Flags

| Command | Flag | Description |
|---------|------|-------------|
| `trace start` | `--screenshots` | Capture screenshots periodically |
| `trace start` | `--snapshots` | Capture HTML snapshots |
| `trace start` | `--bidi` | Record raw BiDi commands in the trace |
| `trace start` | `--name NAME` | Name for the trace |
| `trace stop` | `-o, --output PATH` | Output file path (default: `trace.zip`) |

---

## Next Steps

- [Trace Format](../explanation/trace-format.md) — detailed spec of the zip structure
- [Getting Started](getting-started-js.md) — first steps with Vibium
