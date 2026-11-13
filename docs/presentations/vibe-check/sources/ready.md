# Readiness diagnostics

`vibium ready` checks your local browser installation and tests configured AI.
It reports setup problems and fixes before you start browser work.

```bash
vibium ready
```

Example with an installed browser and no AI settings:

```text
Installed: chrome (stable) — /path/to/chrome
[PASSED] browser.installation: chrome (stable) and its required executable files are installed.
[SKIPPED] browser.connection: Browser launch and BiDi connectivity are not tested by readiness.
[SKIPPED] ai: AI is not configured; optional for direct browser automation.
  Fix: For Run or Check, configure VIBIUM_AI_PROVIDER and VIBIUM_AI_MODEL, then run vibium ready ai.

READY: Browser installation checks passed; launch and BiDi connectivity were not tested.
```

No AI settings means plain `ready` skips AI. Partially configured or failing AI
produces a failure. Run and Check require AI even when browser-only readiness
passes. Readiness does not assess your application's behavior.

## Choose what to test

| Command | Requested checks |
| --- | --- |
| `vibium ready` | Selected browser installation and AI if configured |
| `vibium ready browser` | Selected/default browser installation; AI is ignored |
| `vibium ready browser chrome` | Chrome executable files |
| `vibium ready browser firefox --channel beta` | Firefox beta executable files |
| `vibium ready ai` | AI configuration and provider tool round-trip; no browser required |
| `vibium ready ai anthropic --model your-model` | Anthropic using that model and `ANTHROPIC_API_KEY` |

Run readiness during initial setup, after changing configuration, or when
troubleshooting. You do not need to run it before every browser command.

## Browser readiness

The browser check uses Vibium's existing path resolution to list installations
across Chrome and Firefox channels. It checks that the selected browser and
required driver files exist and are executable. **It never launches a browser
or driver.** Other listed installations are informational.

This keeps setup checks lightweight and avoids browser startup triggering OS
or GPU initialization. Browser launch and BiDi connectivity are reported as
`skipped`, rather than claimed to work based on file checks.

Selection uses `--engine` / `VIBIUM_ENGINE`, `--channel` /
`VIBIUM_ENGINE_CHANNEL`, and the existing version/path settings. A positional
browser overrides the environment's engine. When that changes the engine,
the channel resets to the new engine's default unless `--channel` is explicit.
Conflicting positional and `--engine` selections are errors.
Chrome uses Vibium's cached Chrome for Testing and matching driver;
`VIBIUM_ENGINE_PATH` supports a custom Firefox executable.

Readiness does not install browsers, start a daemon, use your active session,
or inspect a remote browser. A missing installation produces an explicit fix:

```bash
vibium install --engine firefox --channel beta
vibium ready browser firefox --channel beta
```

The second command checks the installed files. Passing does not prove that
the browser can start, connect over BiDi, or run your application. Use the same
engine/channel for subsequent live work.

## AI readiness

AI checks validate shared `VIBIUM_AI_*` settings, authentication, model access,
tool calling, and a structured response using the same provider adapters as
Run and Check. They make at most two model requests with a 60-second budget;
API charges may apply. Invalid settings skip the provider request.

Use [per-call model options](model-providers.md#override-settings-for-one-call)
with `ready` or `ready ai`: `--provider`, `--model`, `--base-url`, and
`--reasoning-effort`. `ready ai <provider>` is equivalent to selecting that
provider with `--provider`. Changing provider requires an explicit model and
clears inherited endpoint/effort defaults. A conflicting provider argument
and flag is an error. No per-call option changes later defaults.

Vibium does not load env files automatically. Source exported assignments in
the same shell that runs readiness. If `~/.config/vibium/ai.env` exists
and AI is missing or invalid, readiness explains how to load it without reading
the file. Credentials and raw provider error bodies are not displayed.

`ready ai` works with no browser installed and ignores browser settings.
Use it before [checking an archive](../tutorials/check-from-a-recording.md).
It does not test screenshot support or application behavior.

## JSON and exit status

```bash
vibium ready --json
vibium ready ai --json
```

Without `--json`, output is readable text. With it, stdout contains one JSON
object using the normal `{ok, result, error?}` CLI envelope:

- `result.ready`: whether all requested checks passed
- `result.scope`: `all`, `browser`, or `ai`
- `result.summary`: concise readiness explanation
- `result.checks`: `{name, status, message, fix?}` entries
- `result.notes`: limitations and setup guidance
- `result.browsers`: discovered browser selections when browser checks ran;
  entries include `engine`, `channel`, `installed`, `selected`, and resolved
  `path` / `driver` when available

Individual check statuses are `passed`, `failed`, or `skipped`. A missing
optional AI configuration is skipped only by plain `ready`.
Exit **0** means the requested checks passed; exit **1** means setup needs
attention. Thus `ready ai` fails when AI is unconfigured, while plain `ready`
can succeed when browser installation checks pass. Browser launch and
connectivity remain untested. Neither result is a Check verdict.
