# Your coding agent's first verification on var.parts

You already use a coding agent to write code. Vibium gives that agent a browser
it can operate from its shell, and `vibium verify` gives it a separate verifier
to check the result.

Your part is to install Vibium, give your agent its browser and verification skills, and configure
the verifier. Then you describe the work. The agent writes the automation,
chooses browser commands, runs the check, and brings back the evidence.

For this first run, you will ask your agent to write a small cart smoke test
against [var.parts](https://var.parts). Using the demo shop lets you learn the
workflow without first building an application. The test will add one Vibium
Battery Pack to the cart and independently verify the product, quantity, and
subtotal. It will finish at the cart.

This simple claim lets you learn the agent-to-verifier handoff. Product,
quantity, and subtotal can also be checked with ordinary assertions, which are
a better fit for repeating a stable regression test. Here, Verify provides a
second opinion and a reviewable record; a PASS is not proof that the shop works.

You need a coding agent with local shell access, Node.js 18 or later with npm,
and an OpenAI API key. The setup examples below use Bash or Zsh on macOS or
Linux. This Verify implementation uses local Chrome and the Vibium CLI.

## 1. Install Vibium

In a terminal, install the CLI:

```bash
npm install -g vibium
vibium --version
```

The package installs the `vibium` command and downloads Chrome for Testing.
Your agent will call that command through its existing shell tool.

Verify is a new capability, so check that your selected binary includes it:

```bash
vibium verify --help
```

You should see help for `verify "<claim>"`.

**Development-build note:** At the time of writing, Verify is implemented in
this working tree. The npm installation route does not by itself establish
that your installed release contains Verify. If the command is missing, have
your agent build the checkout containing this implementation and use its
binary. With Go installed, the build command from that checkout's root is:

```bash
make build-go
```

Tell the agent the absolute path to that checkout's `clicker/bin/vibium` and
have it use that path consistently. You do not need to replace your global
installation to try the development build.

## 2. Give your agent the Vibium skills

Verify and its skill are currently unreleased. From your agent's project,
install both skills from the checkout containing this implementation (replace
the path):

```bash
npx skills add /absolute/path/to/vibium --skill vibe-check verify
```

Choose your coding agent when prompted and install both skills for this project.
The [skills installer](https://github.com/vercel-labs/skills) supports selecting
the target agent and installation scope. Start a new agent session if needed
for it to discover the installed skill.

`vibe-check` teaches casual browser exploration and spot-checks. `verify`
teaches the formal development loop: state an acceptance claim, call the fresh
verifier, report evidence, and fix and rerun when needed. Both use the same CLI.
You will not need to copy element references or selectors into chat yourself.

In agents with slash-command skills, use `/vibe-check` for a casual check and
`/verify` for the formal step. Codex uses `$vibe-check` and `$verify`; you can
also ask for either skill by name.

Add the following to your agent's project instructions, or ask the agent to
save it in the instruction file it uses for this project:

```text
Use the Vibium CLI for browser work in this project. Read the installed
vibe-check and verify skills and resolve the CLI binary before using them.

Use vibe-check for casual exploration. Before completing a user-visible
change, use the verify skill: exercise the relevant flow in
local Chrome. Use vibium verify with a specific, observable claim before
reporting that the behavior works. Keep the same Vibium browser session
throughout the flow and its verification.

Record the run and report the verifier's actual verdict, concise evidence,
and saved recording path. Distinguish execution errors from FAIL or
INCONCLUSIVE verdicts. A successful click alone does not establish that
the feature works.

Keep existing deterministic tests. Report PASS as the verifier's assessment
of the specific claim, and check whether its evidence covers the intended
behavior. Do not weaken an acceptance claim to obtain a pass.

Load verifier settings from ~/.config/vibium/verifier.env in the shell
invocation that runs Verify. Do not display the file, log credentials,
or put API keys in source code or chat.
```

For the development build, add its absolute binary path to these instructions.
These instructions make the tool choice and verification expectation explicit.

## 3. Configure the verifier once

Create a private environment file outside your project:

```bash
mkdir -p ~/.config/vibium
(umask 077; touch ~/.config/vibium/verifier.env)
chmod 600 ~/.config/vibium/verifier.env
```

Open that file in your editor and enter these settings, replacing the key
placeholder with your API key:

```dotenv
VIBIUM_VERIFIER_PROVIDER=openai
VIBIUM_VERIFIER_MODEL=gpt-5.6-sol
VIBIUM_VERIFIER_REASONING_EFFORT=none
OPENAI_API_KEY='replace-with-your-api-key'
```

This example uses `gpt-5.6-sol`, which needs `reasoning_effort=none` for function
tools through Chat Completions. Your API account must have access to the model.
For another model or an OpenAI-compatible endpoint, see the
[configuration how-to](../how-to-guides/verify.md).

The agent should load the settings into the process that runs Verify. For a
Bash or Zsh shell, that means running this prefix and the `vibium verify`
command in the same shell invocation:

```bash
set -a
source ~/.config/vibium/verifier.env
set +a
```

You do not need to execute that prefix before each test yourself. The project
instructions tell the agent to do it. This also avoids relying on a desktop
agent inheriting variables from an unrelated terminal. The environment file
is not loaded automatically by Vibium.

The verifier may use the same provider and account as your coding agent. It
still starts a fresh inference conversation with verifier-specific instructions
and constrained browser tools.

## 4. Check the agent's setup

Open your coding agent in the project and send:

> Read the installed vibe-check and verify skills. Confirm that you can run the Vibium CLI
> from your shell and that `vibium verify --help` is available. Confirm that the
> verifier environment file exists and the required variables are set after
> sourcing it, without printing their values. Use the configured development
> binary if one is specified in the project instructions.

The agent should report that it can find the skill, the CLI, and the verifier
configuration. If it cannot find the CLI, give it the binary's absolute path.
If it cannot find the skill, check the project and agent you selected during
installation. This check confirms setup; the next step exercises the provider
and browser together.

## 5. Ask the agent to write and run the smoke test

Give the agent this task:

> Use the verify skill. Write a small reusable cart smoke test in this project using the Vibium CLI,
> then run it against https://var.parts.
>
> Start an isolated named local Chrome session and record the run. Open the
> Vibium Battery Pack product, add it to the cart exactly once, and open the
> cart. Inspect the live page to choose controls; do not guess element refs.
>
> Use `vibium verify` in that same session to check this claim: “The current
> cart contains exactly one Vibium Battery Pack with quantity 1, and the
> subtotal matches its unit price. Inspect the current cart without changing
> its contents or proceeding to checkout.”
>
> Load the verifier environment file without displaying it. Save the
> recording and close only the session created for this test. Return the
> script path, verifier verdict, concise evidence, and recording path. If the
> script checks the result automatically, inspect the JSON status: PASS,
> FAIL, and INCONCLUSIVE all have CLI exit code zero.

The agent is now doing the implementation work. It may create a shell script
or use an existing project scripting language to invoke the CLI. You do not
need to write the browser commands or introduce an SDK or MCP connection.

As it runs, you should see Chrome open, the product page appear, and the cart
receive one battery pack. The agent may issue commands such as `vibium map`,
`vibium find`, and `vibium click`. Those are the agent's browser tools; the
important handoff is its call to `vibium verify` after it establishes the cart
state.

Vibium supplies a fresh verifier with the current URL, element map,
accessibility tree, and browser tools. That verifier investigates the claim
in the same tab. Allow the run to finish before interacting with its browser.

## 6. Read the result your agent brings back

The cart check passed when this flow was tested on September 4, 2026. Your
agent's response should contain the same kinds of information, with its actual
file paths and observations:

```text
Created: scripts/check-cart.sh
Verification: PASS

Evidence:
- The cart contains one Vibium Battery Pack at quantity 1.
- Its displayed unit price and the cart subtotal match.

Recording: first-verify-<timestamp>.zip
```

For a completed example to compare with your agent's work, see the
[CLI smoke test](../../scripts/var-parts-verify.mjs). It keeps its browser
session isolated and checks the actual JSON status.

Exact wording, script language, and verifier actions can vary. Read the evidence
along with the verdict. This PASS supports the cart claim in that session;
payment and order confirmation have not been checked. The verifier can also
misread evidence or miss a condition, even with a fresh context. This tutorial
demonstrates a working workflow, not a measured improvement in reliability.

FAIL means the verifier observed something that contradicted the claim.
INCONCLUSIVE means it could not establish or refute it. An API or browser
execution error is reported separately. Your agent should preserve those
outcomes in its report rather than describing every completed run as a pass.

Open [Record Player](https://player.vibium.dev) and drop the saved zip onto it.
Find the **Verify** group. You can inspect the agent's product-selection and
Add to Cart actions, then the verifier's observations, verdict, and evidence.
Chrome may report video unavailable; the screenshots and action trace are
sufficient for this tutorial.

## Optional: ask a stronger question

Once the basic test works, ask your agent:

> Extend the smoke test with a second verification after the first. Check:
> “Refreshing the current cart keeps exactly one Vibium Battery Pack in it.
> Check the current cart without adding or removing items or proceeding to
> checkout.” Run the test in a new isolated session, record both checks, and
> report each verdict separately. Keep the observed verdicts even if one fails.

In the tutorial rehearsal, the first check passed and the refresh check failed:

```text
FAIL

The cart did not retain one Vibium Battery Pack after refresh; it became empty.
- Before refresh: one Vibium Battery Pack, quantity 1.
- After refresh: “Your cart is empty.”
```

These results assess different claims. Adding a product worked; keeping it
across a refresh failed. Your result may change as the live site changes.
The stronger claim prompted the useful additional check. A fresh context gives
another opportunity to investigate, but a builder or deterministic test could
also have discovered the failure by reloading the cart.
The second verification can leave the cart empty, which is why the test owns
its session and cleans it up.

For your own application, this is where you would ask the coding agent to
investigate the failure, fix the implementation, and run the check again.
The demo exercise does not require access to the var.parts source code.

## Use the same workflow while building your app

With the setup in place, an ordinary feature request can include the check you
want the agent to perform:

> Implement editing the display name on our account page. Run the app locally,
> exercise the change with the Vibium CLI, and use `vibium verify` to check
> that the saved name remains after refresh. Include the verdict, evidence,
> and recording path in your completion report.

You specify the intended behavior. The coding agent writes the code and
exercises the application. The fresh verifier examines the result. For why
those roles are separated, see
[Why independent verification matters](../explanation/independent-verification.md).
