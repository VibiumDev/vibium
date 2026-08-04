# Record Video

`page.screencast` records the browser viewport to a video file. The browser encodes the video itself, using the WebDriver BiDi `browsingContext.startScreencast` command, so there is nothing extra to install.

## Browser support

Recording requires Firefox 154, which is the current Firefox beta. It reaches regular Firefox on 2026-08-18; from then on a plain `vibium install --engine firefox` is enough. Until then, install the beta:

```
VIBIUM_FIREFOX_CHANNEL=beta vibium install --engine firefox
```

Chrome has not implemented the BiDi screencast command yet. The same code will work on Chrome when it does; today `start()` fails there with an error saying so.

See [Using Firefox](using-firefox.md) for installing and selecting Firefox in general.

## JavaScript

```js
const { firefox } = require('vibium');

const bro = await firefox.start();
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

bro = firefox.start()
vibe = bro.page()

vibe.screencast.start()
vibe.go("https://example.com")
# ... actions to record ...
vibe.screencast.stop(path="run.webm")

bro.stop()
```

## Options

`start()` accepts `mimeType`, `width`, `height`, `frameRate`, and `audio`. Defaults come from the browser; Firefox produces WebM. Firefox 154 rejects `audio: true` ("The audio track is not supported"); recordings are video-only for now.

Firefox writes the file as a live stream, so the WebM header carries no duration. Every player we tried plays it fine, but some show an unknown or zero duration until playback starts.

While recording, Firefox writes the in-progress file to the system Downloads folder (`screencast-<id>.webm`); that location is fixed inside Firefox. Vibium moves or deletes the file when recording stops or the session closes. If the vibium process is force-killed mid-recording, the file can be left behind there.

## Video vs. trace recording

This is different from `recording.start()`, which produces a trace zip (per-action screenshots, DOM snapshots, and sources for the Record Player). Use the trace for debugging test runs, video for demos and reports. See [Recording format](../explanation/recording-format.md).
