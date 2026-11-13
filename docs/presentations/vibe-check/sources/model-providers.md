# Model providers for Run and Check

Run and Check share one AI configuration. Each invocation still starts a fresh
conversation; the operation selects its instructions and tool permissions.

| Setting | Environment variable |
|---------|----------------------|
| Provider | `VIBIUM_AI_PROVIDER` |
| Model | `VIBIUM_AI_MODEL` |
| Optional API base URL | `VIBIUM_AI_BASE_URL` |
| Optional OpenAI reasoning effort | `VIBIUM_AI_REASONING_EFFORT` |

Both operations support `openai`, `anthropic`, `google`, `openai-compatible`,
and `local`. Provider credentials use their native environment variables:

| Provider | Credential | Default API base URL |
|----------|------------|----------------------|
| `openai` | `OPENAI_API_KEY`, required | `https://api.openai.com/v1` |
| `anthropic` | `ANTHROPIC_API_KEY`, required | `https://api.anthropic.com/v1` |
| `google` | `GOOGLE_API_KEY`, required | `https://generativelanguage.googleapis.com/v1beta` |
| `openai-compatible` | `OPENAI_API_KEY`, optional | Explicit base URL required |
| `local` | `OPENAI_API_KEY`, optional | `http://127.0.0.1:8080/v1` |

`GOOGLE_API_KEY` is the canonical Google variable; `GEMINI_API_KEY` is not read.
If an optional key is configured, it is sent to the configured endpoint.
Base URLs must be HTTP(S), without embedded credentials, query, or fragment.
Native base URL overrides use the provider's API version prefix, not the full
messages/generateContent endpoint.

## Override settings for one call

Run, live Check, saved-input Check, and AI readiness accept the same overrides:

| CLI flag | JavaScript / MCP / Java option | Python keyword |
|----------|-------------------------------|----------------|
| `--provider` | `provider` | `provider` |
| `--model` | `model` | `model` |
| `--base-url` | `baseURL` | `base_url` |
| `--reasoning-effort` | `reasoningEffort` | `reasoning_effort` |

Explicit options override the shared AI environment for this invocation only.
Later calls still use their original defaults. Credentials always come from the
selected provider's environment variable; there is no API-key option.

When the provider changes, supply the model explicitly. Vibium clears the old
provider's model, endpoint, and reasoning effort before applying your options.
The new provider's default endpoint is used, or supply `baseURL` for a custom
endpoint. `openai-compatible` always requires an explicit endpoint after a
provider change. If the provider stays the same, omitted options retain the
shared AI defaults.

An empty base URL resets the provider default. An empty reasoning effort clears
an inherited setting. Empty provider/model values are errors. In Python, `None`
omits an option; use `""` for either reset.

For example, after exporting the appropriate API keys:

```bash
vibium ready ai --provider anthropic --model your-claude-model
vibium run "change my timezone to America/Chicago and save it" \
  --provider anthropic --model your-claude-model -o run.zip
# Uses Claude for this goal and saves the run, without changing Run defaults.

vibium check "the saved timezone is America/Chicago after refresh" \
  --provider openai --model gpt-5.6-sol --reasoning-effort none
# Uses OpenAI for a fresh assessment.

vibium check "the order confirmation was shown" -i record.zip \
  --provider local --model your-loaded-model \
  --base-url http://127.0.0.1:8080/v1 --reasoning-effort ""
# Checks saved evidence using the specified local server and its default effort.
```

For evals, specify all four settings explicitly so same-provider environment
defaults cannot affect a run. AI readiness accepts these flags too; use the same
overrides when checking a particular eval configuration.

JavaScript/TypeScript (Browser, Page, and both async/sync APIs):

```js
await page.run('change my timezone to America/Chicago', {
  provider: 'anthropic', model: 'your-claude-model',
});
await page.check('the saved timezone is America/Chicago', {
  provider: 'openai', model: 'gpt-5.6-sol', reasoningEffort: 'none',
});
```

Python (use `await` for the async API):

```python
result = page.check("the saved timezone is America/Chicago", provider="openai",
                     model="gpt-5.6-sol", reasoning_effort="none")
```

Java:

```java
page.run("change my timezone to America/Chicago",
    RunOptions.builder().provider("anthropic").model("your-claude-model").build());
page.check("the saved timezone is America/Chicago",
    CheckOptions.builder().provider("openai").model("gpt-5.6-sol")
        .reasoningEffort("none").build());
```

MCP uses flat arguments on `vibium_run` and `vibium_check`:

```json
{"name":"vibium_check","arguments":{"claim":"the saved timezone is America/Chicago","provider":"openai","model":"gpt-5.6-sol","reasoningEffort":"none"}}
```

Archive SDK calls accept overrides alongside `record`, including module-level
`browser.check(...)` and Java `Vibium.check(...)`.

Recorded Run/Check parent spans contain `params.modelConfig`: resolved
provider, model, reasoning effort, output-token cap, action limit, and timeout.
An empty effort means the provider's default. These are requested settings,
not a guarantee of a provider's model revision or deterministic output.
API keys and endpoint URLs are omitted. Metadata uses the existing recording
redaction machinery. Saved-input checks leave their source archive unchanged;
they do not add a new recording span to it.

## Examples

Export the appropriate credential privately before these examples. Replace
placeholder model names with models available to your account or local server.

OpenAI Run:

```bash
export VIBIUM_AI_PROVIDER=openai
export VIBIUM_AI_MODEL=gpt-5.6-sol
export VIBIUM_AI_REASONING_EFFORT=none
vibium ready ai
```

Use different providers for individual calls without changing the shared defaults:

```bash
vibium run "change my timezone to America/Chicago and save it" \
  --provider anthropic --model 'your-claude-model'
vibium check "the saved timezone persists after reload" \
  --provider google --model 'your-gemini-model'
```

The corresponding keys are `ANTHROPIC_API_KEY` and `GOOGLE_API_KEY`.

Local Check, with a compatible server already running:

```bash
export VIBIUM_AI_PROVIDER=local
export VIBIUM_AI_MODEL='your-loaded-model'
unset VIBIUM_AI_BASE_URL VIBIUM_AI_REASONING_EFFORT
vibium ready ai
```

`local` selects the OpenAI-compatible adapter and supplies a default endpoint.
It does not install, download, launch, configure, or stop a model server or
model files. An API key is not required. Use `openai-compatible` and an explicit
base URL for other compatible servers:

```bash
export VIBIUM_AI_PROVIDER=openai-compatible
export VIBIUM_AI_BASE_URL=http://localhost:1234/v1
export VIBIUM_AI_MODEL='your-tool-capable-model'
vibium ready ai
```

## AI readiness and environment loading

`vibium ready ai` checks the shared AI configuration used by Run and Check.
It accepts `--json`, makes at most two synthetic model requests, and leaves
browsers alone. Use `vibium ready ai anthropic --model your-model` to select a
provider positionally, or use the existing `--provider` flag. Plain `vibium ready`
also tests the selected browser; see [readiness diagnostics](ready.md). Exit 0 and `result.ready: true` indicate that configuration and
the provider tool round-trip passed. It does not test screenshot capability or
application behavior. Unsupported tool protocols produce errors and setup guidance.

Vibium does not load environment files automatically. Use `export NAME=value`
assignments in a private file, then source it in the same shell invocation.
The CLI reads settings on each call; SDK and MCP runtimes read their own process
environment and need restarting after environment defaults change. Per-call
overrides take effect immediately and do not require a runtime restart.

## Protocol and reasoning behavior

OpenAI retains the existing Chat Completions function-tool protocol.
Compatible endpoints must accept function tools, `parallel_tool_calls: false`,
and `max_completion_tokens`. Image input is needed when an operation requests
a screenshot. See [OpenAI function calling](https://developers.openai.com/api/docs/guides/function-calling).

Anthropic uses native Messages `tool_use` and `tool_result` blocks. Google uses
native Gemini `generateContent` function calls and responses. Both adapters
translate requested PNG/JPEG screenshots into the provider's image format.
See [Anthropic tool calls](https://platform.claude.com/docs/en/agents-and-tools/tool-use/handle-tool-calls)
and [Gemini API schemas](https://ai.google.dev/api/generate-content).

Reasoning-effort settings are supported only for OpenAI and compatible
endpoints in this slice. Leave them unset for Anthropic and Google; extended
thinking is not requested from Anthropic. Free-form assistant text accompanying
tool calls and readable reasoning are discarded. Gemini's opaque function-call
signatures are retained only in memory during that invocation and returned to
Google as required by its protocol; they never enter recordings or results.
See [Gemini thought signatures](https://ai.google.dev/gemini-api/docs/generate-content/thought-signatures).

## Acceptance status

Native provider protocols, AI readiness, fresh-context separation, and browser-tool
execution have deterministic fixture coverage. Real OpenAI Run acceptance
covers both an existing browser and standalone browser cleanup.
Real Anthropic, Gemini, and llama.cpp/local-server acceptance remains pending
until the required credentials or server environment is supplied.

Run deterministic coverage with `make test-run`. Real-provider checks are
opt-in via `VIBIUM_AI_LIVE=1` and that provider's `VIBIUM_AI_*` settings:

```bash
VIBIUM_AI_LIVE=1 node --test --test-concurrency=1 tests/run/live.test.js
```
