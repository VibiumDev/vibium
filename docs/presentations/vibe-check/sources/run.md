# Accomplish a browser goal with Run

`vibium run` uses the live browser to carry out a goal and returns
**COMPLETED** or **NOT_COMPLETED**, with concise evidence. Use Check separately
when you want a fresh assessment of the result.

For coding agents, Run guidance is part of the
[`browser` skill](../../skills/browser/SKILL.md). The separate
[`check` skill](../../skills/check/SKILL.md) handles independent verification.

## Configure Run

Run and Check use the same `VIBIUM_AI_*` defaults.
For OpenAI, with your API key already exported:

```bash
export VIBIUM_AI_PROVIDER=openai
export VIBIUM_AI_MODEL=gpt-5.6-sol
export VIBIUM_AI_REASONING_EFFORT=none
vibium ready
```

Wait for READY. For Anthropic, Google, or a local server, see
[model providers](../reference/model-providers.md). Settings files are not
loaded automatically; source a file containing exported assignments in the
same shell that runs Vibium.

For one run, use `--provider`, `--model`, `--base-url`, and `--reasoning-effort`
instead of changing environment defaults. These options also work with
`ready` and `ready ai`. See [per-call settings](../reference/model-providers.md#override-settings-for-one-call).

## Accomplish the goal

Navigate to your app, then describe the change you want:

```bash
vibium go http://localhost:3000/account
vibium run "change my timezone to America/Chicago and save it"
```

Example result:

```text
RUN: change my timezone to America/Chicago and save it

COMPLETED

The timezone was changed and saved.
- The account page reports the saved timezone as America/Chicago.
```

Read the evidence. COMPLETED is the model's assessment, not proof. If the goal
is ambiguous or required information is missing, Run can return
NOT_COMPLETED. Provider, browser, and configuration failures are errors.

Run reuses the existing session and selected page. It can change page
state and does not undo its actions. Supply a clear goal and appropriate test
data. It can enter explicitly supplied test credentials through its browser
tools; Check retains its stricter restriction on password fields.

## Shorthand

A connected Browser or Page in JavaScript/TypeScript and Python is callable:
`vibe(goal, options)` delegates to `vibe.run(goal, options)` (Python uses keyword
options). Use `await` with async APIs. It preserves the same browser session.
Java uses the explicit `run()` method.

```js
import { browser } from 'vibium';

const vibium = await browser.start();
try {
  await vibium("Open https://example.com");
  // Equivalent: await vibium.run("Open https://example.com");
  await vibium.check("The page shows Example Domain");
} finally {
  await vibium.stop();
}
```

Here `vibium` is a connected browser object; importing the package does not
create a browser session.

The CLI accepts one quoted multiword prompt:

```bash
vibium "open example.com and find its contact page"
# Equivalent to vibium run "open example.com and find its contact page".
```

Known subcommands take precedence. Unknown single words and multiple positional
arguments are errors; use `vibium run "stop"` for a one-word prompt.

For shared settings, callable examples in both languages, checkbox methods, and
skills, see [Introducing Run and Check](../updates/2026-09-07-run-and-check.md).

## Save the run

```bash
vibium run "change my timezone to America/Chicago and save it" -o run.zip
```

The recording contains a **Run** group with generated browser actions and
the result. Open the ZIP in [Record Player](https://player.vibium.dev).
Recording output, Firefox WebM capture, existing-recording exports, and
[privacy rules](../reference/check.md#recording-privacy) work as they do for Check.
Use a new filename; existing files are never overwritten.

If Run starts a browser, it closes it after saving evidence, including
when it returns NOT_COMPLETED or encounters an error. A browser already
running stays open. Add `--keep-open` to preserve one it starts:

```bash
vibium run "open https://example.com" -o example.zip --keep-open
```

Close it later with `vibium stop`. Run is live-only: it has no `--input`
or `--report` option. `--json` prints the structured result instead of text.
Completed operations exit 0 for either status; execution errors exit 1.
In automation, inspect `result.status == "completed"` in JSON stdout.

## Follow with Check

Keep the session open for a separate acceptance check:

```bash
vibium run "change my timezone to America/Chicago and save it" --keep-open
vibium check "The saved timezone is America/Chicago after refresh. Inspect without editing the form."
```

Run and Check can use different providers and models. Check starts a
fresh conversation and does not inherit Run's model history.

## Use MCP or a language API

All inference runs in the existing Go runtime. Configure environment defaults
before starting the MCP server or SDK; restart it after changing those defaults.
Per-call model overrides need no restart. Browser and Page instances expose `run(goal)`;
Page pins the operation to that page. SDK calls preserve the caller's browser.

MCP:

```json
{"name":"vibium_run","arguments":{"goal":"change my timezone to America/Chicago and save it"}}
```

An optional `page` context ID selects a specific tab. `record` is not accepted.

JavaScript/TypeScript:

```js
const result = await page.run('change my timezone to America/Chicago and save it');
console.log(result.status, result.evidence);
```

The synchronous API has the same method without `await`.

Python:

```python
result = page.run("change my timezone to America/Chicago and save it")
print(result["status"], result["evidence"])
```

Use `await page.run(...)` with the async API.

Java:

```java
RunResult result = page.run("change my timezone to America/Chicago and save it");
System.out.println(result.status());
```

Import `com.vibium.types.RunResult`. Start and stop an SDK recording around
these calls to save evidence. All interfaces use the same three-minute model
budget, 24-action limit, tool policies, and `completed | not_completed` contract.
