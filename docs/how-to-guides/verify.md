# Verify a claim in local Chrome

Use `vibium verify` to check a claim against the current page in your local
Vibium Chrome session. It returns PASS, FAIL, or INCONCLUSIVE with concise
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
existing session is reused. This implementation supports local Chrome only.

The recording stop command prints the saved zip's location. Open it in
[Record Player](https://player.vibium.dev) and inspect the Verify group, its
browser actions, and its verdict. Recording is optional for live verification;
console and network observation tools require an active recording.

## Read a structured result

Use the global `--json` flag:

```bash
vibium --json verify "changing my display name persists after refresh"
```

The existing CLI envelope contains a structured result:

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
`vibium record stop`. A saved recording can help you review the failure, but
passing `--record` or `--trace` into Verify is not implemented yet.

For details of session reuse, tools, recording metadata, and the current
permissions and privacy limits, see the
[independent verification explanation](../explanation/independent-verification.md).
The executable local Chrome checks live in
[the Verify end-to-end tests](../../tests/daemon/verify.test.js).
