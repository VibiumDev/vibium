# Use Vibium with Claude Code

Give Claude Code browser tools through Vibium's CLI skills. This guide covers
local CLI sessions and local sessions in Claude Desktop's Code tab. For the
Chat tab, use [Claude Desktop setup](using-vibium-with-claude-desktop.md).

## 1. Install Vibium and the skills

Run these commands in your project directory:

```bash
npm install -g vibium
npx skills add https://github.com/VibiumDev/vibium --agent claude-code
vibium ready browser
```

Select all Vibium skills and the project installation scope. Browser readiness
checks installed files without opening a browser. Follow its installation
guidance if anything is missing.

For unpublished skills and a development binary, see
[Install skills from your checkout](../../CONTRIBUTING.md#install-skills-from-your-checkout).

Start Claude Code in this project, or open it in the Desktop app's Code tab.
Type `/` to find `browser` and `check`. These are
[Claude Code skills](https://code.claude.com/docs/en/skills), each invoked as a
slash command. The [Code tab supports these skills too](https://code.claude.com/docs/en/desktop#use-skills).

## 2. Ask Claude to use the browser

Send this as a Claude Code prompt:

```text
/browser Open https://example.com with Vibium, take a screenshot, and
summarize the page. Close the browser session you started afterward.
```

Claude reads the skill and runs Vibium commands. Direct browser commands need
no separate Vibium AI configuration. The browser skill also covers
`vibium run`, which delegates a browser task to a configured model.

If `vibium` isn't found, give Claude the absolute binary path. If the skills
aren't listed, confirm they were installed for this project and start a new
Claude Code session.

## 3. Add an independent check

Run and Check currently require the development build and
[Vibium AI configuration](../tutorials/first-check.md#1-configure-ai-access).
Your Claude login does not configure Vibium's model provider; you can use any
[supported provider](../reference/model-providers.md).

After creating `~/.config/vibium/ai.env` with exported settings, send:

```text
/check Use Vibium to check whether https://var.parts is up and save
its evidence to sitecheck.zip. Source ~/.config/vibium/ai.env in the same
shell invocation before calling Vibium. Run vibium ready ai first and
address setup errors. Do not display the settings file or keys.
Report the verdict and recording path.
```

This example uses Bash or Zsh. For a full development workflow, follow
[Check with your coding agent](../tutorials/check-with-a-coding-agent.md).
