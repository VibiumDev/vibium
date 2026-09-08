# Speaker notes — Run & Check

Use with the HTML deck. Approximate total: 20 minutes, plus questions.

## 01. Run & Check

Opening · 45 seconds

This is a 15–20 minute talk about two browser primitives. Run attempts a goal. Check assesses a claim. The demo uses a deliberately broken local account page, real OpenAI calls, and native Firefox video.

These slides describe the current development build of Vibium. Confirm the installed version exposes these commands before the meetup. The demo is an illustration, not an accuracy benchmark.

## 02. A Saved message can hide a bug

Motivation · 60 seconds

Ask the room what they would check next. The page did everything the literal goal requested: it accepted the timezone and displayed Saved. Its default mode deliberately omits persistence. This is why the stronger acceptance claim matters.

Run did not prove persistence, and this example is not evidence that Run is defective. The task and the acceptance criterion have different scopes.

Source: bundled [actual model results](assets/demo-results.json) and [fixture implementation](demo/account.html).

## 03. Two primitives, two contracts

Contracts · 60 seconds

Both are model assessments. COMPLETED is not proof, and PASS is not proof. INCONCLUSIVE means the evidence is insufficient; it is useful to preserve that distinction instead of forcing a binary answer.

Check can interact with a live page. It is not a general read-only browser sandbox, and it does not roll back state. Archive verification is read-only. Operational failures are errors, separate from model outcomes.

Sources: [Run guide](sources/run.md); [Check reference](sources/check.md).

## 04. Fresh conversation, preserved browser

Independence · 90 seconds

The builder knows its plan, patches, and explanations. A fresh verifier is asked to investigate a claim using browser evidence rather than continue that story. That removes inherited conversation as one source of anchoring.

The boundary is conversational, instructional, and tool-based. It does not remove shared model biases, ambiguous claims, correlated failure modes, or misleading application content. Using another provider might help on a particular workload, but this talk presents no comparative measurement.

Do not oversell the word independent. Check can still be wrong, and both models can miss the same bug. Use it where a new investigation adds information.

Source: [How Run and Check work](sources/run-check-webdriver-bidi-architecture.md).

## 05. The model loop lives in Vibium

Architecture · 90 seconds

The vibium-prefixed commands are Vibium extension commands handled by the runtime. They are not new standard browser-native WebDriver BiDi commands, and Chrome or Firefox does not execute a language model. The runtime calls the provider, validates the selected tool, and dispatches existing browser operations.

The provider sees a constrained tool schema, not the general CLI, shell, filesystem, or an unrestricted script executor. Some recorded internal actions include page.eval because fixed observation handlers use that operation internally; this does not mean arbitrary model-authored JavaScript is permitted.

The same semantic parent names make Run and Check visible in Playwright-compatible recording data. There is no new client/runtime transport.

Source: [How Run and Check work](sources/run-check-webdriver-bidi-architecture.md).

## 06. Install, configure, ready

Setup · 75 seconds

Use the installation instructions for the build you are demonstrating. This deck covers a development build and does not assert that every public npm release has these features. Check command help before presenting. The displayed model is the one used in the recorded demo; choose a model your account can access.

Put real credentials only in a private file outside the repository, with permissions 600. The example key is a placeholder. Vibium does not automatically load this file. Source the file in the same shell that runs the CLI. export makes a variable available to child processes.

Readiness checks browser executable files without launching a browser and makes a small provider tool round trip, so API charges can apply. Browser startup, connectivity, and application correctness remain untested. Never paste real keys into an agent prompt or slide.

Sources: [AI setup tutorial](sources/first-check.md); [provider settings](sources/model-providers.md); [readiness reference](sources/ready.md).

## 07. Start with one claim

First command · 60 seconds

The appeal is a low-friction investigation: give a URL and a small claim, inspect the verdict, then add -o when evidence should be saved. The screenshot is an earlier PinThing capture; the commands now use var.parts. It is not evidence from running those commands or a claim about current availability.

For a fixed uptime requirement, a deterministic HTTP or browser monitor is usually cheaper and easier to interpret. A model check is more interesting when the visible behavior or investigation is less prescribed. Make vague claims more specific as the requirement becomes clearer.

Chrome records screenshots and browser actions. Native continuous WebM is currently available with Firefox 154+. New output paths are required; Vibium does not overwrite existing recordings.

Source: [first check tutorial](sources/first-check.md).

## 08. A goal, then a stronger claim

Demo setup · 60 seconds

Run the server from the presentation folder. A prior go command creates a browser that subsequent commands preserve. Use the same named session if you choose one. Run stop when the workflow is finished.

The actual recorded runs used this fixture on an automatically selected localhost port, Firefox, and explicit provider/model overrides. The commands on the slide use the fixed demo server port for convenient reproduction.

The no-editing instruction matters: the verifier should test persistence, not make the claim true by repairing the form. It narrows intent; it is not a substitute for the runtime's tool policies or for using safe test data.

Source: [capture script](demo/capture.mjs); [exact prompts and results](assets/demo-results.json).

## 09. Demo: the timezone does not persist

Play the first video · 60 seconds

Start playback. Run fills the field and clicks Save. Check starts a fresh model conversation around 8.3 seconds into this recording, reloads the page, and reads the field value.

There is no narration in the media. Explain the pause as model time, and point out the timezone returning to UTC. The video is native Firefox WebM, also transcoded to H.264 MP4 for broad playback compatibility. It has not been sped up.

Source: [original recording](assets/broken.zip); [actual result](assets/demo-results.json).

## 10. Evidence after reload

Evidence · 60 seconds

This is a stronger artifact than a bare FAIL. It connects a particular action—reload—to a specific observation that contradicts the claim. A developer can reproduce the failure and a reviewer can inspect the browser record.

This demo's field value is also easy to assert deterministically. That is intentional: the bug is obvious to the audience. The extra value of a model loop is more situational when it must find its way through an unfamiliar interface or interpret a less rigid observation.

Source: exact summary and evidence in [demo-results.json](assets/demo-results.json).

## 11. Same claim after the fix

Play the second video · 60 seconds

The fixed mode adds localStorage persistence; it is not a backend account system. Fresh browser sessions keep the two demo runs isolated. Both runs use the same literal Run goal and Check claim, with the same provider and model.

The second run passes because the observed timezone survives reload. One fail/pass pair on a purpose-built fixture does not establish a reliability rate. The two durations are demo timings, not latency promises.

Sources: [fixture](demo/account.html); [fixed recording](assets/fixed.zip); [model results](assets/demo-results.json).

## 12. A result you can inspect

Recording · 90 seconds

This screenshot is the real local Record Player, with the Check parent selected, not a mock UI. The parent carries the claim and verdict. Its children show the browser work that led to that verdict.

Start recording after opening the application and keep a single session through the workflow. If a Chrome session is already running, stop it before starting the Firefox example. The full demo capture script adds snapshots, video size and frame rate for presentation quality. The shorter commands show the everyday workflow.

For one operation, -o creates its recording. If recording is already active, -o exports the current chunk and leaves the ongoing recording alone; that export has no continuous video. Use record stop to finalize the complete continuous recording.

Public provider/model settings and limits are recorded, but credentials and endpoint URLs are not put into modelConfig. Inspect artifacts before sharing them.

Sources: [original record](assets/broken.zip); [recording behavior and privacy](sources/check.md).

## 13. Ask a question of saved evidence

Archive verification · 75 seconds

This is useful for post-run triage, reviewing a teammate's reproduction, or reassessing saved evidence under a clearer claim. An archive cannot reveal facts it never captured.

-i / --input reads an existing ZIP; -o / --output creates a live recording. They cannot be combined. --report saves structured verdict JSON and can accompany either mode. The old --record and --trace input flags are not used.

Version 8 refers to the trace format, not the Playwright package version. The report file holds the inner result object, so its status is at the top level. JSON stdout has a result envelope.

Source: [saved recording support](sources/check.md).

## 14. Put it in the development loop

Agent workflow · 75 seconds

The human mainly installs Vibium, configures the model, and instructs the coding agent to use the CLI. The coding agent can handle navigation and individual actions, or delegate a browser goal to Run. Run is optional; Check can follow hand-written browser commands too.

Slash commands are coding-agent skills, not Vibium CLI flags. Their availability depends on the agent and skill installation. Use /browser for browser automation and Run, and /check for an explicit acceptance claim and evidence review.

Source: [coding-agent tutorial](sources/check-with-a-coding-agent.md); repository skills/check and skills/browser.

## 15. Choose the right tool for the question

Tradeoffs · 90 seconds

It is reasonable to ask whether Check is pointless for a simple known assertion. Sometimes it is: a conventional assertion can be faster, cheaper, repeatable, and easier to maintain. The demo uses a simple invariant so the audience can judge the result directly.

The feature is more compelling when investigation itself is work: unfamiliar UI, exploratory behavior, an agent's claimed completion, or a user-facing condition that needs interpretation. Even then, define the claim and inspect the evidence.

Avoid turning every assertion into a model request. Keep unit tests, integration tests, and deterministic browser tests. Model calls add latency, cost, and variance.

Source: [when independent verification helps](sources/run-check-webdriver-bidi-architecture.md).

## 16. Gate on the verdict, not exit zero

Integration · 60 seconds

The Page object is already connected to the application's browser in this excerpt. SDK calls preserve the caller's browser ownership. Async and sync interfaces are available across supported languages; the example uses async JavaScript.

The CLI ok field describes operation completion, not claim truth. The jq command makes a non-PASS fail the shell check. It also avoids parsing human-oriented text. The --report file differs: it contains the inner result, so use .status on a report file.

For production gates, choose an explicit policy for inconclusive results rather than silently treating them as a pass.

Sources: [Run API](sources/run.md); [results and exit codes](sources/check.md).

## 17. Make provider choices explicit

Providers and evals · 60 seconds

Per-call settings are useful for controlled evaluations and do not mutate environment defaults. Switching provider clears inherited model, endpoint and reasoning settings; explicitly supply the new model. Same-provider overrides retain omitted shared defaults. Credentials still come from the provider's environment variables, not CLI key flags.

The local alias uses an already running compatible server; Vibium does not install or start llama.cpp. Native Anthropic and Google adapters and compatible/local paths have deterministic tests. Real-provider acceptance for Anthropic, Gemini and llama.cpp remains pending in this development checkout. The demo in this deck uses real OpenAI only.

Use AI readiness for each configuration. Avoid assuming all models support the same tool, image, or reasoning options.

Source: [model provider reference](sources/model-providers.md).

## 18. Keep the boundaries visible

Boundaries · 75 seconds

The tools intentionally expose a subset of existing browser operations. That is useful containment, but page content can still mislead a model. Use a test environment appropriate for the actions being requested. Check retains stricter password-field restrictions; Run can enter explicitly supplied test credentials.

Recorder-wide known-secret redaction and recognized-sensitive-field visual suppression are implemented. Do not interpret that as detection of every unknown secret in every screenshot. Model requests and saved artifacts deserve a deliberate data boundary.

At the action limit, Check can return INCONCLUSIVE and Run NOT_COMPLETED. Timeouts and provider/browser failures are operational errors, not application verdicts.

Sources: [limits and privacy](sources/check.md); [Run boundaries](sources/run.md).

## 19. Make the claim. Show the evidence.

Close & questions · 45 seconds

Invite a concrete claim from the audience. Tighten it together: which state, after what action, with what evidence? Then discuss whether an assertion, Run, Check, or saved-evidence review is the best fit.

Avoid promising that a model proves correctness. The closing question asks what evidence would actually justify the claim? The portable deck includes the fixture, capture script, original archives and exact results so attendees can inspect the example.
