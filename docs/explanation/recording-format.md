# Recording File Format

## What Recording Does

Recording captures a timeline of everything that happens during a browser session — screenshots, network requests, DOM snapshots, and action groups — and packages it into a single zip file.

## Timestamps

All timestamps in the recording are **relative monotonic milliseconds** — the number of milliseconds elapsed since the recording started. This matches the Playwright trace format and ensures correct rendering in the trace viewer.

- `monotonicTime`, `startTime`, `endTime`, `timestamp`, `time` — relative ms since recording start (small values like `0`, `500`, `3200`)
- `wallTime` — absolute Unix ms (only on `context-options` and `frame-snapshot` events, for calendar time reference)
- `_monotonicTime` (network) — relative ms since recording start (same unit as all other timestamps)

## Zip Structure

A trace zip contains three kinds of entries:

```
record.zip
├── trace.trace             # Main event timeline (newline-delimited JSON)
├── trace.network           # Network events (newline-delimited JSON)
├── resources/
│   ├── page@abc123-1773879004791.jpeg  # Screenshot frames (pageId-timestamp.ext)
│   └── page@abc123-1773879004850.jpeg  # Other resources (pageId-timestamp.ext)
├── video/<context>.webm    # Native video track (when recorded; see below)
└── video/index.json        # Video manifest
```

For chunked recordings, the first chunk uses `trace.trace` / `trace.network`. Subsequent chunks use `<N>.trace` / `<N>.network` (e.g., `1.trace`, `2.trace`).

## Video Entries

When the recording carries a video track (`recording.start({ video })`,
Firefox 154+), the engine-encoded WebM is stored under `video/` with a
manifest. Video entries are additive: existing trace tooling ignores them.

```json
{"version":1,"videos":[{"file":"video/<context>.webm","context":"<browsing context id>","startedAt":1754870000123,"offsetMs":412,"width":1280,"height":720,"mimeType":"video/webm"}]}
```

`offsetMs` is the video's start relative to the recording's t0 — the
wall-clock delta from recording start to the engine's screencast
acknowledgement — and aligns the video timeline with trace event timestamps.
If the video pipeline failed, the entry carries an `error` field and the
file may be absent or partial. Chunk artifacts carry no video file; their
manifest instead records `videoRange: [startMs, endMs]`, the chunk's span
within the session video.

Resource files use Playwright-compatible naming: `<pageId>-<wallTimeMs>.<ext>` (e.g., `resources/page@abc123-1773879004791.jpeg`).

## Event Files

### `trace.trace`

Newline-delimited JSON. Each line is a self-contained event object with a `type` field. Events appear in chronological order.

**`context-options`** — Always the first event. Metadata about the recording session.

```json
{"version":8,"type":"context-options","origin":"library","libraryName":"vibium","libraryVersion":"26.3.18","browserName":"chromium","platform":"darwin","wallTime":1708000000000,"monotonicTime":0,"sdkLanguage":"javascript","title":"my test","contextId":"context@18d5a2b3c00","options":{"viewport":{"width":1280,"height":720}}}
```

| Field | Type | Description |
|-------|------|-------------|
| `libraryName` | string | Always `"vibium"` |
| `libraryVersion` | string | Version of the vibium binary (e.g., `"26.3.18"`) |
| `browserName` | string | Always `"chromium"` |
| `platform` | string | OS: `"darwin"`, `"linux"`, or `"windows"` |
| `wallTime` | number | Absolute Unix timestamp in milliseconds (calendar time) |
| `monotonicTime` | number | Relative ms since recording start (always `0` for the first event) |
| `title` | string | From `start({ title })` or `start({ name })` |
| `contextId` | string | Unique context ID for this recording session (e.g., `"context@18d5a2b3c00"`) |
| `options` | object | Browser context options. Contains `viewport` (`{width, height}`) when available. |
| `sdkLanguage` | string | Always `"javascript"` |
| `version` | number | Trace format version (currently `8`) |
| `origin` | string | Always `"library"` |

**`screencast-frame`** — A screenshot captured during recording. References an image file in `resources/` by name. Default format is JPEG (configurable via `format` and `quality` start options).

```json
{"type":"screencast-frame","pageId":"page@abcdef123","sha1":"page@abcdef123-1773879004791.jpeg","width":1280,"height":720,"timestamp":100}
```

| Field | Type | Description |
|-------|------|-------------|
| `pageId` | string | Browsing context ID of the captured page |
| `sha1` | string | Resource name in `<pageId>-<wallTimeMs>.<ext>` format (Playwright-compatible) |
| `width` | number | Screenshot width in pixels (read from image header) |
| `height` | number | Screenshot height in pixels |
| `timestamp` | number | Relative ms since recording start |

Screenshots are captured per-action in `dispatch()`, with a CAS guard to avoid flooding Chrome with concurrent capture requests.

**`frame-snapshot`** — A DOM snapshot. Contains a nested `snapshot` object with structured HTML as an array tree.

```json
{"type":"frame-snapshot","snapshot":{"callId":"call@3","snapshotName":"before@call@3","pageId":"page@abcdef123","frameId":"page@abcdef123","frameUrl":"https://example.com","doctype":"<!DOCTYPE html>","html":["HTML",{"lang":"en"},["HEAD",{}],["BODY",{},["IMG",{"src":"data:image/jpeg;base64,...","style":"width:100%"}]]],"viewport":{"width":1280,"height":720},"timestamp":200,"wallTime":200,"resourceOverrides":[{"url":"data:image/jpeg;base64,...","sha1":"a1b2c3..."}],"isMainFrame":true}}
```

| Field | Type | Description |
|-------|------|-------------|
| `snapshot.callId` | string | The `call@N` id this snapshot is associated with |
| `snapshot.snapshotName` | string | `"before@call@N"` or `"after@call@N"` |
| `snapshot.pageId` | string | Browsing context ID |
| `snapshot.frameId` | string | Frame ID (same as pageId for main frame) |
| `snapshot.frameUrl` | string | URL of the frame at snapshot time |
| `snapshot.doctype` | string | Document doctype string |
| `snapshot.html` | array | Structured DOM tree as nested arrays: `["TAG", {attrs}, ...children]` |
| `snapshot.viewport` | object | `{width, height}` of the viewport |
| `snapshot.resourceOverrides` | array | Maps resource URLs to SHA1 hashes in `resources/` |
| `snapshot.isMainFrame` | boolean | Always `true` (sub-frames not yet supported) |
| `snapshot.timestamp` | number | Relative ms since recording start |
| `snapshot.wallTime` | number | Relative ms since recording start |

**`before`** / **`after`** — Paired markers that bracket an operation. There are three kinds: actions, action groups, and BiDi commands. All share a single monotonic counter with `call@N` format.

#### Actions (auto-recorded)

Every vibium command emits a `before`/`after` pair automatically — both mutations (`click`, `fill`, `navigate`) and read-only queries (`text`, `isVisible`, `getAttribute`).

```json
{"type":"before","callId":"call@1","startTime":300,"class":"Page","method":"vibium:page.navigate","pageId":"page@abcdef123","params":{"url":"https://example.com"},"title":"Page.navigate"}
{"type":"after","callId":"call@1","endTime":400,"afterSnapshot":"after@call@1"}
{"type":"before","callId":"call@2","startTime":500,"class":"Element","method":"vibium:element.click","pageId":"page@abcdef123","params":{"selector":"#login"},"beforeSnapshot":"before@call@2","title":"Element.click"}
{"type":"input","callId":"call@2","point":{"x":640,"y":360},"box":{"x":600,"y":340,"width":80,"height":40}}
{"type":"after","callId":"call@2","endTime":600}
{"type":"before","callId":"call@3","startTime":700,"class":"Element","method":"vibium:element.text","pageId":"page@abcdef123","params":{"selector":".result"},"title":"Element.text"}
{"type":"after","callId":"call@3","endTime":750,"afterSnapshot":"after@call@3"}
```

**`before` fields:**

| Field | Type | Description |
|-------|------|-------------|
| `callId` | string | `call@<N>` — same id for both `before` and `after`. `N` is a monotonic counter shared across actions, groups, and BiDi commands. |
| `title` | string | Human-readable name like `Element.click`, `Page.navigate`, `Element.text`. Derived from the vibium method by `apiNameFromMethod()`. |
| `class` | string | `Element`, `Page`, `Browser`, `BrowserContext`, `Network`, `Dialog`, etc. |
| `method` | string | The raw vibium method (e.g., `vibium:element.click`, `vibium:element.text`). |
| `params` | object | The command parameters as sent by the client. |
| `startTime` | number | Relative ms since recording start. |
| `pageId` | string | Browsing context ID of the page the action targets. Present on actions, absent on groups. |
| `parentId` | string | `call@<N>` of the enclosing action group, if the action is inside a `startGroup()`/`stopGroup()` span. Absent for top-level actions. |
| `beforeSnapshot` | string | `"before@call@<N>"` — references the `snapshotName` of a `frame-snapshot` event captured before the action ran. Present on interaction actions (`click`, `fill`, `hover`, etc.) that resolve an element — the snapshot is taken after scrolling the element into view. Click-like actions (`click`, `dblclick`, `hover`, `tap`, `check`, `uncheck`, `dragTo`) have `beforeSnapshot` only. Fill-like actions (`fill`, `type`, `press`, `clear`, `selectOption`) have both `beforeSnapshot` and `afterSnapshot`. Query actions (`find`, `text`, `navigate`, etc.) have `afterSnapshot` only. |

**`input` fields** (emitted for element-targeting actions only):

| Field | Type | Description |
|-------|------|-------------|
| `callId` | string | `call@<N>` — matches the corresponding `before`/`after` pair. |
| `point` | object | `{x, y}` — center of the resolved element's bounding box (viewport coordinates). Compatible with Playwright's trace viewer click-dot overlay. |
| `box` | object | `{x, y, width, height}` — full bounding box of the resolved element (viewport coordinates). Used by the record player to draw a highlight rectangle. |

Present for actions that resolve an element (`click`, `fill`, `hover`, `type`, `check`, etc.). Absent for non-element actions (`page.navigate`, `page.eval`, etc.) and action groups.

**`after` fields:**

| Field | Type | Description |
|-------|------|-------------|
| `callId` | string | `call@<N>` — matches the corresponding `before` event. |
| `endTime` | number | Relative ms since recording start. |
| `afterSnapshot` | string | `"after@call@<N>"` — references the `snapshotName` of a `frame-snapshot` event captured after the action completed. Present on fill-like actions (`fill`, `type`, `press`, `clear`, `selectOption`) and all non-interaction actions (`navigate`, `find`, `text`, etc.). Absent on click-like actions (`click`, `dblclick`, `hover`, `tap`, `check`, `uncheck`, `dragTo`) which use `beforeSnapshot` only, and absent on groups. |

The `dispatch()` wrapper in the router records these markers — every vibium command that goes through `dispatch()` gets recorded. Recording commands themselves (`recording.start`, `recording.stop`, etc.) are excluded since they control recording.

#### Action groups (user-defined)

Groups are named spans from `startGroup()` / `stopGroup()`. They wrap multiple actions under a single label in the timeline.

```json
{"type":"before","callId":"call@4","startTime":300,"class":"Tracing","method":"tracingGroup","params":{"name":"login flow"},"title":"login flow"}
{"type":"before","callId":"call@5","startTime":350,"class":"Element","method":"vibium:element.fill","pageId":"page@abcdef123","parentId":"call@4","params":{"selector":"#user","value":"admin"},"title":"Element.fill"}
{"type":"after","callId":"call@5","endTime":380,"afterSnapshot":"after@call@5"}
{"type":"after","callId":"call@4","endTime":500}
```

Groups are nestable. The `before` and `after` events share the same `call@N` id. Actions inside a group have a `parentId` field pointing to the group's `callId` (see `parentId` in the field table above).

#### BiDi commands (opt-in)

When recording is started with `bidi: true`, every raw BiDi command sent to the browser via `sendInternalCommand` is also recorded. This is useful for debugging low-level protocol issues.

```json
{"type":"before","callId":"call@6","title":"browsingContext.navigate","class":"BiDi","method":"browsingContext.navigate","params":{"context":"ABC123","url":"https://example.com"},"startTime":350}
{"type":"after","callId":"call@6","endTime":390}
```

| Field | Type | Description |
|-------|------|-------------|
| `callId` | string | `call@<N>` — same monotonic counter as actions and groups. |
| `title` | string | The BiDi method name (e.g., `browsingContext.navigate`). |
| `class` | string | Always `"BiDi"`. |

BiDi markers nest inside action markers — a single vibium command (like `Page.navigate`) may produce several BiDi commands internally.

**`event`** — A BiDi browser event (context creation, dialog, log entry, etc.) recorded as-is.

```json
{"type":"event","method":"browsingContext.contextCreated","params":{...},"time":150,"class":"BrowserContext"}
```

These are all non-network BiDi events that flow through the router while recording is active.

### `trace.network`

Newline-delimited JSON. Each line is a `resource-snapshot` event containing a HAR 1.2 Entry object. This format is compatible with both [player.vibium.dev](https://player.vibium.dev) and the standard [Playwright trace viewer](https://trace.playwright.dev).

Raw BiDi network events (`network.beforeRequestSent`, `network.responseCompleted`, `network.fetchError`) are automatically correlated by request ID and transformed into HAR entries during recording.

```json
{"type":"resource-snapshot","snapshot":{"startedDateTime":"2024-02-15T10:00:00.000Z","time":45,"request":{"method":"GET","url":"https://example.com/","httpVersion":"HTTP/1.1","cookies":[],"headers":[{"name":"Host","value":"example.com"}],"queryString":[],"headersSize":250,"bodySize":0},"response":{"status":200,"statusText":"OK","httpVersion":"HTTP/1.1","cookies":[],"headers":[{"name":"Content-Type","value":"text/html"}],"content":{"size":1234,"mimeType":"text/html"},"redirectURL":"","headersSize":-1,"bodySize":1234},"cache":{},"timings":{"send":-1,"wait":45,"receive":-1},"_monotonicTime":300,"_frameref":"page@abc123"}}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"resource-snapshot"` |
| `snapshot.startedDateTime` | string | ISO 8601 timestamp of the request |
| `snapshot.time` | number | Total elapsed time in milliseconds |
| `snapshot.request` | object | HAR request with `method`, `url`, `httpVersion`, `headers`, `queryString`, etc. |
| `snapshot.response` | object | HAR response with `status`, `statusText`, `headers`, `content`, etc. |
| `snapshot.cache` | object | Always `{}` |
| `snapshot.timings` | object | `{send, wait, receive}` — `wait` is the request→response delta |
| `snapshot._monotonicTime` | number | Relative ms since recording start (same unit as all other timestamps) |
| `snapshot._frameref` | string | BiDi browsing context ID |

For failed requests (`network.fetchError`), the response has `status: 0` and a `_failureText` field with the error message.

## Resources Directory

Binary assets referenced by name from the event files. Resource files include their image extension (`.jpeg` or `.png`).

| Content | Source |
|---------|--------|
| Screenshot frames (JPEG by default, or PNG) | `screencast-frame` events |
| DOM snapshot images | `frame-snapshot` resourceOverrides |

Resource names use Playwright-compatible format: `<pageId>-<wallTimeMs>.<ext>` (e.g., `resources/page@abc123-1773879004791.jpeg`). Each screenshot gets a unique name based on its page and capture time.

## How Recording Works

### Session lifecycle

```
Client                       Proxy                            Browser
  │                            │                                 │
  │  recording.start           │                                 │
  ├───────────────────────────>│  Create Recorder       │
  │                            │  Start screenshot goroutine     │
  │                            │                                 │
  │  page.navigate, click, …   │                                 │
  ├───────────────────────────>│  dispatch() (see below)         │
  │                            │                                 │
  │                            │<─── BiDi events ────────────────│
  │                            │  RecordBidiEvent()              │
  │                            │  (network → .network,           │
  │                            │   other → .trace)               │
  │                            │                                 │
  │  recording.stop            │                                 │
  ├───────────────────────────>│  Stop screenshot goroutine      │
  │                            │  Capture final DOM snapshot     │
  │                            │  Build zip                      │
  │<──── zip (base64 or file) ─│                                 │
```

### Inside `dispatch()` for a single command

```
dispatch(session, cmd, handler)
  │
  ├── RecordAction(method, params)       // before marker
  │
  │   handler(session, cmd)
  │     │
  │     ├── sendInternalCommand ────────────────> Browser
  │     │   ├── [if bidi: true] RecordBidiCommand()
  │     │   │   ···wait for response···
  │     │   └── [if bidi: true] RecordBidiCommandEnd()
  │     │
  │     ├── sendInternalCommand ────────────────> Browser
  │     │   └── (same pattern, one per BiDi call)
  │     │
  │     └── sendSuccess / sendError
  │
  └── RecordActionEnd()                  // after marker
```

The `dispatch()` wrapper records action markers around every vibium command handler. Inside the handler, `sendInternalCommand` optionally records BiDi command markers when `bidi: true` was passed to `recording.start`. The `routeBrowserToClient` loop independently records BiDi *events* (browser-initiated messages like context creation, network activity, log entries) — these are passive observations that require no extra round-trips. Screenshots are captured per-action in `dispatch()` with a CAS guard.

## Chunks vs. Single Trace

By default, `start()` → `stop()` produces one zip covering the entire session.

Chunks split a recording into segments without stopping the recording:

```
start()  ──────────────────────────────────────────────  stop()
           │                    │                    │
      events A             events B             events C
           │                    │                    │
       stopChunk()         startChunk()          (final)
       → zip with A        stopChunk()
                           → zip with B
                                                → zip with C
```

Each chunk gets its own `context-options` header and chunk index. The monotonic base is reset on each `startChunk()`, so timestamps within each chunk start from `0`. Resources (screenshots, snapshots) are shared across chunks — the `resources/` map is not cleared on `startChunk()`, so a stopChunk zip may contain resources referenced by earlier chunks too.

## Viewing Recordings

Open a recording in [Record Player](https://player.vibium.dev):

1. Go to [player.vibium.dev](https://player.vibium.dev)
2. Drop your `record.zip` file onto the page

The viewer renders a timeline with screenshots, actions, network waterfall, and DOM snapshots. Every vibium command appears as an individual action in the timeline (e.g., `Page.navigate`, `Element.click`, `Element.text`). Action groups from `startGroup()`/`stopGroup()` appear as labeled spans that wrap multiple actions. With `bidi: true`, raw BiDi commands are also visible as nested entries within their parent actions.
