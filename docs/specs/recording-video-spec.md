# Spec: video as a recording track

Folds screen recording into `recording.start()` / `recording.stop()`
and retires the separate `page.screencast` API. Resolves #310; #311's
CLI/MCP surfaces then target this shape instead of the screencast one.

## Why one API

- **One artifact.** The record (`record.zip`) is the unit of evidence
  — "not done until the results are checked in" needs one atomic
  thing to check in. A loose video beside a zip is evidence separated
  from its case file. (Playwright ships this seam: trace.zip carries
  filmstrip frames while the video lands loose in a `videosDir`;
  users trip on it.)
- **One entry point for agents.** An agent that knows
  `recording.start()` discovers video by reading one options object.
  A sibling namespace is a second thing to know exists.
- **Congruent grammar.** `RecordingStartOptions` is already a set of
  capture-track toggles (`screenshots`, `snapshots`, `sources`,
  `bidi`). Video is one more track, not a new concept.

## API

```ts
// JS — Python/Java mirror naming conventions
await recording.start();   // video when the engine can, record.zip, crash-durable — zero config
await recording.start({ video: true });          // require video: fails fast if the engine can't
await recording.start({ video: { width: 1280, height: 720, frameRate: 30 }, path: 'runs/login.zip' });
await recording.stop();                          // delivers to the declared path
await recording.stop({ path: 'checkout-failure-run1234.zip' });  // override wins
```

**Declare at start, refine at stop.** `start.path` declares the
destination and anchors crash durability (below); `stop.path`
overrides it — outcome-aware naming stays possible because the final
word comes last. Precedence: `stop.path` > `start.path` > bytes mode
(neither declared). Clients resolve relative paths before the wire —
the daemon's working directory is not the caller's.

- `video?: boolean | { width?, height?, frameRate? }` — three
  states. **Omitted (the default): record video if the engine can**
  — the "browser visible by default" philosophy extended to
  after-the-fact: see what the AI did. Where the engine can't, the
  recording proceeds and the result says so (below) — annotated,
  never silent. **Explicit `true`: strict** — fails fast if the
  engine can't deliver. **Explicit `false`: off.** Dimensions
  default to the viewport — faithful to what was on screen, and it
  eliminates the letterbox padding a fixed pre-layout size causes.
- Video-specific options nest in the object — the flat `format` /
  `quality` options stay screenshot-only.
- `audio` is reserved, not exposed: Firefox 154 rejects it
  (`unsupported operation`). Add when an engine accepts it.
- `recording.stop()` is unchanged in shape. The zip gains the track.

## Wire protocol

`vibium:recording.start` gains the `video` param (same bool-or-object
shape). The router starts `browsingContext.startScreencast` when the
recording starts, stops it when the recording stops, and moves the
browser-written file into the zip. `vibium:screencast.start/stop` are
removed — they shipped after the last tagged release, so nothing
public breaks.

## Zip layout and sync contract

Correction from source: record.zip is a **Playwright trace-format**
zip (`trace.trace` + `trace.network` + resources), and the MCP tool
advertises trace-viewer compatibility. That compatibility is a
requirement, not an accident: any tool that reads Playwright traces
reads vibium records for free.

Video is therefore **additive**: entries Playwright's viewer ignores
and vibium's surfaces read.

```
record.zip
├── trace.trace            # unchanged
├── trace.network          # unchanged
├── video/<context>.webm   # the recording
└── video/index.json       # vibium's video metadata
```

```json
{ "videos": [ {
  "file": "video/<context>.webm",
  "context": "<browsing context id>",
  "startedAt": 1754870000123,
  "offsetMs": 412,
  "width": 1280, "height": 720,
  "mimeType": "video/webm"
} ] }
```

An array from day one — v1 writes one entry; multi-cam appends.
`offsetMs` is the video's start relative to the recording's t0 — the
number the viewer needs to scrub steps and video in sync (trace
events already carry timestamps). Acceptance test: a video-carrying
record still opens in Playwright's trace viewer unchanged.

## Format: WebM inside, forever

The video is stored as the engine produced it. Every consumer of a
record's video is a browser surface (the local viewer, the web UI,
agents reading bytes) and browsers play WebM natively — so **capture
never depends on an encoder**. MP4 is an export concern for handing a
human a loose file (`vibium record export video record.zip -o
clip.mp4` — separate spec; that is where ffmpeg lives, and where the
demo-clip flow lands: record with only the video track on, then
export).

## Engine and connection semantics

- **Firefox 154+, local:** works.
- **Chrome, or Firefox < 154:** with the default (video omitted),
  the recording runs without video and the stop result carries
  `videoUnavailable` with the engine's reason — degradation is
  annotated, never silent. With explicit `video: true`, `start`
  **fails fast** with today's explanatory message (Chrome: not
  implemented; Firefox: requires 154+, and the message names the
  exact install command). An explicit request the engine can't honor
  is an error — a record that quietly lacks the video it promised is
  the same lie as a WebM renamed to `.mp4`.
- **Remote connections (`--connect`):** video errors with the
  existing remote-screencast message (the browser writes the file on
  the remote host). Unchanged from today; revisit if grids grow a
  BiDi screencast story.

## What the camera sees: context binding

The video records the browsing context that was active at
`recording.start()` and **does not follow focus** — a `window.open`
that takes focus never appears in the video. Consequences the spec
commits to:

- The manifest's video block gains `"context"`, naming the recorded
  context.
- Steps already carry their context; the viewer can therefore mark
  "action moved off-camera" ranges by comparing step contexts against
  the video's — divergence is visible, never silent.
- Dimensions default to the viewport, which avoids the padding
  artifact we observed (a fixed pre-layout size letterboxes with a
  black band until first layout). Callers pinning explicit
  dimensions accept possible padding.
- Leading warm-up frames (compositor start, about:blank flash, first
  paint of the destination) are part of the video and stay: with
  `offsetMs` and nav timestamps the viewer annotates them as the
  page-load phase — accidental paint-timing evidence, not junk.

## Durability: spill at start, package at stop

When `start.path` is declared, capture spills incrementally to a
spool directory beside it (`<path>.parts/`): trace events append to
NDJSON (valid up to the last complete line, by format), screenshots
land as files, and the engine live-muxes the video (a crash-orphaned
WebM is still playable). `stop()` packages the spool into the zip, renames it into
place atomically, and removes the spool. Nothing ever half-exists at
the declared path.

If the client or vibium dies mid-recording, the spool survives.
`vibium record recover <path>.parts/` packages what was captured —
partial video annotated, truncated tail dropped — into a record that
says honestly how far it got. Evidence survives the death of the
witness; a crashed run still leaves something to check in.

`start.path` **defaults to `record.zip` in the caller's working
directory** — "screenshots save to a sensible location
automatically," applied to records: crash durability is the
zero-config state, not a reward for reading the docs. Bytes-only
capture (no file, no spool, no durability) is the explicit opt-out:
`path: null`.

## The stop moment

The instant after `stop()` is a designed surface, not an
implementation detail — it is where Playwright earned a decade of
goodwill with one printed line.

- **Wire result** (path mode): `{ path, steps, durationMs,
  videos: [{ context, durationMs, width, height }] }` — or
  `videoUnavailable: "<engine reason>"` when the default couldn't
  deliver. An agent learns what was captured without unzipping;
  with the if-available default this is load-bearing, not garnish.
- **CLI** prints the human version and the next move:
  `Saved record.zip (23 steps, 14s video) — view: vibium play record.zip`
- **MCP** returns that same sentence as its text result, so an agent
  can relay it verbatim to the human it works for.
- `vibium play` itself — the app-mode viewer — is specced
  separately; this spec owns only the hint that points at it.

## Close is not a crash

`browser.stop()` (or session end) with a recording active
**auto-finalizes**: the spool packages and delivers to the declared
path, exactly as if `recording.stop()` had been called. Forgetting
to stop a recording costs nothing — "it's at record.zip anyway."
Only explicit bytes-only recordings (`path: null`) are lost on
close, which is the tradeoff those callers chose.

## Failure and lifecycle semantics

- **A broken video track never loses the record.** If the screencast
  dies mid-recording (browser crash, engine write error),
  `recording.stop()` still delivers the zip — video absent or
  partial, and the manifest records `"video": { "error": "…" }`.
  Fail-fast applies only at the explicit `start`; after capture has
  begun, degradation is annotated, never fatal and never silent.
- **Concurrency: one camera per context.** Two contexts can record
  simultaneously. (The "one active
  recording per browser session" line in the current client docs
  describes vibium's single-slot implementation, not the engine —
  correct it when this lands.) Concurrent recordings in different
  user-contexts can therefore each hold their own camera. v1 keeps
  one video per recording; the per-context limit only means two
  recordings can never film the same context.
- **Cleanup:** the engine writes the video into its own temp dir;
  session close and abandoned-recording paths delete stale files
  (same discipline the screencast handlers have today).
- **Manifest versioning:** the video block is additive; bump the
  manifest schema version so viewer and Store parsers can gate on it.
- **`offsetMs` definition:** wall-clock delta from recording t0 to
  the `startScreencast` acknowledgement, accurate to about one frame.
  Firefox exposes no first-frame timestamp; if it ever does, prefer
  it.

## Chunks and groups

The video is one continuous session track. Chunks slice the timeline
logically: a chunk's manifest records `videoRange: [startMs, endMs]`
into the session video rather than carrying its own file — WebM can't
be split mid-stream without transcoding, and capture stays
encoder-free. Groups are annotations and are unaffected. Honest
limitation: a chunk artifact alone does not contain its video; the
range resolves against the final zip.

## Surfaces: every door, same room (all at once, per the one-binary rule)

The logic lives once in the binary; each surface maps its house
style onto the same wire param. This was review flag #2 on #307; the
fold-in is the moment to close it everywhere. Shapes:

**JS (async and sync share the options type):**
```ts
await recording.start({ video: true });
await recording.start({ video: { width: 1280, height: 720, frameRate: 30 } });
```

**Python (async and sync, kwargs house style; snake_case keys map to
the wire's camelCase, same as mime_type→mimeType):**
```python
await recording.start(video=True)
await recording.start(video={"width": 1280, "height": 720, "frame_rate": 30})
```

**Java (flat fluent setters — the client's house style has no nested
option classes, so video flattens like every other option; setting
any video option implies video on):**
```java
recording.start(new RecordingOptions().video(true));
recording.start(new RecordingOptions().videoSize(1280, 720).videoFrameRate(30));
```
The client maps the flat setters onto the wire's nested `video`
param — flattening is per-surface sugar; the wire stays one shape.

**CLI (flag house style; nested options flatten with a prefix; new
options ship with the example CLAUDE.md requires):**
```
vibium record start --video -o record.zip
vibium record start --video --video-size 1280x720 --video-fps 30
# Recording started (video: on, spooling beside record.zip)
vibium record stop
# Saved record.zip
vibium record stop -o checkout-failure.zip   # override wins
```

(`-o` at start is the durability anchor on every surface: `path` in
JS/Python/MCP, `.path()` in Java.)

**MCP (`browser_record_start` gains flat properties, matching its
existing flat schema style):**
- `video` (boolean, optional): "Omit to record video when the
  engine supports it (Firefox 154+). Set true to require video —
  fails with an explanatory error on Chrome. Set false to disable."
- `video_width`, `video_height`, `video_frame_rate` (numbers,
  optional; defaults follow the viewport).

**Skill (the seventh surface):** the vibe-check SKILL.md gains
recording-with-video guidance in the same change — it is where
agents actually learn vibium, and a feature the skill never mentions
is a feature agents never use.

Acceptance: all seven surfaces land in the same change; no surface
ships a release behind another.

## Deliberately absent: pause()

Considered and excluded from v1. The real use cases resolve elsewhere:
sensitive input needs **redaction across all tracks** (pausing video
while snapshots still capture the DOM is false safety — and a
redaction story must exist before records are uploaded anywhere);
task boundaries are **chunks**; idle time is nearly free (still
frames and event-driven tracks cost ~nothing). Mechanically, BiDi
screencast has no pause — only stop/start — so pause would force
multi-segment videos and viewer stitching, the exact complexity the
chunk design avoids. If demand emerges post-redaction, pause must
write explicit paused-ranges into the manifest: an evidence timeline
may have gaps, but never silent ones.

## Open questions

1. Does `stopChunk` need to optionally wait for a keyframe so
   `videoRange` cuts land clean in the viewer?
2. Multi-context video: the engine supports one screencast per
   context simultaneously, so "film every tab in the
   recorded user-context" is feasible today. Shape when it comes:
   the manifest's `video` block becomes an array (one entry per
   context, each with its own `context` and `offsetMs`), the zip
   carries `video/<context>.webm` per tab, cameras attach
   event-driven (`contextCreated` starts a new tab's camera; a
   closed tab finalizes a shorter file), and the viewer cuts between
   cameras following step contexts — fully solving the off-camera
   problem. Wants a scope knob (`video: { contexts: 'all' |
   'active' | [...] }`) since every camera is an encoder in the
   browser and agents fan out to many tabs. Deferred from v1 for
   scope, not capability; the single-context binding ships first,
   documented. Note the non-video tracks are already multi-tab —
   one record.zip already carries every tab's steps and screenshots.
