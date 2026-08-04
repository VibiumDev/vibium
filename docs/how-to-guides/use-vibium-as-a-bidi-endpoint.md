# Use vibium as a WebDriver BiDi endpoint

If you already have a WebDriver BiDi client — Selenium, WebdriverIO, Puppeteer's
BiDi mode, or something you wrote — you do not need a vibium client library to
use vibium. `vibium serve` is a BiDi endpoint that manages its own browser.

```bash
vibium serve --headless --port 9633
# Server listening on ws://localhost:9633
```

Point your client at that URL and speak standard BiDi. There is no chromedriver
to install, and no `POST /session` handshake — connect the WebSocket and send
commands.

```json
→ {"id":1,"method":"session.status","params":{}}
← {"id":1,"result":{"build":{"version":"151.0.7922.71"},"ready":false}}

→ {"id":2,"method":"browsingContext.getTree","params":{}}
← {"id":2,"result":{"contexts":[{"context":"E1B0A6CA...","url":"about:blank"}]}}

→ {"id":3,"method":"browsingContext.navigate",
   "params":{"context":"E1B0A6CA...","url":"https://example.com","wait":"complete"}}
← {"id":3,"result":{"navigation":"1496dc0f-...","url":"https://example.com/"}}
```

Events arrive unsolicited, as usual:

```json
← {"method":"browsingContext.contextCreated","params":{...}}
← {"method":"browsingContext.navigationStarted","params":{...}}
← {"method":"network.beforeRequestSent","params":{...}}
← {"method":"browsingContext.load","params":{...}}
```

vibium downloads and launches Chrome for Testing on the first connection and
cleans it up when you disconnect.

---

## Which transport do I want?

`serve` is one of two ways to reach vibium, and it is the right one only for
some jobs.

| You have | Use | Why |
|---|---|---|
| An existing BiDi client | `vibium serve` | It already speaks BiDi over a WebSocket. Nothing to integrate. |
| A vibium client library (JS, Python, Java) | those clients | They spawn `vibium pipe` for you. |
| A new client library you are writing | `vibium pipe` | No port, no auth, browser lifetime tied to your process. See the [client implementation guide](../explanation/client-implementation-guide.md). |

Do not chain them. `vibium pipe --connect ws://localhost:9633` works, but it
inserts a second router that translates nothing.

---

## Extension commands

`serve` also accepts vibium's `vibium:` commands on the same connection —
actionability-aware clicks, semantic locators, recording. Those are additions to
BiDi, not replacements, so a plain BiDi client can ignore them entirely.

```json
→ {"id":4,"method":"vibium:browser.page","params":{}}
← {"id":4,"type":"success","result":{"context":"B0EC708F...","userContext":"default"}}
```

See the [API reference](../reference/api.md) for the full list.

---

## Reaching it from elsewhere

`serve` binds to `127.0.0.1` and rejects WebSocket upgrades whose `Origin` is
not local. Browser automation is a remote code execution surface, so it is not
exposed to the network by default.

To reach it from a container or another machine, forward a local port rather
than binding wider — for example `ssh -L 9633:localhost:9633 user@host`, or
Docker's `-p 127.0.0.1:9633:9633` with the server inside the container.

Clients that are not browsers (most BiDi libraries) send no `Origin` header and
connect normally.

---

## When it does not fit

**You want one browser shared by several clients.** Each connection to `serve`
gets its own browser and its own session. For concurrent CLI use, see
[running concurrent CLI sessions](concurrent-cli-sessions.md).

**You want the browser to outlive the connection.** It does not. Disconnecting
closes the browser. The daemon behind the `vibium` CLI is what keeps a browser
alive between commands.

---

## Related

- [Client implementation guide](../explanation/client-implementation-guide.md) — writing a client against `vibium pipe`
- [Remote browser](../tutorials/remote-browser.md) — pointing vibium at someone else's browser, the opposite direction
- [API reference](../reference/api.md) — the `vibium:` extension commands
