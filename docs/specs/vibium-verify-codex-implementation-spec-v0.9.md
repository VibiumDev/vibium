# Vibium Verify — Codex Implementation Spec v0.9

**Status:** local CLI, archive, MCP, and SDK implementation; acceptance coverage in `tests/verify` and `tests/daemon/verify*`
**Audience:** Codex / Vibium engineering
**Scope:** implement the smallest useful local `verify()` loop in the existing Vibium codebase.

---

## 1. Goal

Add one new high-level operation:

```bash
vibium verify "<claim>"
```

The operation gives the builder agent an **independent second opinion** from a fresh verifier inference context with a verifier-specific role and constrained permissions.

Using a different model, provider, or account can strengthen that independence, but none of those are required for the first implementation.

Valid examples:

```text
Codex/OpenAI = builder
fresh OpenAI verifier context = verifier
```

```text
Claude Code = builder
OpenAI = verifier
```

The first example is sufficient for v1 as long as the verifier invocation starts with a fresh context and does not inherit the builder conversation.

For the first implementation slice, the verifier must use the **same live Vibium browser session** the builder is already using.

The design should also support a second observation source shortly afterward: an existing `record.zip`. In that mode Vibium verifies a claim against recorded evidence rather than a live browser.

It may inspect and operate the browser as needed, then return:

```text
PASS | FAIL | INCONCLUSIVE
+ concise explanation
+ recorded evidence
```

The verification and all browser actions it causes must be captured in the existing Playwright-compatible `record.zip`.

---

## 2. First milestone only

Implement this first:

```text
existing local Chrome session
        +
explicit verification claim
        +
one BYO verifier adapter
        ↓
independent verifier tool loop
        ↓
PASS / FAIL / INCONCLUSIVE
        +
record.zip evidence
```

For the first proof, it is enough for:

- Claude Code or Codex to be the builder
- the Vibium CLI to already control a local Chrome session
- an OpenAI/OpenAI-compatible model to be the verifier, including the same provider/model family used by the builder if desired
- `vibium verify "<claim>"` to reuse that exact browser
- the verifier to have only browser-oriented Vibium tools
- the result to return to the original builder
- the verification to be visible in the Vibium recording

### First acceptance slice vs. immediate follow-on work

The **first acceptance slice** is intentionally:

```text
local Chrome + CLI + one BYO verifier
```

That is only to prove the end-to-end Verify loop with the smallest amount of new plumbing.

Immediately after that slice works, add parity through the same underlying implementation:

```text
Firefox
MCP
JavaScript / TypeScript
Python
Java
```

These are part of the product surface, not optional long-term ideas.

Do **not** block the first acceptance slice on:

- `vibium.com`
- hosted Vibium accounts
- application memory / Cortex
- git diff / PR context
- automatic GitHub verification
- cloud browsers
- `perform()`
- multiple production-grade verifier adapters
- payments/email/database capabilities
- a new networking transport

---

## 3. Existing architecture constraint

Do not introduce a new client-to-runtime transport for Verify.

Use Vibium's **existing command/router architecture**.

Conceptually:

```text
Claude Code / Codex
        │
        │ shell
        ▼
vibium verify "..."
        │
        ▼
existing Vibium CLI / Go binary command router
        │
        ├───────────────→ existing live Chrome session
        │                    via existing BiDi machinery
        │
        └───────────────→ verifier adapter
                              │
                              ▼
                         second model/account
```

Do **not** add a second BiDi proxy layer merely to support Verify.

Represent Verify internally with the semantic method name:

```text
vibium:verify.run
```

where useful in the router, recording, protocol metadata, or future remote compatibility.

Verify is a high-level Vibium semantic command built **on top of** the existing WebDriver BiDi browser machinery.

---

## 4. CLI

Add:

```bash
vibium verify "<claim>"
```

Example:

```bash
vibium go http://localhost:3000/account
vibium verify "changing my display name persists after refresh"
```

For the first milestone:

- an explicit claim is required
- use the currently active/default Vibium browser session
- if there is no usable browser session, use existing daemon/browser autostart behavior if natural; otherwise return a clear error
- do not implement cloud fallback
- optional `--json` should follow existing CLI conventions if easy

### Input, recording output, and reports

```text
-i, --input <archive.zip>     # existing Vibium recording or Playwright trace
-o, --output <recording.zip>  # new recording of a live verification run
--report <verdict.json>       # separate structured verification result
--json                       # structured stdout, using the CLI envelope
```

`--record` and `--trace` are not input flags. Both sound like requests to
start recording, so the unreleased CLI uses `--input` for either archive format.

```bash
vibium verify "changing my display name persists after refresh"
vibium verify "changing my display name persists after refresh" -o verification.zip
vibium verify -i record.zip "checkout completed successfully"
vibium verify --input trace.zip --report verification.json "checkout completed successfully"
```

The first command verifies the live browser. The second saves its verification
run in a new ZIP. The last two inspect immutable recorded evidence without
launching a browser. All return a verdict with concise evidence.

Output contract:

- default: concise human-readable stdout; no new artifact
- `--json`: structured JSON to stdout
- `--report`: write the verification result to a separate JSON file
- `--output`: save a new recording ZIP of live verification, including the
  Verify parent span, child actions, observations, and verdict
- never overwrite an existing output/report file, including a symlink or hardlink
- reject `--input` together with `--output` for now; a derived archive of an
  inspection is a separate future feature, not a replay of browser actions

For live `--output`, start and finish a recording if none is active. Capture
screenshots and browser actions; continuous video is not required. If recording
is already active, export its current chunk through the end of Verify using
the existing recorder. Preserve that recording's groups, state, video track,
and declared destination; do not stop, reset, or redirect it. This export
includes earlier actions in that chunk and excludes the continuous video file.
The output path must differ from the active recording's destination. Save
available evidence even on an operational verifier error, but do not write a
JSON verdict report for an operation that failed to complete.

### Recorded/trace-session verification

Both Vibium `record.zip` and compatible Playwright `trace.zip` use `--input`:

- do not require or launch a live browser
- treat the archive as read-only evidence
- detect the trace/archive shape and version where practical
- inspect available actions, screenshots, DOM/accessibility snapshots, network
  events, console events, timestamps, and other trace data
- do not silently perform additional browser actions
- return INCONCLUSIVE when the recording cannot establish or refute the claim
- preserve the source archive unchanged; write no new archive

Future versions may combine a live session with prior recordings, but that is
not required now.

---

## 5. Verifier configuration

Implement a small internal verifier abstraction.

Conceptually:

```go
type Verifier interface {
    Verify(ctx context.Context, req VerificationRequest, tools ToolExecutor) (VerificationResult, error)
}
```

Exact Go shape should fit existing repository conventions.

Initial configuration:

```bash
VIBIUM_VERIFIER_PROVIDER=openai
VIBIUM_VERIFIER_MODEL=<model>
OPENAI_API_KEY=...
```

Optionally support an OpenAI-compatible base URL:

```bash
VIBIUM_VERIFIER_PROVIDER=openai-compatible
VIBIUM_VERIFIER_BASE_URL=http://localhost:1234/v1
VIBIUM_VERIFIER_MODEL=<model>
```

The model must be configurable rather than hard-coded.

For the first implementation, one working verifier adapter is enough.

Do **not** require separate-account authentication or a different model/provider for v1. The same OpenAI credentials may be used by the builder environment and the verifier adapter, provided the verifier call is a fresh inference context and receives only the verifier-specific context/tools defined in this spec.

Future adapters may include Anthropic, Gemini, local runtimes, and `vibium`.

---

## 6. Independence is part of the feature

The minimum required independence boundary is:

```text
fresh verifier inference context
+
verifier-specific system instruction
+
constrained permissions/tools
+
no inherited builder conversation
```

The verifier must use a **fresh inference context**.

Do not simply ask the builder agent the verification question again in the builder's existing conversation.

The verifier should receive:

- the verification claim
- a concise verifier system instruction
- current browser/session observations
- constrained browser tools

It should **not** automatically receive:

- the builder's full conversation
- source-code write access
- arbitrary shell access
- git mutation access
- package install access
- deployment mutation access

The first implementation may use the same provider and model family as the builder. It only needs to be a distinct verifier invocation with a fresh context and verifier-specific permissions.

For example, this is valid for v1:

```text
Codex builds
→ vibium verify
→ fresh OpenAI verifier context
```

Stronger optional independence boundaries include:

```text
same provider, different model
different provider/model
different account or credentials
```

For example:

```text
Claude Code builds
OpenAI verifies
```

or vice versa.

These stronger boundaries should remain possible, but they are not prerequisites for the first slice.

---

## 7. Verifier objective

Use a verifier-specific system instruction along these lines:

> You are an independent software verifier. Your job is to determine whether the supplied claim about the running application is true. Do not modify source code. Use only the provided browser tools. Gather enough observable evidence to return PASS, FAIL, or INCONCLUSIVE. Do not assume the builder's claim is correct. If the available evidence is insufficient, return INCONCLUSIVE rather than guessing.

Do not expose or persist model chain-of-thought.

Persist only:

- actions
- observations
- concise evidence summaries
- final verdict
- final explanation

---

## 8. Browser/session behavior

If the user has already done:

```bash
vibium go http://localhost:3000
vibium click ...
vibium fill ...
```

then:

```bash
vibium verify "..."
```

must continue in **that exact browser session**.

Preserve:

- current page
- cookies
- authentication
- localStorage/sessionStorage
- browser history/state
- current DOM
- existing application state

Do not silently launch a fresh browser or restart the workflow.

---

## 9. Browser tool surface

Expose a constrained tool set to the verifier by adapting existing Vibium command implementations.

Do not create a second browser automation engine.

Use existing Vibium commands/actions wherever possible.

Minimum useful tools should include existing equivalents for:

```text
map / inspect
go / navigate
click
fill / type
press / keys
scroll
find
screenshot
a11y tree / structured page inspection
console messages / page errors
network observations where already available
```

Consider whether `eval` should be exposed in v0.1; if it is, keep it intentionally constrained.

Return concise structured tool results to the model. Avoid dumping huge raw DOM/network payloads when summarized or filtered results are enough.

---

## 10. Agent loop

Conceptually:

```text
claim
  ↓
fresh verifier context
  ↓
initial browser observation
  ↓
verifier chooses browser tool
  ↓
execute through existing Vibium router/runtime
  ↓
return observation
  ↓
verifier chooses next step
  ↓
...
  ↓
PASS / FAIL / INCONCLUSIVE
```

Add sensible limits:

- overall timeout
- max tool turns / actions
- max payload sizes

Use existing Vibium timeout/config conventions where possible.

Do not optimize heavily before the loop works.

---

## 11. Result model

Use a shared internal result shape approximately like:

```json
{
  "status": "passed",
  "claim": "changing my display name persists after refresh",
  "summary": "The display name remained changed after the page was refreshed.",
  "evidence": [
    {
      "type": "observation",
      "summary": "Display name before edit: Jason"
    },
    {
      "type": "observation",
      "summary": "Display name after refresh: Jason Test"
    }
  ]
}
```

Statuses:

```text
passed
failed
inconclusive
```

Execution/configuration/provider failures are errors, not verification verdicts.

A verification that cannot safely establish the claim should be `inconclusive`, not `passed`.

This is especially important for record-based verification. For example, if `record.zip` shows that the user clicked "Place order" and the network request returned 200, but the recording ends before any confirmation state appears, Vibium should be willing to return:

```text
INCONCLUSIVE

The recording shows that the order request succeeded,
but it does not contain enough evidence to confirm that checkout completed.
```

---

## 12. CLI output

Example:

```text
$ vibium verify "changing my display name persists after refresh"

VERIFY: changing my display name persists after refresh

PASS

The display name remained changed after the page was refreshed.
```

Failure:

```text
FAIL

The display name appeared to save, but reverted after refresh.

Observed:
- save action completed
- refreshed page showed previous value
```

Keep output concise.

---

## 13. Recorded / Playwright-trace verification

This is **not required for the first live-browser PR**, but it should be the next small implementation after the live loop works.

The observation source is a Playwright-compatible trace archive.

Supported intended inputs:

```text
Vibium record.zip
Playwright trace.zip
```

A Vibium `record.zip` may contain additional Vibium metadata and embedded verification spans. A vanilla Playwright `trace.zip` may not, but can still provide actions, snapshots, screenshots, network data, console/page errors, URLs, and timing evidence.

### Input

```bash
vibium verify --input record.zip "checkout completed successfully"

vibium verify --input trace.zip "checkout completed successfully"
```

### Behavior

1. open the supplied Playwright-compatible trace archive
2. detect whether it is a Vibium record or vanilla/compatible Playwright trace
3. index/read the trace data needed for verification
4. construct a read-only observation source for the verifier
5. expose trace-inspection tools rather than live browser-action tools
6. let the verifier gather enough evidence from the archive
7. return PASS, FAIL, or INCONCLUSIVE

### Trace-inspection tool surface

Prefer tools such as:

```text
list actions
inspect action
inspect page snapshot
inspect screenshot
inspect network events
inspect console/page errors
inspect URL/navigation history
search trace
```

Do not expose live mutation tools such as `click`, `fill`, or `navigate` when the observation source is a recording.

### Privacy / payload discipline

Do not automatically send the entire raw `record.zip` to a model provider.

Extract or summarize only the evidence needed for the verifier's current question/tool call. Large screenshots or trace payloads should be provided only when requested and supported.

The same verifier permission principle applies: the verifier receives evidence, not arbitrary local filesystem access.

### Compatibility model

Use one internal observation-source abstraction for both formats:

```text
TraceObservationSource
├── Playwright trace.zip
└── Vibium record.zip
      └── optional Vibium metadata / embedded verifications
```

The verifier should not need separate reasoning logic for Playwright traces vs. Vibium records.

Playwright trace compatibility should be version-aware where practical. If an unsupported trace version or malformed archive is encountered, return a clear execution/configuration error rather than a verification verdict.

### Embedded verifications in records

A Vibium `record.zip` may already contain verification spans if those verifications happened while the recording was being created. A vanilla Playwright `trace.zip` normally will not contain those Vibium-specific spans.

Example:

```text
record.zip
├── navigate
├── click
├── verify: "checkout works"
│   ├── inspect
│   ├── click
│   ├── network observation
│   └── verdict: PASS
├── more browser actions
└── verify: "order appears in history"
    ├── inspect
    └── verdict: FAIL
```

Those embedded verification spans are part of the available evidence when later verifying the record.

For example:

```bash
vibium verify --input record.zip   "all critical checkout verifications passed"
```

or:

```bash
vibium verify --input record.zip   "the evidence supports the earlier checkout PASS"
```

The verifier may inspect:

- previous verification claims
- previous verdicts
- actions nested under those verifications
- observations/evidence associated with them

### Output

The existing source archive remains unchanged by default.

Default:

```bash
vibium verify --input record.zip   "checkout completed successfully"
```

returns a concise human-readable verdict to stdout.

Structured output:

```bash
vibium verify --json --input record.zip   "checkout completed successfully"
```

returns the common `VerificationResult` as JSON to stdout.

Persistent result:

```bash
vibium verify --input record.zip   --report verification.json   "checkout completed successfully"
```

writes the result to the requested path.

The source archive remains unchanged in all cases.

A later version may support explicitly creating a **derived** record containing the new verification result, but that must be opt-in. Do not mutate the source `record.zip` or `trace.zip` in v0.8.

---

## 14. Recording integration

`record.zip` is already Playwright-trace-compatible and already records Vibium commands through the existing recording/dispatch path.

Do **not** replace the trace format or introduce an incompatible custom event stream.

Desired Record Player timeline:

```text
Verify: "changing my display name persists after refresh"
    ├── map
    ├── fill
    ├── click
    ├── navigate/reload
    └── map
PASS
```

Recommended implementation:

1. record Verify as a named parent span/action group
2. run verifier-generated Vibium actions through the existing `dispatch()` path
3. ensure child actions inherit the Verify group's `parentId`
4. preserve normal screenshots, DOM snapshots, network events, and timestamps
5. associate the final verdict/summary with the Verify span

Prefer existing action-group/recording structures so the trace remains Playwright-compatible.

If the existing trace schema cannot safely store the final verdict/summary without breaking compatibility, add a **Vibium-namespaced sidecar entry inside the zip** keyed by the parent `callId`.

Conceptually:

```text
record.zip
├── trace.trace
├── trace.network
├── resources/...
└── vibium/
    └── verifications.jsonl
```

Example:

```json
{
  "callId": "call@42",
  "claim": "changing my display name persists after refresh",
  "status": "passed",
  "summary": "The new display name persisted after refresh."
}
```

Only add the sidecar if needed. Reuse any existing Vibium extension mechanism first.

A clean rule for all record behavior:

> **Vibium records may contain verifications. Verifying a record or Playwright trace does not alter the source archive unless the user explicitly requests a derived artifact.**

Never record:

- API keys
- passwords
- raw model chain-of-thought
- provider credentials

---

## 15. WebDriver BiDi relationship

Verify is built on top of WebDriver BiDi, but the first implementation should not require browser vendors to implement a new command.

The browser still sees normal BiDi operations.

Vibium handles the semantic command:

```text
vibium:verify.run
```

and the verifier composes normal Vibium/BiDi operations underneath it:

```text
vibium:verify.run
        ↓
Vibium verifier loop
        ↓
existing Vibium actions
        ↓
WebDriver BiDi
        ↓
Chrome
```

---

## 16. MCP

Do not block the first CLI proof on MCP.

After the CLI loop works, expose the same operation as:

```text
vibium_verify
```

Minimum live-session input:

```json
{
  "claim": "checkout still works"
}
```

Recorded/trace-session input:

```json
{
  "claim": "checkout completed successfully",
  "record": "record.zip"
}
```

MCP uses the generic `record` field; CLI `--input` normalizes to that internal archive field.

The MCP implementation must call the same internal Verify code as the CLI.
An optional `page` context ID pins a live check; omit it to use the active page.

`record` and live-session selection should be mutually exclusive in the first implementation.

---

## 17. Language APIs

Do not block the first CLI proof on SDK parity.

After the core loop works:

JavaScript / TypeScript:

```js
const result = await vibe.verify("checkout still works")

const recorded = await vibe.verify(
  "checkout completed successfully",
  { record: "./record.zip" }
)
```

Python:

```python
result = vibe.verify("checkout still works")

recorded = vibe.verify(
    "checkout completed successfully",
    record="./record.zip",
)
```

Java:

```java
VerificationResult result = vibe.verify("checkout still works");

VerificationResult recorded = vibe.verify(
    "checkout completed successfully",
    VerifyOptions.builder().record(Path.of("record.zip")).build()
);
```

All must delegate to the same underlying Vibium runtime.

`perform()` is a separate high-level operation and is **out of scope for this implementation task**.

---

## 18. Future hosted verifier

Do not implement this now, but preserve the abstraction.

Later:

```bash
VIBIUM_VERIFIER_PROVIDER=vibium
VIBIUM_API_URL=https://vibium.com
VIBIUM_API_KEY=...
```

should make the same command:

```bash
vibium verify "checkout still works"
```

use the Vibium-hosted verifier.

That verifier can eventually add:

- prior verification history
- previous `record.zip` evidence
- app maps
- known journeys
- prior failures
- git/PR context
- team policies
- managed test capabilities
- persistent application memory

The local/BYO verifier proves the interaction.

The hosted verifier becomes valuable because it knows more.

---

## 19. Acceptance test — first PR

Setup:

- Vibium built locally
- Chrome working through Vibium
- a test web app running locally
- an OpenAI API key/model configured
- the builder may also be Codex/OpenAI; a different provider/account is not required
- recording enabled

Example:

```bash
export VIBIUM_VERIFIER_PROVIDER=openai
export VIBIUM_VERIFIER_MODEL=<model>
export OPENAI_API_KEY=...

vibium record start -o record.zip
vibium go http://localhost:3000/account
```

Run:

```bash
vibium verify "changing my display name persists after refresh"
```

Required behavior:

1. reuse the existing Chrome session
2. start a fresh verifier model context with no inherited builder conversation
3. give that verifier only the verifier-specific instructions and constrained Vibium/browser tools
4. allow the verifier to inspect and operate the current page
5. return `PASS`, `FAIL`, or `INCONCLUSIVE`
6. return a concise evidence-backed explanation
7. record the Verify parent span in `record.zip`
8. record verifier-driven child browser actions under that span
9. preserve Playwright trace compatibility
10. avoid recording secrets or model chain-of-thought

Then:

```bash
vibium record stop
```

must produce a recording that lets the Vibium Record Player reconstruct:

```text
what was asked
→ what browser actions were performed
→ what was observed
→ what the verifier concluded
```

### Acceptance test — recorded/trace session (second slice)

Run both:

```bash
vibium verify --input record.zip "checkout completed successfully"
vibium verify --input trace.zip "checkout completed successfully"
```

Both must:

1. avoid launching a browser
2. leave the source archive unchanged
3. expose only read-only trace evidence to the verifier
4. return PASS, FAIL, or INCONCLUSIVE
5. return INCONCLUSIVE when the trace lacks enough evidence
6. use embedded verification spans when present in a Vibium record
7. work with a vanilla compatible Playwright trace that has no Vibium-specific metadata
8. write only to stdout by default, JSON stdout with `--json`, or a separate JSON file with `--report`
9. avoid sending the entire raw zip to the model by default

---

## 20. Suggested implementation order

1. Inspect the existing CLI/router/daemon and recording `dispatch()` implementation.
2. Add internal `VerificationRequest`, `VerificationResult`, and `Verifier` types.
3. Add one OpenAI/OpenAI-compatible verifier adapter.
4. Add `verify` CLI parsing.
5. Route `verify` through the existing Vibium command router.
6. Bind the verifier to the current session.
7. Expose a small set of existing Vibium actions as verifier tools.
8. Implement the bounded verifier tool loop.
9. Return the structured verdict.
10. Wrap verifier actions in an existing recording action group/span.
11. Add verdict metadata using existing trace capabilities or a Vibium sidecar if necessary.
12. Add focused unit tests.
13. Add one end-to-end local Chrome test.
14. Add `TraceObservationSource` for Playwright-compatible archives.
15. Support both Vibium `record.zip` and vanilla Playwright `trace.zip`.
16. Add `vibium verify -i <archive> "<claim>"` (`--input`) for both recording formats.
17. Add live recording output with `-o`/`--output` and separate JSON reports with `--report`.
18. Add end-to-end tests for both Vibium records and Playwright traces.
19. Add MCP using the same underlying Verify implementation.
20. Add JS/TS, Python, and Java APIs using the same underlying Verify implementation.
21. Add Firefox parity immediately after the first local-Chrome loop is stable.

---

## 21. Design principle

Keep the first proof small:

> **Builder agent writes the software. A fresh verifier context gets independent evidence — from the live browser or an existing recording — and gives it a second opinion. Different models/providers/accounts can strengthen the boundary, but fresh context and constrained role are the minimum requirement.**

For the first proof, use the live browser. Then add Playwright-compatible trace archives (`record.zip` / `trace.zip`) as the second observation source.

Do not build the future Vibium Verify platform before proving those loops are useful.

## Implementation boundaries

The reader currently supports Playwright trace **version 8**, including real
Playwright 1.58.2 archives and Vibium recordings. Unsupported versions produce
an explicit error. CLI recordings supply screenshots and action observations;
Playwright DOM snapshots are decoded when the input contains them.

Known credentials and structural secret fields are redacted across recording
text. Visual artifacts are omitted for recordings that encounter sensitive
form fields. No implementation can guarantee detection of every unknown
secret embedded in arbitrary application text or opaque images; that absolute
privacy guarantee remains a limitation, not an assertion made by this build.
Hosted verification, Cortex, PR integration, remote/cloud live verification,
`perform()`, and derived input-plus-output archives remain explicitly deferred.
