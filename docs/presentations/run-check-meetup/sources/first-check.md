# Part 1: Check a live website

**Check tutorials:** Part 1 · [Part 2: Coding agents](check-with-a-coding-agent.md) · [Part 3: Recordings and traces](check-from-a-recording.md)

Check whether a website is up, then save the result, screenshots, and browser
actions in a recording. You don't need a coding agent or a test script.

## Before you start

You need Node.js with npm and a Vibium build that includes Check. Install
Vibium using the [installation instructions](../../README.md#agent-setup), then confirm:

```bash
vibium check --help
```

The shell examples use Bash or Zsh on macOS or Linux.
These tutorials describe the development build. If your installed version
has no `check` command, use the development binary supplied with your checkout
(`./clicker/bin/vibium`) in place of `vibium` below. Build instructions are
available for [macOS](../contributing/local-dev-setup-mac.md) and
[Linux](../contributing/local-dev-setup-x86-linux.md).

## 1. Configure AI access

Run and Check use Vibium's own model configuration. Your coding agent's
subscription does not configure it automatically. This example uses OpenAI;
for other providers, use the [provider guide](../reference/model-providers.md).

If you already have working settings, keep them and load them below.
Otherwise, create a private settings file:

```bash
mkdir -p ~/.config/vibium
(umask 077; touch ~/.config/vibium/ai.env)
chmod 600 ~/.config/vibium/ai.env
```

Open `~/.config/vibium/ai.env` in your editor and add:

```bash
export VIBIUM_AI_PROVIDER=openai
export VIBIUM_AI_MODEL=gpt-5.6-sol
export VIBIUM_AI_REASONING_EFFORT=none
export OPENAI_API_KEY='replace-with-your-api-key'
```

Use a model your API account can access. This model uses `none` for tool calls;
other models may use different settings. Keep `export` on each line so the
settings reach Vibium, and keep your actual key in the file rather than chat
or project source code.

Load the file in the terminal you'll use for this tutorial:

```bash
source ~/.config/vibium/ai.env
vibium ready
```

Wait for browser installation and AI checks to pass. Readiness checks browser
files without opening a browser. If one is missing, run the suggested
`vibium install` command, then retry. Other failures also include a suggested fix.
Vibium does not load the file automatically, so source it again in each new
shell. These shared settings work for both Run and Check.

## 2. Check a website

```bash
vibium check "https://var.parts is up"
```

A browser opens and the result appears in your terminal. A successful result
might look like this (illustrative output):

```text
CHECK: https://var.parts is up

PASS

https://var.parts is reachable and renders successfully, including after a reload.
- Navigation completed at https://var.parts/.
- The page rendered visible content.
- After reloading, the page rendered again.
```

**PASS** means the evidence supports the claim. **FAIL** means it contradicts
the claim. **INCONCLUSIVE** means the model couldn't decide. Your result may
differ; this is a small availability check, not a test of every site feature.

A browser started by Check closes afterward. An existing Vibium browser stays
open. Add `--keep-open` to preserve a newly started browser too.

## 3. Save a recording

Repeat the check with `-o`:

```bash
vibium check "https://var.parts is up" -o sitecheck.zip
```

You'll get the verdict and evidence again, followed by:

```text
Recording saved to sitecheck.zip
```

Open [Record Player](https://player.vibium.dev) and drop the ZIP onto it.
Expand the **Check** group to inspect the result and browser actions, and step
through the screenshots. Use a new output filename for each run.

This earlier Chrome capture used **PinThing**, a different site. Your recording
will show var.parts:

![PinThing rendered during a Chrome check](../images/first-check-pinthing.jpg)

Keep `sitecheck.zip` for Part 3.

## Optional: include video with Firefox

Once the browser from this exercise is closed, use Firefox beta for this
development build's video example:

```bash
vibium install --engine firefox --channel beta
vibium check "https://var.parts is up" -o sitecheck-firefox.zip --engine firefox --channel beta
```

If you used `--keep-open`, close that browser with `vibium stop` before switching
engines. An unrelated existing session should be preserved; finish this exercise
in a separate session if needed.

Firefox 154 or newer includes WebM video. Chrome currently records actions and
screenshots without continuous video. Record Player can play the video from
your ZIP.

<details>
<summary>See an earlier Firefox run on PinThing</summary>

In this earlier PinThing run, the site responded but the demo stayed blank because
Firefox couldn't create a WebGL context. It is a different run from the
rendered Chrome screenshot above.

![Video preview of a Firefox check](../images/first-check-pinthing.gif)

[Watch the original WebM video](../images/first-check-pinthing.webm).

</details>

Next, [use Check with your coding agent](check-with-a-coding-agent.md), or
[check the saved recording](check-from-a-recording.md).
