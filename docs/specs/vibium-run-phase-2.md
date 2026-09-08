# Vibium Run + Provider Expansion — Follow-up Implementation Spec

**Status:** implemented in the development checkout; real Anthropic, Gemini, and local-runtime acceptance remains pending. See [Implementation review](#implementation-review-2026-09-07).

**Audience:** Codex / Vibium engineering

**Baseline:** [Check v0.9](vibium-check-codex-implementation-spec-v0.9.md), including its documented evidence and privacy limits.

This spec uses the current names: **Run** (originally Perform) and **Check**
(originally Verify). Those old unshipped API names are not aliases. The implementation order below
records the original sequence; the review identifies the work still pending.

Decisions made during implementation are part of this updated spec:

- shared `VIBIUM_AI_*` defaults, with per-call provider/model overrides
- `vibium "<multiword goal>"` and callable JS/Python session objects alias Run
- `browser` skill for browser work and Run; `check` skill for acceptance claims
- checkbox operations use Set/Unset rather than the high-level Check name

---

## 1. Implemented v0.9 baseline — preserve exactly

Assume the following behavior is already implemented. This is context, not work to redo.

### Check file/output flags

```text
-i, --input <path>   read a Vibium recording or compatible Playwright trace
-o, --output <path>  create/save the live verification recording
--report <path>      save the verification verdict/report separately
```

Rules:

- `--record` and `--trace` are not accepted CLI flags
- SDK/MCP inputs retain the field name `record`
- do not rename or reinterpret these existing options in this slice

### Browser lifecycle

- standalone CLI Check closes a browser that Check itself started, after evidence is saved
- this cleanup occurs on success, verification failure, and execution error
- an already-existing browser/session stays open
- `--keep-open` preserves a browser newly started by Check

Run follows the same ownership principle:

- never close a browser/session it did not start
- standalone Run uses the existing autostart path and cleans up the browser it starts unless `--keep-open` applies

Do not invent a second lifecycle policy.

### Setup diagnostics

`vibium ready ai` provides the provider diagnostic described here.
Plain `vibium ready` also checks the selected local browser executable files;
`vibium ready browser [engine]` runs only installation checks. Neither launches
a browser or driver; browser launch and BiDi connectivity remain untested.
See [readiness diagnostics](../reference/ready.md).

It:

- checks configuration
- checks provider tool support
- may use up to two synthetic model requests
- does not open a browser

Provider additions in this spec must integrate with `vibium ready ai`.

### Configuration loading

Vibium does not automatically load env files.

Documentation/examples may use:

```bash
export NAME=value
source ~/.config/vibium/ai.env
```

Do not add automatic dotenv/env-file loading in this slice.

### Video

Check-owned and Run-owned recordings automatically include native WebM on
Firefox 154+. Chrome records actions and screenshots without continuous video.

Exporting an already-active recording still excludes its continuous video and preserves the original recording.

Do not change these recording/video semantics in this slice.

### Privacy boundary

Existing recorder-wide redaction covers:

- known credentials
- recognized secret fields

Sensitive form fields suppress visual artifacts.

The implementation does not claim it can detect every unknown secret.

Run-generated actions and recordings must go through the same existing redaction/privacy machinery. Do not build a separate privacy layer for Run.

### Archive compatibility

Current trace input support implements Playwright trace format version 8.

Unsupported trace versions return a clear error.

Do not broaden trace compatibility as part of this slice.

### Agent skills

The skills have distinct roles:

```text
/browser   browser automation, exploration, debugging, recording, and Run
/check     independent assessment of an explicit acceptance claim
```

These are skill names, not CLI subcommands. Agent invocation syntax varies;
Codex uses `$browser` and `$check`. See the repository
[`browser`](../../skills/browser/SKILL.md) and
[`check`](../../skills/check/SKILL.md) instructions.

### Transport / semantic command

Preserve the existing transport/router architecture.

Check's semantic command remains:

```text
vibium:check.run
```

Run's semantic command is:

```text
vibium:run.run
```

These are **Vibium runtime semantic operations**, not browser-vendor WebDriver BiDi commands.

The intended layering is:

```text
CLI / MCP / JS / Python / Java
        ↓
existing Vibium client→runtime transport
        ↓
Clicker / Go runtime
        ↓
vibium:check.run
or
vibium:run.run
        ↓
operation-specific model/tool loop
        ↓
configured LLM provider API
        ↕
existing Vibium browser actions/tools
        ↓
ordinary WebDriver BiDi commands
        ↓
Chrome / Firefox / other supported browser
```

For Check specifically:

```text
client calls check(...)
        ↓
runtime dispatches vibium:check.run
        ↓
runtime creates a fresh verifier inference context
        ↓
runtime calls the configured model provider
        ↓
model requests Vibium browser tools
        ↓
runtime executes those tools against the existing browser session via BiDi
        ↓
observations return to the model
        ↓
PASS / FAIL / INCONCLUSIVE returns to the caller
```

For Run:

```text
client calls run(...)
        ↓
runtime dispatches vibium:run.run
        ↓
runtime creates the Run model/tool loop
        ↓
runtime calls the configured model provider
        ↓
model requests Vibium browser tools
        ↓
runtime executes those tools against the existing browser session via BiDi
        ↓
observations return to the model
        ↓
completed / not_completed returns to the caller
```

Do not:

- send `vibium:check.run` or `vibium:run.run` directly to Chrome/Firefox as if they were native browser commands
- require browser vendors to implement Vibium extension commands
- introduce a second client→runtime transport
- introduce a second browser automation layer

The browser should continue to see ordinary WebDriver BiDi operations generated by Vibium's existing browser-action implementation.

Check must continue to use:

- a fresh inference context
- constrained verifier tools
- provider-independent Check semantics

---

## 2. Goal

Add two focused capabilities on top of the existing v0.9 implementation:

1. `vibium run "<goal>"`
2. additional model providers:
   - Anthropic
   - Google Gemini
   - OpenAI-compatible
   - `local` as a convenience alias for OpenAI-compatible

Preserve all existing Check behavior.

---

## 3. Add `run()`

Add:

```bash
vibium run "<goal>"
```

Semantics:

> Use the current live Vibium browser session to accomplish this goal.

Examples:

```bash
vibium run "log in with the test account"
vibium run "change my timezone to America/Chicago"
vibium run "add the blue widget to the cart"
```

### Requirements

`run()` must:

- reuse the current live Vibium browser session
- preserve cookies, auth, storage, current page, and application state
- use existing Vibium browser actions/tools
- continue until the goal is completed, cannot reasonably be completed, or execution limits are reached
- return a concise structured result
- when recording is active, create a Run parent span/action group with generated child browser actions beneath it
- use the same existing runtime/router/browser machinery as Check
- not introduce a second browser automation engine
- not introduce a new client/runtime transport

Do not add persisted/offline Run input. Run operates on a live session.

### Result contract

Recommended shape:

```json
{
  "status": "completed",
  "goal": "change my timezone to America/Chicago",
  "summary": "The timezone was changed successfully.",
  "evidence": [
    {
      "type": "observation",
      "summary": "Profile page shows timezone America/Chicago."
    }
  ]
}
```

Statuses:

```text
completed
not_completed
```

Execution/configuration/provider failures remain errors.

If the agent cannot establish that the goal was completed, return `not_completed` rather than claiming success.

---

## 4. Keep Run and Check semantically distinct

They may share implementation infrastructure, but not behavior.

```text
run(goal)
    → accomplish a goal

check(claim)
    → independently judge whether a claim is true
```

Check must retain all v0.9 behavior, including:

- fresh verifier inference context
- no inherited builder conversation
- verifier-specific instructions
- constrained browser permissions/tools
- PASS / FAIL / INCONCLUSIVE
- existing recording behavior
- existing record/trace verification behavior

Do not weaken Check while extracting shared code.

---

## 5. Shared model/tool-loop refactor

Reuse the v0.9 Check implementation where practical.

The preferred shape is:

```text
provider
+
operation-specific instructions
+
input goal/claim
+
tool policy
+
result/termination contract
        ↓
shared model/tool loop
```

with separate policies/contracts for:

```text
Run
Check
```

Do not build a separate orchestration stack for Run.

---

## 6. Provider support

Support these provider names:

```text
openai
anthropic
google
openai-compatible
local
```

The provider layer should be usable by both Run and Check.

### OpenAI

Preserve the existing v0.9 OpenAI behavior.

### Anthropic

Add native Anthropic tool-use support.

Credential:

```bash
ANTHROPIC_API_KEY=...
```

### Google

Add native Gemini function-calling/tool support.

Canonical credential:

```bash
GOOGLE_API_KEY=...
```

Supporting `GEMINI_API_KEY` as a fallback alias is acceptable if convenient, but document one canonical variable.

The implementation reads `GOOGLE_API_KEY` only; the optional fallback was not added.

### OpenAI-compatible

Support explicitly configured OpenAI-compatible endpoints:

```bash
export VIBIUM_AI_PROVIDER=openai-compatible
export VIBIUM_AI_BASE_URL=http://host:port/v1
export VIBIUM_AI_MODEL='your-tool-capable-model'
```

These defaults apply to both Run and Check.

Only rely on the subset of the API required for Vibium's tool-calling loop.

If the endpoint does not support required tool behavior, return a clear capability/configuration error.

### `local`

`local` is user-facing convenience syntax for the OpenAI-compatible adapter.

It is not a separate protocol.

Example:

```bash
export VIBIUM_AI_PROVIDER=local
export VIBIUM_AI_MODEL='your-loaded-model'
export VIBIUM_AI_BASE_URL=http://127.0.0.1:8080/v1
```

If `VIBIUM_AI_BASE_URL` is omitted for `provider=local`, default to:

```text
http://127.0.0.1:8080/v1
```

`local` must not require an API key.

This is intended to support local runtimes exposing a sufficiently compatible OpenAI-style API, such as llama.cpp-style servers.

Vibium must not install, download, launch, stop, configure, or manage local model runtimes or model files in this slice.

---

## 7. Shared AI configuration and per-call overrides

Run, Check, and AI readiness read the same defaults:

```text
VIBIUM_AI_PROVIDER
VIBIUM_AI_MODEL
VIBIUM_AI_BASE_URL             optional; required for openai-compatible
VIBIUM_AI_REASONING_EFFORT     optional; OpenAI/compatible only
```

The original proposal for separate `VIBIUM_PERFORM_*` and `VIBIUM_VERIFIER_*`
settings was replaced before release. Those variables do not configure either
operation. Shared configuration does not mean shared conversation history.

Provider-native credentials remain:

```text
OPENAI_API_KEY
ANTHROPIC_API_KEY
GOOGLE_API_KEY
```

Use explicit options to select a different provider/model for one invocation:

```bash
vibium run "change my timezone to America/Chicago and save it" \
  --provider anthropic --model your-claude-model
vibium check "the saved timezone is America/Chicago after refresh" \
  --provider openai --model your-openai-model
```

The first call uses Anthropic for the goal; the second asks OpenAI to assess
the result. Both use the existing browser when one is already running.

CLI flags are `--provider`, `--model`, `--base-url`, and `--reasoning-effort`.
SDK/MCP options are `provider`, `model`, `baseURL`, and `reasoningEffort`;
Python uses `base_url` and `reasoning_effort`. Credentials have no per-call
argument and come from the selected provider's environment variable.

Overrides affect only that invocation. Changing provider requires an explicit
model and clears the inherited endpoint and reasoning effort. Unchanged
providers retain unspecified defaults. Empty base URL/effort strings reset
those settings; empty provider/model values are errors. These options apply to
live and archive Check, Run, and CLI `ready ai`. See
[model providers](../reference/model-providers.md#override-settings-for-one-call).

This allows:

```text
Run   → Claude
Check → OpenAI
```

or:

```text
Run   → local model
Check → Gemini
```

The two roles may also use the same provider/model.

---

## 8. Interface parity

`run()` is a first-class Vibium primitive and must be exposed through every existing public interface that exposes Check.

All interfaces must delegate to the same underlying runtime implementation and common Run result contract. Do not implement independent model/tool loops in MCP or language SDKs.

### CLI

Add:

```bash
vibium run "<goal>"
```

For Run:

- use existing Vibium CLI conventions
- `-o/--output` saves the live Run recording, consistent with Check
- `--keep-open` follows the existing browser-ownership semantics for a browser Run starts
- do not add `-i/--input` for Run
- Run has no `--report`; use `--json` for its structured result
- support the per-call provider/model options in section 7

Both `completed` and `not_completed` exit 0. With `--json`, inspect
`result.status` in the CLI `{ok, result}` envelope. Execution errors exit 1.

### Run shorthand

One quoted multiword CLI argument also invokes Run:

```bash
vibium "open https://example.com and find its contact page"
# Same operation as vibium run with this goal.
```

Known subcommands take precedence. Unknown single words and multiple positional
arguments are errors. Use explicit `vibium run "stop"` for a one-word goal;
shell quote style does not survive argument parsing.

Connected Browser and Page objects in JavaScript/TypeScript and Python are
callable: `vibium(goal)` or `vibe(goal)` delegates to that object's `run(goal)`.
The variable name is arbitrary; importing the package does not create a
session. Options, async/sync behavior, and browser identity are preserved.
Java retains the explicit `run()` method.

### Checkbox naming

High-level `check(claim)` is distinct from checkbox state. Elements use
`set()` / `set(false)` / `unset()` and `isSet()` in JavaScript/Java,
or `is_set()` in Python. CLI uses `vibium set`, `vibium unset`, and
`vibium is set`; MCP uses `browser_set`, `browser_unset`, and `browser_is_set`.
See [API reference](../reference/api.md) for signatures.

### MCP

Add:

```text
vibium_run
```

Minimum input:

```json
{
  "goal": "change my timezone to America/Chicago"
}
```

Return the common Run result shape.

MCP must call the same internal Run implementation as the CLI.

Do not add persisted/offline input-record behavior to MCP Run.

### JavaScript / TypeScript

Add:

```js
const result = await vibe.run(
  "change my timezone to America/Chicago"
)
```

Return the common Run result shape and delegate to the existing Vibium runtime.

Do not implement a separate SDK-side model/tool loop.

### Python

Add:

```python
result = vibe.run(
    "change my timezone to America/Chicago"
)
```

Return the common Run result shape and delegate to the existing Vibium runtime.

### Java

Add an idiomatic equivalent:

```java
RunResult result = vibe.run(
    "change my timezone to America/Chicago"
);
```

Use existing repository naming/builders/result conventions if they differ from this illustrative type name.

### Cross-interface contract

Across CLI, MCP, JavaScript/TypeScript, Python, and Java:

- the goal has the same meaning
- the same live browser/session semantics apply
- the same provider selection/configuration applies
- the same execution limits apply
- the same `completed | not_completed` result semantics apply
- the same recording/privacy behavior applies
- browser ownership and cleanup rules remain consistent
- all surfaces call the same runtime implementation
- no surface gets separate Run orchestration

---

## 9. Recording

When recording is active, Run should appear similarly to Check:

```text
Run: "change my timezone to America/Chicago"
    ├── map
    ├── fill
    ├── click
    └── map
COMPLETED
```

Reuse existing parent span/action-group machinery.

Do not create a new trace format.

---

## 10. Acceptance tests

Add focused tests proving:

1. CLI `vibium run "<goal>"` works with OpenAI
2. Run uses the existing live browser session
3. Run records a parent span with nested child actions
4. MCP `vibium_run` delegates to the same runtime/result contract
5. JavaScript/TypeScript `vibe.run(...)` delegates to the same runtime/result contract
6. Python `vibe.run(...)` delegates to the same runtime/result contract
7. Java `vibe.run(...)` delegates to the same runtime/result contract
8. Anthropic can execute at least one Vibium browser tool call
9. Google can execute at least one Vibium browser function/tool call
10. `openai-compatible` works against a configurable compatible endpoint
11. `local` normalizes to the OpenAI-compatible adapter
12. `local` does not require an API key
13. Run and Check can use different providers/models
14. `vibium ready ai` can validate Anthropic configuration/tool support
15. `vibium ready ai` can validate Google configuration/tool support
16. `vibium ready ai` can validate OpenAI-compatible/local configuration/tool support
17. all existing v0.9 Check tests continue to pass
18. Check still uses a fresh inference context after any shared-loop refactor
19. Run recordings pass through the existing redaction/privacy machinery
20. equivalent Run calls through different public surfaces produce equivalent runtime behavior and result semantics

---

## 11. Suggested implementation order

1. Inspect the existing v0.9 Check model/tool loop, provider code, and public interface bindings.
2. Extract only the minimum provider-neutral/shared loop needed for reuse.
3. Re-run existing Check tests before adding new behavior.
4. Add Run-specific request/result/policy types.
5. Add the internal `vibium:run.run` runtime/router path.
6. Add `run` CLI parsing and routing.
7. Bind Run to the current live browser session.
8. Add Run parent-span recording.
9. Add one end-to-end OpenAI Run test.
10. Expose Run through MCP using the same runtime implementation.
11. Expose Run through JavaScript/TypeScript.
12. Expose Run through Python.
13. Expose Run through Java.
14. Add focused surface-parity tests.
15. Add Anthropic provider and integrate it with `vibium ready ai`.
16. Add Google provider and integrate it with `vibium ready ai`.
17. Generalize OpenAI-compatible configuration if needed.
18. Add `local` normalization/defaults and AI readiness coverage.
19. Run provider-focused and AI readiness tests.
20. Confirm Run uses existing recorder redaction/privacy handling.
21. Re-run the complete existing Check acceptance suite.

---

## 12. Explicit non-goals

Do not implement in this slice:

- hosted Vibium provider
- Cortex/application memory
- PR/GitHub verification
- cloud browser fallback
- automatic local-model installation
- automatic llama.cpp/Ollama/LM Studio/vLLM management
- additional providers beyond OpenAI, Anthropic, Google, OpenAI-compatible/local
- new `--record` or `--trace` flags
- changing existing `-i` / `--input`, `-o` / `--output`, `--report`, or `--keep-open` semantics
- automatic env-file loading
- Playwright trace versions beyond the currently supported version 8
- changing existing video capture/export behavior
- redesigning recorder redaction/privacy behavior
- persisted/offline Run execution
- interface-specific Run semantics or separate SDK/MCP orchestration loops
- unrelated Check redesign

---

## 13. Design principle

> **Run accomplishes a browser goal. Check independently judges a claim. They may share provider/tool-loop infrastructure, but they remain distinct primitives.**

## Implementation review (2026-09-07)

The review found implementation paths for all required Phase 2 capabilities,
using the updated names and configuration decisions above. Real-provider
acceptance remains incomplete; fixture success is not a substitute for it.

| Acceptance requirements (section 10) | Implementation and coverage |
| --- | --- |
| 1–3: CLI Run, same browser, nested recording | [Run CLI](../../clicker/cmd/clicker/run.go), [shared browser ownership and recording](../../clicker/internal/agent/check.go), and [CLI tests](../../tests/run/cli.test.js); [opt-in real acceptance](../../tests/run/live.test.js) covers OpenAI existing and standalone browsers |
| 4–7, 20: MCP and language interfaces | [Runtime bindings](../../clicker/internal/api/model_operations.go) and [surface tests](../../tests/run/surfaces.test.js) cover MCP, JS async/sync, Python async/sync, Java, and a Firefox SDK case |
| 8–12: native Anthropic/Gemini and compatible/local tools | [Provider adapters](../../clicker/internal/verifier/providers.go), [configuration](../../clicker/internal/verifier/verifier.go), [protocol tests](../../clicker/internal/verifier/providers_test.go), and CLI fixture tests execute browser tool calls for all five provider names |
| 13: different provider/model per operation | [Per-call resolution](../../clicker/internal/verifier/config_overrides.go), [override tests](../../tests/run/overrides.test.js), and surface tests prove overrides work without changing defaults |
| 14–16: provider readiness | [Shared probe](../../clicker/internal/verifier/probe.go), [probe tests](../../clicker/internal/verifier/probe_test.go), native protocol tests, and CLI fixture tests exercise the synthetic tool round trip |
| 17–18: preserve Check and fresh context | [Operation-neutral loop](../../clicker/internal/verifier/loop.go), [Run contract](../../clicker/internal/run/run.go), provider/context unit tests, and `make test-check`; no SDK/MCP-side model orchestration |
| 19: common recording/privacy path | Run uses the same recorder as Check; CLI and SDK privacy cases verify sensitive input redaction and visual suppression. The v0.9 privacy limitation still applies. |
| Added naming and shorthand decisions | [Naming tests](../../tests/naming/cli.test.js), [callable-object tests](../../tests/naming/callable.test.js), and [API surface tracker](../trackers/arewewebdriveryet.md) |

### Pending acceptance and explicit limits

- **Anthropic:** native adapter, AI readiness, and browser-tool fixtures pass;
  real model acceptance awaits credentials and a model selection.
- **Google Gemini:** native function calling, image conversion, in-memory
  function-call signatures, AI readiness, and browser-tool fixtures pass; real
  model acceptance awaits credentials and a model selection.
- **llama.cpp/local:** alias normalization, default URL, key-free requests,
  AI readiness, and browser-tool fixtures pass; real acceptance awaits a running
  compatible model server. The adapter does not manage that server.
- **OpenAI-compatible:** configurable endpoint behavior is fixture-tested.
  Compatibility depends on the server implementing the required tool and
  image protocol; this is not a claim that every compatible server/model works.
- **Privacy and evidence:** the limitations in the
  [v0.9 review](vibium-check-codex-implementation-spec-v0.9.md#implementation-review-2026-09-07)
  remain. Run's completion assessment does not replace Check. Action-budget
  exhaustion returns `not_completed`; timeout/browser/provider failures are
  execution errors. Limits are currently fixed at three minutes and 24 model
  actions, shared across surfaces.

Real OpenAI Run and Check have previously been exercised, including the
captured [meetup example](../presentations/vibe-check/README.md#what-was-actually-captured).
This review reruns deterministic tests, not real-model acceptance. The normal
Run suite skips all five opt-in real cases, including OpenAI; that skip does
not erase the earlier OpenAI results or validate the three pending providers.

To finish a pending provider acceptance, configure its native credential (if
required), shared AI settings, and any local server, then run:

```bash
VIBIUM_AI_LIVE=1 node --test --test-concurrency=1 tests/run/live.test.js
```

Only the configured provider's real cases run. Success must include a browser
tool action, completion evidence, existing-session preservation, and a recorded
Run parent with child actions. Do not mark this pending work complete based
only on AI readiness or fixture tests.

No additional unimplemented in-scope primitive was found. Hosted Vibium,
Cortex, PR integration, runtime installation/management, offline Run, and
broader trace versions remain the explicit non-goals in section 12.

### Checks run for this review

- `go test ./...` in `clicker`: passed.
- Fresh `go test -count=1` for verifier, Run, agent, API/recorder, and CLI packages: passed.
- `make test-run`: 30 passed; five real-provider cases skipped as opt-in.
- `make test-check`: 31 main acceptance tests and 15 Firefox tests passed.
- Documentation validation: 50 local links/anchors and 41 shell/JSON/Python/JavaScript snippets passed; executable examples contain no obsolete feature names.

These checks cover shared runtime behavior and the sampled browser/interface
combinations; they are not an exhaustive provider × browser × SDK certification.
