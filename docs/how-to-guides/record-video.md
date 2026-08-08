# Record Video

Recordings can include a video track of the session. Pass `video` to
`recording.start()` and the browser encodes the viewport to WebM itself,
using the WebDriver BiDi `browsingContext.startScreencast` command — nothing
extra to install. The video lands inside the recording zip next to the trace:

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

Video requires Firefox 154, which is the current Firefox beta. It reaches regular Firefox on 2026-08-18; from then on a plain `firefox.start()` is enough. Until then, pass `channel: 'beta'` when starting the browser, as in the examples below. The channel applies at install and at launch: it picks which Firefox to download and which one to run, so a plain `firefox.start()` still launches stable Firefox even after the beta is installed.

On macOS and Linux, the clients install the selected channel automatically on
first use. Windows users must install Firefox themselves and set
`VIBIUM_FIREFOX_PATH`. To install it ahead of time on a supported platform:

```
vibium install --engine firefox --firefox-channel beta
```

The `VIBIUM_FIREFOX_CHANNEL` env var does the same as the flag and option, for cases where you cannot change the code.

Chrome has not implemented the BiDi screencast command yet. The same code will work on Chrome when it does; today `video: true` fails there with an error saying so, and an omitted `video` records the trace without a video track.

Video is not recorded on remote browser connections (`--connect`) because the
browser writes the video on the remote host and Vibium cannot retrieve that
file. Use a local browser, or record without video for a trace with
screenshots.

On Linux, Vibium gives Firefox a private Downloads directory inside the
temporary browser profile because Firefox's native command requires one. It is
removed with the profile when the browser closes; Vibium does not create a
Downloads folder in your home directory. Recording works in headed and
headless modes.

See [Using Firefox](using-firefox.md) for installing and selecting Firefox in general.

## JavaScript

```js
const { firefox } = require('vibium');

// channel: 'beta' is needed until Firefox 154 reaches stable on 2026-08-18
const bro = await firefox.start({ channel: 'beta' });
const vibe = await bro.page();

await vibe.context.recording.start({ video: true, path: 'runs/login.zip' });
await vibe.go('https://example.com');
// ... actions to record ...
await vibe.context.recording.stop();

await bro.stop();
```

`path` defaults to `record.zip` in the working directory; `stop({ path })`
overrides the path declared at start. `path: null` selects bytes-only
capture — `stop()` returns the zip as a `Buffer` and nothing is written.

## Python

```python
from vibium import firefox

# channel="beta" is needed until Firefox 154 reaches stable on 2026-08-18
bro = firefox.start(channel="beta")
vibe = bro.page()

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
`path`.

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
[Recording format](../explanation/recording-format.md) for the zip layout.
