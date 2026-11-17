# Introducing Run and Check

Vibium can carry out a browser goal and independently check the result.
**Run** describes what you want done. **Check** describes what should be true.
Run acts through your existing Vibium browser session. Check can inspect that
session or a saved recording, even after the browser has closed.

```bash
vibium run "Open http://localhost:3000/account and change my timezone to America/Chicago and save" --keep-open
vibium check "The timezone is America/Chicago after refreshing. Inspect without editing the form."
vibium stop
```

Run returns COMPLETED or NOT_COMPLETED with evidence. Check returns PASS,
FAIL, or REVIEW when the evidence can't decide it (`inconclusive` in JSON).
Keeping these operations separate lets a coding agent
act on a goal, then request a fresh assessment of its work. Check does not
inherit the builder's or Run's model conversation.

That separation helps catch mistakes, but a model's judgment is not proof.
Use precise claims, read the evidence, and keep deterministic tests for
behavior you can assert directly. A different model can bring a different
perspective; it does not guarantee independent errors.

Run and Check shipped in Vibium 26.11. This post was updated for that
release: see [What's new since this post](#whats-new-since-this-post).

| Interface | Accomplish a goal | Assess a claim |
|-----------|------------------|----------------|
| CLI | `vibium run "<goal>"` | `vibium check "<claim>"` |
| Browser/Page | `run(goal)` | `check(claim)` |
| MCP | `vibium_run` | `vibium_check` |

Live operations use the existing Vibium browser session and transport. For
complete examples, see [Run](../how-to-guides/run.md) and [Check](../how-to-guides/check.md).

## Check a saved recording

You can ask Check about a run that's already finished. It reads **Vibium
recordings and Playwright traces**, so the browser doesn't need to be open
and the original run doesn't need to be reproduced.

Save a live check, then assess the recording:

```bash
vibium check "https://var.parts is up" -o sitecheck.zip
vibium check "https://var.parts was up during the recorded run" -i sitecheck.zip
```

`-o` creates a recording. `-i` reads an existing one. The input can also be a
trace from your Playwright tests. For a trace containing a checkout flow:

```bash
vibium check "The order confirmation was visible in the recorded run" -i trace.zip
```

This lets you ask new questions about a saved run, investigate a CI failure,
or get a second model's assessment of the same evidence. Each Check starts
with a fresh model context. It inspects the saved actions, screenshots, and
other available evidence without replaying the browser or changing the ZIP.
It still needs a configured model provider.

To keep the new verdict and evidence as JSON:

```bash
vibium check "https://var.parts was up during the recorded run" -i sitecheck.zip --report sitecheck-result.json
```

The verdict describes **what was recorded**. It cannot establish whether the
site works now, and missing evidence can lead to REVIEW. The reader
currently supports trace format version 8; see the
[recording and trace tutorial](../tutorials/check-from-a-recording.md) for details.

## Shared AI settings

Run and Check use one set of defaults. The quickest way to set them is:

```bash
vibium setup
```

It asks for a provider, a model and a key, writes them to
`~/.config/vibium/ai.env`, and checks the result. Vibium loads that file
for any AI variable that is empty, so there's nothing to source.

To set them by hand instead, with your OpenAI API key already exported:

```bash
export VIBIUM_AI_PROVIDER=openai
export VIBIUM_AI_MODEL=gpt-5.6-sol
export VIBIUM_AI_REASONING_EFFORT=none
# Optional: VIBIUM_AI_BASE_URL
vibium ready ai
```

Credentials use the provider's native variable, such as `OPENAI_API_KEY`.
Providers are `openai`, `anthropic`, `google`, `xai`, `openai-compatible`
and `local`.

Use per-call provider and model overrides when you want Run and Check to use
different models. See [model providers](../reference/model-providers.md).
Restart SDK/MCP runtimes after changing environment defaults; CLI invocations
read the current environment. Per-call overrides need no restart.

Shared defaults do not share inference history. Check uses verifier-specific
instructions and permissions and does not inherit Run's model conversation.
See [how Run and Check work](../explanation/run-check-webdriver-bidi-architecture.md).

## Callable objects

Connected Browser and Page objects are callable in JavaScript/TypeScript and
Python. Calling the object is an alias for its `run` method, with the same
options, result, selected page, and recording behavior.

```js
import { browser } from 'vibium';

const vibium = await browser.start();
try {
  await vibium("Open https://example.com");
  // Equivalent: await vibium.run("Open https://example.com");

  const vibe = await vibium.page();
  await vibe("Find the contact page", { model: "your-model" });
  await vibe.check("The contact page provides an email address");
} finally {
  await vibium.stop();
}
```

The synchronous JavaScript API has the same callable syntax without `await`.
TypeScript understands both the call signature and the object's methods.

```python
from vibium import browser

vibium = browser.start()
try:
    vibium("Open https://example.com")
    vibe = vibium.page()
    vibe("Find the contact page")
    vibe.check("The contact page provides an email address")
finally:
    vibium.stop()
```

Use `await` with the Python async API. Java uses the explicit `run()` method.
These examples name a connected object `vibium`; importing the package does
not create a browser session.

Public SDK types include `RunOptions`, `RunResult`, `CheckOptions`, and
`CheckResult` where applicable. See the [API reference](../reference/api.md)
for each language's signatures.

## CLI prompt shorthand

```bash
vibium "open example.com and find the contact page"
vibium "open example.com and find the contact page" -o run.zip
```

Both use the Run command, including its options and browser cleanup. Known
commands take precedence. Exactly one positional argument containing
whitespace becomes a Run prompt. Unknown single words and multiple positional
arguments remain errors, so command typos do not silently start a model run.
Quote the prompt so the shell passes it as one argument.

Use explicit Run for one-word prompts and command-name collisions:

```bash
vibium run "stop"
```

Single and double quotes are both removed by the shell before CLI parsing.
For scripts and coding agents, the explicit `vibium run "<goal>"` form is
often clearest.

## Checkbox state

`set()` and `set(true)` select a checkbox or radio. `set(false)` and
`unset()` clear a checkbox. They are idempotent; they do not toggle blindly.
Use `fill()` for text fields and `selectOption()` / `select_option()` for
select menus.

```js
await consent.set();
await consent.set(false);
await consent.unset();
const selected = await consent.isSet();
```

Java uses the same explicit methods. Python uses `set(False)` and `is_set()`.

```bash
vibium set "#consent"
vibium unset "#consent"
vibium is set "#consent"
```

MCP provides `browser_set`, `browser_unset`, and `browser_is_set`.
`browser_set` accepts an optional boolean `value`, defaulting to true.

## Recordings and skills

Recordings show **Run** and **Check** parent groups with their generated
browser actions underneath. Their method metadata is `vibium:run.run` and
`vibium:check.run`. Checkbox actions use `vibium:element.set`,
`vibium:element.unset`, and `vibium:element.isSet`.

Install the browser skill using `vibium add-skill browser` and the formal
Check skill using `vibium add-skill check`. Use `/browser` for browser work,
including Run, and `/check` for an independent acceptance check. See
[checking with a coding agent](../tutorials/check-with-a-coding-agent.md).

## What's new since this post

Run and Check kept growing between this post and the 26.11 release.

**Gate CI on the verdict.** `--expect pass` turns the verdict into an exit
code, so a pipeline doesn't have to parse JSON. ([#773](https://github.com/VibiumDev/vibium/pull/773))

```bash
vibium check "the cart contains one battery pack" --expect pass
```

| Verdict | Exit code with `--expect pass` |
|---------|--------------------------------|
| PASS | 0 |
| FAIL | 2 |
| REVIEW | 3 |
| Execution error | 1 |

REVIEW fails the gate. A pipeline can retry on 3 and hard-fail on 2.
`vibium run` takes `--expect completed`, and NOT_COMPLETED exits 2.

**REVIEW.** The third verdict is now shown as REVIEW: it needs a person to
look. JSON, reports and SDK results still say `inconclusive`. ([#775](https://github.com/VibiumDev/vibium/pull/775))

**Reasons.** A REVIEW or NOT_COMPLETED result can carry a `reason`, such as
`action_limit`, `insufficient_evidence` or `safety_stop`. ([#774](https://github.com/VibiumDev/vibium/pull/774))

**Point Run and Check at your site.** `--base-url` names the site under test, so one
claim works against localhost, a preview deploy, or production. The AI
provider's endpoint moved to `--ai-base-url`. ([#577](https://github.com/VibiumDev/vibium/pull/577), [#578](https://github.com/VibiumDev/vibium/pull/578))

```bash
vibium check "checkout completes" --base-url http://localhost:3000
vibium check "checkout completes" --base-url "$PREVIEW_URL"
```

**Reports from Run.** `vibium run --report run.json` saves the result and
evidence as JSON, like Check. ([#772](https://github.com/VibiumDev/vibium/pull/772))

**Setup and providers.** `vibium setup` writes your AI settings, and Vibium
loads them on its own. xAI and Google joined OpenAI, Anthropic, and local
models. ([#542](https://github.com/VibiumDev/vibium/pull/542), [#549](https://github.com/VibiumDev/vibium/pull/549), [#550](https://github.com/VibiumDev/vibium/pull/550), [#567](https://github.com/VibiumDev/vibium/pull/567))

See the [Check reference](../reference/check.md) for the full details.
