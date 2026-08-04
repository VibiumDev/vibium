# Using Firefox

Vibium launches Chrome by default. Firefox is supported as an alternative engine, using Firefox's native WebDriver BiDi — no driver binary involved.

## Install

```
$ vibium install --engine firefox
Installing Firefox v153.0.3 (release channel)...
Downloading Firefox from https://ftp.mozilla.org/pub/firefox/releases/153.0.3/mac/en-US/Firefox%20153.0.3.dmg...
Installation complete!
Firefox: /Users/you/Library/Caches/vibium/firefox/release/153.0.3/Firefox.app/Contents/MacOS/firefox
```

Firefox installs into the vibium cache next to Chrome for Testing. The JavaScript,
Python, and Java clients auto-install the selected engine on first launch on macOS
and Linux, same as Chrome. On Windows, Firefox auto-install is not available:
install Firefox yourself and point `VIBIUM_FIREFOX_PATH` at `firefox.exe`.

### Release channels

`--firefox-channel beta` (or `VIBIUM_FIREFOX_CHANNEL=beta`) selects the Firefox beta instead of the release build. The channel applies to install and launch alike: each channel is cached separately, and only the selected one is run, so an installed beta never shadows stable. In the clients, pass `channel` when starting the browser:

```js
const bro = await firefox.start({ channel: 'beta' });
```

```python
bro = firefox.start(channel="beta")
```

The main use today is [video recording](record-video.md), which needs Firefox 154 while it is still in beta.

## Launch

CLI — every command accepts `--engine`, or set it once with `VIBIUM_ENGINE=firefox`:

```
$ vibium start --engine firefox
```

JavaScript — a named launcher, or the engine option:

```js
const { firefox } = require('vibium');
const bro = await firefox.start();

// equivalent:
const bro = await browser.start({ engine: 'firefox' });
```

Python:

```python
from vibium import firefox
bro = firefox.start()

# equivalent:
bro = browser.start(engine="firefox")
```

Java:

```java
Browser bro = Vibium.start(new StartOptions().engine("firefox"));
```

MCP — `browser_start` takes an `engine` argument (`chrome` or `firefox`).

## Selecting Firefox with an environment variable

The `VIBIUM_ENGINE` env var changes the default engine of the vibium binary
itself, so code that uses browser-neutral APIs can select Firefox without a
code change:

```
$ VIBIUM_ENGINE=firefox node my-script.js
```

Browser-specific features still differ. In particular, native video recording
currently requires Firefox, while PDF output may differ between engines.

## Environment variables

| Variable | Effect |
|----------|--------|
| `VIBIUM_ENGINE` | Default engine (`chrome` or `firefox`) when `--engine` is not given |
| `VIBIUM_FIREFOX_PATH` | Use this Firefox executable instead of the vibium cache |
| `VIBIUM_FIREFOX_CHANNEL` | Channel to install and run: `release` (default) or `beta`; same as `--firefox-channel` or the clients' `channel` option |

## Feature notes

- Screenshots, navigation, clicking, and the rest of the automation API work the same on both engines. It is all standard WebDriver BiDi.
- Video recording (`page.screencast`) currently works on Firefox only: see [Record Video](record-video.md).
- PDF printing (`page.pdf`) support may differ between engines.
