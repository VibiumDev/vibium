# Record Video

`page.screencast` records the browser viewport to a video file. The browser encodes the video itself, using the WebDriver BiDi `browsingContext.startScreencast` command, so there is nothing extra to install.

Only one screencast can be active in a browser session at a time. Starting a
recording from a second page fails until the active recording is stopped.

## Browser support

Recording requires Firefox 154, which is the current Firefox beta. It reaches regular Firefox on 2026-08-18; from then on a plain `firefox.start()` is enough. Until then, pass `channel: 'beta'` when starting the browser, as in the examples below. The channel applies at install and at launch: it picks which Firefox to download and which one to run, so a plain `firefox.start()` still launches stable Firefox even after the beta is installed.

On macOS and Linux, the clients install the selected channel automatically on
first use. Windows users must install Firefox themselves and set
`VIBIUM_FIREFOX_PATH`. To install it ahead of time on a supported platform:

```
vibium install --engine firefox --firefox-channel beta
```

The `VIBIUM_FIREFOX_CHANNEL` env var does the same as the flag and option, for cases where you cannot change the code.

Chrome has not implemented the BiDi screencast command yet. The same code will work on Chrome when it does; today `start()` fails there with an error saying so.

Screen recording is not supported on remote browser connections because the
browser writes the video on the remote host and Vibium cannot retrieve that
file. Use a local browser, or `recording.start()` for a trace with screenshots.

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

await vibe.screencast.start();
await vibe.go('https://example.com');
// ... actions to record ...
await vibe.screencast.stop({ path: 'run.webm' });

await bro.stop();
```

`stop()` without a path returns the video as a `Buffer` instead of writing a file.

## Python

```python
from vibium import firefox

# channel="beta" is needed until Firefox 154 reaches stable on 2026-08-18
bro = firefox.start(channel="beta")
vibe = bro.page()

vibe.screencast.start()
vibe.go("https://example.com")
# ... actions to record ...
vibe.screencast.stop(path="run.webm")

bro.stop()
```

The Java client can launch Firefox but does not expose the screencast API yet.

## Options

JavaScript `start()` accepts `mimeType`, `width`, `height`, `frameRate`, and
`audio`. Python uses `mime_type`, `width`, `height`, `frame_rate`, and `audio`.
Defaults come from the browser; Firefox produces WebM. Firefox 154 rejects
audio recording ("The audio track is not supported"); recordings are
video-only for now.

Firefox writes the file as a live stream, so the WebM header carries no duration. Every player we tried plays it fine, but some show an unknown or zero duration until playback starts.

While recording, Firefox writes the in-progress file to the system Downloads folder (`screencast-<id>.webm`); that location is fixed inside Firefox. Vibium moves or deletes the file when recording stops or the session closes. If the vibium process is force-killed mid-recording, the file can be left behind there.

If `stop()` fails to deliver the video, for example because the destination path is not writable, the recording is kept: call `stop()` again with a different path, or with no path to get the video inline. The file is cleaned up when the session closes either way.

## Video vs. trace recording

This is different from `recording.start()`, which produces a trace zip (per-action screenshots, DOM snapshots, and sources for the Record Player). Use the trace for debugging test runs, video for demos and reports. See [Recording format](../explanation/recording-format.md).
