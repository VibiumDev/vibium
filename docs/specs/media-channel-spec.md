# Spec: Media Channel — Async Screenshots, Live Streaming, and Remote Input

> **Status: Draft.** This document describes proposed behavior. It does
> not reflect what is currently implemented in vibium. Do not rely on it
> as documentation of existing functionality.

## Context

Recording with screenshots is slow. Benchmarking saucedemo E2E:

| | With recording | Without recording | Overhead |
|--|---------------|-------------------|----------|
| **Playwright** | 17.3s | 16.3s | +1s |
| **Vibium** | 27.9s | 12.4s | +15.5s |

Without recording, Vibium (12.4s) is faster than Playwright (16.3s). The entire slowdown is screenshot capture blocking the action pipeline.

### Why it's slow

The current `dispatch()` flow in `router.go`:

1. `RecordAction()` — before marker (fast)
2. `handler()` — actual browser action
3. `endTime := time.Now()` — capture real end time
4. `captureActionSnapshot()` — DOM snapshot if enabled
5. `CaptureRecordingScreenshot()` — **blocking screenshot on main BiDi channel** (50–200ms)
6. `RecordActionEnd()` — after marker
7. Release `dispatchMu`

The BiDi WebSocket is shared. A fat base64 screenshot response sitting in the pipe delays the next action's BiDi commands. Chrome processing the capture and pushing megabytes of image data back through the same connection is the real bottleneck.

## Design: Second BiDi WebSocket

Open a **second BiDi WebSocket connection** to the same browser endpoint. BiDi is just a WebSocket — Chrome handles concurrent connections fine (same as having multiple DevTools tabs open).

The media channel serves three purposes:

1. **Recording** — async screenshot capture, fire-and-forget from `dispatch()`, unblocking the action pipeline
2. **Streaming** — continuous capture loop, relay frames over a WebSocket server to external viewers
3. **Remote input** — accept mouse/keyboard/touch from stream viewers, relay as BiDi `input.performActions`

All three use **only BiDi commands** (`browsingContext.captureScreenshot`, `input.performActions`). No CDP.

---

## Phase 1: Async Recording

The immediate goal: move screenshot capture off the main BiDi channel so `dispatch()` doesn't block.

### New file: `clicker/internal/api/media.go`

```go
type MediaChannel struct {
    conn       *bidi.Connection
    mu         sync.Mutex
    closed     bool
    nextID     int
    pending    map[int]chan json.RawMessage
    pendingMu  sync.Mutex
    stopChan   chan struct{}
    inflightWg sync.WaitGroup
}
```

| Field | Type | Purpose |
|-------|------|---------|
| `conn` | `*bidi.Connection` | Second WebSocket to the same browser |
| `nextID` | `int` | Command ID counter (starts at 2,000,000 to avoid collision) |
| `pending` | `map[int]chan json.RawMessage` | Response routing by command ID |
| `inflightWg` | `sync.WaitGroup` | Tracks in-flight async screenshots so `Drain()` can wait |
| `stopChan` | `chan struct{}` | Signals the read loop to exit |

#### Command ID ranges

| Source | ID range | Notes |
|--------|----------|-------|
| Client (passthrough) | 1 – 999,999 | Client-assigned IDs forwarded to browser |
| Main channel internal | 1,000,000 – 1,999,999 | `sendInternalCommand` in router.go |
| Media channel | 2,000,000+ | Media channel commands |

#### Methods

- `OpenMediaChannel(wsURL string) (*MediaChannel, error)` — create connection, start read loop
- `SendCommand(method string, params map[string]interface{}) (json.RawMessage, error)` — 5s timeout
- `CaptureScreenshotAsync(recorder, context, opts)` — fire goroutine, increment `inflightWg`, capture via `browsingContext.captureScreenshot` on media channel, decode, call `recorder.AddScreenshot()`
- `Drain()` — `inflightWg.Wait()`, blocks until all in-flight screenshots complete
- `Close()` — close `stopChan`, then close the WebSocket connection

### How `dispatch()` changes

| Aspect | Before | After |
|--------|--------|-------|
| Screenshot channel | Main BiDi connection | Media channel |
| Blocking | `CaptureRecordingScreenshot` blocks dispatch | `CaptureScreenshotAsync` returns immediately |
| `screenshotInFlight` | Atomic to prevent overlapping | Removed (media channel handles concurrency) |
| `dispatchMu` hold time | Includes screenshot wait | Releases after handler + fire-and-forget |

Before-snapshots stay on the main channel synchronously — they only apply to interaction handlers (click, fill) and the ~2s timeout is already short.

### Lifecycle

1. **Creation**: Lazy — opened on `recording.start` with `screenshots: true`
2. **WebSocket URL**: Stored on `BrowserSession` as `wsURL`, set during `OnClientConnect`
3. **Read loop**: Routes BiDi responses by command ID, discards events
4. **Stop**: `handleRecordingStop` calls `mc.Drain()` then `recorder.Stop()` then `mc.Close()`
5. **Teardown**: `closeSession()` closes media channel if still open

### Graceful fallback

If media channel fails to open (nil check), fall back to current synchronous behavior. Log a warning, no error to the client.

### Wire format

```json
{"id":2000001,"method":"browsingContext.captureScreenshot","params":{"context":"ABC123","format":{"type":"image/jpeg","quality":0.5}}}
```

### Files to modify

| File | Changes |
|------|---------|
| `clicker/internal/api/media.go` | **New file.** MediaChannel, OpenMediaChannel, readLoop, SendCommand, CaptureScreenshotAsync, Drain, Close |
| `clicker/internal/api/router.go` | Add `mediaChannel` and `wsURL` to BrowserSession. Modify dispatch() for async screenshots |
| `clicker/internal/api/handlers_recording.go` | Open media channel in handleRecordingStart. Drain + close in handleRecordingStop |
| `clicker/internal/agent/handlers.go` | Add `mediaChannel` to Handlers. Async screenshots in Call() |

### Verification

1. `make test` — all existing tests pass
2. Re-run saucedemo benchmark, expect recording time to drop from ~28s to ~14–15s
3. Inspect zip — `screencast-frame` events present, resources have screenshots
4. Rapid actions + immediate stop — all pending frames in zip
5. Force media channel failure — verify sync fallback works
6. Close session mid-recording — verify goroutines cleaned up

---

## Phase 2: Live Streaming + Remote Input

Stream the browser viewport via WebSocket for live preview or pair browsing.

### Activation

```bash
VIBIUM_STREAM_PORT=9223 vibium start
# or
vibium start --stream-port 9223
```

### Capture loop

BiDi has no push-based screencast, so the media channel runs a **polling loop**:

```
every <interval>:
    browsingContext.captureScreenshot on media channel
    → broadcast to connected stream viewers
    → if recording with allFrames, also feed to recorder
```

Default: 100ms (~10 fps). Configurable. Runs on the second BiDi connection — never blocks the main channel.

### Frame message (server → client)

```json
{
  "type": "frame",
  "data": "<base64-encoded-jpeg>",
  "metadata": {
    "width": 1280,
    "height": 720,
    "pageId": "ABC123",
    "timestamp": 1708000000100
  }
}
```

### Status message (server → client)

```json
{
  "type": "status",
  "connected": true,
  "streaming": true,
  "width": 1280,
  "height": 720,
  "pageId": "ABC123"
}
```

### Input messages (client → server)

Stream viewers send input events. Vibium translates each to a BiDi `input.performActions` command on the media channel.

**Mouse:**
```json
{"type": "input_mouse", "eventType": "mousePressed", "x": 100, "y": 200, "button": "left", "clickCount": 1}
{"type": "input_mouse", "eventType": "mouseReleased", "x": 100, "y": 200, "button": "left"}
{"type": "input_mouse", "eventType": "mouseMoved", "x": 150, "y": 250}
{"type": "input_mouse", "eventType": "mouseWheel", "x": 100, "y": 200, "deltaX": 0, "deltaY": 100}
```

**Keyboard:**
```json
{"type": "input_keyboard", "eventType": "keyDown", "key": "Enter", "code": "Enter"}
{"type": "input_keyboard", "eventType": "keyUp", "key": "Enter", "code": "Enter"}
{"type": "input_keyboard", "eventType": "char", "text": "a"}
{"type": "input_keyboard", "eventType": "keyDown", "key": "c", "code": "KeyC", "modifiers": 2}
```

Modifier bitmask: 1=Alt, 2=Ctrl, 4=Meta, 8=Shift.

**Touch:**
```json
{"type": "input_touch", "eventType": "touchStart", "touchPoints": [{"x": 100, "y": 200, "id": 0}]}
{"type": "input_touch", "eventType": "touchMove", "touchPoints": [{"x": 150, "y": 250, "id": 0}]}
{"type": "input_touch", "eventType": "touchEnd", "touchPoints": []}
```

### BiDi translation

All input maps to `input.performActions`:

| Stream event | BiDi source type | BiDi action |
|-------------|-----------------|-------------|
| `mousePressed` | `pointer` (mouse) | `pointerDown` |
| `mouseReleased` | `pointer` (mouse) | `pointerUp` |
| `mouseMoved` | `pointer` (mouse) | `pointerMove` |
| `mouseWheel` | `wheel` | `scroll` |
| `keyDown` | `key` | `keyDown` |
| `keyUp` | `key` | `keyUp` |
| `touchStart` | `pointer` (touch) | `pointerDown` |
| `touchMove` | `pointer` (touch) | `pointerMove` |
| `touchEnd` | `pointer` (touch) | `pointerUp` |

Remote input goes on the **media channel** — automation commands on the main channel and human input on the media channel never contend for the same WebSocket.

### Concurrent viewers

Multiple clients connect simultaneously, all receive the same frame broadcast. Input from any viewer is forwarded (last-writer-wins, no arbitration in v1).

---

## Phase 3: Frame Retention + Video Export

### Recording options

```json
recording.start({
  "screenshots": true,
  "allFrames": false,
  "video": null
})
```

| Option | Default | Behavior |
|--------|---------|----------|
| `allFrames: false` | `false` | One frame per action (async capture or nearest from capture loop). Same density as today. |
| `allFrames: true` | — | Every capture-loop frame goes into the zip. Smooth filmstrip in Record Player. |
| `video: "path.mp4"` | `null` | Encode frames to MP4/WebM via ffmpeg pipe. Separate from zip. |

### allFrames

When the capture loop is running (streaming active or `allFrames` requested), every frame feeds into `recorder.AddScreenshot()`. Zip size: ~15MB for a 30s recording at 10fps/50KB.

When the capture loop is *not* running, `allFrames: true` starts it solely for the recorder.

### Video export

A `VideoEncoder` goroutine pipes JPEG frames to ffmpeg stdin as an MJPEG stream. ffmpeg handles re-encoding and container format. Activated by `recording.start({ video: "output.mp4" })` or `vibium record --video output.mp4`. Graceful error if ffmpeg is not installed.

---

## Use cases

- **Pair browsing** — human watches and assists AI agent in real-time
- **Remote preview** — view browser output in a separate UI (web dashboard, IDE panel)
- **Manual intervention** — human takes over when the agent gets stuck, then hands back
- **Recording filmstrip** — `allFrames: true` for smooth scrubbing in Record Player
- **Video export** — `video: "test.mp4"` for CI reports, bug reports, demos
- **Mobile testing** — inject touch events for mobile emulation
- **Combined** — zip + video + live stream all at once
