# Recording Browser Sessions

Record a timeline of screenshots, network requests, DOM snapshots, action groups — and, on supported engines, a video track — then view it in Record Player.

---

## What You'll Learn

How to capture a recording of a browser session and view it as an interactive timeline. For the video track's engine requirements and options, see the [Record Video](../how-to-guides/record-video.md) how-to guide.

---

## Quick Start

The fastest way to record a session — use `page.context` to access recording without creating an explicit context:

```javascript
const { browser } = require('vibium')

async function main() {
  const bro = await browser.start()
  const vibe = await bro.page()

  await vibe.context.recording.start()

  await vibe.go('https://example.com')
  await vibe.find('a').click()

  await vibe.context.recording.stop()
  await bro.stop()
}

main()
```

<details>
<summary>Sync JS</summary>

```javascript
const { browser } = require('vibium/sync')

const bro = browser.start()
const vibe = bro.page()

vibe.context.recording.start()

vibe.go('https://example.com')
vibe.find('a').click()

vibe.context.recording.stop()
bro.stop()
```

</details>

<details>
<summary>Async Python</summary>

```python
import asyncio
from vibium.async_api import browser

async def main():
    bro = await browser.start()
    vibe = await bro.page()

    await vibe.context.recording.start()

    await vibe.go('https://example.com')
    await vibe.find('a').click()

    await vibe.context.recording.stop()
    await bro.stop()

asyncio.run(main())
```

</details>

<details>
<summary>Sync Python</summary>

```python
from vibium import browser

bro = browser.start()
vibe = bro.page()

vibe.context.recording.start()

vibe.go('https://example.com')
vibe.find('a').click()

vibe.context.recording.stop()
bro.stop()
```

</details>

<details>
<summary>Java</summary>

```java
import com.vibium.Vibium;

var bro = Vibium.start();
var vibe = bro.page();

vibe.context().recording().start();

vibe.go("https://example.com");
vibe.find("a").click();

vibe.context().recording().stop();

bro.stop();
```

</details>

<details>
<summary>CLI</summary>

```bash
vibium record start

vibium go https://example.com
vibium click 'a'

vibium record stop
```

</details>

<details>
<summary>MCP</summary>

Ask your AI assistant:

> "Start a recording, go to example.com and click the first link, then stop and save it as record.zip"

The assistant drives the same operations through the `browser_record_start` and `browser_record_stop` tools.

</details>

Open `record.zip` in [Record Player](https://player.vibium.dev) to see a timeline of screenshots and actions.

---

## Basic Recording

Recording lives on `BrowserContext`, not `Page`. The Quick Start above uses `page.context` as a shortcut — under the hood, every page belongs to a context, and `page.context` gives you direct access to it. This is equivalent to creating an explicit context:

```javascript
const { browser } = require('vibium')

async function main() {
  const bro = await browser.start()
  const ctx = await bro.newContext()
  const vibe = await ctx.newPage()

  await ctx.recording.start({ name: 'my-session' })

  await vibe.go('https://example.com')
  await vibe.find('a').click()

  await ctx.recording.stop()   // lands at record.zip by default

  await bro.stop()
}

main()
```

<details>
<summary>Sync JS</summary>

```javascript
const { browser } = require('vibium/sync')

const bro = browser.start()
const ctx = bro.newContext()
const vibe = ctx.newPage()

ctx.recording.start({ name: 'my-session' })

vibe.go('https://example.com')
vibe.find('a').click()

ctx.recording.stop()   // lands at record.zip by default

bro.stop()
```

</details>

<details>
<summary>Async Python</summary>

```python
import asyncio
from vibium.async_api import browser

async def main():
    bro = await browser.start()
    ctx = await bro.new_context()
    vibe = await ctx.new_page()

    await ctx.recording.start(name='my-session')

    await vibe.go('https://example.com')
    await vibe.find('a').click()

    await ctx.recording.stop()   # lands at record.zip by default

    await bro.stop()

asyncio.run(main())
```

</details>

<details>
<summary>Sync Python</summary>

```python
from vibium import browser

bro = browser.start()
ctx = bro.new_context()
vibe = ctx.new_page()

ctx.recording.start(name='my-session')

vibe.go('https://example.com')
vibe.find('a').click()

ctx.recording.stop()   # lands at record.zip by default

bro.stop()
```

</details>

Use an explicit context when you need multiple pages in the same recording, or when you want to configure context options (viewport, locale, etc.). Use `page.context` when you just want to record a single page quickly.

The recording zip lands at `record.zip` in your working directory by default. Declare a different destination at start, or override it at stop — the stop path wins:

```javascript
await ctx.recording.start({ path: 'runs/login.zip' })
// ...
await ctx.recording.stop({ path: 'runs/login-failed.zip' })  // override wins
```

Declaring the path at start also protects the recording: if the browser session ends before `stop()`, the recording auto-finalizes and delivers there. `stop()` additionally returns the zip as a `Buffer` (`bytes` in Python); pass `path: null` (`path=None`) at start for bytes-only capture with no file written.

Screenshots are on by default. Add `snapshots` for a more complete recording (`screenshots: false` turns the filmstrip off):

```javascript
await ctx.recording.start({ snapshots: true })
```

- **screenshots** — captures the page after each action (on by default), creating a visual filmstrip.
- **snapshots** — captures the page around each action, so you can inspect it in the viewer.

To reduce recording size, use JPEG format with a lower quality setting:

```javascript
await ctx.recording.start({
  format: 'jpeg',
  quality: 0.3,
})
```

The default format is JPEG at 0.5 quality. Lowering `quality` produces smaller files — useful for long-running recordings or CI where file size matters.

---

## Video

On engines that support it (Firefox 154+, local browsers), the recording can include a video track — the browser encodes the viewport to WebM and the file lands inside the zip next to the trace:

```javascript
await ctx.recording.start({ video: true })
// ...
await ctx.recording.stop()   // record.zip now contains video/<context>.webm
```

With `video` omitted, video is recorded whenever the engine supports it and skipped otherwise. `video: true` requires it — `start()` fails with an explanatory error on Chrome. `video: false` turns it off. Dimensions default to the viewport (`video: { width, height, frameRate }` to override).

Engine requirements, Firefox channel setup, and the zip layout are covered in [Record Video](../how-to-guides/record-video.md).

---

## Actions

Every vibium command (`click`, `fill`, `navigate`, etc.) is automatically recorded in the recording as an action marker. You don't need to wrap commands in groups to see them — they show up individually in the timeline.

```javascript
await ctx.recording.start()

await vibe.go('https://example.com')       // recorded as Page.navigate
await vibe.find('#btn').click()             // recorded as Element.click
await vibe.find('#input').fill('hello')     // recorded as Element.fill

await ctx.recording.stop()
```

<details>
<summary>Sync JS</summary>

```javascript
ctx.recording.start()

vibe.go('https://example.com')       // recorded as Page.navigate
vibe.find('#btn').click()             // recorded as Element.click
vibe.find('#input').fill('hello')     // recorded as Element.fill

ctx.recording.stop()
```

</details>

<details>
<summary>Async Python</summary>

```python
await ctx.recording.start()

await vibe.go('https://example.com')       # recorded as Page.navigate
await vibe.find('#btn').click()             # recorded as Element.click
await vibe.find('#input').fill('hello')     # recorded as Element.fill

await ctx.recording.stop()
```

</details>

<details>
<summary>Sync Python</summary>

```python
ctx.recording.start()

vibe.go('https://example.com')       # recorded as Page.navigate
vibe.find('#btn').click()             # recorded as Element.click
vibe.find('#input').fill('hello')     # recorded as Element.fill

ctx.recording.stop()
```

</details>

<details>
<summary>CLI</summary>

```bash
vibium record start

vibium go https://example.com
vibium click '#btn'
vibium fill '#input' 'hello'

vibium record stop
```

</details>

To also record the raw BiDi protocol commands sent to the browser (e.g. `input.performActions`, `script.callFunction`), enable `bidi`:

```javascript
await ctx.recording.start({ bidi: true })
```

This is useful for debugging low-level protocol issues but makes recordings larger.

---

## Action Groups

Use `startGroup()` and `stopGroup()` to label sections of your recording. Groups show up as named spans in the timeline.

```javascript
await ctx.recording.start()
await vibe.go('https://example.com')

await ctx.recording.startGroup('fill login form')
await vibe.find('#username').fill('alice')
await vibe.find('#password').fill('secret')
await ctx.recording.stopGroup()

await ctx.recording.startGroup('submit')
await vibe.find('button[type="submit"]').click()
await ctx.recording.stopGroup()

await ctx.recording.stop()
```

<details>
<summary>Sync JS</summary>

```javascript
ctx.recording.start()
vibe.go('https://example.com')

ctx.recording.startGroup('fill login form')
vibe.find('#username').fill('alice')
vibe.find('#password').fill('secret')
ctx.recording.stopGroup()

ctx.recording.startGroup('submit')
vibe.find('button[type="submit"]').click()
ctx.recording.stopGroup()

ctx.recording.stop()
```

</details>

<details>
<summary>Async Python</summary>

```python
await ctx.recording.start()
await vibe.go('https://example.com')

await ctx.recording.start_group('fill login form')
await vibe.find('#username').fill('alice')
await vibe.find('#password').fill('secret')
await ctx.recording.stop_group()

await ctx.recording.start_group('submit')
await vibe.find('button[type="submit"]').click()
await ctx.recording.stop_group()

await ctx.recording.stop()
```

</details>

<details>
<summary>Sync Python</summary>

```python
ctx.recording.start()
vibe.go('https://example.com')

ctx.recording.start_group('fill login form')
vibe.find('#username').fill('alice')
vibe.find('#password').fill('secret')
ctx.recording.stop_group()

ctx.recording.start_group('submit')
vibe.find('button[type="submit"]').click()
ctx.recording.stop_group()

ctx.recording.stop()
```

</details>

<details>
<summary>CLI</summary>

```bash
vibium record start
vibium go https://example.com

vibium record group start 'fill login form'
vibium fill '#username' 'alice'
vibium fill '#password' 'secret'
vibium record group stop

vibium record group start 'submit'
vibium click 'button[type="submit"]'
vibium record group stop

vibium record stop
```

</details>

Groups can be nested:

```javascript
await ctx.recording.startGroup('checkout flow')

  await ctx.recording.startGroup('shipping')
  // ... fill shipping form
  await ctx.recording.stopGroup()

  await ctx.recording.startGroup('payment')
  // ... fill payment form
  await ctx.recording.stopGroup()

await ctx.recording.stopGroup()
```

---

## Chunks

Chunks split a long recording into segments without stopping the recording. Each chunk produces its own zip.

```javascript
await ctx.recording.start()

// First chunk: login
await vibe.go('https://example.com/login')
await vibe.find('#username').fill('alice')
const loginZip = await ctx.recording.stopChunk({ path: 'login.zip' })

// Second chunk: dashboard
await ctx.recording.startChunk({ name: 'dashboard' })
await vibe.go('https://example.com/dashboard')
const dashboardZip = await ctx.recording.stopChunk({ path: 'dashboard.zip' })

// Final stop
await ctx.recording.stop()
```

<details>
<summary>Sync JS</summary>

```javascript
ctx.recording.start()

// First chunk: login
vibe.go('https://example.com/login')
vibe.find('#username').fill('alice')
ctx.recording.stopChunk({ path: 'login.zip' })

// Second chunk: dashboard
ctx.recording.startChunk({ name: 'dashboard' })
vibe.go('https://example.com/dashboard')
ctx.recording.stopChunk({ path: 'dashboard.zip' })

// Final stop
ctx.recording.stop()
```

</details>

<details>
<summary>Async Python</summary>

```python
await ctx.recording.start()

# First chunk: login
await vibe.go('https://example.com/login')
await vibe.find('#username').fill('alice')
await ctx.recording.stop_chunk(path='login.zip')

# Second chunk: dashboard
await ctx.recording.start_chunk(name='dashboard')
await vibe.go('https://example.com/dashboard')
await ctx.recording.stop_chunk(path='dashboard.zip')

# Final stop
await ctx.recording.stop()
```

</details>

<details>
<summary>Sync Python</summary>

```python
ctx.recording.start()

# First chunk: login
vibe.go('https://example.com/login')
vibe.find('#username').fill('alice')
ctx.recording.stop_chunk(path='login.zip')

# Second chunk: dashboard
ctx.recording.start_chunk(name='dashboard')
vibe.go('https://example.com/dashboard')
ctx.recording.stop_chunk(path='dashboard.zip')

# Final stop
ctx.recording.stop()
```

</details>

<details>
<summary>CLI</summary>

```bash
vibium record start

# First chunk: login
vibium go https://example.com/login
vibium fill '#username' 'alice'
vibium record chunk stop -o login.zip

# Second chunk: dashboard
vibium record chunk start --name dashboard
vibium go https://example.com/dashboard
vibium record chunk stop -o dashboard.zip

# Final stop
vibium record stop
```

</details>

---

## Viewing Recordings

Open a recording in [Record Player](https://player.vibium.dev):

1. Go to [player.vibium.dev](https://player.vibium.dev)
2. Drop your `record.zip` file onto the page

The viewer shows:
- **Timeline** — scrub through screenshots frame by frame
- **Actions** — see group markers from `startGroup()`/`stopGroup()`
- **Network** — waterfall of all HTTP requests
- **Snapshots** — inspect the DOM at capture time

---

## CLI Usage

All recording features are available from the command line. The daemon is automatically started when needed.

```bash
# Start recording with screenshots
vibium record start --snapshots --name my-session

# Do some work
vibium go https://example.com
vibium click '#btn'
vibium fill '#input' 'hello'

# Action groups
vibium record group start 'login'
vibium fill '#username' 'alice'
vibium fill '#password' 'secret'
vibium record group stop

# Chunks
vibium record chunk stop -o chunk1.zip
vibium record chunk start --name next-chunk

# Stop and save the recording
vibium record stop
```

---

## Reference

### start() Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `name` | string | `"record"` | Name for the recording |
| `title` | string | — | Title shown in Record Player |
| `screenshots` | boolean | `true` | Capture a screenshot after each action |
| `snapshots` | boolean | `false` | Capture DOM snapshots around each action |
| `sources` | boolean | `false` | Reserved for future use |
| `bidi` | boolean | `false` | Record raw BiDi commands in the recording |
| `format` | `'jpeg'` \| `'png'` | `'jpeg'` | Screenshot image format |
| `quality` | number | `0.5` | JPEG quality 0.0–1.0 (ignored for PNG) |
| `video` | boolean \| object | — | Video track (Firefox 154+). Omitted: on when supported. `true`: required. `{width, height, frameRate}` to size it |
| `path` | string \| null | `record.zip` | Where the zip lands at stop. `null`: bytes-only, no file |

### stop() Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `path` | string | start's `path` | Overrides the path declared at start |

`stop()` also returns the zip as a `Buffer`. `stopChunk()` takes a `path` too; without one it returns the chunk inline.

### CLI Flags

| Command | Flag | Description |
|---------|------|-------------|
| `record start` | `--screenshots` | Capture screenshots after each action (default on) |
| `record start` | `--snapshots` | Capture HTML snapshots |
| `record start` | `--bidi` | Record raw BiDi commands in the recording |
| `record start` | `--name NAME` | Name for the recording |
| `record start` | `--video` | Require video (`--video=false` to disable; omit for auto) |
| `record start` | `--video-size WxH` | Video dimensions (default: viewport) |
| `record start` | `--video-fps N` | Video frame rate |
| `record start` | `-o, --output PATH` | Where the zip lands at stop (default: `record.zip`) |
| `record stop` | `-o, --output PATH` | Overrides the path declared at start |

### Java

Java mirrors the same options as flat fluent setters:

```java
vibe.context().recording().start(
    new RecordingOptions().video(true).videoSize(1280, 720).path("run.zip"));
```

### MCP Tools

`browser_record_start`, `browser_record_stop`, `browser_record_start_group`, `browser_record_stop_group`, `browser_record_start_chunk`, `browser_record_stop_chunk`. The start tool takes the `start()` options as flat properties: `video`, `video_width`, `video_height`, `video_frame_rate`, `path`.

---

## Next Steps

- [Record Video](../how-to-guides/record-video.md) — the video track: engine support, Firefox channels, options
- [Recording Format](../explanation/recording-format.md) — detailed spec of the zip structure
- [Getting Started](getting-started-js.md) — first steps with Vibium
