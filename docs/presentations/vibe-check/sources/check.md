# Check reference

Use this page to look up options, configuration, API examples, and limits.
For everyday commands, see [Check a live browser or saved recording](../how-to-guides/check.md).
For a first website check, start with the [tutorial](../tutorials/first-check.md).

## Setup diagnostics

```bash
vibium ready             # Selected browser and configured AI
vibium ready ai --json   # Require AI readiness; no browser needed
```

Use `ready` for initial live setup and `ready ai` for archive-only checks.
AI diagnostics test configuration and a provider tool round-trip, with up to
two model requests. Browser diagnostics launch and close an isolated headless
session. Neither command tests your application's acceptance claims.

Plain `ready` allows missing AI for direct browser automation; Run and Check
require AI. See [Readiness diagnostics](ready.md) for selection, installation
fixes, JSON output, and exit codes.

## CLI options

```bash
vibium check "<claim>" [options]
```

| Option | Purpose | Example |
|--------|---------|---------|
| No file option | Check the current live browser | `vibium check "the cart contains one battery pack"` |
| `-i`, `--input` | Read a saved recording or trace; no browser launch | `vibium check "the order confirmation is visible" -i record.zip` |
| `-o`, `--output` | Save a recording of live verification | `vibium check "the cart contains one battery pack" -o verification.zip` |
| `--keep-open` | Keep a browser started by Check open after the run | `vibium check "https://example.com is up" --keep-open` |
| `--report` | Save the verdict and evidence as JSON | `vibium check "the cart contains one battery pack" --report verdict.json` |
| `--json` | Print the CLI result as JSON | `vibium check "the cart contains one battery pack" --json` |

`--keep-open` cannot be combined with `--input`.
`--input` and `--output` cannot be combined. You can combine either one with
`--report` and `--json`. The old `--record` and `--trace` flags are not accepted.
Paths are relative to the caller's working directory. Output and report files
must be new and must have different paths; existing files are never replaced.

## Provider configuration

Check supports OpenAI, Anthropic, Google Gemini, OpenAI-compatible servers,
and the `local` alias. See [model providers](model-providers.md) for native
credentials, endpoint defaults, and shared AI settings. The examples
below retain the OpenAI setup.

Per-call overrides are available as `--provider`, `--model`, `--base-url`, and
`--reasoning-effort` for live and saved-input checks, and for AI readiness.
SDKs and MCP accept equivalent options. See
[override settings for one call](model-providers.md#override-settings-for-one-call).

For OpenAI, export the provider, a model that supports Chat Completions function tools, and
your API key in the terminal running the CLI. For example, with a key already
available as `OPENAI_API_KEY`:

```bash
export OPENAI_API_KEY
export VIBIUM_AI_PROVIDER=openai
export VIBIUM_AI_MODEL=gpt-5.6-sol
export VIBIUM_AI_REASONING_EFFORT=none
```

`gpt-5.6-sol` requires `none` for function tools through Chat Completions.
`VIBIUM_AI_REASONING_EFFORT` is optional for the adapter and is forwarded
only when set; choose a value supported by your model. For a model that does
not accept this setting, unset the variable.

For an OpenAI-compatible server:

```bash
export VIBIUM_AI_PROVIDER=openai-compatible
export VIBIUM_AI_BASE_URL=http://localhost:1234/v1
export VIBIUM_AI_MODEL='your-tool-capable-model'
unset VIBIUM_AI_REASONING_EFFORT
```

`OPENAI_API_KEY` is optional in compatible mode; if set, it is sent as a Bearer
authorization header to the configured endpoint. The server must support
function tools and `max_completion_tokens`. Image input support is needed for
the screenshot tool.

Configuration is read by each CLI invocation, so changing it does not require
restarting the daemon or browser.

Vibium does not load environment files automatically. To use a file with Bash
or Zsh, put `export NAME=value` assignments in it, then source it in the same
shell invocation as the CLI command:

```bash
source ~/.config/vibium/ai.env
vibium check "the cart contains one battery pack"
```

Keep the file outside your repository. The [tutorial](../tutorials/first-check.md#1-configure-ai-access)
shows how to create it with private permissions.

## Results and exit codes

| Displayed verdict | JSON `status` | Meaning | Exit code |
|-------------------|---------------|---------|-----------|
| PASS | `passed` | Evidence supports the claim. | 0 |
| FAIL | `failed` | Evidence contradicts the claim. | 0 |
| INCONCLUSIVE | `inconclusive` | Evidence is insufficient to decide. | 0 |
| Execution error | No verdict | Configuration, provider, browser, or timeout failure. | 1 |

With `--json`, a completed check prints this CLI envelope:

```json
{
  "ok": true,
  "result": {
    "status": "passed",
    "claim": "the changed timezone persists after refresh",
    "summary": "The new timezone persisted after refresh.",
    "evidence": [
      {"type": "observation", "summary": "The saved timezone remained visible after reload."}
    ]
  }
}
```

`ok: true` means the check completed. To require a PASS in automation, inspect
`result.status == "passed"`. Execution errors print `{"ok":false,"error":"..."}`.

`--report` writes the inner result object, without `ok` or `result`. Inspect
`status` at the top level of that file. An execution error leaves no verdict
report. Language APIs also return the result directly and raise exceptions for
execution errors.

## Live browser and recording behavior

Live checks support local Chrome and Firefox. They reuse the existing Vibium
session and selected page. When no browser exists, the CLI starts one and
closes it after recording finalization, including on FAIL, INCONCLUSIVE, or an
execution error. `--keep-open` keeps that newly started browser open instead.
A browser already running before Check is always preserved. The ownership
check, launch, verification, and cleanup are serialized in the daemon; its
presence alone does not count as an existing browser. The daemon remains
available for subsequent commands under its normal idle timeout.

Keep the same `VIBIUM_SESSION` value throughout a workflow. The verifier can
interact with the page and does not restore its starting state.

Recording is optional. Console and network observation tools need an active
recording; without one, they report that the evidence is unavailable.

When no recording is active, `--output` starts one with screenshots and browser
actions, then saves and stops it when Check finishes. It saves available
evidence even if verification encounters an execution error. Video is included
when the browser supports it (Firefox 154+ produces WebM). It is finalized
before the browser closes. Chrome currently records without video.

When recording is already active, `--output` exports the **entire current
recording chunk**, including actions before Check. It leaves the ongoing
recording, open groups, video track, and original destination intact. The
export has no continuous video file. Choose a destination different from the
active recording's path.

## Saved recording and trace support

`--input` accepts Vibium recordings and Playwright traces in **trace format
version 8**. This is a trace format version, not a Playwright package version.
Other formats and malformed archives return execution errors.

Archive verification uses read-only tools. It does not launch a browser,
replay actions, modify the source ZIP, or create a new artifact by default.
The verifier can inspect recorded actions, DOM snapshots where present,
screenshots, network summaries, console events, and earlier Check groups.
The entire archive is not automatically sent to the provider.

Missing evidence can produce INCONCLUSIVE. An earlier PASS is an assessment
to examine, not proof. An archive describes its recorded run and cannot
establish what a later deployment or live session does.

## Run limits

- Claims must be nonempty and at most 4,000 bytes.
- Each run has a three-minute budget and at most 24 model-selected actions.
- Text and image payloads are bounded; truncated text is marked.
- Exhausting the action limit returns INCONCLUSIVE. A timeout returns an error.
- Browser polling or recording cleanup can take additional time after a timeout.

## MCP and language APIs

Configure the verifier environment before starting the MCP server or SDK
runtime. These surfaces read configuration in the Go process; restart that
runtime after changing its environment. All invoke the same verifier adapter,
result validation, and bounded tool loop used by the CLI.

### MCP

MCP exposes `vibium_check`:

```json
{"claim":"changing my timezone persists after refresh","page":"existing-context-id"}
```

Omit `page` to use the active page. To inspect an archive, pass
`{"claim":"the order confirmation is visible","record":"/absolute/path/record.zip"}`.
`record` and `page` cannot be combined. Archive mode does not start a browser.
Use existing recording tools around live MCP checks to save their evidence.

### JavaScript and TypeScript

```js
import { browser } from 'vibium';
const bro = await browser.start({ headless: true });
const page = await bro.page();
await page.go('http://localhost:3000/account');
await page.context.recording.start({ path: 'verification.zip', video: false });
try {
  const result = await page.check('changing my timezone persists after refresh');
  console.log(result.status, result.evidence);
} finally {
  await page.context.recording.stop();
  await bro.stop();
}
// Standalone archive inspection: no browser installation or launch.
const recorded = await browser.check('the order confirmation is visible', { record: './record.zip' });
```

The synchronous API has the same methods via `vibium/sync`; omit `await`.
`Browser.check()` uses the active page, while `Page.check()` pins that Page.
Both also accept `{ record: './record.zip' }`, which selects read-only evidence
instead of inspecting their live browser.

### Python

```python
from vibium import browser
bro = browser.start(headless=True)
try:
    page = bro.page()
    page.go('http://localhost:3000/account')
    result = page.check('changing my timezone persists after refresh')
    print(result['status'], result['evidence'])
finally:
    bro.stop()
recorded = browser.check('the order confirmation is visible', record='./record.zip')
```

For async Python, import from `vibium.async_api` and await browser operations
and Check. Use `page.context.recording.start()` / `.stop()` around a live
check to save its evidence.

### Java

```java
Browser bro = Vibium.start(new StartOptions().headless(true));
try {
    Page page = bro.page();
    page.go("http://localhost:3000/account");
    CheckResult result = page.check("changing my timezone persists after refresh");
    System.out.println(result.status());
} finally {
    bro.stop();
}
CheckResult recorded = Vibium.check(
    "the order confirmation is visible",
    CheckOptions.builder().record(Path.of("record.zip")).build()
);
```

Import `com.vibium.*`, `com.vibium.types.*`, and `java.nio.file.Path`. Use
`page.context().recording()` for live recording. A Browser or Page can also
call `check(claim, options)` with an archive. Returned statuses and errors
have the same meaning across all language APIs.

## Recording privacy

Export masks credential fields and known credential values in textual
recording events, including earlier actions, console observations, and raw
BiDi events. Credentials from verifier configuration are registered for
redaction without being stored as verifier metadata. Sensitive form fields
cause visual artifacts for that recording to be omitted, including video;
the ZIP retains the action timeline and a privacy notice. Existing recordings
remain active, and export never changes browser form values.

This is a defined redaction policy, not universal secret detection. Arbitrary
secrets in unrecognized application content or opaque images cannot be
reliably identified. Prefer test data and inspect artifacts before sharing.
