# Part 2: Check with your coding agent

**Check tutorials:** [Part 1: Live websites](first-check.md) · Part 2 · [Part 3: Recordings and traces](check-from-a-recording.md)

Ask your coding agent to write and run a small cart test on
[var.parts](https://var.parts). It will add one Vibium Battery Pack, then hand
a claim about the cart to Check. You'll get a script, a verdict, and a recording.

Your job is to install the skills and describe the task. The agent handles the
code and browser commands.

## Before you start

Complete [Part 1's setup](first-check.md#before-you-start), or reuse a working
Vibium installation and AI configuration. You also need a coding agent with
local shell access. These examples use Bash or Zsh and local Chrome.

The commands below assume the settings file from Part 1:
`~/.config/vibium/ai.env`. If yours is elsewhere, give the agent that path.
If you're using `clicker/bin/vibium`, give it the absolute binary path too.

## 1. Install the browser and check skills

From the project where your agent will write the test, install the skills from
the checkout containing this development build:

```bash
npx skills add /absolute/path/to/vibium --skill browser check
```

Replace the path, choose your coding agent, and install for this project.
Start a new agent session if needed to discover the skills.

- **browser** handles navigation, interaction, debugging, and recording. It
  also covers `vibium run` when a model should choose the browser steps.
- **check** runs `vibium check` against an explicit claim and reports its verdict
  and evidence.

In agents with slash-command skills, use `/browser` and `/check`. In Codex,
use `$browser` and `$check`. Asking for the installed skill by name also works
in agents that support it.

## 2. Configure your agent once

Ask it to save the following in its project instructions:

```text
Use the Vibium CLI for browser work. Use the browser skill for navigation,
interaction, debugging, and recording. Use explicit CLI commands for known
steps, or vibium run to delegate a browser goal.

Use the check skill when an independent assessment of an acceptance claim is
needed. Run's completed result does not replace a Check verdict.

Before Run, Check, or AI readiness, source ~/.config/vibium/ai.env in the
same shell invocation. Do not display the settings file or log credentials.
Keep the same Vibium session throughout a browser workflow and its check.
```

Then ask it to confirm setup:

> Read the installed browser and check skills. Confirm that the configured
> Vibium binary supports `run` and `check`. Load the AI settings file without
> displaying it, then run `vibium ready` in that same shell invocation.
> Report whether it says READY, or what setup needs fixing.

Continue when browser installation and AI checks pass. Readiness inspects
browser files and contacts the model provider. It does not open a browser,
test browser connectivity, or check your application.

## 3. Ask the agent to write and run the test

Send this task:

> Use the browser skill to write a small reusable cart smoke test in this
> project using the Vibium CLI. Run it against https://var.parts in a new named
> local Chrome session, and record the flow.
>
> Find the Vibium Battery Pack, add it exactly once, and open the cart. Inspect
> the live page to choose the controls; do not proceed to checkout.
>
> Then use the check skill in that same session with this claim:
> “The current cart contains exactly one Vibium Battery Pack with quantity 1,
> and the subtotal matches its unit price. Inspect without changing the cart
> or proceeding to checkout.”
>
> Save the recording even if the check fails. Close only the session this test
> created. Return the script path, actual verdict, concise evidence, and
> recording path. If the script gates on success, inspect the JSON status;
> PASS, FAIL, and INCONCLUSIVE all have CLI exit code zero.

Chrome opens and the agent works through the product and cart pages. It writes
the script and chooses the CLI commands; you don't need to supply selectors.

When it calls `vibium check`, a fresh model context examines the same browser
page. It doesn't inherit the coding agent's conversation. It can use the same
model provider as your coding agent.

For this small flow, direct CLI commands are enough. Run is available through
the browser skill for goals where choosing the steps needs more investigation;
you don't need to insert it before every Check.

## 4. Review the result

A successful agent response might look like this:

```text
Created: scripts/check-cart.sh
Check: PASS

Evidence:
- The cart contains one Vibium Battery Pack at quantity 1.
- Its displayed unit price and the cart subtotal match.

Recording: first-check-<timestamp>.zip
```

Your agent should report the actual verdict, including FAIL or INCONCLUSIVE.
This claim covers the cart contents and subtotal, not payment or order confirmation.

Open [Record Player](https://player.vibium.dev) and drop the ZIP onto it. Look
for the agent's browser actions and the **Check** group containing the model's
child actions and verdict. A Chrome recording includes screenshots and the
action timeline; it has no continuous video.

For a working script to compare with your agent's code, see the
[CLI cart example](../../scripts/var-parts-check.mjs).

## Try it in your development loop

On your own app, combine the two skills in a coding request:

> Implement editing the timezone on our account page. Run the app locally and
> use the browser skill to exercise the change. Then use the check skill to
> assess whether the saved timezone remains after refresh. Report the verdict,
> evidence, and recording path. If it fails, investigate and fix the cause,
> then rerun the same claim.

A useful finding can become a deterministic regression test. Keep those tests
for behavior you need to check repeatedly.

Next, [check evidence from a recording or trace](check-from-a-recording.md).
For the underlying model and browser flow, read
[How Run and Check work](../explanation/run-check-webdriver-bidi-architecture.md).
