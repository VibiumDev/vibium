# Why independent verification matters

A builder can successfully run a browser command and still be wrong about what
the application does. A click can land on the intended button while the save
fails. A success message can appear while a refresh restores the old value. A
cart badge can increase while the wrong product ends up in the cart.

Vibium Verify adds a separate check of the result. The builder supplies a claim
about the running application. A fresh verifier context inspects the browser,
chooses browser actions, and returns a verdict with observable evidence.

The purpose is to make completion depend on another examination of the
application. The verifier is instructed to investigate what happened, including
the possibility that the claim is false or cannot be established. Its verdict
is a model's assessment of the evidence and can be wrong.

Think of Verify as an independent acceptance check for coding agents. It can
provide a useful second opinion before the builder declares completion. It
does not prove that the software works or replace automated tests. The current
implementation demonstrates the workflow; we have not yet measured how much
reliability it adds compared with the builder checking its own work.

For a hands-on introduction, see [Your coding agent's first verification on var.parts](../tutorials/first-verification.md).
For configuration and commands, see [Verify a claim in local Chrome](../how-to-guides/verify.md).

## Why asking the builder again can miss the problem

While building a feature, an agent accumulates a detailed account of how that
feature should work. It has seen the requirements, chosen an implementation,
responded to failures, and explained its fixes. That context is useful for
making changes. It can also carry assumptions into the agent's evaluation of
its own work.

Consider a display-name form. The builder fixes its submit handler, sees a
"Saved" message, and concludes that the name persists. When asked to check its
work again in the same conversation, it still has the successful edit and its
explanation of the fix in context. It may look for the same success message
again. The missing check is whether the name survives a reload.

A separate verifier starts with the claim and the application as it exists.
It does not automatically receive the builder's story about the implementation.
That gives it an opportunity to choose a different observation: change the
name, save, reload, and read the value again.

This separation can help expose assumptions. It is not a guarantee that a model
will choose a good test. The verifier can also overlook a condition or
misinterpret a page. Its evidence and the recording remain essential to judging
its conclusion.

## What “independent” means in this implementation

Vibium establishes independence through the invocation's context, role, and
permissions:

| Boundary | What Vibium provides | Why it matters |
|----------|----------------------|----------------|
| Fresh inference context | A new message history for every verification | Earlier builder messages and explanations do not carry into the check. |
| Verifier-specific role | Instructions to gather evidence, question the claim, and admit uncertainty | The task is to assess behavior, rather than finish the implementation. |
| Constrained tools | An enforced set of browser operations and validated arguments | The verifier can investigate the application without access to source edits, shell commands, or deployment tools. |
| Explicit result | PASS, FAIL, or INCONCLUSIVE with concise evidence | The caller gets an assessment it can inspect and act on. |

The same OpenAI provider, model, and account can be used for both building and
verification. Each Verify invocation still starts a distinct conversation with
its own instructions and tool boundary.

Using a different model or provider may add another perspective, but it does
not automatically make the result more reliable. Models can share similar
assumptions or miss the same ambiguity. A fresh context is a concrete isolation
boundary; it does not imply that the builder's and verifier's mistakes are
statistically independent.

The claim also deserves scrutiny. If the builder submits an incomplete
requirement, a verifier can correctly assess that narrow claim while missing
the user's actual goal. “The button shows a success message” is a much weaker
requirement than “The changed name remains visible after a reload.”

## Why the browser session stays the same

The verifier's conversation is fresh. The application state is deliberately
preserved.

The builder may have signed in, selected a product, filled a form, or navigated
to a particular account. Those actions establish the situation being checked.
Opening a different browser could lose the relevant cookies, storage, cart,
unsaved input, or authentication. The verifier would then be examining a
different situation.

Verify therefore uses the daemon's existing local Chrome session and pins its
tools to the active tab. The browser process and its state are shared with the
builder. The daemon serializes the verification and its child actions so other
daemon commands wait until the check completes.

This choice also defines the reach of the evidence. A cart surviving a reload
in one browser demonstrates persistence across that reload. It does not by
itself establish persistence across another device, a new browser profile, or
a server restart. Those require different claims and observations.

Verification can change the page. A verifier checking persistence may edit a
value, click Save, or navigate. The implementation does not promise to restore
the starting state. A caller should use a suitable test environment and inspect
the final state before continuing a workflow.

## How the verification loop works

The CLI routes the semantic operation `vibium:verify.run` through the existing
local daemon connection. The daemon keeps ownership of the browser while a
provider adapter runs the inference loop:

```text
Builder supplies a claim
          |
          v
Existing Vibium daemon --------> Existing Chrome session
          |                            ^
          v                            |
Fresh verifier context                 |
          |                            |
          +-- chooses browser tool ----+
          |         |
          |         +-- observation returns to verifier
          |
          +-- chooses another tool, or returns a verdict
```

Before the first inference, Vibium obtains the current URL, an interactive
element map, and an accessibility tree. The provider receives those observations,
the claim, verifier instructions, and the tool definitions. It does not receive
the builder's full conversation.

The verifier can then request existing Vibium operations such as finding an
element, reading text or a value, clicking, filling, pressing a key, scrolling,
navigating, or reloading. Each accepted call runs through the same browser
command implementation used by ordinary CLI automation. There is no second
browser automation engine or separate browser session for the model.

Text observations are bounded and marked when truncated. Screenshots are sent
only when requested and within the image limit. Console messages, page errors,
and network summaries can be read from the active live recording; if recording
is off, those tools report that the evidence is unavailable. The entire archive
is not automatically sent to the provider.

The current adapter uses OpenAI-compatible Chat Completions function tools.
The model is configurable. Every verification creates its own in-memory
conversation, and the adapter discards free-form assistant content accompanying
tool calls and provider reasoning fields. It retains the tool interaction
needed for the current run and parses the final structured verdict.

## The tool boundary and its limits

Tool restrictions are enforced in code. The verifier cannot invoke arbitrary
Vibium commands simply by naming them: the executor checks the allowlist,
argument names, and argument types. File-output parameters are removed from the
screenshot tool. Navigation accepts HTTP(S) URLs without embedded credentials.
Password input targets are unavailable.

The provided tools do not include arbitrary JavaScript evaluation, shell
execution, source-code changes, browser lifecycle operations, or recording
controls. These restrictions keep the verifier's capabilities aligned with
browser observation and interaction.

Some restrictions are instructions rather than mechanical guarantees. The
verifier is instructed to stay within the claim, treat page content as
untrusted evidence, and avoid purchases, messages, and other irreversible
actions. A generic click tool cannot determine every possible consequence of a
button on every website. The current implementation therefore should not be
understood as a complete policy engine for safe interaction with arbitrary
production applications.

The run is also bounded: a three-minute budget, at most 24 model-selected
actions, and payload limits constrain how much investigation it can perform.
An action-limit exhaustion returns INCONCLUSIVE. Timeout, provider, malformed
response, and browser execution failures are reported as errors. Existing
browser polling and recording cleanup may take additional time after the
verification budget expires.

## What the verdict establishes

The verdict describes the supplied claim in the observed situation:

| Verdict | Meaning | Cart example |
|---------|---------|--------------|
| PASS | The verifier found evidence supporting the claim. | After reloading the cart, the named product remains at quantity one. |
| FAIL | The verifier found evidence contradicting the claim. | After reloading, the cart is empty. |
| INCONCLUSIVE | The available evidence does not establish or refute the claim. | The page loads, but its contents do not identify the selected product or quantity clearly enough. |

A provider authentication error is an execution failure, rather than evidence
that the application failed. The CLI distinguishes these cases: completed
verdicts have a structured status, while execution failures return an error.
All three completed verdicts currently exit with code zero. A caller that needs
automatic pass/fail gating must inspect the JSON status.

An effective claim names the behavior and the observable outcome. “Checkout
works” leaves product selection, quantity, payment, confirmation, and order
history unspecified. “Refreshing the current cart keeps exactly one Vibium
Battery Pack in it” gives the verifier a bounded question and a visible
criterion. It also leaves later checkout behavior outside that conclusion.

A concise explanation should connect the conclusion to observations. A button
click proves that an action was attempted. A successful network response is
another piece of evidence. The resulting page state may still need to be
checked before a broader claim is supported.

The var.parts tutorial illustrates why the claim matters. In its rehearsal,
the current-cart check passed: the expected product, quantity, and subtotal
were present. A second claim about retaining that product after refresh
failed. The first PASS did not establish persistence. Asking the stronger
question exposed a different behavior that needed checking.

That example demonstrates useful evidence gathering, but it does not establish
that only a separate verifier could have found the problem. A builder or a
deterministic test could also reload the cart. The intended benefit of the
fresh context is another opportunity to question assumptions; it does not
automatically improve the claim or ensure a thorough investigation.

## Why the recording is part of verification

A verdict is more useful when another person can examine what led to it.
While recording is active, Verify becomes a named parent group in the existing
Playwright-compatible trace. Its browser actions inherit the group's
`parentId`, alongside the recorder's normal screenshots and browser events.

For a cart check, the timeline might contain:

```text
Builder: open product, add it to cart, open cart
Verify: "Refreshing the current cart keeps exactly one ..."
    Read URL and page structure
    Inspect product and quantity
    Reload
    Inspect product and quantity again
    PASS with concise evidence
```

The exact actions are chosen by the model, so their order and number can vary.
The claim is stored in the group's parameters. Concise child observations and
the final result use standard trace `after.result` fields, and the completed
group title includes the verdict and evidence for the existing Record Player.
The current implementation needs no extra sidecar or trace-format version.

The recording captures actions, observations, and the conclusion. Provider
credentials, raw model responses, and private model reasoning are not passed
to the recorder. Record export also masks common credential fields in HAR
and raw BiDi events: authorization and API-key headers, cookies, password
fields, and token query parameters. Password input values in structured DOM
snapshots are masked when present.

This is structural redaction, not general secret detection. Sensitive text
elsewhere on a page, in a console message, in an unusually named field, or in
a screenshot can still reach the provider or recording. Use test data when
checking sensitive applications; review recordings before sharing them. The
specification's absolute “never record secrets” requirement is not fully
guaranteed for arbitrary application content.

An archived check can be reviewed later, but it describes the run that produced
it. A recorded PASS does not prove that a later deployment or a different
session has the same behavior. Verifying an existing `record.zip` or Playwright
trace as input is a separate capability and is not implemented in this slice.

## How this fits alongside tests and review

Verify is a promising fit when the desired outcome is easy to state but the right
sequence of observations depends on the current page. It lets a builder ask a
bounded question without having to prescribe every locator and assertion.
Examples include exploring a changed user flow or assessing a visible outcome
that needs judgment. These checks benefit from reviewing the observations
alongside the verdict.

For simple, stable assertions that run repeatedly, prefer deterministic tests.
They avoid repeated inference cost and model variation, and are usually faster
to run. Checking a known cart's product, quantity, and subtotal is a reasonable
example; using a model for it in the tutorial teaches the handoff without
establishing that a model is necessary for that assertion.

A verifier finding that a name reverts after reload can point to a concrete
regression test: edit, save, reload, and assert the value. That test can then
run on subsequent changes, while an independent verifier explores another
claim or examines behavior in a particular live session.

Each Verify invocation adds model latency and, depending on the provider,
inference cost. A vague claim or a check that merely repeats an adequate
deterministic assertion may add little useful evidence. Use the formal step
where another investigation is valuable, and retain the tests that already
cover known behavior.

Code review, automated tests, and browser verification provide different kinds
of evidence. A browser check can demonstrate a user-visible result while
leaving implementation quality, security properties, and unexercised cases
unexamined. The current product boundary is local Chrome through the CLI;
MCP, SDK verification methods, Firefox, hosted verification, and recorded-trace
input remain outside this implementation.

## What remains to be demonstrated

Passing an end-to-end acceptance test shows that the CLI, browser, provider,
and recording work together. It does not measure the verifier's ability to
detect defects or avoid incorrect verdicts. The current rehearsals establish
a working mechanism and a plausible benefit, not a measured reliability gain.

A useful evaluation would compare the builder's own checks with the additional
Verify step on applications containing known defects and on working controls.
Both should be assessed against the same acceptance criteria. Measure which
defects the builder declares complete, which of those Verify catches, false
PASS and false FAIL verdicts, INCONCLUSIVE results, and added latency and cost.
Repeat the checks to expose model variability, and distinguish results for the
same model from results for different builder and verifier models.

Those measurements would show where another model investigation earns its
cost and where ordinary tests or a better acceptance claim do more of the work.
