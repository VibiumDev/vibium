# Spec: Video as a Recording Track

> **Status: Draft.** This document describes proposed behavior. It does
> not reflect what is currently implemented in vibium. Do not rely on it
> as documentation of existing functionality.

## Context

`recording.start()` gains a `video` option; the video lands inside
record.zip next to the other capture tracks. The separate
`page.screencast` API and the `vibium:screencast.start/stop` wire
commands are removed — they have not shipped in a tagged release
(latest: v26.5.31, 2026-06-01; screencast merged 2026-08-05).
Resolves #310. #311's CLI/MCP recording surfaces target this shape.

## API

```ts
await recording.start();                // video if the engine supports it; path record.zip
await recording.start({ video: true }); // require video: start fails if unsupported
await recording.start({ video: { width: 1280, height: 720, frameRate: 30 }, path: 'runs/login.zip' });
await recording.stop();                                          // delivers to the declared path
await recording.stop({ path: 'checkout-failure-run1234.zip' });  // override wins
```

- `video?: boolean | { width?, height?, frameRate? }` — omitted:
  record video if the engine supports it; otherwise the recording
  proceeds and the stop result reports `videoUnavailable`. `true`:
  `start` fails if the engine can't deliver. `false`: off.
- Dimensions default to the viewport. Explicit dimensions that
  mismatch the window aspect are letterboxed by the engine.
- `path` defaults to `record.zip` in the caller's working directory.
  `path: null` selects bytes-only capture: no file, no spool, no
  crash durability. Clients resolve relative paths before sending.
- Path precedence: `stop.path` > `start.path`.
- `audio` is reserved, not exposed; Firefox 154 rejects it.
- The flat `format`/`quality` options remain screenshot-only.

## Wire protocol

`vibium:recording.start` gains the `video` param, same shape. The
router calls `browsingContext.startScreencast` when the recording
starts, `stopScreencast` when it stops, and moves the engine-written
file into the zip.

## Zip layout

record.zip strives to be Playwright trace-compatible. Video entries
are additive; existing trace tooling ignores them. Success
criterion: a video-carrying record.zip opens in the Playwright trace
viewer.

```
record.zip
├── trace.trace
├── trace.network
├── video/<context>.webm
└── video/index.json
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

`videos` is an array; v1 writes one entry. `offsetMs` is the video's
start relative to the recording's t0: wall-clock delta from t0 to the
`startScreencast` acknowledgement, accurate to about one frame. Trace
events carry their own timestamps; `offsetMs` aligns the two
timelines. Bump the manifest schema version.

## Video format

Stored as the engine produced it (WebM). Capture never invokes an
encoder. MP4 conversion is an export operation —
`vibium record export video record.zip -o clip.mp4` — specced
separately; a video-only clip is a recording with the other tracks
disabled, then an export.

## Engine semantics

- Firefox 154+, local: supported.
- Chrome, or Firefox < 154: with `video` omitted, no video is
  recorded and the stop result carries `videoUnavailable` with the
  engine's reason. With `video: true`, `start` fails; the message
  names the engine gap and the install command that fixes it.
- Remote connections (`--connect`): `video: true` errors with the
  existing remote-screencast message; the engine writes the file on
  the remote host.

## Context binding

The video records the browsing context active at `recording.start()`
and does not follow focus. `video/index.json` names the recorded
context. Step events name theirs; the viewer marks ranges where the
action left the recorded context. Warm-up frames (compositor start,
about:blank, first paint) remain in the video; the viewer labels
them using `offsetMs` and navigation timestamps.

## Durability

With a path declared, capture spills incrementally to `<path>.parts/`:
trace events append as NDJSON, screenshots land as files, the engine
live-muxes the video. `stop()` packages the spool into the zip,
renames it into place atomically, and removes the spool.

A crashed client leaves the spool. `vibium record recover
<path>.parts/` packages what was captured; a partial video is
annotated, a truncated NDJSON tail is dropped at the last complete
line.

## Stop result

- Wire (path mode): `{ path, steps, durationMs, videos: [{ context,
  durationMs, width, height }] }`, or `videoUnavailable: "<engine
  reason>"` in place of `videos`.
- CLI: `Saved record.zip (23 steps, 14s video) — view: vibium play record.zip`
- MCP: the same sentence as the tool's text result.
- `vibium play` is specced separately.

## Close with an active recording

`browser.stop()` or session end auto-finalizes: the spool packages
and delivers to the declared path as if `recording.stop()` had been
called. `path: null` recordings are lost on close.

## Failure and lifecycle

- If the screencast dies mid-recording, `recording.stop()` still
  delivers the zip; the video is absent or partial and
  `video/index.json` records `"error"`. Fail-fast applies only at
  `start`.
- One camera per context. Two contexts can record simultaneously;
  two recordings cannot film the same context. (The "one active
  recording per browser session" line in current client docs
  describes the old single-slot implementation; correct it.)
- Engine temp files are deleted on session close and when an
  abandoned recording is superseded.

## Chunks and groups

The video is one continuous session track. A chunk's manifest
records `videoRange: [startMs, endMs]` into the session video;
chunk artifacts carry no video file. Groups are unaffected.

## Surfaces

The logic lives once in the binary; each surface maps its house
style onto the same wire param.

**JS (async and sync share the options type):**
```ts
await recording.start({ video: true });
await recording.start({ video: { width: 1280, height: 720, frameRate: 30 } });
```

**Python (async and sync; snake_case keys map to the wire's
camelCase):**
```python
await recording.start(video=True)
await recording.start(video={"width": 1280, "height": 720, "frame_rate": 30})
```

**Java (flat fluent setters; setting any video option implies video
on; the client maps them onto the wire's nested `video` param):**
```java
recording.start(new RecordingOptions().video(true));
recording.start(new RecordingOptions().videoSize(1280, 720).videoFrameRate(30));
```

**CLI:**
```
vibium record start --video -o record.zip
vibium record start --video --video-size 1280x720 --video-fps 30
# Recording started (video: on, spooling beside record.zip)
vibium record stop
# Saved record.zip
vibium record stop -o checkout-failure.zip
```

**MCP** — `browser_record_start` gains flat properties:
- `video` (boolean, optional): "Omit to record video when the engine
  supports it (Firefox 154+). Set true to require video — fails with
  an explanatory error on Chrome. Set false to disable."
- `video_width`, `video_height`, `video_frame_rate` (numbers,
  optional; defaults follow the viewport).

**Skill:** the vibe-check SKILL.md gains recording-with-video
guidance in the same change.

The start-side path exists on every surface: `-o` in the CLI, `path`
in JS/Python/MCP, `.path()` in Java.

Acceptance: all seven surfaces land in the same change.

## pause() is excluded

v1 has no `pause()`. Sensitive input is a redaction concern — every
track captures it, not just video. Task boundaries are chunks. Idle
time costs little in any track. BiDi screencast has no pause
primitive, so pause would require multi-segment video and viewer
stitching. If added later, paused ranges must be recorded in the
manifest.

## Open questions

1. Should `stopChunk` optionally wait for a keyframe so `videoRange`
   cuts land clean in the viewer?
2. Multi-context video. Feasible now — the engine runs one
   screencast per context simultaneously. Shape: one `videos` entry
   and one `video/<context>.webm` per tab; cameras attach on
   `contextCreated` and finalize on context close; the viewer cuts
   between cameras by step context. Needs a scope knob
   (`video: { contexts: 'all' | 'active' | [...] }`) — each camera
   is an encoder in the browser. Deferred from v1.
