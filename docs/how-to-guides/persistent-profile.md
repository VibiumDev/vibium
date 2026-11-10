# Reuse a logged-in profile

Vibium launches the browser with a temporary profile and wipes it on close. Cookies, logins, and localStorage are gone on the next run. A persistent profile keeps them: pass a directory, log in once, and reuse the session across runs.

This is opt-in. Without the option, nothing changes: launches stay ephemeral, and `newContext` still gives you an isolated, incognito-like context inside a session.

## How it works

- The directory is created if it is missing. Relative paths resolve against your working directory.
- Vibium never deletes the directory. Close only shuts the browser down — gracefully, so profile data is flushed to disk.
- One browser per profile: a second browser MUST NOT run on the same directory at the same time.
- The option is ignored in connect mode, where the browser is already running elsewhere.
- Works with both engines. Do not point Chrome and Firefox at the same directory.

## CLI

Every command accepts `--user-data-dir`:

```
$ vibium go https://example.com --user-data-dir ~/vibium-profiles/work
```

Log in by hand or with `vibium run`, stop the browser, and the next run with the same directory is still logged in:

```
$ vibium stop
$ vibium go https://example.com/account --user-data-dir ~/vibium-profiles/work
https://example.com/account
```

The running browser keeps the profile it was launched with. Asking for a different `--user-data-dir` while it runs is an error; run `vibium stop` first.

## JavaScript

```js
const { browser } = require('vibium');

const bro = await browser.start({ userDataDir: '/path/to/profile' });
// ... log in, cookies land in the profile ...
await bro.stop();

// Later: same profile, still logged in.
const again = await browser.start({ userDataDir: '/path/to/profile' });
```

## Python

```python
from vibium import browser

bro = browser.start(user_data_dir="/path/to/profile")
bro.stop()

bro = browser.start(user_data_dir="/path/to/profile")  # still logged in
```

The async API takes the same keyword.

## Java

```java
Browser bro = Vibium.start(new StartOptions().userDataDir("/path/to/profile"));
```

## MCP

`browser_start` takes a `userDataDir` property:

```json
{"name": "browser_start", "arguments": {"userDataDir": "/path/to/profile"}}
```

A server started with `vibium mcp --user-data-dir /path/to/profile` uses that directory as the default for every launch.

## Notes

- Keep profile directories out of the system temp dir. Vibium's startup sweep of orphaned browser temp dirs only matches its own temp-dir patterns, but temp dirs are fair game for the OS.
- A profile holds real credentials. Treat the directory like a password store.
