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

Firefox installs into the vibium cache next to Chrome for Testing. On Windows, Mozilla ships no archive build, so install Firefox yourself and point `VIBIUM_FIREFOX_PATH` at `firefox.exe`. The clients auto-install the selected engine on first launch, same as Chrome.

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

## Running a whole suite against Firefox

The `VIBIUM_ENGINE` env var changes the default engine of the vibium binary itself, so code that never mentions an engine — including an existing Chrome test suite — runs on Firefox unmodified:

```
$ VIBIUM_ENGINE=firefox make test
```

This mirrors how Playwright projects re-run the same tests per browser via `browserName` config.

## Environment variables

| Variable | Effect |
|----------|--------|
| `VIBIUM_ENGINE` | Default engine (`chrome` or `firefox`) when `--engine` is not given |
| `VIBIUM_FIREFOX_PATH` | Use this Firefox executable instead of the vibium cache |
| `VIBIUM_FIREFOX_CHANNEL` | Install channel: `release` (default) or `beta` |

## Feature notes

- Screenshots, navigation, clicking, and the rest of the automation API work the same on both engines. It is all standard WebDriver BiDi.
- Video recording (`page.screencast`) currently works on Firefox only: see [Record Video](record-video.md).
- PDF printing (`page.pdf`) support may differ between engines.
