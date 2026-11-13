# vibe-check — speaker notes

## 01. vibe.check()

Opening · 45 seconds

This is a 15–20 minute talk about two browser primitives. Check asks a question and investigates the evidence. Run does a browser task. The demo uses a deliberately broken local shop, real OpenAI calls, and native Firefox video.

These slides describe the current development build of Vibium. Confirm the installed version exposes these commands before the meetup. The demo is an illustration, not an accuracy benchmark.

## 02. Is GitHub down?

Opening example · 60 seconds

Start with a familiar question: GitHub will not load, and you want to know what the browser can observe. The command checks the claim that the site is down. PASS supports that claim; FAIL means the evidence contradicts it, such as the site loading successfully. INCONCLUSIVE means there is not enough evidence.

This is an example command, not a captured GitHub outage or a current availability report. One browser observation does not establish a global outage. A dedicated uptime monitor is a better fit for continuous availability checks.

## 03. Check asks a question.

Check · 60 seconds

The prompt asks whether the cart contains exactly one battery pack. Check returns a verdict about whether that condition holds. Check investigates the existing state and returns evidence. Its job is to assess the claim, not make it true by adding an item.

PASS supports the claim, FAIL contradicts it, and INCONCLUSIVE means the evidence is insufficient. Provider and browser failures are execution errors, separate from those verdicts. PASS remains a model assessment, not proof.

This example assumes the cart is already open in a Vibium browser. Check can follow manual work, a coding agent, or individual browser commands. Run is not required.

Source: [Check reference](sources/check.md).

## 04. Check looks for evidence.

Independence · 90 seconds

The builder knows its plan, patches, and explanations. A fresh verifier is asked to investigate a claim using browser evidence rather than continue that story. That removes inherited conversation as one source of anchoring.

The boundary is conversational, instructional, and tool-based. It does not remove shared model biases, ambiguous claims, correlated failure modes, or misleading application content. Using another provider might help on a particular workload, but this talk presents no comparative measurement.

Do not oversell the word independent. Check can still be wrong, and both models can miss the same bug. Use it where a new investigation adds information.

Source: [How Run and Check work](sources/run-check-webdriver-bidi-architecture.md).

Verifier-specific instructions and constrained tools accompany the fresh context. The same provider or model is allowed. A fresh conversation does not make errors statistically independent.

## 05. Run does a thing.

Run · 60 seconds

Run takes a task and works toward it: navigate to the shop, find the battery pack, and add it. Check asks whether the resulting cart satisfies a claim. Both can use browser tools, but their purpose differs: Run changes the state toward a goal; Check investigates whether a claim holds.

--keep-open preserves the browser started by this command so you can follow it with the cart Check from the earlier slide. A completed Run is the model's assessment that it accomplished the task, not an independent confirmation of the outcome.

The CLI shorthand is vibium "<browser task>". In JavaScript and Python, vibe(goal) is also shorthand for vibe.run(goal). Run returns evidence along with COMPLETED or NOT_COMPLETED.

Check may navigate, reload, or interact to answer its question. It is not a general read-only sandbox and does not roll back changes. Checking an existing archive is read-only.

Source: [Run guide](sources/run.md).

## 06. How Check answers your question

How it works · 60 seconds

Verified against the implementation: live Check starts a new model conversation with verifier instructions, the question, and initial observations of the page URL, element map, and accessibility tree. The model can request allowed tools, such as clicking the cart link or reading visible text. Vibium validates and executes each request in the existing browser session, then sends the result back. Steps 2 and 3 repeat until the model returns its verdict or a limit or error stops the operation. Clicking does not automatically give the model a screenshot or the entire updated page; it can request further observations. Run uses the same request-and-result mechanism with task-specific instructions and completion results. Archive Check uses recorded evidence tools instead of a live browser.

The vibium-prefixed commands are Vibium extension commands handled by the runtime. They are not new standard browser-native WebDriver BiDi commands, and Chrome or Firefox does not execute a language model. The runtime calls the provider, validates the selected tool, and dispatches existing browser operations.

The provider sees a constrained tool schema, not the general CLI, shell, filesystem, or an unrestricted script executor. Some recorded internal actions include page.eval because fixed observation handlers use that operation internally; this does not mean arbitrary model-authored JavaScript is permitted.

The same semantic parent names make Run and Check visible in Playwright-compatible recording data. There is no new client/runtime transport.

Source: [How Run and Check work](sources/run-check-webdriver-bidi-architecture.md).

The extension commands are `vibium:run.run` and `vibium:check.run`. The recorder groups child browser actions under these semantic parents.

## 07. Getting ready

Setup · 75 seconds

Use the installation instructions for the build you are demonstrating. This deck covers a development build and does not assert that every public npm release has these features. Check command help before presenting. The displayed model is the one used in the recorded demo; choose a model your account can access.

Put real credentials only in a private file outside the repository, with permissions 600. The example key is a placeholder. Vibium does not automatically load this file. Source the file in the same shell that runs the CLI. export makes a variable available to child processes.

Readiness checks browser executable files without launching a browser and makes a small provider tool round trip, so API charges can apply. Browser startup, connectivity, and application correctness remain untested. Never paste real keys into an agent prompt or slide.

Sources: [AI setup tutorial](sources/first-check.md); [provider settings](sources/model-providers.md); [readiness reference](sources/ready.md).

Before sourcing ai.env, create it privately and configure the shared AI settings:



```
export OPENAI_API_KEY='your-key'
export VIBIUM_AI_PROVIDER=openai
export VIBIUM_AI_MODEL=gpt-5.6-sol
export VIBIUM_AI_REASONING_EFFORT=none
```

Run and Check share these defaults. For the source checkout, use ./clicker/bin/vibium. Confirm both commands exist with check --help and run --help.

## 08. Your first Check

First command · 60 seconds

The appeal is a low-friction investigation: give a URL and a small claim, inspect the verdict, then add -o when evidence should be saved. The screenshot comes from a real Firefox Check of “https://var.parts is up” on September 7, 2026. Check returned PASS after observing the storefront. It demonstrates that run’s result, not a guarantee of current availability.

For a fixed uptime requirement, a deterministic HTTP or browser monitor is usually cheaper and easier to interpret. A model check is more interesting when the visible behavior or investigation is less prescribed. Make vague claims more specific as the requirement becomes clearer.

Chrome records screenshots and browser actions. Native continuous WebM is currently available with Firefox 154+. New output paths are required; Vibium does not overwrite existing recordings.

Source: [first check tutorial](sources/first-check.md).

A standalone Check closes the browser it starts. An existing browser stays open. Add --keep-open to preserve a newly started browser.

## 09. Add it. Then check the cart.

Cart demo · 60 seconds

This local demo store deliberately shows Added to cart and a badge of 1 without saving the cart item. It is a controlled fixture inspired by the var.parts examples, not a bug report about var.parts.

Start node demo/server.mjs from the presentation folder. Open the shop with vibium go http://127.0.0.1:4177 so Run and Check reuse that browser. The recording starts before Run and stops after Check, so flow.zip contains both operations. Adding -o only to Check would create a recording of that operation, omitting the earlier Run. Use Firefox 154+ for the continuous video shown here. The slide shows short prompts; the capture uses these complete prompts:



```
vibium run "Add one battery pack to the cart. Confirm the Added to cart message, then stay on the product page."
vibium check "Does the cart contain exactly one battery pack? Open the cart and inspect its contents without adding or removing anything."
```

Run stops after the product page confirms the addition. Check independently opens the cart and inspects the actual contents. The Run verdict reflects the misleading confirmation it saw. It is not proof that the item was saved.

Sources: [demo store](demo/shop.html); [capture script](demo/capture.mjs); [exact prompts and results](assets/demo-results.json).

## 10. Added. But the cart is empty.

Play the broken video · 60 seconds

Play the 15.7-second recording at normal speed. Run clicks Add to cart. Check starts around 8.2 seconds, opens the cart, and reads its contents.

Point out the Added to cart message and badge of 1, followed by the empty cart. Check looks beyond the confirmation without adding or removing items. This is a real OpenAI result on an intentionally broken local fixture.

The silent native Firefox WebM and its MP4 copy show the same run, without speed changes. This comparison illustrates one bug, not an accuracy or latency benchmark.

Sources: [original recording](assets/broken.zip); [actual results](assets/demo-results.json).

## 11. Evidence behind the FAIL

Evidence · 60 seconds

The screenshot and quoted page message come from the cart opened during Check. The model opened the cart and observed 0 items and Your cart is empty. The header still shows Cart 1, making the inconsistent UI visible.

This is also a good candidate for a deterministic regression test. The point is that a separate investigation can uncover a problem hidden by the completion message.

Source: [exact summary and evidence](assets/demo-results.json).

## 12. After the fix: PASS

Play the fixed video · 60 seconds

Play the 13.9-second recording at normal speed. Run clicks Add to cart. Check starts around 7.5 seconds, opens the cart, and reads its contents.

The fixed mode saves the cart quantity as well as updating the confirmation. Both captures use identical prompts and model settings in separate browser sessions. Check now sees 1 item and Quantity: 1. The fix uses sessionStorage in this demo, not a production backend.

The silent native Firefox WebM and its MP4 copy show the same run, without speed changes. This comparison illustrates one bug, not an accuracy or latency benchmark.

Sources: [original recording](assets/fixed.zip); [actual results](assets/demo-results.json).

## 13. The recording shows the work.

Recording · 90 seconds

This screenshot is the real local Record Player, with the Check parent selected, not a mock UI. The parent carries the claim and verdict. Its children show the browser work that led to that verdict.

Start recording after opening the application and keep a single session through the workflow. If a Chrome session is already running, stop it before starting the Firefox example. The full demo capture script adds snapshots, video size and frame rate for presentation quality. The shorter commands show the everyday workflow.

For one operation, -o creates its recording. If recording is already active, -o exports the current chunk and leaves the ongoing recording alone; that export has no continuous video. Use record stop to finalize the complete continuous recording.

Public provider/model settings and limits are recorded, but credentials and endpoint URLs are not put into modelConfig. Inspect artifacts before sharing them.

Sources: [original record](assets/broken.zip); [recording behavior and privacy](sources/check.md).

To record a complete workflow:



```
vibium --engine firefox go http://127.0.0.1:4177
vibium record start --video -o flow.zip
vibium run "<goal>"
vibium check "<claim>"
vibium record stop
```

Firefox 154+ supports native WebM. Chrome records screenshots and actions.

## 14. Check a recording or trace.

Archive verification · 75 seconds

Tested with the slide’s exact prompt and flags against both bundled recordings: broken returned FAIL, fixed returned PASS. Both created verdict.json with references to the recorded actions. Reports: [broken](assets/broken-archive-report.json); [fixed](assets/fixed-archive-report.json).

This is useful for post-run triage, reviewing a teammate's reproduction, or reassessing saved evidence under a clearer claim. An archive cannot reveal facts it never captured.

-i / --input reads an existing ZIP; -o / --output creates a live recording. They cannot be combined. --report saves structured verdict JSON and can accompany either mode. The old --record and --trace input flags are not used.

Version 8 refers to the trace format, not the Playwright package version. The report file holds the inner result object, so its status is at the top level. JSON stdout has a result envelope.

Source: [saved recording support](sources/check.md).

Supported inputs include Vibium recordings and Playwright traces in trace format version 8. Missing evidence should produce INCONCLUSIVE. The verdict covers the recorded run, not the current application state.

## 15. In the development loop

Agent workflow · 75 seconds

The human mainly installs Vibium, configures the model, and instructs the coding agent to use the CLI. The coding agent can handle navigation and individual actions, or delegate a browser goal to Run. Run is optional; Check can follow hand-written browser commands too.

Slash commands are coding-agent skills, not Vibium CLI flags. Their availability depends on the agent and skill installation. Use /browser for browser automation and Run, and /check for an explicit acceptance claim and evidence review.

Source: [coding-agent tutorial](sources/check-with-a-coding-agent.md); repository skills/check and skills/browser.

## 16. Run. Check. Review.

Tradeoffs · 90 seconds

It is reasonable to ask whether Check is pointless for a simple known assertion. Sometimes it is: a conventional assertion can be faster, cheaper, repeatable, and easier to maintain. The demo uses a simple invariant so the audience can judge the result directly.

The feature is more compelling when investigation itself is work: unfamiliar UI, exploratory behavior, an agent's claimed completion, or a user-facing condition that needs interpretation. Even then, define the claim and inspect the evidence.

Avoid turning every assertion into a model request. Keep unit tests, integration tests, and deterministic browser tests. Model calls add latency, cost, and variance.

Source: [when independent verification helps](sources/run-check-webdriver-bidi-architecture.md).

## 17. Automation reads the verdict.

Integration · 60 seconds

The Page object is already connected to the application's browser in this excerpt. SDK calls preserve the caller's browser ownership. Async and sync interfaces are available across supported languages; the example uses async JavaScript.

The CLI ok field describes operation completion, not claim truth. JSON stdout goes directly to jq, which makes a non-PASS fail the shell check. In Bash or Zsh, set -o pipefail also preserves a Vibium execution failure as a pipeline failure. No intermediate JSON file is needed. The --report file differs: it contains the inner result, so use .status on a report file.

For production gates, choose an explicit policy for inconclusive results rather than silently treating them as a pass.

Sources: [Run API](sources/run.md); [results and exit codes](sources/check.md).

PASS, FAIL, and INCONCLUSIVE all exit 0 when Check completes. Execution errors exit 1. The following JavaScript example uses an already connected Page:



```
const action = await page.run(
  "Add one battery pack to the cart");
if (action.status !== "completed") {
  throw new Error(action.summary);
}

const check = await page.check(
  "Does the cart contain exactly one battery pack?");
if (check.status !== "passed") {
  throw new Error(check.summary);
}
```

## 18. Choose the model per call.

Providers and evals · 60 seconds

Per-call settings are useful for controlled evaluations and do not mutate environment defaults. Switching provider clears inherited model, endpoint and reasoning settings; explicitly supply the new model. Same-provider overrides retain omitted shared defaults. Credentials still come from the provider's environment variables, not CLI key flags.

The local alias uses an already running compatible server; Vibium does not install or start llama.cpp. Native Anthropic and Google adapters and compatible/local paths have deterministic tests. Real-provider acceptance for Anthropic, Gemini and llama.cpp remains pending in this development checkout. The demo in this deck uses real OpenAI only.

Use AI readiness for each configuration. Avoid assuming all models support the same tool, image, or reasoning options.

Source: [model provider reference](sources/model-providers.md).

The same provider flags work on Run and AI readiness. --base-url can select a compatible endpoint. For evaluations, reset the fixture, keep the claim and conditions fixed, and compare repeated runs for verdicts, evidence, latency, and cost.

## 19. Check can still be wrong.

Boundaries · 75 seconds

The tools intentionally expose a subset of existing browser operations. That is useful containment, but page content can still mislead a model. Use a test environment appropriate for the actions being requested. Check retains stricter password-field restrictions; Run can enter explicitly supplied test credentials.

Recorder-wide known-secret redaction and recognized-sensitive-field visual suppression are implemented. Do not interpret that as detection of every unknown secret in every screenshot. Model requests and saved artifacts deserve a deliberate data boundary.

At the action limit, Check can return INCONCLUSIVE and Run NOT_COMPLETED. Timeouts and provider/browser failures are operational errors, not application verdicts.

Sources: [limits and privacy](sources/check.md); [Run boundaries](sources/run.md).

Live operations can change page state and do not undo their actions. Browser observations go to the configured provider. The default limits are 24 selected actions and a three-minute model budget.

## 20. vibe.check()

Close & questions · 45 seconds

The talk demonstrates the CLI, but Check is also available as the `vibium_check` MCP tool and through the JavaScript/TypeScript, Python, and Java SDKs. In the SDK example, `vibe` is the connected browser or page object. Async JavaScript uses `await vibe.check(question)`; Python supports async and sync APIs; Java uses `vibe.check(question)`.

MCP takes the question in its `claim` argument. The interfaces use the same underlying Check operation and return a verdict with evidence. Run is available through the CLI, MCP, and SDKs too.

Sources and setup: [Check interfaces](sources/check.md); [Run interfaces](sources/run.md); [examples and recordings](README.md).
