# Source Linking Specification

> **Status: Draft.** This document describes proposed behavior. It does
> not reflect what is currently implemented in vibium. Do not rely on it
> as documentation of existing functionality.

Link recording trace events to the automation source code that triggered them, enabling step-through navigation in the Playwright trace viewer.

---

## 1. Overview

When a user writes an automation script (JS, Python, Java, etc.) and records a trace, each action in the trace should link back to the exact file and line in their script. The Playwright trace viewer already supports this — clicking an action highlights the source line, and vice versa.

Vibium needs three things to enable this:

1. **Clients send source locations** with each command
2. **Daemon writes locations** into trace events and a `trace.stacks` file
3. **Optionally embed source files** in the trace zip

---

## 2. Protocol Change

### Request Format

Add an optional `source` field to the JSON-RPC request params. Clients include this when recording is active.

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "browser_click",
    "arguments": { "selector": "#login" },
    "source": [
      { "file": "/Users/dev/test.mjs", "line": 42, "column": 23, "function": "runLogin" }
    ]
  }
}
```

### Source Frame Object

```
{
  "file":     string  (required) — absolute path to source file
  "line":     number  (required) — 1-based line number
  "column":   number  (optional) — 0-based column number, default 0
  "function": string  (optional) — function/method name, default ""
}
```

The `source` field is an array of frames ordered from innermost (user call site) to outermost (entry point). Most calls will have 1-3 frames.

When `source` is absent, the daemon behaves exactly as today — no source linking in the trace.

---

## 3. Client Implementations

Each client captures the call site at the point where the user's code calls the API, before sending the command to the daemon.

### JavaScript

```javascript
// Capture call site from V8 stack trace
function captureSource() {
  const err = {};
  Error.captureStackTrace(err, captureSource);
  const frame = err.stack.split('\n')[1]; // first frame outside this function
  const match = frame.match(/at (?:(.+?) )?\(?(.+?):(\d+):(\d+)\)?/);
  if (!match) return [];
  return [{
    file: match[2],
    line: parseInt(match[3], 10),
    column: parseInt(match[4], 10),
    function: match[1] || ''
  }];
}
```

### Python

```python
import traceback

def capture_source():
    frames = traceback.extract_stack()
    # Walk backwards to find the first frame outside the vibium package
    for frame in reversed(frames):
        if '/vibium/' not in frame.filename:
            return [{"file": frame.filename, "line": frame.lineno,
                      "column": 0, "function": frame.name}]
    return []
```

### Java

```java
static List<SourceFrame> captureSource() {
    StackTraceElement[] stack = Thread.currentThread().getStackTrace();
    // Skip getStackTrace() and captureSource() frames, find user code
    for (StackTraceElement el : stack) {
        if (!el.getClassName().startsWith("com.vibium.")) {
            return List.of(new SourceFrame(
                el.getFileName(), el.getLineNumber(), 0, el.getMethodName()));
        }
    }
    return List.of();
}
```

### Other Languages

Any language that can produce `(file, line)` tuples works:

| Language | API |
|----------|-----|
| Go | `runtime.Callers()` + `runtime.CallersFrames()` |
| Rust | `std::backtrace::Backtrace` |
| Swift | `#file`, `#line` literals |
| Nim | `getStackTraceEntries()` |

### CLI

CLI invocations from agents are ad-hoc — there is no script file. Source linking does not apply. The CLI does not send `source` frames.

---

## 4. Daemon Changes

### 4.1 Parse Source from Request

When the daemon receives a request with a `source` field, pass it through to the recorder.

In `daemon/router.go`, extend `ToolsCallParams`:

```go
type ToolsCallParams struct {
    Name      string          `json:"name"`
    Arguments json.RawMessage `json:"arguments"`
    Source    []SourceFrame   `json:"source,omitempty"`
}

type SourceFrame struct {
    File     string `json:"file"`
    Line     int    `json:"line"`
    Column   int    `json:"column,omitempty"`
    Function string `json:"function,omitempty"`
}
```

### 4.2 Write Source into Trace Events

In `recording.go`, add `source` to `before` events when present:

```go
event := recordEvent{
    "type":      "before",
    "callId":    callId,
    "startTime": now,
    "title":     title,
    "class":     class,
    "method":    method,
    "params":    params,
}
if len(source) > 0 {
    event["stack"] = source  // "stack" for Playwright viewer compatibility
}
```

The field must be named `stack` in the trace output — that's what the Playwright trace viewer reads.

### 4.3 Build `trace.stacks` File

Add a `trace.stacks` entry to the recording zip alongside `<n>-trace.trace` and `<n>-trace.network`.

Format (compact, matches Playwright):

```json
{
  "files": ["/Users/dev/test.mjs", "/Users/dev/helpers.mjs"],
  "stacks": [
    [1, [[0, 42, 23, "runLogin"]]],
    [2, [[0, 45, 5, ""], [1, 10, 0, "helper"]]]
  ]
}
```

- `files`: deduplicated array of file paths, indexed by ordinal
- `stacks`: array of `[callIdNumber, frames]` tuples
  - `callIdNumber`: numeric part of the callId (e.g., `7` from `"call@7"`)
  - Each frame: `[fileOrdinal, line, column, functionName]`

In `buildZipLocked()`:

```go
// Build trace.stacks from collected source frames
stacksName := fmt.Sprintf("%d-trace.stacks", t.chunkIndex)
sw, _ := zw.Create(stacksName)
sw.Write(t.buildStacks())
```

### 4.4 Embed Source Files (Optional)

When the recording is started with `sources: true`, the daemon reads each unique file referenced in source frames and embeds it in the zip.

```go
if t.options.Sources {
    for _, filePath := range t.uniqueSourceFiles() {
        data, err := os.ReadFile(filePath)
        if err != nil {
            continue
        }
        hash := sha1Hex(data)
        rw, _ := zw.Create("resources/src@" + hash + ".txt")
        rw.Write(data)
    }
}
```

The `resources/src@<sha1>.txt` naming matches what the Playwright trace viewer expects.

---

## 5. Recording Options Change

Add `sources` to `RecordingStartOptions` (already defined but not wired):

```go
type RecordingStartOptions struct {
    // ... existing fields ...
    Sources bool `json:"sources"` // embed automation source files in trace
}
```

Client API:

```javascript
await recording.start({ screenshots: true, sources: true });
```

```python
recording.start(screenshots=True, sources=True)
```

```java
recording.start(new RecordingOptions().screenshots(true).sources(true));
```

---

## 6. Trace Viewer Compatibility

The Playwright trace viewer (trace.playwright.dev) reads these fields:

| What | Where | Format |
|------|-------|--------|
| Per-action source | `before` event `stack` field | `[{file, line, column, function}]` |
| Compressed stacks | `<n>-trace.stacks` file | `{files: [...], stacks: [...]}` |
| Embedded sources | `resources/src@<sha1>.txt` | Raw source file content |
| Source detection | Zip entry names containing `src@` | Sets `hasSource` flag in viewer |

The viewer prioritizes:
1. `trace.stacks` compressed format (loaded after trace events)
2. Falls back to inline `stack` field on `before` events
3. Source files loaded on demand from `resources/src@*.txt`

Both `stack` and `trace.stacks` can coexist. The viewer merges them: `action.stack = action.stack || callMetadata.get(action.callId)`.

---

## 7. Implementation Order

1. **Daemon**: Parse `source` from requests, write `stack` field on `before` events
2. **Daemon**: Build `trace.stacks` file in zip
3. **JS client**: Capture and send source frames
4. **Verify**: Record a trace with source, open in trace.playwright.dev, confirm source linking works
5. **Daemon**: Implement `sources: true` file embedding
6. **Python client**: Capture and send source frames
7. **Java client**: Capture and send source frames

---

## 8. Wire Format Example

Full round-trip for a single action:

**Client sends:**
```json
{
  "jsonrpc": "2.0", "id": 5,
  "method": "tools/call",
  "params": {
    "name": "browser_click",
    "arguments": {"selector": "#login"},
    "source": [{"file": "/Users/dev/test.mjs", "line": 42, "column": 23, "function": ""}]
  }
}
```

**Trace event written:**
```json
{
  "type": "before",
  "callId": "call@5",
  "startTime": 1773819270479,
  "title": "click #login",
  "class": "Element",
  "method": "vibium:element.click",
  "params": {"selector": "#login"},
  "stack": [{"file": "/Users/dev/test.mjs", "line": 42, "column": 23, "function": ""}]
}
```

**trace.stacks entry:**
```json
{
  "files": ["/Users/dev/test.mjs"],
  "stacks": [[5, [[0, 42, 23, ""]]]]
}
```

**Embedded source (if sources: true):**
```
resources/src@a1b2c3d4e5f6.txt  →  contents of /Users/dev/test.mjs
```
