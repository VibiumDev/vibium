# Part 2: Verify with your coding agent

**Verify tutorials:** [Part 1: Live websites](first-verification.md) · Part 2 · [Part 3: Recordings and traces](verification-from-a-recording.md)

In this tutorial, you'll give your coding agent a browser and ask it to write
and run a small cart test on [var.parts](https://var.parts). The agent will add
one Vibium Battery Pack, then use `vibium verify` to check the cart with a fresh
verifier. You'll get a script, a verdict, and a recording to review.

Your job is setup and describing the task. The agent writes the code and runs
the browser commands through the CLI.

If you completed Part 1, reuse your Vibium installation and verifier settings.
Add the agent skills in step 1, then continue at step 3.

You need a coding agent with local shell access, Node.js 18 or later with npm,
and an OpenAI API key. These instructions use Bash or Zsh on macOS or Linux
and local Chrome. **Verify currently requires the development build in this
checkout**, so you also need Go if your installed CLI lacks the command.

## 1. Install Vibium and the agent skills

Install the CLI and its browser:

```bash
npm install -g vibium
vibium verify --help
```

If Verify is missing, ask your agent to run `make build-go` from the Vibium
checkout containing this implementation. Have it use the resulting
`clicker/bin/vibium` by absolute path throughout this tutorial. The global
installation can stay in place.

From the project where your agent will write the test, install both skills.
Replace the path with the location of that Vibium checkout:

```bash
npx skills add /absolute/path/to/vibium --skill vibe-check verify
```

Choose your agent and install the skills for this project. Start a new agent
session if it needs one to discover them.

The skills have different jobs:

- **vibe-check**: casual browser exploration and spot-checks.
- **verify**: a formal check of a stated outcome, with a verdict and evidence.

In agents with slash-command skills, invoke `/vibe-check` or `/verify`. In
Codex, use `$vibe-check` or `$verify`. You can also ask for a skill by name.

## 2. Configure the verifier

The verifier needs its own API settings, even if your coding agent already
has a subscription. Create a private settings file outside your project:

```bash
mkdir -p ~/.config/vibium
(umask 077; touch ~/.config/vibium/verifier.env)
chmod 600 ~/.config/vibium/verifier.env
```

Open the file in your editor and add:

```bash
export VIBIUM_VERIFIER_PROVIDER=openai
export VIBIUM_VERIFIER_MODEL=gpt-5.6-sol
export VIBIUM_VERIFIER_REASONING_EFFORT=none
export OPENAI_API_KEY='replace-with-your-api-key'
```

Keep `export` on each line so Vibium receives the settings when you source
the file. Use your actual API key in the file, without putting it in chat or
source code.
Your API account needs access to the selected model. The `none` setting above
is required for this model's tool calls. For other models or compatible
servers, see the [configuration reference](../reference/verify.md#provider-configuration).

Vibium does not load this file automatically. The next step tells your agent
to load it whenever it runs Verify. You won't need to do that by hand each time.

## 3. Tell your agent how to use Vibium

Ask the agent to save these instructions in the project instruction file it
normally uses. Include the absolute CLI path if you built it in step 1:

```text
Use the Vibium CLI for browser work. Read the installed vibe-check and verify
skills. Use vibe-check for casual checks. Use verify as an acceptance check
when a user-visible change needs browser investigation, and keep existing tests.

Before running Verify, load ~/.config/vibium/verifier.env into the same shell
invocation: source ~/.config/vibium/verifier.env.
Do not display the file or log credentials.

Exercise the relevant flow, then call vibium verify with a specific claim
about the intended outcome. Use the same Vibium session for both. Record the
run and report the actual verdict, evidence, and recording path. Check whether
the evidence covers the requirement; do not weaken the claim to obtain a PASS.
```

Then ask it to run the setup check:

> Read the installed vibe-check and verify skills. Confirm that you can run
> `vibium verify --help`. Load the verifier settings file without displaying
> it, then run `vibium soundcheck` in that same shell invocation. Use the
> configured development binary if one is specified and report any setup fixes.

Soundcheck tests configuration, API access, and the model's tool support. It
makes up to two small model requests (API charges may apply) without opening
a browser. Wait for READY before continuing. If a check fails, follow its fix.
You should now have a CLI the agent can call, two skills it can read, and
verifier settings it can load.

## 4. Ask the agent to write and run the test

Send this task:

> Use the verify skill. Write a small reusable cart smoke test in this project
> using the Vibium CLI, then run it against https://var.parts.
>
> Start an isolated named local Chrome session and record the run. Find the
> Vibium Battery Pack, add it to the cart exactly once, and open the cart.
> Choose controls by inspecting the live page.
>
> In that same session, use `vibium verify` to check: “The current cart contains
> exactly one Vibium Battery Pack with quantity 1, and the subtotal matches its
> unit price. Inspect the current cart without changing its contents or
> proceeding to checkout.”
>
> Save the recording even if the check fails, and close only the session this
> test created. Return the script path, actual verdict, concise evidence, and
> recording path. If the script gates on success, inspect the JSON status:
> PASS, FAIL, and INCONCLUSIVE all have CLI exit code zero.

You should see Chrome open and the agent work through the product and cart
pages. Let it finish before interacting with the browser. The agent chooses
the CLI commands and writes the script; you don't need to supply selectors.

When it calls `vibium verify`, a fresh verifier examines the same browser tab.
It gets the claim and browser observations, without the coding agent's
conversation. The verifier may use the same model or provider as your agent.

## 5. Review the result

A successful run might look like this:

```text
Created: scripts/check-cart.sh
Verification: PASS

Evidence:
- The cart contains one Vibium Battery Pack at quantity 1.
- Its displayed unit price and the cart subtotal match.

Recording: first-verify-<timestamp>.zip
```

Read the evidence as well as the verdict. This check covers the cart's product,
quantity, and subtotal; it doesn't cover payment or order confirmation.

- **PASS**: the verifier found evidence supporting the claim.
- **FAIL**: it found evidence contradicting the claim.
- **INCONCLUSIVE**: it didn't have enough evidence to decide.

API or browser errors are reported separately. Your agent should report the
actual outcome, even when it differs from the example.

Open [Record Player](https://player.vibium.dev) and drop the ZIP onto it. You'll
see the agent's actions followed by a **Verify** group containing the verifier's
actions and result. Continuous video is optional; screenshots and the action
timeline are enough for this exercise.

For a working script to compare with your agent's code, see the
[CLI smoke test](../../scripts/var-parts-verify.mjs).

## Optional: check what happens after refresh

Ask the agent to add a second check:

> Extend the test with another verification: “Refreshing the current cart keeps
> exactly one Vibium Battery Pack in it. Check without adding or removing items
> or proceeding to checkout.” Run both checks in a new isolated session and
> report each verdict, even if one fails.

In the tutorial rehearsal, the cart check passed but the refresh check failed:
the cart became empty after reload. Your result may differ as the site changes.
The two verdicts answer different questions—adding the product and retaining
it after refresh. On your own app, a failure is the point to investigate, fix
the code, and rerun the same claim.

## Use it while building your own app

With setup done, you can include verification in an ordinary coding request:

> Implement editing the display name on our account page. Run the app locally,
> exercise the change with the Vibium CLI, and use the verify skill to check
> that the saved name remains after refresh. Report the verdict, evidence,
> and recording path.

Verify gives you a second opinion, and it can be wrong. This tutorial's simple
cart claim teaches the handoff; an ordinary assertion is a better choice for
repeating that stable check. Keep your tests and use Verify where additional
browser investigation is useful.

Next, use a saved ZIP in
[Part 3: Verify a recording or trace](verification-from-a-recording.md).
For everyday commands, use the [how-to guide](../how-to-guides/verify.md). Read
[Why independent verification matters](../explanation/independent-verification.md)
for the benefits and limits of a fresh verifier.
