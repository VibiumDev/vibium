# Cloud browsers

There are two ways to get a browser in the cloud. You can rent the
**computer** and run the browser yourself — that's
[cloud computers](../cloud-computers/). Or you
can rent the **browser**: managed vendors run it and hand you a
session. These guides cover connecting vibium to the second kind.

vibium connects to any vendor that speaks a W3C protocol — either a
WebDriver BiDi WebSocket URL, or a classic WebDriver endpoint (vibium
creates the session with `webSocketUrl: true` and attaches to the
BiDi socket it returns). The generic mechanics live in the
[remote browser tutorial](../../tutorials/remote-browser.md); these
pages add each vendor's specifics.

| Vendor | Connects via | Guide |
|---|---|---|
| Kernel | WebDriver BiDi URL from their API | [kernel.md](kernel.md) |
| BrowserStack | classic WebDriver endpoint | [browserstack.md](browserstack.md) — no BiDi screenshots |
| Sauce Labs | classic WebDriver endpoint | [saucelabs.md](saucelabs.md) |
| TestMu (LambdaTest) | classic WebDriver endpoint | [testmu.md](testmu.md) |
| TestingBot | classic WebDriver endpoint | [testingbot.md](testingbot.md) |

Vendors that expose only the Chrome DevTools Protocol don't currently
have a connect path — vibium speaks the W3C standards.

To test an app that isn't publicly reachable from a vendor's browser,
see [testing private sites](../testing-private-sites.md).

## Credentials

`vibium config init cloud` writes a commented template listing every provider's
variables to `~/.config/vibium/cloud-browser.env`, readable only by you:

```
$ vibium config init cloud
Wrote /Users/you/.config/vibium/cloud-browser.env (0600) — credentials for cloud browser vendors
Edit it, then: source /Users/you/.config/vibium/cloud-browser.env
```

Fill in the providers you use, delete the rest, and source it in the shell that
runs vibium — the file is not loaded automatically.
