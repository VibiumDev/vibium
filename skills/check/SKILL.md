---
name: check
description: Independently check application acceptance criteria in a live browser or saved recording with the Vibium CLI. Use for a formal verification step in the development loop, with PASS, FAIL, or INCONCLUSIVE and recorded evidence.
---

# Check with Vibium

Use `vibium check "<claim>"` to hand a specific acceptance claim to a fresh
verifier context. The coding agent builds and fixes the application; Vibium's
verifier investigates the running behavior with constrained browser tools.
Your own browser inspection does not substitute for this invocation.

## Setup

Use the project's configured Vibium binary. Otherwise try `vibium`,
`./clicker/bin/vibium`, then `./node_modules/.bin/vibium`. Confirm
`vibium check --help` works. Local Chrome is the tested baseline for live checks.
For initial live setup, run `vibium ready browser --json` with the selected
engine/channel. It inspects browser and driver files without launching them
or touching the current browser; launch and BiDi connectivity remain untested. Follow its install/fix guidance if needed. Saved-input checks do not
require a browser; skip browser readiness for them.

For OpenAI, export `VIBIUM_AI_PROVIDER=openai`,
`VIBIUM_AI_MODEL`, and `OPENAI_API_KEY`.
For Anthropic, select `anthropic` and export `ANTHROPIC_API_KEY`; for Google,
select `google` and export `GOOGLE_API_KEY`. Set a model available to that
provider. Leave `VIBIUM_AI_REASONING_EFFORT` unset for Anthropic/Google.
`local` uses the compatible adapter, defaults to `http://127.0.0.1:8080/v1`,
and requires no key. Never manage or install a model runtime as part of Check.

An OpenAI-compatible service uses `VIBIUM_AI_PROVIDER=openai-compatible`
and `VIBIUM_AI_BASE_URL`. Use the project's existing configuration;
Vibium does not load environment files automatically. If an environment file
is configured, source it in the same shell invocation as Check. Its shell
assignments must export the settings (`export NAME=value`) so the CLI receives
them. Never print or log credentials. Do not choose another provider or model to work around a
missing configuration without the user's direction.

After loading settings, run `vibium ready ai --json` during initial setup or
when provider configuration changes. Exit 0 and `result.ready: true` mean the
configuration and provider tool round-trip passed; exit 1 includes failed
checks and fixes. It makes up to two small model requests and does not launch
or change a browser. It does not check the application or replace Check.
Resolve reported setup problems before the browser workflow; do not rerun it
before every claim when the configuration is unchanged.

Run accomplishes a browser goal; its `completed` result does not replace a
Check verdict. Check never inherits Run's conversation. For guidance on Run
and general browser automation, use the `browser` skill if installed.

For a user-requested provider comparison or eval, pass `--provider`, `--model`,
`--base-url`, and `--reasoning-effort` per invocation. Pass the same overrides
to `vibium ready ai`. Changing provider clears the inherited model, endpoint, and
effort, so supply a model and any custom endpoint explicitly. Same-provider
calls keep unspecified defaults; use `--base-url ""` or `--reasoning-effort ""`
to reset them. Credentials stay in the runtime environment. Overrides do not
change later calls or weaken Check's independence.

## Development loop

1. Turn the requested behavior into a concrete, observable claim. Keep the
   acceptance condition faithful to the request. For example: “Changing the
   timezone persists after refresh.” State any boundaries needed for the
   task, such as inspecting a cart without proceeding to checkout. If several
   independent behaviors matter, check them separately.
2. Run the application and exercise the relevant flow through the existing
   Vibium CLI. Use `go`, `map`, `find`, `click`, and `fill`; inspect fresh refs
   after page changes. Reuse the current browser, page, and `--session` or
   `VIBIUM_SESSION` throughout setup and verification. Create a named session
   only when starting a new isolated test, not when verifying existing state.
3. Record the flow. If the user already has a recording active, keep it active.
   Otherwise start one with `vibium record start` (the default filename is
   unique) and stop/save the recording you started after verification, even
   when the check fails. Do not stop someone else's browser or recording.
4. Run `vibium check --json "<claim>"` in that same session. The command
   starts a fresh verifier conversation; do not send the builder transcript,
   source code, or a suggested verdict. It may operate the page to test the
   claim. Let it finish before continuing browser work.
5. Read the JSON result and report the actual verdict, concise evidence, and
   recording path. The Check group contains verifier-driven child actions.
   A passed check applies to the stated claim and observed session.
6. When a check fails and fixing the application is within the user's task,
   investigate the evidence, fix the cause, and rerun the original claim.
   Preserve the earlier result. Do not weaken the claim or retry unchanged
   behavior until a model happens to pass it. For INCONCLUSIVE, address the
   missing evidence or report the limitation. Stop when the requested claims
   pass or a concrete blocker requires user input.

## Browser cleanup

Check preserves an existing browser session. If it starts a browser itself,
it closes that browser after saving any requested recording, on all verdicts
and execution errors. Use `--keep-open` when a standalone check should leave
the browser available for inspection or more work. Do not assume it stays
open just because `--output` saved a recording. `--keep-open` is live-only and
cannot be combined with `--input`.

## Recording and report flags

Use `-o verification.zip` to record the verification run automatically when no
recording is active. It includes video when supported (Firefox 154+), finalized
before browser cleanup. With an active recording, it exports the whole current
chunk through Check without stopping or resetting it; the export excludes
continuous video. Choose a new path different from the active recording's
output. Use `--report verdict.json` for a separate JSON verdict; `--json` still
controls stdout. Files are not overwritten, and available recording evidence
is saved even if the verifier encounters an operational error.

When asked to verify saved evidence, use
`vibium check -i record.zip "<claim>"` instead of the live browser steps above.
`--input` accepts Vibium recordings and compatible Playwright version 8 traces.
Inspect the actual evidence behind earlier verdicts; do not treat an embedded
PASS as proof. Report that the result concerns the recorded run. `--input`
cannot be combined with `--output`; use `--report` to save this verdict.
`--record` and `--trace` are not input flags.

## Result contract

Successful execution returns:

```json
{"ok":true,"result":{"status":"passed","claim":"…","summary":"…","evidence":[{"type":"observation","summary":"…"}]}}
```

`result.status` is `passed`, `failed`, or `inconclusive`, displayed as PASS,
FAIL, or INCONCLUSIVE. **All three verdicts have CLI exit code zero.** A script
that gates completion must inspect `result.status`; exit zero alone is not a
pass. Provider, configuration, and browser execution failures are errors,
not FAIL verdicts. Report them separately and retain available evidence.

For browser automation, exploration, and recording, use the `browser` skill if installed.
It supplies a broader CLI reference; this skill can run on its own using
`vibium <command> --help` when needed.
