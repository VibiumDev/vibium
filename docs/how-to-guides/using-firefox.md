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

```java
Browser bro = Vibium.start(
    new StartOptions().engine("firefox").channel("beta")
);
```

The main use today is [video recording](record-video.md), which needs Firefox 154 while it is still in beta.

## Launch

CLI — every command accepts `--engine`, or set it once with `VIBIUM_ENGINE=firefox`:

```
$ vibium start --engine firefox
```

JavaScript — a named launcher, or the engine option:

```js
const { browser, firefox } = require('vibium');
const bro = await firefox.start();

// equivalent:
const bro = await browser.start({ engine: 'firefox' });
```

The synchronous JavaScript API supports the same two forms:

```js
const { browser, firefox } = require('vibium/sync');
const bro = firefox.start();

// equivalent:
const bro = browser.start({ engine: 'firefox' });
```

Python:

```python
from vibium import browser, firefox
bro = firefox.start()

# equivalent:
bro = browser.start(engine="firefox")
```

Java:

```java
Browser bro = Vibium.start(new StartOptions().engine("firefox"));
```

MCP — `browser_start` takes an `engine` argument (`chrome` or `firefox`) and
an optional Firefox `channel` argument (`release` or `beta`).

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
| `VIBIUM_FIREFOX_PATH` | Use this Firefox executable instead of the vibium cache; when set, channel selection does not apply |
| `VIBIUM_FIREFOX_CHANNEL` | Channel to install and run: `release` (default) or `beta`; same as `--firefox-channel` or the clients' `channel` option |

## Feature notes

| Capability | Chrome | Firefox |
|------------|--------|---------|
| Navigation, elements, input, pages, screenshots, storage, and trace recording | Supported | Supported and covered by the Firefox core suite |
| Native video (`page.screencast`) | Not implemented by Chrome yet | Firefox 154+; see [Record Video](record-video.md) |
| Dialog callbacks and `capture.dialog()` | Supported | Not supported reliably by Vibium's native Firefox path yet |
| Network events and request interception | Supported | Not supported reliably by Vibium's native Firefox path yet |
| PDF printing (`page.pdf`) | Supported | Output and support may differ |

CI runs the full suite on Chrome, plus the browser-neutral CLI core and focused
installation, launch, navigation, screenshot, channel, and screencast tests on
Firefox. New browser-neutral CLI tests belong in `CLI_CORE_TESTS` in the
Makefile; engine-specific behavior stays in its focused suite.
