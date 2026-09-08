# How Run and Check work

Run carries out a browser goal. Check investigates a claim about the result.
Both use a model to choose from Vibium’s existing browser operations.

| Operation | You provide | You get back |
|-----------|-------------|--------------|
| Run | A goal: “Change the timezone and save it.” | `completed` or `not_completed`, with evidence |
| Check | A claim: “The saved timezone survives a reload.” | PASS, FAIL, or INCONCLUSIVE, with evidence |

For example, with your app running at `http://localhost:3000`:

```bash
vibium go http://localhost:3000/settings
vibium run "Set the timezone to America/Chicago and save it"
vibium check "The saved timezone is America/Chicago after reloading" -o sitecheck.zip
vibium stop
```

Run may report completion after seeing a “Saved” message even if the application
loses the setting on reload. Check asks for evidence of persistence.

For installation and a first check, start with the
[live website tutorial](../tutorials/first-check.md).

## A fresh context, the same browser

A coding agent knows what it changed and why it expects the change to work.
Check starts a new model conversation with the claim, browser observations,
and instructions to investigate. It does not inherit the builder’s conversation
or a preceding Run’s model history. This gives it a chance to notice something
the builder missed, such as the need to reload after saving.

That is what “independent” means here. The same model and provider can do both
jobs. A fresh context can offer a useful second look, but the two calls can
still make the same mistake; a reliability improvement has not been measured.

The browser session stays the same, including its cookies, login, and storage.
Check’s tools are pinned to the selected page. Both operations can interact
with the application, and Check may change values while testing a claim; it
does not restore the starting state.

Live Run and Check currently support local Chrome and Firefox. When the CLI
starts a browser for either operation, it closes it afterward. An existing
browser stays open; `--keep-open` also preserves a newly started one.

## The model chooses tools; Vibium runs them

```text
CLI / SDK / MCP
       ↓
Vibium runtime ⇄ Model in a fresh context
       ↓          requests allowed browser tools
Existing Vibium browser operations
       ↓
WebDriver BiDi ⇄ Chrome or Firefox
```

Vibium starts a live check with the page URL, an interactive element map, and
an accessibility tree. The model can request tools to read a value, click,
fill, reload, or take a screenshot. Vibium validates each request, executes
the browser operation, and returns the observation. The loop continues until
the model returns a result or reaches a limit.

Run and Check share this loop and the configured model adapters, with different
instructions, permissions, and result types. Tools are constrained in code:
the model has no shell, source-editing tools, arbitrary JavaScript execution,
or unrestricted browser-protocol access. Page observations go to the configured
model provider. Settings use shared `VIBIUM_AI_*` defaults and support
[per-call overrides](../reference/model-providers.md#override-settings-for-one-call).

The custom commands **`vibium:run.run`** and **`vibium:check.run`** identify the
whole operation inside Vibium. They travel through the existing CLI daemon,
SDK pipe, or MCP handling path. Vibium handles them; the browser receives
ordinary WebDriver BiDi commands such as `browsingContext.reload` and
`input.performActions`. No second browser automation engine is involved.

This separation could support remote browsers later without new browser-side
commands. Live Run and Check currently reject remote browser connections.

## Results you can inspect

Check returns PASS when its evidence supports the claim, FAIL when evidence
contradicts it, and INCONCLUSIVE when it cannot decide. Configuration, provider,
or browser failures return an execution error. For automated gating, inspect
the JSON status: all three completed verdicts exit with code zero.

With recording active, Run and Check appear as parent groups in the existing
Playwright-compatible recording. Each group includes the goal or claim,
model-selected child actions, and the final result with evidence. Record
Player can expand the group so you can see what led to the conclusion.

The example above uses `-o sitecheck.zip` to save that evidence. Chrome records
actions and screenshots; supported Firefox versions also record WebM video.
Console and network observation tools require an active recording. Credentials
and private model reasoning are excluded from model metadata; export applies
[recording redaction](../reference/check.md#recording-privacy), which cannot
detect every secret in arbitrary page content.

Check can also inspect an existing Vibium recording or supported Playwright
trace with `-i record.zip`. It reads saved evidence without launching a browser
or replaying actions. Its verdict describes that recorded run. See the
[recording tutorial](../tutorials/check-from-a-recording.md).

## When it earns its place

Check is useful when investigating the browser is part of the work: exploring
a changed flow, assessing an agent’s claimed fix, or examining an unfamiliar
page. A specific claim gives it something observable to test.

For stable assertions you run repeatedly, deterministic tests are usually
faster, cheaper, and more repeatable. If Check finds that a setting reverts on
reload, turn that finding into a regression test. Use another model investigation
where it can add evidence, and review the evidence behind its verdict.

For commands and setup, see [Run](../how-to-guides/run.md) and
[Check](../how-to-guides/check.md). Detailed options, recording behavior, and
limits live in the [Check reference](../reference/check.md).
