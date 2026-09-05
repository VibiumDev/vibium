# Verify a live browser or saved recording

Use `vibium verify` to check a claim against the current page in your local
Vibium Chrome or Firefox session. It returns PASS, FAIL, or INCONCLUSIVE with concise
evidence. You need a CLI build that includes Verify and a configured verifier
model.

For a first guided run, follow [Your coding agent's first verification on var.parts](../tutorials/first-verification.md).
For context and design, read [Why independent verification matters](../explanation/independent-verification.md).

## Choose a useful claim

Use Verify for an additional acceptance check when inspecting the running
behavior requires exploration or judgment. For a simple, stable assertion you
need to repeat, prefer a deterministic test. Verify adds model latency and
possibly inference cost; it should add useful evidence to the workflow.

State the intended outcome, not just a successful action: “The changed display
name persists after refresh” checks more than “Clicking Save shows a message.”
Keep the claim faithful to the requirement. Split a broad requirement into
separate checks without dropping its acceptance conditions.

## Configure a verifier

Export the provider, a model that supports Chat Completions function tools, and
your API key in the terminal running the CLI. For example, with a key already
available as `OPENAI_API_KEY`:

```bash
export OPENAI_API_KEY
export VIBIUM_VERIFIER_PROVIDER=openai
export VIBIUM_VERIFIER_MODEL=gpt-5.6-sol
export VIBIUM_VERIFIER_REASONING_EFFORT=none
```

`gpt-5.6-sol` requires `none` for function tools through Chat Completions.
`VIBIUM_VERIFIER_REASONING_EFFORT` is optional for the adapter and is forwarded
only when set; choose a value supported by your model. For a model that does
not accept this setting, unset the variable.

For an OpenAI-compatible server:

```bash
export VIBIUM_VERIFIER_PROVIDER=openai-compatible
export VIBIUM_VERIFIER_BASE_URL=http://localhost:1234/v1
export VIBIUM_VERIFIER_MODEL='your-tool-capable-model'
unset VIBIUM_VERIFIER_REASONING_EFFORT
```

`OPENAI_API_KEY` is optional in compatible mode; if set, it is sent as a Bearer
authorization header to the configured endpoint. The server must support
function tools and `max_completion_tokens`. Image input support is needed for
the screenshot tool.

Configuration is read by each CLI invocation, so changing it does not require
restarting the daemon or browser.

## Run the check in your existing session

Keep using the terminal and `VIBIUM_SESSION` value that control your browser.
Enable recording before the workflow if you want a saved timeline:

```bash
vibium record start --name account-verification
vibium go http://localhost:3000/account
vibium verify "changing my display name persists after refresh"
vibium record stop
```

Substitute your application URL and claim. The claim must be nonempty and at
most 4,000 bytes. The verifier can operate the page and may leave it changed.
Normal CLI browser autostart applies if no browser has been launched; an
existing session is reused. Local Chrome and Firefox are supported for live verification.

The recording stop command prints the saved zip's location. Open it in
[Record Player](https://player.vibium.dev) and inspect the Verify group, its
browser actions, and its verdict. Recording is optional for live verification;
console and network observation tools require an active recording.

## Save just the verification run

Use `-o` / `--output` to create a recording without separate start/stop commands:

```bash
vibium verify "changing my display name persists after refresh" -o verification.zip
```

Verify starts a recording with screenshots and browser actions, then saves and
stops it when verification finishes. It also saves available evidence if the
provider or another part of verification fails. This automatic recording does
not include continuous video.

If recording is already active, `--output` exports the **entire current chunk
through Verify**, including earlier builder actions. It leaves the ongoing
recording, open groups, video track, and original destination intact. The export
contains no continuous video file. Choose a different destination from the
active recording. Output files must be new; existing files are never replaced.

## Verify an existing recording or Playwright trace

Use `-i` / `--input` for either kind of saved evidence:

```bash
vibium verify -i record.zip "the changed display name remained visible after reload"
vibium verify --input trace.zip "the order confirmation is visible"
```

These commands use read-only inspection tools and do not launch a browser or
replay actions. The source ZIP stays unchanged, and no new artifact is created
by default. The current reader supports Playwright trace format version 8;
malformed archives and unsupported versions return execution errors.

The verifier can inspect actions, DOM snapshots, requested screenshots,
network summaries, console events, and earlier Verify spans. A prior PASS is
an assessment to examine, not proof. Missing evidence should produce
INCONCLUSIVE. A recording cannot establish what a later live session does.

`--record` and `--trace` are not accepted flags. `--input` and `--output` cannot
be combined yet; use `--report` to save an archive verification verdict.

## Read a structured result

Use the global `--json` flag:

```bash
vibium --json verify "changing my display name persists after refresh"
```

To save the verdict separately, use `--report`:

```bash
vibium verify -i record.zip "the order confirmation is visible" --report verdict.json
vibium verify "changing my display name persists after refresh" -o verification.zip --report verdict.json --json
```

The first command writes only a JSON report. The second saves a live recording
and JSON report and prints JSON to stdout. `--report` writes the result object
shown below, without the outer CLI envelope. Its path must be new and different
from the recording output; an operational error leaves no verdict report.

The existing CLI stdout envelope contains a structured result:

```json
{
  "ok": true,
  "result": {
    "status": "passed",
    "claim": "changing my display name persists after refresh",
    "summary": "The new display name persisted after refresh.",
    "evidence": [
      {"type": "observation", "summary": "The saved name remained visible after reload."}
    ]
  }
}
```

Completed verdicts have status `passed`, `failed`, or `inconclusive`. All three
exit with code zero: `ok: true` means the verification completed. To gate a
workflow on success, check `result.status == "passed"`.

A PASS is the verifier's assessment of the stated claim in this session. Read
the evidence and, when available, the recording to judge whether the check
actually exercised the acceptance condition. The model can be wrong; retain
existing tests and do not treat a PASS as proof of overall correctness.

Configuration, provider, browser execution, and timeout failures return errors
and exit 1. In JSON mode they use `{"ok":false,"error":"..."}`. They are not
verification verdicts.

## Handle an incomplete run

A run has a three-minute budget and at most 24 model-selected actions. If the
claim needs too much investigation, split it into bounded checks that preserve
the original acceptance conditions. Text observations and screenshots have
payload limits; an INCONCLUSIVE result can reflect insufficient evidence.
Polling or recording cleanup may take additional time after a timeout.

Check the reported provider error and your configuration if inference fails.
For browser failures, inspect the page and save the active recording with
`vibium record stop`. If Verify started its own recording with `--output`,
it already stopped and saved that recording. Inspect it to understand which
actions completed before the error.

For details of session reuse, tools, recording metadata, and the current
permissions and privacy limits, see the
[independent verification explanation](../explanation/independent-verification.md).
The executable local Chrome checks live in
[the Verify end-to-end tests](../../tests/daemon/verify.test.js).

## MCP and language APIs

Configure the verifier environment before starting the MCP server or SDK
runtime. These surfaces read configuration in the Go process; restart that
runtime after changing its environment. All invoke the same verifier adapter,
result validation, and bounded tool loop used by the CLI.

MCP exposes `vibium_verify`:

```json
{"claim":"changing my display name persists after refresh","page":"existing-context-id"}
```

Omit `page` to use the active page. To inspect an archive, pass
`{"claim":"the order confirmation is visible","record":"/absolute/path/record.zip"}`.
`record` and `page` cannot be combined. Archive mode does not start a browser.
Use existing recording tools around live MCP checks to save their evidence.

JavaScript / TypeScript:

```js
import { browser } from 'vibium';
const bro = await browser.start({ headless: true });
const page = await bro.page();
await page.go('http://localhost:3000/account');
await page.context.recording.start({ path: 'verification.zip', video: false });
try {
  const result = await page.verify('changing my display name persists after refresh');
  console.log(result.status, result.evidence);
} finally {
  await page.context.recording.stop();
  await bro.stop();
}
// Standalone archive inspection: no browser installation or launch.
const recorded = await browser.verify('the order confirmation is visible', { record: './record.zip' });
```

The synchronous API has the same methods via `vibium/sync`; omit `await`.
`Browser.verify()` uses the active page, while `Page.verify()` pins that Page.
Both also accept `{ record: './record.zip' }`, which selects read-only evidence
instead of inspecting their live browser.

Python:

```python
from vibium import browser
bro = browser.start(headless=True)
try:
    page = bro.page()
    page.go('http://localhost:3000/account')
    result = page.verify('changing my display name persists after refresh')
    print(result['status'], result['evidence'])
finally:
    bro.stop()
recorded = browser.verify('the order confirmation is visible', record='./record.zip')
```

For async Python, import from `vibium.async_api` and await browser operations
and Verify. Use `page.context.recording.start()` / `.stop()` around a live
check to save its evidence.

Java:

```java
Browser bro = Vibium.start(new StartOptions().headless(true));
try {
    Page page = bro.page();
    page.go("http://localhost:3000/account");
    VerificationResult result = page.verify("changing my display name persists after refresh");
    System.out.println(result.status());
} finally {
    bro.stop();
}
VerificationResult recorded = Vibium.verify(
    "the order confirmation is visible",
    VerifyOptions.builder().record(Path.of("record.zip")).build()
);
```

Import `com.vibium.*`, `com.vibium.types.*`, and `java.nio.file.Path`. Use
`page.context().recording()` for live recording. A Browser or Page can also
call `verify(claim, options)` with an archive. Returned statuses and errors
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
