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

Firefox installs into the vibium cache next to Chrome for Testing. The vibium
binary auto-installs the selected engine on first launch on macOS and Linux,
same as Chrome, so all clients get it for free. On Windows, Firefox auto-install is not available:
install Firefox yourself and point `VIBIUM_ENGINE_PATH` at `firefox.exe`.

### Release channels

`--channel beta` (or `VIBIUM_ENGINE_CHANNEL=beta`) selects the Firefox beta instead of the release build. The channel applies to install and launch alike: each channel is cached separately, and only the selected one is run, so an installed beta never shadows stable. In the clients, pass `channel` when starting the browser:

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

The release channel covers everything vibium supports, including [video recording](record-video.md) (Firefox 154+); the beta is for trying upcoming Firefox changes early.

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
| `VIBIUM_ENGINE_PATH` | Use this Firefox executable instead of the vibium cache; when set, channel selection does not apply |
| `VIBIUM_ENGINE_CHANNEL` | Channel to install and run: `release` (default) or `beta`; same as `--channel` or the clients' `channel` option |
| `VIBIUM_ENGINE_VERSION` | Pin the exact version to install and run (e.g. `153.0.4`) instead of the channel's latest; keeps CI and fleets on one version until you move the pin |

The names are engine-neutral, but only Firefox implements them today. Setting
any of them with `--engine chrome` is an error rather than a silent no-op.

## Feature notes

| Capability | Chrome | Firefox |
|------------|--------|---------|
| Navigation, elements, input, pages, screenshots, storage, and trace recording | Supported | Supported and covered by the Firefox core suite |
| Native video (`recording.start()`) | Not implemented by Chrome yet | Firefox 154+; see [Record Video](record-video.md) |
| Dialog callbacks and `capture.dialog()` | Supported | Supported and covered by the cross-engine suites |
| Network events and request interception | Supported | Supported and covered by the cross-engine suites |
| PDF printing (`page.pdf`) | Supported | Output and support may differ |
| Console and page error events (`onConsole`, `onError`) | Supported | Supported and covered by the cross-engine suites |
| Download events (`onDownload`, `capture.download()`) | Supported | Supported and covered by the cross-engine suites |
| New tab and popup events (`onPage`, `onPopup`) | Supported | Supported and covered by the cross-engine suites |
| Navigation capture (`capture.navigation()`) | Supported | Supported and covered by the cross-engine suites |

CI runs the full suite on Chrome. A separate Firefox job runs capability-selected
CLI and client tests plus focused installation, launch, channel, and video coverage.
Browser-driving cross-engine tests declare their requirements through the shared
capability adapters; unsupported features are reported as skips with reasons.
