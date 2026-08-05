# Client Conformance Suite

**Goal:** Make the client contract executable, so adding Ruby (#68), C# (#90), Nim, or Rust is a weekend rather than a liability, and so a client that drifts from the protocol fails a test instead of a user's script.

**Status:** Spec. Not started. Written 2026-08-05.

---

## Motivation

`page.expose` was broken in every client, differently, for months. Nobody noticed until a user filed a bug.

| Client | Defect | Found by |
|---|---|---|
| JS, Python | function vanished on the next navigation | reading the handler while fixing something else (#297) |
| Java | never sent `fn` at all, so the server had nothing to inject | a user (#135) |

Each client had a test for `expose`. Every one passed. The JS test navigated *before* exposing and never navigated again, so the navigation bug was invisible to it. Java had no test for the method at all.

That is the shape of the problem: three implementations of 114 `vibium:` methods, verified only by tests each author wrote independently against a prose description. Nothing compares what a client sends to what the server expects. Coverage is not comparable either — 35 test files for JS, 33 for Python, 15 for Java measures nothing about the contract.

A fourth and fifth client multiply this. The suite exists so they do not.

---

## Layer 1 — Protocol manifest

A machine-readable contract generated from the implementation, so it cannot drift from it.

**Artifact:** `docs/reference/protocol.json`, generated and checked in.

**Schema:**

```json
{
  "version": "26.5.31",
  "methods": [
    {
      "name": "vibium:page.expose",
      "aliases": ["vibium:element.expose"],
      "params": {
        "context": { "type": "string", "required": false, "resolved": true },
        "name":    { "type": "string", "required": true },
        "fn":      { "type": "string", "required": true }
      },
      "result": { "exposed": { "type": "boolean" } },
      "errors": ["name is required", "fn is required"]
    }
  ]
}
```

- `aliases` — the router dispatches several names to one handler (`case "vibium:element.find", "vibium:page.find":`). A client may send any alias.
- `resolved: true` — the server fills this from session state if absent, so a client omitting it is conforming.
- `errors` — literal messages the handler can return, for negative cases in Layer 3.

**Generator:** `clicker/cmd/protocolgen`, walking the `case "vibium:...":` arms in `api/router.go` and the handler each dispatches to. Params come from the `cmd.Params[...]` reads in the handler body; anything it cannot infer is listed in a `TODO` block in the output rather than silently omitted.

**Acceptance:**
- [ ] `make protocol-manifest` regenerates it
- [ ] every one of the 114 dispatched methods appears
- [ ] CI fails when the checked-in file differs from a fresh generation
- [ ] unresolvable params are reported, not dropped

### Alternative considered

Hand-writing the manifest. Rejected: a hand-written contract is a third thing to keep in sync, and `api.md` already demonstrates the failure mode — its Protocol column names `browsingContext.handleUserPrompt` for dialog accept while the router also accepts `vibium:dialog.accept`, and no reader can tell which is canonical.

---

## What happens to api.md

`api.md` stays. It answers a question the manifest cannot: *how do I do this in each surface?* Its 150 rows map a capability across CLI, MCP, JS, and Python, which is the table a human reaches for and which no generated wire contract replaces.

What changes is that it stops being a second, unverified source of truth.

**The Wire Command column gets a rule.** Today it mixes 109 `vibium:` methods, 15 raw BiDi commands, and some prose. That is why dialog accept is ambiguous: row 137 names `browsingContext.handleUserPrompt` while the router also accepts `vibium:dialog.accept`, and a client author cannot tell which to send.

The rule: **name what a client should send.** Where a `vibium:` extension exists, name it. Raw BiDi only where there is no extension and the command is forwarded to Chrome untouched. Prose only where no single command applies, as with browser launch.

**And it gets checked.** Once the manifest exists, `report.js` can verify the table rather than trusting it:

- every manifest method appears in exactly one row
- every `vibium:` Wire Command resolves to a manifest method or alias
- every raw BiDi Wire Command is genuinely absent from the manifest
- the JS and Python columns match the naming conventions in `client-implementation-guide.md:222`

That last check is worth having on its own. The naming table is the thing a Ruby or Nim author will read most closely, and it is currently advisory prose that nothing tests.

**Acceptance:**
- [ ] Wire Command rule documented at the top of `api.md`
- [ ] the 15 raw-BiDi rows audited: extension where one exists, left alone where not
- [ ] `make conformance` checks api.md against the manifest
- [ ] new columns for future clients are checked by the same rule, not added by hand and hoped for

---

## Layer 2 — Wire recorder

Measures what each client actually sends, by watching its existing tests. Requires no client changes, which is the point.

**Artifacts:**
```
tests/conformance/record.js     wraps a client run, tees stdio
tests/conformance/report.js     diffs the recording against the manifest
```

**Mechanism:** `record.js` sets `VIBIUM_BINARY` to a shim that spawns the real binary and copies both directions of the ndjson stream to a session log. The client is unmodified and unaware.

**Recording format**, one JSON object per line:

```json
{"t": 1234, "dir": "out", "msg": {"id": 7, "method": "vibium:page.expose", "params": {"name": "add"}}}
{"t": 1240, "dir": "in",  "msg": {"id": 7, "type": "error", "error": "error", "message": "fn is required"}}
```

**Report output:**

```
client   methods  covered  param-mismatch  unknown-method
js       114      88 (77%)  0               0
python   114      81 (71%)  0               0
java     114      34 (30%)  1               1
                            ^ page.expose sends {name}, manifest requires {name, fn}
                                            ^ waits on vibium:page.exposedFunction, not in manifest
```

**Acceptance:**
- [ ] `make conformance` records all three clients and prints the table
- [ ] param mismatches name the method, the sent params, and the required ones
- [ ] a method sent but absent from the manifest is reported, not ignored
- [ ] CI publishes the numbers; failing on regression waits until a baseline exists

### Alternative considered

Asking each client to emit its own coverage. Rejected: it needs per-client work before producing any value, and a client that mis-sends a param is exactly the client whose self-report cannot be trusted.

---

## Layer 3 — Conformance battery

What a *new* client implements to prove itself. Layer 2 measures what exists; this defines what correct means.

**Artifacts:**
```
tests/conformance/cases/*.json          language-agnostic scenarios
tests/conformance/drivers/<lang>/       one small driver per client
```

**Case schema:**

```json
{
  "name": "page.expose sends name and fn",
  "requires": ["page.expose"],
  "steps": [
    {
      "call": "page.expose",
      "args": { "name": "add", "fn": "(a,b) => a+b" },
      "expect_wire": {
        "method": "vibium:page.expose",
        "params": { "name": "add", "fn": "(a,b) => a+b" }
      }
    },
    { "call": "page.goto", "args": { "url": "$BASE/example" } },
    {
      "call": "page.evaluate",
      "args": { "expression": "window.add(2,3)" },
      "expect_result": 5
    }
  ]
}
```

- `expect_wire` asserts the message the client sends — catches Java's missing `fn` without running a browser.
- `expect_result` asserts observable behaviour — catches the JS navigation bug, because the step order puts the navigation *between* expose and use.
- `$BASE` resolves to `tests/helpers/test-server.js`. No external sites.

**Driver contract.** A client implements one executable that reads cases on stdin and reports per-step results on stdout. It maps `"page.expose"` to that language's binding and nothing else:

```
in:  {"call": "page.expose", "args": {"name": "add", "fn": "(a,b) => a+b"}}
out: {"ok": true, "result": null}
out: {"ok": false, "error": "NoSuchMethod: page.expose"}
```

Method names in cases are canonical dotted form. The driver applies the naming convention for its language, so `page.setViewport` becomes `setViewport()` in JS and Nim, `set_viewport()` in Python and Ruby, `visible?` in Ruby. The conventions are already tabulated in `client-implementation-guide.md:222`; Layer 3 makes that table testable rather than advisory.

**Acceptance:**
- [ ] case format and driver contract documented in the client implementation guide
- [ ] JS driver as the reference
- [ ] cases covering every manifest method, including negative cases from `errors`
- [ ] Python and Java drivers
- [ ] a case that fails against pre-#297 and pre-#135 behaviour

### Alternative considered

Reusing each client's existing tests as the battery. Rejected: they encode each author's assumptions, which is the original problem. The JS `expose` test passed throughout the navigation bug because of the order its author happened to choose.

---

## Constraints

- Go stdlib only for the generator. No new dependencies.
- The recorder must not require client changes. If it does, the design is wrong.
- Cases are data. A case needing a language-specific escape hatch belongs in that client's own suite.
- No external sites — `tests/helpers/test-server.js` only.
- Layer 2 does not fail CI until a baseline exists. The first run will be red, and that is the finding.

---

## Predictions

Recorded now so the first run tests the recorder as much as the clients.

1. Java's `page.expose` fails param validation, sending `{name}` against a required `{name, fn}`.
2. Java's method coverage lands well below JS and Python — nearer a third than the two thirds those reach.
3. Some `vibium:` methods are exercised by no client, only by the CLI or MCP.
4. Every client omits at least one `resolved` param, and that is conforming rather than a defect.

If 1 and 2 do not appear, the recorder is not seeing everything and the numbers should not be trusted.

---

## Related

- #135 — Java `expose`; #297 — the JS and Python half of the same method
- #298 — exposed functions, which adds a bidirectional case Layer 3 must cover
- #68 Ruby, #90 C#, #81 Robot Framework — what this is meant to make cheap
- `docs/explanation/client-implementation-guide.md` — the prose contract this makes executable
