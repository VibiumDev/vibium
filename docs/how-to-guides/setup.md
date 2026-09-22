# Set up Vibium

`vibium setup` is the happy path after install. It asks for AI settings and
writes `~/.config/vibium/ai.env`, can install agent skills, and sets up the
local browser, then runs [`vibium ready`](../reference/ready.md). Prompts come
first; the browser download runs after them, so the questions are done in the
first seconds.

```bash
npm install -g vibium
vibium setup
```

Agents and CI:

```bash
vibium setup --non-interactive
```

`--quick` fills only what is missing. Sections can run on their own:

```bash
vibium setup browser
vibium setup ai
vibium setup skills
```

## AI

Setup prompts for provider, model, and (when needed) an API key. The key is
not echoed, and Enter skips it; setup then ends by naming the file to put it
in. openai-compatible and local also prompt for a base URL.

It writes `~/.config/vibium/ai.env` at mode 0600. If that file already exists,
the previous copy is kept as `ai.env.bak`. The setup process uses those values
immediately so the following `ready` check in the same run sees them. Later
commands load empty AI variables from that file automatically; see
[model providers](../reference/model-providers.md).

`--non-interactive` never prompts and does not overwrite an existing
`ai.env`. With no prompts and no file, that section is skipped; re-run
`vibium setup` in a terminal to configure AI. `vibium config init` still writes
the commented template. Values are never printed.

## Skills

Setup installs the `browser` and `check` skills for each agent already
present: `~/.claude/skills` when `~/.claude` exists, and `~/.grok/skills`
(or `$GROK_HOME/skills`) when `~/.grok` exists or `GROK_HOME` is set. This
matches `vibium add-skill`. Non-interactive setup installs skills only when
at least one agent is present.

## Browser

If the selected engine is missing, setup runs the same install path as
`vibium install`. If the files are already present, it says so. It then checks
those files the same way `vibium ready browser` does. It does not launch
Chrome or Firefox.

Use `--engine` and `--channel` for the same selection as live commands.

## Readiness

Setup finishes by running the matching `vibium ready` checks (files only for
the browser; up to two model requests for AI when a key is present). A fully
ready run ends with a first command to try. Without an API key the credential
check reads as skipped, and the summary names the one step left: add the key
to `ai.env`, then run `vibium ready ai`. Use `--json` for one `{ok, result}`
envelope with section statuses and paths.
