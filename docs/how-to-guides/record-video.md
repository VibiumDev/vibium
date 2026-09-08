# Record Video

Recordings include a video track wherever the engine supports it — no option
needed. The browser encodes the viewport to WebM itself, using the WebDriver
BiDi `browsingContext.startScreencast` command, so there is nothing extra to
install. The video lands inside the recording zip next to the trace:

```
record.zip
├── trace.trace
├── trace.network
├── video/<context>.webm
└── video/index.json
```

With `video` omitted, vibium records video whenever the engine supports it;
otherwise the recording proceeds without it and the stop result reports
`videoUnavailable` with the engine's reason. `video: true` requires video:
`start` fails with an explanatory error if the engine can't deliver.
`video: false` turns it off.

The video films the browsing context that was active at `recording.start()`
and does not follow focus. One camera per context: a second recording cannot
film a context that is already being recorded.

This guide covers the video track specifically — engine support, Firefox
channels, and video options. For recording in general (screenshots,
snapshots, groups, chunks, the viewer), start with the
[Recording tutorial](../tutorials/recording.md).

## Browser support

Video requires Firefox 154 or newer, which regular Firefox ships since 2026-08-18. A plain `firefox.start()` is enough: it installs and launches the release channel.

On macOS and Linux, the clients install Firefox automatically on
first use. Windows users must install Firefox themselves and set
`VIBIUM_ENGINE_PATH`. To install it ahead of time on a supported platform:

```
vibium install --engine firefox
```

To record against the Firefox beta instead, select the channel with `--channel beta`, the clients' `channel` option, or the `VIBIUM_ENGINE_CHANNEL` env var; see [Using Firefox](using-firefox.md#release-channels).

Chrome has not implemented the BiDi screencast command yet. The same code will work on Chrome when it does; today `video: true` fails there with an error saying so, and an omitted `video` records the trace without a video track.

No video is recorded on remote browser connections (`--connect`). The
engine would write the file on the remote host's disk, where Vibium cannot
reach it — the WebDriver BiDi protocol has no way to retrieve files — so
the screencast is never started at all: with `video` omitted the recording
proceeds and the stop result reports `videoUnavailable`; `video: true`
fails at start. No file is left behind on the remote host, and there is
nothing to fetch separately. Every other track — actions, screenshots,
snapshots, network — records fully over remote connections; only the video
needs a local browser.

For a remote host you control, `video: { remote: 'keep' }` opts out of the
refusal: the screencast records on the remote host and stays there. The
stop result and `video/index.json` carry `remotePath` — the file's
location on that machine — and vibium never reads, moves, or deletes it;
retrieval (scp, a mounted volume) and cleanup are yours. The remote engine
must still support the screencast (Firefox 154+). CLI:
`--video-remote keep`; MCP: `video_remote: "keep"`; Java:
`.videoRemote("keep")`.

On Linux, Vibium gives Firefox a private Downloads directory inside the
temporary browser profile because Firefox's native command requires one. It is
removed with the profile when the browser closes; Vibium does not create a
Downloads folder in your home directory. Recording works in headed and
headless modes.

See [Using Firefox](using-firefox.md) for installing and selecting Firefox in general.

## JavaScript

```js
const { firefox } = require('vibium');

const bro = await firefox.start();
const vibe = await bro.page();

// video: true because this script exists to produce a video -- fail
// rather than quietly write a trace without one.
await vibe.context.recording.start({ video: true, path: 'runs/login.zip' });
await vibe.go('https://example.com');
// ... actions to record ...
await vibe.context.recording.stop();

await bro.stop();
```

`path` defaults to a timestamped `record-YYYYMMDD-HHMMSS.zip` in the
working directory, so a rerun never overwrites the previous artifact;
`stop({ path })` overrides the path declared at start. `stop()` returns a
result — `path`, `steps`, `durationMs`, `videos`/`videoUnavailable`.
`path: null` selects bytes-only capture: nothing is written and the result
carries the zip itself.

## Python

```python
from vibium import firefox

bro = firefox.start()
vibe = bro.page()

# video=True because this script exists to produce a video -- fail
# rather than quietly write a trace without one.
vibe.context.recording.start(video=True, path="runs/login.zip")
vibe.go("https://example.com")
# ... actions to record ...
vibe.context.recording.stop()

bro.stop()
```

## Java

```java
recording.start(new RecordingOptions().video(true).path("runs/login.zip"));
// ... actions to record ...
recording.stop();
```

Setting any video option (`videoSize(1280, 720)`, `videoFrameRate(30)`)
implies video on.

## CLI

```
vibium record start --video -o run.zip
# ... actions ...
vibium record stop
# Saved run.zip (23 steps, 14s video)
```

The MCP tool `browser_record_start` takes the same options as flat
properties: `video`, `video_width`, `video_height`, `video_frame_rate`,
`path`. Without a `path`, the recording lands in the MCP server's working
directory when that is a real, writable place (Claude Code launches the
server in your project, so it lands next to your code), and in
`~/Documents/Vibium` otherwise; the stop result names the absolute path
either way.

## Options

Video dimensions default to the viewport. Explicit dimensions that mismatch
the window aspect are letterboxed by the engine. `frameRate` uses the
engine's default when omitted. Firefox produces WebM; audio is not exposed
(Firefox 154 rejects it: "The audio track is not supported").

Firefox writes the file as a live stream, so the WebM header carries no duration. Every player we tried plays it fine, but some show an unknown or zero duration until playback starts.

While recording, Firefox writes the in-progress file to the system Downloads folder (`screencast-<id>.webm`); that location is fixed inside Firefox. Vibium moves the file into the recording zip when recording stops and deletes leftovers when the session closes. If the vibium process is force-killed mid-recording, the file can be left behind there.

If the browser session ends with a recording still active, the recording
auto-finalizes and delivers to the declared path as if `stop()` had been
called; `path: null` recordings are lost on close. If the video pipeline
dies mid-recording, `stop()` still delivers the zip — the video is absent or
partial and `video/index.json` records the error.

## Video vs. trace tracks

The video is one more track in the same recording zip that carries
per-action screenshots, DOM snapshots, and network events for the Record
Player. `video/index.json` records the video's `offsetMs` from the
recording's start so viewers can align the two timelines. See the
[Recording tutorial](../tutorials/recording.md) for the other tracks and
[Recording format](../reference/recording-format.md) for the zip layout.
