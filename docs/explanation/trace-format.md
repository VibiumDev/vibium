# Trace File Format

## What Tracing Does

Tracing captures a timeline of everything that happens during a browser session — screenshots, network requests, DOM snapshots, and action groups — and packages it into a single zip file.

## Zip Structure

A trace zip contains three kinds of entries:

```
trace.zip
├── 0-trace.trace           # Main event timeline (newline-delimited JSON)
├── 0-trace.network         # Network events (newline-delimited JSON)
└── resources/
    ├── a1b2c3d4e5f6...     # Screenshot frames (named by SHA1, no extension)
    └── f6a7b8c9d0e1...     # Other resources (named by SHA1, no extension)
```

The number prefix (`0-`) is the chunk index. Each `stopChunk()` call produces a zip with the current chunk's events. The first chunk is `0`, the next is `1`, etc.

Resource files have **no extension** — they are stored as bare SHA1 hex hashes (e.g., `resources/a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0`).

## Event Files

### `<n>-trace.trace`

Newline-delimited JSON. Each line is a self-contained event object with a `type` field. Events appear in chronological order.

**`context-options`** — Always the first event. Metadata about the recording session.

```json
{"type":"context-options","browserName":"chromium","platform":"darwin","wallTime":1708000000000,"monotonicTime":1708000000000,"title":"my test","options":{},"sdkLanguage":"javascript","version":8,"origin":"library"}
```

| Field | Type | Description |
|-------|------|-------------|
| `browserName` | string | Always `"chromium"` |
| `platform` | string | OS: `"darwin"`, `"linux"`, or `"windows"` |
| `wallTime` | number | Unix timestamp in milliseconds |
| `monotonicTime` | number | Monotonic timestamp in milliseconds (same as `wallTime`) |
| `title` | string | From `start({ title })` or `start({ name })` |
| `options` | object | Browser context options (currently `{}`) |
| `sdkLanguage` | string | Always `"javascript"` |
| `version` | number | Trace format version (currently `8`) |
| `origin` | string | Always `"library"` |

**`screencast-frame`** — A screenshot captured during recording. References an image file in `resources/` by SHA1 hash. Default format is JPEG (configurable via `format` and `quality` start options).

```json
{"type":"screencast-frame","pageId":"ABCDEF123","sha1":"a1b2c3d4e5f6...","width":1280,"height":720,"timestamp":1708000000100}
```

| Field | Type | Description |
|-------|------|-------------|
| `pageId` | string | Browsing context ID of the captured page |
| `sha1` | string | Lowercase hex SHA1 of the image data |
| `width` | number | Screenshot width in pixels (read from image header) |
| `height` | number | Screenshot height in pixels |
| `timestamp` | number | Unix ms when the screenshot was taken |

Screenshots are captured per-action in `dispatch()`, with a CAS guard to avoid flooding Chrome with concurrent capture requests. Identical frames are deduplicated by SHA1 — if the page doesn't change, only one image is stored in `resources/`.

**`frame-snapshot`** — A DOM snapshot. Contains a nested `snapshot` object with structured HTML as an array tree.

```json
{"type":"frame-snapshot","snapshot":{"callId":"call@3","snapshotName":"before@call@3","pageId":"ABCDEF123","frameId":"ABCDEF123","frameUrl":"https://example.com","doctype":"<!DOCTYPE html>","html":["HTML",{"lang":"en"},["HEAD",{}],["BODY",{},["IMG",{"src":"data:image/jpeg;base64,...","style":"width:100%"}]]],"viewport":{"width":1280,"height":720},"timestamp":1708000000200,"wallTime":1708000000200,"resourceOverrides":[{"url":"data:image/jpeg;base64,...","sha1":"a1b2c3..."}],"isMainFrame":true}}
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
| `snapshot.timestamp` | number | Unix ms when the snapshot was taken |
| `snapshot.wallTime` | number | Wall clock time in ms |

**`before`** / **`after`** — Paired markers that bracket an operation. There are three kinds: actions, action groups, and BiDi commands. All share a single monotonic counter with `call@N` format.

#### Actions (auto-recorded)

Every vibium command emits a `before`/`after` pair automatically — both mutations (`click`, `fill`, `navigate`) and read-only queries (`text`, `isVisible`, `getAttribute`).

```json
{"type":"before","callId":"call@1","title":"Page.navigate","class":"Page","method":"vibium:page.navigate","params":{"url":"https://example.com"},"wallTime":1708000000300,"startTime":1708000000300}
{"type":"after","callId":"call@1","endTime":1708000000400}
{"type":"before","callId":"call@2","title":"Element.click","class":"Element","method":"vibium:click","params":{"selector":"#login"},"wallTime":1708000000500,"startTime":1708000000500}
{"type":"after","callId":"call@2","endTime":1708000000600}
{"type":"before","callId":"call@3","title":"Element.text","class":"Element","method":"vibium:el.text","params":{"selector":".result"},"wallTime":1708000000700,"startTime":1708000000700}
{"type":"after","callId":"call@3","endTime":1708000000750}
```

| Field | Type | Description |
|-------|------|-------------|
| `callId` | string | `call@<N>` — same id for both `before` and `after`. `N` is a monotonic counter shared across actions, groups, and BiDi commands. |
| `title` | string | Human-readable name like `Element.click`, `Page.navigate`, `Element.text`. Derived from the vibium method by `apiNameFromMethod()`. |
| `class` | string | `Element`, `Page`, `Browser`, `BrowserContext`, `Network`, `Dialog`, etc. |
| `method` | string | The raw vibium method (e.g., `vibium:click`, `vibium:el.text`). |
| `params` | object | The command parameters as sent by the client. |

The `dispatch()` wrapper in the router records these markers — every vibium command that goes through `dispatch()` gets traced. Tracing commands themselves (`tracing.start`, `tracing.stop`, etc.) are excluded since they control tracing.

#### Action groups (user-defined)

Groups are named spans from `startGroup()` / `stopGroup()`. They wrap multiple actions under a single label in the timeline.

```json
{"type":"before","callId":"call@4","title":"login flow","class":"Tracing","method":"group","params":{"name":"login flow"},"wallTime":1708000000300,"startTime":1708000000300}
{"type":"after","callId":"call@4","endTime":1708000000500}
```

Groups are nestable. The `before` and `after` events share the same `call@N` id. Actions inside a group have a `parentId` field pointing to the group's `callId`.

#### BiDi commands (opt-in)

When tracing is started with `bidi: true`, every raw BiDi command sent to the browser via `sendInternalCommand` is also recorded. This is useful for debugging low-level protocol issues.

```json
{"type":"before","callId":"call@5","title":"browsingContext.navigate","class":"BiDi","method":"browsingContext.navigate","params":{"context":"ABC123","url":"https://example.com"},"wallTime":1708000000350,"startTime":1708000000350}
{"type":"after","callId":"call@5","endTime":1708000000390}
```

| Field | Type | Description |
|-------|------|-------------|
| `callId` | string | `call@<N>` — same monotonic counter as actions and groups. |
| `title` | string | The BiDi method name (e.g., `browsingContext.navigate`). |
| `class` | string | Always `"BiDi"`. |

BiDi markers nest inside action markers — a single vibium command (like `Page.navigate`) may produce several BiDi commands internally.

**`event`** — A BiDi browser event (context creation, dialog, log entry, etc.) recorded as-is.

```json
{"type":"event","method":"browsingContext.contextCreated","params":{...},"time":1708000000150,"class":"BrowserContext"}
```

These are all non-network BiDi events that flow through the router while tracing is active.

### `<n>-trace.network`

Newline-delimited JSON. Each line is a `resource-snapshot` event containing a HAR 1.2 Entry object. This format is compatible with both [trace.vibium.dev](https://trace.vibium.dev) and the standard [Playwright trace viewer](https://trace.playwright.dev).

Raw BiDi network events (`network.beforeRequestSent`, `network.responseCompleted`, `network.fetchError`) are automatically correlated by request ID and transformed into HAR entries during recording.

```json
{"type":"resource-snapshot","snapshot":{"startedDateTime":"2024-02-15T10:00:00.000Z","time":45,"request":{"method":"GET","url":"https://example.com/","httpVersion":"HTTP/1.1","cookies":[],"headers":[{"name":"Host","value":"example.com"}],"queryString":[],"headersSize":250,"bodySize":0},"response":{"status":200,"statusText":"OK","httpVersion":"HTTP/1.1","cookies":[],"headers":[{"name":"Content-Type","value":"text/html"}],"content":{"size":1234,"mimeType":"text/html"},"redirectURL":"","headersSize":-1,"bodySize":1234},"cache":{},"timings":{"send":-1,"wait":45,"receive":-1},"_monotonicTime":1708000000.3,"_frameref":"ABC123"}}
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
| `snapshot._monotonicTime` | number | Request start time in seconds (epoch) |
| `snapshot._frameref` | string | BiDi browsing context ID |

For failed requests (`network.fetchError`), the response has `status: 0` and a `_failureText` field with the error message.

## Resources Directory

Binary assets referenced by SHA1 hash from the event files. Files have **no extension** — they are stored as bare hex SHA1 hashes.

| Content | Source |
|---------|--------|
| Screenshot frames (JPEG by default, or PNG) | `screencast-frame` events |
| DOM snapshot images | `frame-snapshot` resourceOverrides |

The file name is the full lowercase hex SHA1 hash of the content (e.g., `resources/a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0`). This provides natural deduplication — if two screenshots are pixel-identical, they share one file.

## How Recording Works

### Session lifecycle

```
Client                       Proxy                            Browser
  │                            │                                 │
  │  tracing.start             │                                 │
  ├───────────────────────────>│  Create TraceRecorder           │
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
  │  tracing.stop              │                                 │
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

The `dispatch()` wrapper records action markers around every vibium command handler. Inside the handler, `sendInternalCommand` optionally records BiDi command markers when `bidi: true` was passed to `tracing.start`. The `routeBrowserToClient` loop independently records BiDi *events* (browser-initiated messages like context creation, network activity, log entries) — these are passive observations that require no extra round-trips. Screenshots are captured per-action in `dispatch()` with a CAS guard.

## Chunks vs. Single Trace

By default, `start()` → `stop()` produces one zip covering the entire session.

Chunks split a trace into segments without stopping the recording:

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

Each chunk gets its own `context-options` header and chunk index. Resources (screenshots, snapshots) are shared across chunks — the `resources/` map is not cleared on `startChunk()`, so a stopChunk zip may contain resources referenced by earlier chunks too.

## Viewing Traces

Open a trace in [Vibium Trace](https://trace.vibium.dev):

1. Go to [trace.vibium.dev](https://trace.vibium.dev)
2. Drop your `trace.zip` file onto the page

The viewer renders a timeline with screenshots, actions, network waterfall, and DOM snapshots. Every vibium command appears as an individual action in the timeline (e.g., `Page.navigate`, `Element.click`, `Element.text`). Action groups from `startGroup()`/`stopGroup()` appear as labeled spans that wrap multiple actions. With `bidi: true`, raw BiDi commands are also visible as nested entries within their parent actions.
