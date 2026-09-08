# Use Vibium with Codex

Give Codex browser tools through the Vibium CLI and its `browser` and `check`
skills. Use a local Codex session with shell access to your project. The terminal
examples below use Bash or Zsh.

## 1. Install Vibium and the skills

In your project directory, run:

```bash
npm install -g vibium
npx skills add https://github.com/VibiumDev/vibium --agent codex
vibium ready browser
```

Select all Vibium skills and the project installation scope. The browser
readiness command checks installed files without opening a browser. Follow
its installation guidance if anything is missing.

For unpublished skills and a development binary, see
[Install skills from your checkout](../../CONTRIBUTING.md#install-skills-from-your-checkout).

Open the same project in Codex. In Codex CLI or the IDE extension, type `$` to
select an installed skill, or use `/skills`. If the skills don't appear,
restart Codex. See [OpenAI's skill instructions](https://learn.chatgpt.com/docs/build-skills#how-chatgpt-and-codex-use-skills).

## 2. Ask Codex to use the browser

Send this as a Codex prompt, not a terminal command:

```text
$browser Open https://example.com with Vibium, take a screenshot, and
summarize the page. Close the browser session you started afterward.
```

Codex reads the skill and runs Vibium commands. This direct browser work needs
no separate Vibium AI configuration.

If Codex cannot find `vibium`, give it the absolute path to your installed
binary, or to `clicker/bin/vibium` when using a development checkout.

## 3. Add an independent check

The `check` skill calls `vibium check` for a fresh assessment of a claim.
Check and `vibium run` (which carries out browser tasks from instructions)
currently require the development build and Vibium's own AI settings.
Signing into Codex does not configure those settings.

Follow [Configure AI access](../tutorials/first-check.md#1-configure-ai-access)
to create `~/.config/vibium/ai.env` with exported settings. Then ask Codex:

```text
$check Use Vibium to check whether https://var.parts is up and save
its evidence to sitecheck.zip. Before invoking Vibium, source
~/.config/vibium/ai.env in the same shell invocation. Run vibium ready ai
first and address any setup errors. Do not display the settings file or keys.
Report the verdict and the recording path.
```

Expect PASS, FAIL, or INCONCLUSIVE with concise evidence. For a complete coding
workflow, follow [Check with your coding agent](../tutorials/check-with-a-coding-agent.md).
