# Spec: `vibium play`

> **Status: Draft.** This document describes proposed behavior. It does
> not reflect what is currently implemented in vibium. Do not rely on it
> as documentation of existing functionality.

## Context

Records and screencast videos have no local viewer. Recordings are
WebM, which macOS QuickTime cannot open; record.zip contents are
only viewable by uploading to Record Player. `vibium play` opens
either in a local app window using the browser vibium already
manages. The recording spec's stop message points at it:
`Saved record.zip (23 steps, 14s video) — view: vibium play record.zip`.

## CLI

```
vibium play record.zip
# Opens the record viewer: step timeline, screenshots, video, synced scrubbing

vibium play demo.webm
# Plays a video file
```

## Architecture

No new runtime. The binary serves an embedded viewer page
(`go:embed`, no network dependencies) on an ephemeral `127.0.0.1`
port and launches the cached Chrome with
`--app=http://127.0.0.1:<port>` — a chromeless standalone window.
The process exits when the window closes.

- The viewer always runs on Chrome regardless of which engine
  recorded; Firefox has no `--app` equivalent.
- If Chrome is not installed, `vibium play` opens the URL in the
  default browser instead (plain tab, same viewer).
- The server binds localhost only and serves only the opened
  artifact.

## Viewer (v1 scope)

- record.zip: step timeline with screenshots; video playback synced
  to steps via `video/index.json`'s `offsetMs`; off-camera ranges
  marked where step contexts diverge from the video's context;
  warm-up frames labeled as the page-load phase using navigation
  timestamps.
- .webm: plain video playback.
- Browsers play WebM natively; the viewer needs no transcoding.

## Out of scope for v1

- Editing, blessing, or diffing records.
- MCP/client-library surfaces — this is a human-facing window; the
  CLI is the only door.
- Remote records: the argument is a local path.

## Open questions

1. Share viewer code with the Record Player web app, or keep a
   minimal bespoke page embedded in the binary?
2. `vibium play record.zip.parts/` — live view of an in-progress
   recording by tailing the spool. The spool format supports it;
   defer until the spool ships.
3. Accept a URL argument (a record on Record Player) or keep the
   local/hosted split strict?
