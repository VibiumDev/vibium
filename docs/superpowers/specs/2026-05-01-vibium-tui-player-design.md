# Vibium TUI Player — Design Spec

**Status:** Draft
**Date:** 2026-05-01
**Author:** Paul (with Claude)

## Goal

A terminal UI application that plays back a Vibium recording (`record.zip`) in
terminals that support the Kitty Graphics protocol (Kitty, WezTerm, Ghostty).
It is a TUI counterpart to the existing web-based player at
[player.vibium.dev](https://player.vibium.dev).

The prototype is intentionally limited in scope — it shows screenshots, an
action timeline, and a click-target overlay. Network and DOM-snapshot panes are
deferred.

## Placement

New top-level Go module:

```
/Users/paul/projects/vibium/player/
├── go.mod                   # module github.com/vibium/player
├── cmd/player/main.go       # CLI entrypoint
└── internal/
    ├── recording/           # zip parsing, trace event types, indexing
    ├── kitty/               # Kitty graphics protocol encoder + capability probe
    ├── overlay/             # click-box drawing on a JPEG/PNG frame
    └── tui/                 # Bubble Tea model, view, update
```

Self-contained — no dependency on `clicker` or any other in-repo module. Runs
as `player path/to/record.zip`. If the prototype graduates, it can be
moved/renamed.

## Stack

- **Go 1.26+**, stdlib first.
- **github.com/charmbracelet/bubbletea** — TUI event loop.
- **github.com/charmbracelet/lipgloss** — pane styling/borders/colors.
- Stdlib only for everything else: `archive/zip`, `encoding/json`,
  `image`, `image/jpeg`, `image/png`, `image/color`, `encoding/base64`,
  `bytes`, `bufio`, `os`, `time`.
- No CGo. No third-party image, base64, or terminal-graphics libraries.
- Kitty graphics implemented inline as escape-sequence emission.

## CLI

```
player path/to/record.zip
```

That is the entire CLI surface for the prototype. No flags. If the terminal
does not support Kitty graphics, exit with a clear error message and a hint
about supported terminals.

Exit codes:
- 0 — clean exit (q / Esc).
- 1 — usage error or file not found.
- 2 — recording parse error.
- 3 — terminal does not support Kitty graphics.

## Recording Format Recap

A `record.zip` contains:
- `<n>-trace.trace` — newline-delimited JSON event stream.
- `<n>-trace.network` — HAR entries (ignored by this prototype).
- `resources/<sha1>` — JPEG/PNG screenshot frames and other binary assets.

Multi-trace zips are **out of scope** — if a zip contains more than one
`*-trace.trace` entry, we exit with a parse error. A standalone chunk zip from
`stopChunk()` is valid even when its single trace file is `1-trace.trace`,
`2-trace.trace`, etc.; the loader accepts exactly one numbered trace file and
uses the matching `<n>-trace.network` if present.

Event types we consume:

| Type | Purpose |
|------|---------|
| `context-options` | Title, wallTime offset, sanity-check version. |
| `screencast-frame` | A frame: `{pageId, sha1, width, height, timestamp}`. The `sha1` field is the resource filename — used **verbatim** as the path key under `resources/`. Recordings made after vibium 26.3.18 embed the file extension (e.g. `"abc123.jpeg"`); older recordings, including the `var-parts-trace.zip` fixture, use bare hex (e.g. `"abc123"`). The loader does no rewriting. |
| `frame-snapshot` | Parsed only to index before/after screenshot resources for actions; DOM tree rendering is deferred. |
| `before` | Action start: `{callId, parentId?, beforeSnapshot?, title, method, params, pageId?, startTime, wallTime}`. |
| `after` | Action end: `{callId, afterSnapshot?, endTime}`. |
| `input` | Click coords: `{callId, point, box{x,y,width,height}}`. |

`event` and BiDi-command markers are parsed but ignored in v0.

## Data Model

Loaded once at startup, all in memory:

```go
type Recording struct {
    Title    string
    StartMs  int64                 // earliest event timestamp; t=0 of playback
    EndMs    int64                 // latest event timestamp
    Frames   []Frame               // sorted by Timestamp ascending
    Actions  []Action              // ordered by StartTime ascending
    Boxes    map[string]Rect       // callID -> bounding box (from input events)
    Resources map[string][]byte    // sha1 -> raw bytes (loaded lazily)
}

type Frame struct {
    Timestamp int64
    SHA1      string
    Width     int
    Height    int
}

type Action struct {
    CallID         string
    ParentID       string            // "" if top-level
    BeforeImageSHA string            // image resource from beforeSnapshot, if present
    AfterImageSHA  string            // image resource from afterSnapshot, if present
    Title          string            // e.g. "Element.click"
    Method         string            // e.g. "vibium:element.click"
    Params         map[string]any    // for status-bar display
    StartTime      int64
    EndTime        int64             // 0 if no matching after
    Depth          int               // computed from ParentID chain
}

type Rect struct{ X, Y, W, H int }
```

The `Resources` map starts empty; bytes are read from the zip on demand and
cached. `Frame.Timestamp`, `Action.StartTime`, `Action.EndTime`, `StartMs`, and
`EndMs` are absolute recording timestamps from the trace. The TUI's
`Model.virtTime` is the only relative time value; it is always measured in
milliseconds since `StartMs`.

For `frame-snapshot` events, the loader does not retain the DOM tree. It only
indexes `snapshot.snapshotName` to the first JPEG/PNG `resourceOverrides[].sha1`
entry, then copies that image key onto actions that reference the snapshot via
`beforeSnapshot` or `afterSnapshot`.

## Module Boundaries

Each internal package has one job and a small interface.

### `internal/recording`

```go
func Open(path string) (*Recording, error)
func (r *Recording) FrameAt(t int64) (Frame, bool)                // absolute ms; most recent frame ≤ t
func (r *Recording) ActionAt(t int64) (Action, bool)              // absolute ms; action whose [Start,End] contains t
func (r *Recording) Resource(sha1 string) ([]byte, string, error) // lazy load + cache; returns (bytes, contentType, err)
```

`Resource`'s `sha1` argument is passed through verbatim to the zip path
(prepended with `resources/`). The returned `contentType` is `"image/jpeg"`,
`"image/png"`, or `""` (unknown), derived from the filename extension if
present, otherwise from magic bytes (`FF D8` → JPEG, `89 50 4E 47` → PNG).

Owns: zip handle (kept open for the life of the program), trace parsing, index
construction.

Tested in isolation against the sample `var-parts-trace.zip` and a synthetic
fixture.

### `internal/kitty`

```go
// Supported runs the capability probe. Must be called BEFORE the TUI program
// acquires the TTY; calling it concurrently with bubbletea is undefined.
// `in` and `out` should be the controlling terminal's stdin/stdout. Pass nil
// to skip the probe and rely solely on env sniffing.
func Supported(in, out *os.File) bool

func Display(w io.Writer, png []byte, cell Cell, size Size) error
func Clear(w io.Writer) error              // clears all images

type Cell struct{ Row, Col int }           // 1-indexed terminal cell origin
type Size struct{ Cols, Rows int }         // size in terminal cells
```

Owns: chunked base64 encoding of PNG, escape-sequence emission, capability
detection. No knowledge of recordings or actions.

Capability probe sends a 1×1 RGB query (`\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\`
— the `AAAA` payload is 3 base64-decoded zero bytes representing a single
black pixel) and waits up to 200ms for an `OK` response on stdin. Falls back
to `$KITTY_WINDOW_ID` / `$TERM` (`xterm-kitty`, `xterm-ghostty`) /
`$TERM_PROGRAM` (`WezTerm`, `ghostty`) env sniffing if stdin is not a
terminal or the probe times out.

### `internal/overlay`

```go
func DrawBox(frame []byte, contentType string, box recording.Rect, frameSize image.Point) ([]byte, error)
```

Decodes the JPEG/PNG (using `contentType` to pick the decoder; falls back to
`image.Decode`'s magic-byte sniff if `contentType == ""`), draws a 2-pixel
red opaque rectangle outline by direct `SetRGBA` calls, re-encodes as PNG.
`frameSize` is the screencast-frame's `(width,height)` so we can scale the
box if the frame's pixel dimensions differ from the box coordinate system
(they should match for chromium/BiDi, but we guard for it).

Pure function. No state. Trivially testable against a known image.

### `internal/tui`

Owns the Bubble Tea model:

```go
type Model struct {
    rec        *recording.Recording
    selIdx     int           // index into rec.Actions
    frameIdx   int           // index into rec.Frames (for j/k scrubbing)
    playing    bool
    speed      float64       // 0.5, 1, 2, 4
    virtTime   int64         // current playback position, ms since rec.StartMs
    lastTick   time.Time
    imgCache     map[string][]byte // (imageSHA + ":" + selectedCallID) -> PNG bytes
    lastImageKey string            // last image+overlay+pane key sent to the terminal
    imageDirty   bool              // true when Kitty output must be refreshed
    paneSize     layout
    err          error
}
```

Update handlers map keystrokes to state mutations and mark `imageDirty` when
the selected image, overlay, or pane geometry changes. View renders the textual
layout; a render command sends Kitty escape sequences for the image pane only
when `imageDirty == true` or the computed image key differs from
`lastImageKey`.

Cache invalidation: on terminal resize, clear `imgCache`, set `imageDirty` to
`true`, and reset `lastImageKey` (size changed → must re-encode and resend at
new dimensions).

## Layout

```
┌────────────────────────────────────────┬─────────────────────┐
│                                        │ Actions             │
│                                        │ ▶ Page.navigate     │
│        [screenshot rendered            │   Element.click     │
│         via Kitty graphics,            │     Element.fill    │
│         with click-box overlay]        │   Element.text      │
│                                        │                     │
│                                        │                     │
├────────────────────────────────────────┴─────────────────────┤
│ ▶ 00:01.234 / 00:05.678  1.0×                                │
│ Element.click  selector="#login"                             │
│ ████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ │
│ space=play/pause  ←/→=step  j/k=frame  +/-=speed  q=quit     │
└──────────────────────────────────────────────────────────────┘
```

- Image pane: ~70% width × ~85% height of the terminal.
- Action list: ~30% width, same height. Indented by `Depth` for grouped
  actions. Current action highlighted with reverse video.
- Status bar: fixed 4 lines tall — time/speed line, action description,
  progress bar, key hints.

The image pane's pixel dimensions are derived from cell dimensions × the
terminal's reported cell size (Kitty's response to a CSI 14t / 16t query).
We probe once at startup; if the terminal won't tell us, we assume 8×16 px
per cell, which is wrong but produces something visible.

## Action ↔ Frame Mapping

- Selected action determines the displayed image:
  1. If the action has a click box and a `beforeSnapshot` image resource, display
     that image. Interaction handlers capture this snapshot after scrolling the
     element into view, so it is the best match for the box coordinates.
  2. Otherwise, if the action has an `afterSnapshot` image resource, display
     that image.
  3. Otherwise, pick the latest screencast frame whose timestamp is inside the
     action range `[StartTime, effectiveEnd]`, where `effectiveEnd` is
     `EndTime` when present and `StartTime` otherwise. Current Vibium recordings
     emit action-end screencast frames at `EndTime`, so this avoids selecting a
     stale pre-action screenshot.
  4. If no in-range frame exists, pick the first screencast frame after
     `StartTime`; if none exists, fall back to the most recent frame before
     `StartTime`; if still none exists, use the first frame.
- `j` / `k` step through `Frames[]` directly, decoupling the view from action
  selection. The action list highlight follows: the action whose
  `[StartTime, EndTime]` range contains the current frame's timestamp is
  selected; if none, the most recent action with `StartTime ≤ frame.Timestamp`.
- After any `j` / `k` step, `virtTime` is set to
  `Frames[frameIdx].Timestamp - StartMs`. After any `←` / `→` / `g` / `G`
  step, `virtTime` is set to `Actions[selIdx].StartTime - StartMs`. This
  keeps the playback engine in sync — pressing Space after stepping resumes
  from where the user left off, never jumping back.

## Playback Engine

A `tea.Tick` fires every 50 ms while `playing == true`. On each tick:

```
elapsed = now - lastTick
virtTime += elapsed * speed
lastTick = now
absTime = rec.StartMs + virtTime
```

Then:
- Find the action whose `[StartTime, EndTime]` contains `absTime`. Update
  `selIdx`.
- Find the frame for `absTime`. Update `frameIdx`.
- If `absTime >= EndMs`, set `playing = false` and clamp `virtTime` to
  `EndMs - StartMs`.

**Idle-gap compression:** if the next event of interest (next action start OR
next distinct frame SHA) is more than `idleGapThreshold` (2000 ms) ahead of
`absTime`, snap `virtTime` forward to `nextEventTime - idleGapPad - StartMs`
(500 ms before the next event, still stored as a relative offset). This keeps
long page loads watchable. Both thresholds are package-level constants in
`internal/tui` (`idleGapThreshold`, `idleGapPad`) so they're easy to tune.

`Space` toggles `playing`. Stepping keys (`←/→/j/k/g/G`) implicitly set
`playing = false` and adjust `virtTime` to match the new selection.

## Click-Box Overlay

When the selected action has an entry in `rec.Boxes[callID]`:

1. Resolve the selected action's displayed image as described in "Action ↔
   Frame Mapping".
2. Cache key is `imageSHA + ":" + callID`. If hit, use cached PNG.
3. Otherwise, read the resource bytes, call `overlay.DrawBox`, store the
   resulting PNG in `imgCache`.
4. Send the PNG via Kitty only if the image key or pane geometry differs from
   `lastImageKey`, or if `imageDirty` is set.

When the selected action has **no** box (e.g. `Page.navigate`):

1. Cache key is `imageSHA + ":none"`. If hit, use cached PNG.
2. Otherwise, decode the JPEG and re-encode as PNG (Kitty wants PNG via
   `f=100`), store, send.
3. As above, terminal output is skipped when the resolved image key has already
   been sent for the current pane geometry.

## Kitty Graphics Protocol

We use the file-descriptor-free path: PNG bytes, base64-encoded, chunked
across multiple escape sequences.

Per dirty image:
1. `\x1b_Ga=d,d=A\x1b\\` — clear all previously displayed images.
2. Move cursor to `(row, col)` of the image pane origin.
3. Send chunks: each chunk is **at most 4096 base64 characters** (the Kitty
   protocol's documented hard limit on chunk payload size), framed as
   `\x1b_Gf=100,a=T,m=<0|1>;<base64>\x1b\\`. `m=1` on all but the last,
   `m=0` on the final chunk. On the first chunk, also include
   `c=<cols>,r=<rows>` to fix the display size in cells.

Capability probe (run once at startup, before Bubble Tea takes over the TTY):
- Put stdin into raw mode.
- Send `\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\`.
- Read with a 200 ms timeout. Look for `\x1b_G...;OK\x1b\\`.
- Restore stdin.
- Fallback: check `$KITTY_WINDOW_ID` is set, `$TERM` is `xterm-kitty` or
  `xterm-ghostty`, or `$TERM_PROGRAM` is `WezTerm` or `ghostty`. (Same set
  as in §Module Boundaries / `internal/kitty`.)

If neither path confirms support, exit with code 3.

## Keymap

| Key | Action |
|---|---|
| `Space` | toggle play / pause |
| `←` `→` | previous / next action (single-step; pauses playback) |
| `j` `k` | previous / next screencast frame (pauses playback) |
| `g` `G` | jump to first / last action |
| `+` `-` | cycle speed: 0.5× → 1× → 2× → 4× and back |
| `r` | restart from beginning (sets virtTime=0, plays) |
| `q` `Esc` | quit cleanly |

## Error Handling

| Failure | Response |
|---|---|
| File missing / not a zip | Print error, exit 1. |
| Multi-trace zip detected | Print "multi-trace recordings not supported in this prototype", exit 2. |
| No `*-trace.trace` entry or unreadable trace | Print parse error with offending line number when available, exit 2. |
| Resource SHA1 missing from zip | Skip that frame; log to stderr at quit time. Don't crash. |
| Kitty unsupported | Print "this terminal does not support the Kitty graphics protocol; tested terminals: kitty, wezterm, ghostty", exit 3. |
| Terminal too small (< 40 cols × 15 rows) | Render a "terminal too small" message instead of the full UI; resume on resize. |

## Out of Scope (v0)

Documented here so the prototype stays small:

- Network panel (HAR display).
- DOM snapshot rendering.
- Multi-trace zips containing more than one `*-trace.trace` file.
- Multi-page recordings — we always show the temporally-nearest frame
  regardless of `pageId`, which means the click-box overlay may render on a
  screenshot from a different tab if the recording spans multiple pages.
- Sub-frames (`isMainFrame: false`).
- Console / log output.
- Click animation, easing, motion paths between frames.
- Mouse input / click-to-seek.
- Search / filter on actions.
- Export to GIF or video.

These can be layered on later if the prototype proves out.

## Testing Strategy

- `internal/recording`: the **synthetic fixture** is the primary test
  vehicle — `var-parts-trace.zip` has only 1 frame, 1 action, 0 input events,
  and 2 resources, so it cannot exercise scrolling, multi-frame stepping, or
  the click-box path at all. The synthetic fixture covers: multiple frames,
  groups (`parentId`), input events with boxes, missing resources, and both
  bare-hex and `.jpeg`-suffixed `sha1` values. It also covers accepting a
  single non-zero trace file like `1-trace.trace` while rejecting zips with
  multiple trace files. `var-parts-trace.zip` is used only as a smoke test
  confirming we can open a real recording.
- `internal/overlay`: feed a 100×100 solid-color JPEG, draw a known box,
  decode the result, assert the rectangle pixels are the box color.
- `internal/kitty`: golden-test the encoder against a known PNG byte string —
  asserts we emit the right escape-sequence framing. Capability probe is not
  unit-tested (requires a real terminal).
- `internal/tui`: Bubble Tea models are testable by sending `tea.Msg` values
  and asserting state transitions. Cover: keymap behavior, idle-gap
  compression, absolute/relative time conversion, action ↔ frame mapping,
  restart, end-of-recording, and avoiding duplicate Kitty image sends when
  ticks do not change the displayed image.

Manual smoke test: `player clients/javascript/var-parts-trace.zip` in Kitty,
WezTerm, and Ghostty.

## Open Questions

None blocking. All deferred to "v1 if it graduates."
