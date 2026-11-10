# Vibium Find: Unified Element Search

> Give this file alone to the Vibium coding agent. The Roadie agent should use the separate Roadie Native UI Tree spec. Coordinate the tree transport schema before implementing either side.

## Implementation prompt for coding agent

### Goal

Extend Vibium's existing `find` command and SDK methods into a unified element-location primitive. Find elements through natural-language descriptions, explicit structural locators, or reference-image matching; register results in the existing Vibium element map; and let subsequent commands act on those elements via `@eN` references.

**Core principle: Find by intent. Interact by reference.**

Implement across the Go CLI/core/daemon, JavaScript/TypeScript (sync and async), Python (sync and async), Java, MCP server, documentation, agent instructions, tests, migration guide, and a short tutorial using https://var.parts.

**Implement the feature, not just a design.** Inspect the actual repository first, and adapt examples below to its current conventions.

### Product decisions (non-negotiable)

1. A bare string in `find` means a **natural-language description** across CLI and SDKs. This is an intentional breaking change from positional CSS.
2. Explicit `css`, `tag`, `text`, `role`, and other existing locator parameters retain their established meanings; `image` adds reference-image matching. Do **not** guess locator types from string syntax or file extensions.
3. CLI and SDK interpretation is **independent of target**: the same query has the same meaning on browsers and Roadie/mobile.
4. Natural-language `find` is **a bounded visual-grounding primitive, not an agentic task runner**. The outer agent (Claude, Codex, etc.) plans actions; `find` locates candidates and returns references/boxes.
5. Reuse the **existing AI configuration and provider adapters** from nightly `vibium run` and `vibium check`, including credentials, models, base URLs, and override precedence. Permit per-call model/provider overrides for evaluation. No new configuration subsystem in v1.
6. `find(image=...)` defaults to `strategy=auto`: try **local SIFT-based feature matching** first, then fall back to a configured vision-language model (VLM) if no verified match. `strategy=cv` is feature matching only; `strategy=vlm` is model grounding only. **Do not implement template matching.**
7. Return results through the **existing map and its `@eN` allocator**. Do not create a separate visual-reference namespace.
8. `find` **prints matches by default**; `--json` exposes structured results with bounding boxes. Preserve the existing `find`/`find --all` and SDK `find`/`findAll` cardinality conventions.
9. Keep the default response concise. Confidence diagnostics are optional and **raw algorithm scores must not be described as calibrated probabilities**.
10. The public `strategy` parameter applies to the entire `find` family; values are `auto` (default), `cv`, `vlm`, and `tree`. `tree` uses browser accessibility/DOM or native mobile UI hierarchy, never a fabricated HTML DOM.
10. Do not add mandatory Python, OpenCV, CGO, or external vision-model weights to Vibium's normal distribution. If OpenCV is necessary, make the backend optional. Review code, dependencies, and weights for Apache-2.0-compatible distribution.
11. An OmniParser-like pipeline (e.g., YOLOv9-E UI detector + Florence-2 captioner) is a **future optional extension**, not part of this initial build.

---

## 1. Repository reconnaissance

Repository: https://github.com/VibiumDev/vibium

Documentation to inspect:

- https://vibium.com/docs/commands/find/
- https://vibium.com/docs/commands/map/
- https://vibium.com/docs/commands/click/
- https://vibium.com/docs/commands/tap/
- https://vibium.com/docs/client-libraries/
- https://vibium.com/docs/mcp-integration/

Read repository guidance (including `CLAUDE.md`, if present) and inspect the **current code**, especially the nightly `run` / `check` implementations. Do not treat illustrations in this spec as proof of exact current API spellings.

Determine and preserve:

- CLI locator parsing, `--all`, `--limit`, `--json` (if present), `--timeout`, exit codes, and human-readable output.
- Current JS/TS, Python, and Java `find` and `findAll`/`find_all` signatures, return types, sync/async behavior, and locator composition.
- Existing map allocation, stable reference rules, `@eN` resolution, page/session scoping, and invalidation.
- Element interaction methods, coordinate-based `click`/`tap`, `highlight`, and screenshot geometry.
- Daemon JSON-RPC / WebDriver BiDi boundaries, MCP tool schema, and image-binary transport patterns.
- AI provider configuration and overrides used by `run`/`check`, including model capabilities and any configured reasoning effort.
- Build/release constraints (`CGO_ENABLED=0`, supported OS/architectures) and package/licensing requirements.

Document deviations when an existing repository constraint requires a narrower implementation.

---

## 2. Public locator contract

### CLI

```bash
# Bare string = description
vibium find "the Add to Cart button"
vibium find "the settings icon"
vibium find "the main heading"

# Explicit locators
vibium find description "the Add to Cart button"
vibium find image ./thumbnail.png
vibium find css "h1"
vibium find css "button.primary"
vibium find tag h1
vibium find text "Add to Cart"
vibium find role button
```

`vibium find h1` is now a **description of “h1”**, not a CSS selector. `vibium find thumbnail.png` is a description, not an automatic file lookup. Use explicit `css` and `image` forms to avoid ambiguity. No heuristics, target-dependent dispatch, or CSS-first/fallback-to-AI behavior.

### SDKs

**Python (illustrative of the intended public API):**

```python
vibe.find("the Add to Cart button")
vibe.find(description="the Add to Cart button")
vibe.find(image="thumbnail.png")
vibe.find(css="h1")
vibe.find(tag="h1")
vibe.find(text="Add to Cart")
vibe.find(role="button", text="Add to Cart")

matches = vibe.find_all(image="thumbnail.png")
for match in matches:
    print(match.box)
    match.click()
```

**JavaScript/TypeScript:** use established options-object syntax, including existing sync and async variants.

```javascript
const element = await vibe.find('the Add to Cart button');
await element.click();

await vibe.find({ description: 'the Add to Cart button' });
await vibe.find({ image: 'thumbnail.png' });
await vibe.find({ css: 'h1' });
await vibe.find({ tag: 'h1' });
await vibe.find({ role: 'button', text: 'Add to Cart' });

const matches = await vibe.findAll({ image: 'thumbnail.png' });
```

**Java:** preserve existing overload/builder/typed-parameter conventions. `find(String)` now means description; explicit locators are provided through the repository's existing idiom.

```java
var element = vibe.find("the Add to Cart button");
element.click();
// Use existing options/builder style for css, image, description, etc.
```

**Do not introduce** `findImage`, `findVisual`, `find_image`, or `find_visual`. Extend the existing `find` family with parameters.

Preserve existing supported combinations such as `role + text`. Initially require `description` and `image` each to be a standalone locator; reject unsupported combinations explicitly. A CSS query remains real CSS and should fail on targets without CSS support rather than being silently approximated visually. `tag` is HTML-specific; `role` is the more general cross-platform semantic abstraction.

### Cardinality and no-match behavior

- CLI `find`: first established match by default; `--all` produces multiple established matches; follow current ordering and limit conventions.
- SDK `find`: one Element-compatible object; `findAll`/`find_all`: collection. Follow existing SDK no-match/error conventions instead of changing their return types.
- JSON CLI output: **always an array**, including one match (`[ ... ]`) or zero matches (`[]`).
- MCP: preserve existing tool naming/cardinality where possible and clearly expose single versus all-match behavior.
- Do not silently choose a dangerous or poorly distinguished candidate when uncertainty/ambiguity prevents a reliable match.
- Operational errors (missing file, screenshot failure, unsupported model, timeout) are **not** empty matches.

---

## 3. CLI results and element map

`find` both **updates the map** and **prints the result**, by default. Example output (references illustrative):

```text
$ vibium find "the Add to Cart button"
@e7  button  "Add to Cart"

$ vibium find "product thumbnail" --all
@e7  image  "Product thumbnail"
@e8  image  "Product thumbnail"
```

Do not reset numbering to `@e1` for each call. Reuse the existing map allocator, identity semantics, session scope, persistence, and reference lifecycle. Avoid map regeneration that unnecessarily renumbers or replaces unrelated entries.

Structured output:

```bash
vibium find image thumbnail.png --json
```

```json
[
  {
    "ref": "@e7",
    "role": "image",
    "box": { "x": 120, "y": 80, "width": 160, "height": 90 }
  }
]
```

When no matches are established, JSON prints `[]`. Follow existing project stdout/stderr and exit-code conventions, with distinct treatment for errors. Do not clutter default output with box coordinates, inference-provider metadata, or raw confidence scores.

Add or reuse `--details` for diagnostics such as source, algorithm, raw score, inlier count/ratio, and verification evidence. **A SIFT score, detector classification score, and VLM confidence are not interchangeable probabilities.** Only use a probability label if empirically calibrated.

### Unified mapped element

Extend the existing mapped element representation, instead of creating another map. Conceptually a reference can point to:

- An actual DOM-backed element.
- A semantic/accessibility-backed element.
- A visual region identified by SIFT or model grounding.
- Future native or detector-derived elements.

Useful normalized metadata: `ref`, source/provenance, role, text/name, bounding box, context/session ID, underlying target or visual search metadata, and validity state. Preserve real DOM-native interaction when available. Visual-only elements may use coordinate-based interaction.

### Geometry and interactions

- Store boxes as `{x, y, width, height}`.
- Browser boxes must correspond to **viewport CSS coordinates**, accounting for device-pixel ratio, screenshot scaling/crops, and scrolling.
- Roadie/mobile adapters can translate the same public box into target interaction coordinates later.
- Default click/tap location is the box center. Support an optional positional override such as `--position bottom-right` if it fits existing interaction interfaces.
- `vibium click @e7`, `vibium tap @e7`, and `vibium highlight @e7` should work where appropriate, while preserving existing numeric-coordinate forms and native DOM interactions.
- A stale visual reference must not blindly click an old coordinate after navigation, scrolling, resizing, animation, or device rotation. Use lightweight validity checks and re-detection where feasible; otherwise return the appropriate stale-reference error.
- Scope references to the correct session and browsing/device context, across successive CLI calls and SDK/MCP operations.

---

## 4. Natural-language find: bounded visual grounding

**This is not a second autonomous agent.** Claude, Codex, and other outer-loop agents may already plan, reason, and use screenshots. `find` is a focused, independently optimized operation with a stable contract:

```text
find(description, current screen/context, options)
    -> zero or more located candidate elements
    -> map references + bounding boxes
```

It must not generate a task plan, navigate, click, execute arbitrary tool loops, or call `vibium run` as a shortcut. Local deterministic candidate narrowing and bounded validation/refinement are appropriate.

### Resolution strategy

1. Inspect existing map, DOM, accessibility, labels, roles, and text for reliable candidates.
2. Honor meaningful visual constraints (e.g. color, position, icon appearance); **do not** claim to satisfy “the red Add to Cart button” solely because text says “Add to Cart.”
3. If necessary, capture the relevant screenshot and invoke a configured **vision-capable** multimodal model for localization.
4. Request structured candidate bounding boxes with coordinates and optional role/label hints.
5. Validate output schema, dimensions, coordinate system, bounds, and duplicated/degenerate boxes.
6. Refine/verify approximate boxes when practical, then register established candidates in the map.
7. Return no established match if the target cannot be reliably located. When AI is necessary but unavailable/unsupported, report an explicit capability/configuration error.

### AI configuration

Inspect and **reuse the exact existing nightly `run`/`check` AI settings and precedence rules**. These may include:

```bash
VIBIUM_AI_PROVIDER
VIBIUM_AI_MODEL
VIBIUM_AI_BASE_URL
VIBIUM_AI_REASONING_EFFORT
```

Reuse existing provider-specific API keys, `~/.config/vibium/ai.env` if supported, model adapters, retry/timeouts, and CLI overrides such as `--provider` and `--model`. Do not create a separate model-configuration system in v1.

Examples:

```bash
vibium find "the Add to Cart button"
vibium find "the Add to Cart button" --provider openai --model <vision-capable-model>
```

Validate image input and grounding suitability, not simply that a model can answer text. Ensure screenshot/data transfer follows existing privacy/security practices. Explicit CSS, tag, role, and text matching must **not** send screenshots to remote models. `find(image=..., strategy=auto)` **may** send the screenshot and reference image to the configured VLM after local CV fails. `strategy=cv` never sends images; `strategy=vlm` does so explicitly. Document provider use and potential cost.

### Why independent model selection matters

An outer agent could be Claude Code while Vibium `find` uses a faster/cheaper/more accurate grounding model. Design the inference adapter so it can be benchmarked and swapped independently of the outer agent. **Do not add a `VIBIUM_FIND_MODEL` setting yet** unless existing configuration genuinely cannot provide per-invocation overrides; record this as a possible future enhancement.

### Evaluation harness

Build a reproducible grounding benchmark with real representative browser/mobile screenshots, descriptions, and labeled target boxes. Different models/providers should receive equivalent inputs and produce the same normalized output schema.

Measure:

- Localization accuracy (e.g., IoU and click-point-in-target rate).
- False-positive and miss rates, especially for visually similar/repeated elements.
- Accuracy of single versus multiple candidate localization.
- Median/p95 end-to-end latency and model-call latency.
- Token/inference cost per find and **cost per successful localization**.
- Failure modes: hallucinated boxes, out-of-range coordinates, invalid JSON/schema, and stale observations.

Make models independently configurable for the benchmark. Do not select a supposedly optimal model without recorded measurements. Keep benchmarking outside the hot path and don't require all providers to be available for normal tests.

---

## 5. Reference-image find: local SIFT-first matching

```bash
vibium find image thumbnail.png
vibium find image thumbnail.png --all
```

The user is asking to locate an image that visually corresponds to the supplied reference. Implement **local SIFT feature matching** first in `strategy=auto`, with optional VLM fallback when no verified local match and a configured vision-capable provider is available. **Do not implement template matching**, including as a preliminary fast path. This model fallback is deliberate and documented, not an agentic `run` or `check` loop. `strategy=cv` is strictly local and `strategy=vlm` requires a configured model. Reject or clearly label semantic lookalikes that cannot be verified as corresponding to the reference.

Pipeline:

1. Capture the screen in a defined pixel coordinate system.
2. Decode the reference image and screenshot; validate dimensions/resource limits.
3. Preprocess as useful (grayscale and contrast normalization, retaining appropriate alternatives where they improve reliability).
4. Extract SIFT keypoints/descriptors.
5. Match descriptors using FLANN or brute-force nearest neighbors; benchmark whether FLANN is needed for typical thumbnail sizes.
6. Apply Lowe-style ratio filtering and geometric verification (RANSAC / an appropriate transform).
7. Find **multiple distinct occurrences**, not just a single best homography; consider spatial clustering, repeated inlier removal, or another verified multi-instance method.
8. Reject implausible transformations, weak geometric support, impossible/degenerate boxes, and duplicate overlaps.
9. Translate coordinates into Vibium's action coordinate system; register accepted matches in the existing map.

Handle tiny/flat/low-feature targets by reporting an honest no-match or supported limitation. Do not quietly add brittle template matching as a workaround.

### Packaging/backend decision

Investigate maintained, appropriately licensed SIFT implementations before writing a new one. OpenCV can be used as a benchmark/reference, but Vibium must not require users to install system OpenCV or Python just to invoke `find image`.

Preferred order:

1. Validated pure-Go SIFT and matcher implementation, compatible with the standard binary.
2. Narrow, appropriately licensed native implementation/bundled helper if pure Go is inadequate.
3. Optional OpenCV backend if necessary (never a mandatory base runtime requirement).

FLANN is an implementation choice, not an API requirement. Brute-force descriptor matching may be sufficient for small reference images. Avoid reimplementing all of FLANN without data showing it is beneficial. Preserve cross-platform distribution and make backend selection testable.

### Image-matcher benchmarks

Use representative fixtures: resized thumbnails, brightness/contrast shifts, slightly altered renderings, partial occlusion, multiple copies, similar nonmatches, flat icons, high-DPI screenshots, and no-match cases.

Report false positives, recall, box localization error, median/p95 latency, memory consumption, binary-size overhead, and supported cross-builds. Compare against OpenCV SIFT-based matching as a reference, without making it a production dependency.

---

## 6. API and protocol integration

### JavaScript / TypeScript

Extend the existing options/locator overloads, synchronous and asynchronous APIs, TypeScript declarations, and Element objects. Do not invent a parallel `findImage()` interface. Maintain `find()` returning an Element-compatible object and `findAll()` returning a collection.

### Python

Support matching keyword parameters on sync/async `find` and `find_all`, consistent with existing SDK style. Preserve Element object actions and its no-match contract.

### Java

Preserve the current Java API style (overloads, builders, or typed options), introducing `description`, `image`, `css`, `tag` as appropriate, without inventing a separate visual-find family.

### Image inputs from SDKs

A client-local path is not necessarily available to a remote Go daemon. Accept friendly local paths at SDK boundaries, but transfer decoded/bounded image bytes through an existing or narrowly extended protocol. Do not make arbitrary daemon file reads available to MCP callers. Validate payload sizes and formats.

### MCP

Extend existing find tool(s) with explicit typed locator fields, including `description`, `image`, `css`, `role`, and existing options. MCP callers should not rely on heuristic parsing of ambiguous text. Preserve tool names and existing schemas where possible. Support image content/resource transport consistent with the MCP implementation, structured boxes, map references, single/all modes, and actionable error types. Interactions must resolve the same `@eN` references.

Use one shared Go-side find implementation; SDKs and MCP must not reimplement matching or grounding separately.

---

## 7. Cross-platform architecture and future visual UI tree

The same locator semantics should work regardless of target type; the actual information sources differ:

| Backend | Useful sources |
|---|---|
| Browser | DOM, accessibility tree, screenshot |
| Roadie | Screenshot; future OCR/UI detector |
| Android (future) | UiAutomator/accessibility, screenshot |
| iOS (future) | WebDriverAgent/XCUITest, screenshot |

`role="button"` means a semantic role. `tag="button"` means an actual HTML tag, where supported. `css="button.primary"` means genuine CSS, not a fabricated query against an inferred screen layout.

Design map entries to accept provenance, role, name/text, box, optional confidence evidence, and optional relationships. **Do not build a complete pseudo-DOM now.** In the future, a UI detector can populate a flat semantic map and only later add inferred hierarchy.

### Future OmniParser-style enhancement (out of scope)

Possible later pipeline:

```text
Screenshot
   -> YOLOv9-E-style UI region detector
   -> OCR and Florence-2-style icon captioner
   -> semantic regions / optional hierarchy
   -> existing Vibium element map
   -> find / map / click / tap
```

This may improve repeated visual grounding or generate a map for Roadie without a native accessibility tree, but it is **not a substitute for arbitrary reference-image correspondence**. Do not integrate it in v1, bundle weights, or make it a prerequisite for SIFT/image finding.

---

## 8. Licensing and dependency review

Vibium is Apache-2.0 licensed. Review the **exact versions and distribution artifacts** of any adopted computer-vision or model components:

- Source-code and binary library licenses.
- Transitive dependencies and required notice/attribution files.
- Whether code, model checkpoints, and weights have **different** licenses.
- Redistribution and commercial use terms of weights and any restrictions.
- GPL/AGPL obligations if encountered, including potential network-service implications.
- Compatibility with the Go release/build and distribution model.
- Whether a native helper, optional backend, or hosted model changes licensing obligations.

Do not assume that a permissively licensed inference framework implies permissively licensed pretrained weights. Review exact YOLOv9-E and Florence-2 variants/checkpoints separately **before any future integration**. Avoid casually including AGPL code or unverified weights in Vibium's standard distribution.

Deliver a concise dependency/license report and unresolved issues. Flag any material uncertainty requiring legal review.

---

## 9. Error handling, security, and correctness

Distinguish no match from:

- Missing or invalid image file; unsupported format; resource limit.
- Screenshot capture or coordinate translation failure.
- Missing AI configuration when visual inference is needed.
- Configured model lacking vision capability.
- Provider/inference timeout or invalid model output.
- Unsupported locator on current target.
- Invalid locator-parameter combination.
- Stale element reference or unsupported action.

Do not silently treat these as `[]`. Respect existing error and CLI exit-code conventions. Bound memory, decoded image size, number of features, inference time, and returned candidate count. Do not expose arbitrary server-side file access through MCP. External screenshot transmission must follow the invoked model-provider semantics; local image searches never upload data on their own.

---

## 10. Tests and acceptance

### Required tests

**API migration:** Verify that bare `find("h1")` means description and `find(css="h1")` means CSS in CLI and SDKs. Preserve explicit `role`, `text`, `label`, and composed locators. Do not inadvertently change unrelated APIs such as `wait` without an explicit decision.

**Image matching:** Test `auto` local-first, optional fallback with/without configured provider, `cv` never invoking a model, `vlm` failing clearly without provider, and fixtures for exact/resized/contrast-shifted/partially occluded/multiple/similar-nonmatching/tiny/flat/high-DPI/no-match images; check false positives, box accuracy, deduplication, and deterministic output.

**Map:** DOM and visual entries share the allocator; no overwrite or premature renumbering; session isolation; sequential CLI calls; navigation/scroll/resize stale handling; correct coordinate interaction for `click @eN` and supported `tap @eN`.

**AI grounding:** Deterministic matching avoids unnecessary inference; current Run/Check settings and overrides are respected; unsupported vision models and missing credentials produce clear errors; bounded invocation doesn't take autonomous actions; malformed/invalid boxes are rejected; no unexpected model calls for explicit `cv` image matching; expected and tracked VLM fallback for image `auto` when CV fails.

**SDK/MCP:** Sync/async parity where applicable; return types; correct transport of local image bytes; map reference resolution; structured metadata; no-match/error behavior.

**Build/licensing/performance:** Supported `CGO_ENABLED=0` binaries, no mandatory runtime OpenCV/Python, binary size, benchmark measurements, notices, and license report.

### Acceptance criteria

- Bare string search invokes **description** semantics in CLI and all SDKs.
- Explicit `css`/`tag` and existing semantic locators work as documented.
- `find(image=...)` with default `auto` uses local SIFT first and optional configured VLM fallback, without template matching; it still works without any AI provider. `cv` is guaranteed local, while `vlm` requires configuration.
- Description search reuses existing Run/Check AI provider configuration and per-call overrides.
- The model operation is bounded and independently benchmarkable rather than a second planning loop.
- `find` prints concise `@eN` matches by default and adds them to the existing map.
- `--all` and SDK find-all return multiple results; JSON CLI output is an array with boxes; diagnostics are opt-in.
- Visual references work in relevant click/tap/highlight operations and are safe against stale coordinates.
- JS/TS, Python, Java, and MCP have feature parity within their established API conventions.
- Supported Go builds remain available without mandatory system CV dependencies.
- Tests, migration guide, updated docs/skills, var.parts tutorial, benchmarks, and licensing review are included.

---

## 11. Documentation and migration

Update the `find`, `map`, `click`, and `tap` references, quickstarts, all SDK references, MCP docs/tool descriptions, and agent skills/examples. Search the docs for positional CSS `find("...")` examples and migrate those intentionally.

Create **“Vibium Find: Natural Language by Default”** migration guidance:

```python
# Before: positional CSS
vibe.find("h1")
vibe.find(".submit")
vibe.find("#login")

# After: explicit CSS
vibe.find(css="h1")
vibe.find(css=".submit")
vibe.find(css="#login")

# New: description default
vibe.find("the main heading")
vibe.find("the submit button")
```

CLI equivalents:

```bash
vibium find css h1
vibium find css .submit
vibium find css '#login'
vibium find "the main heading"
```

Explain that the migration is syntactically simple but the new description locator does not guarantee the same match semantics as CSS.

### Tutorial: “Finding Things by Sight with Vibium”

Use https://var.parts for approachable examples. **Inspect the current site and verify all selectors/descriptions rather than assuming that the site has a particular control.** Keep the tutorial short and reproducible:

1. `vibium go https://var.parts`.
2. Locate a visible UI element with a description and read its actual `@eN` reference.
3. `vibium click @eN` on that returned reference.
4. Capture a screenshot and crop/save a sample `thumbnail.png`.
5. `vibium find image thumbnail.png` and show `--json` bounding boxes.
6. Demonstrate multiple matches with `--all`, if the verified page supports it; otherwise use a deterministic local fixture.
7. Contrast description matching with explicit `find css ...`.
8. Provide short JS, Python, and Java equivalents.

Include a local HTML fixture with repeated images for deterministic tests and an alternative tutorial when the external page changes. Use real command output rather than invented examples.

---

## 12. Implementation order and final report

1. Inspect the repo, nightly AI infrastructure, existing locator/map contracts, and release constraints.
2. Identify and record the public breaking change and protocol boundaries.
3. Implement normalized locators and description-first semantics.
4. Extend the map and reference-backed visual interactions.
5. Select/implement and benchmark SIFT-based local matching (no template matching).
6. Add a bounded vision-grounding adapter using existing Run/Check AI settings.
7. Add CLI/JSON behavior and typed MCP support.
8. Update JS/TS, Python, and Java libraries, including image-byte transport.
9. Add tests, deterministic fixtures, benchmark harness, and cross-platform builds.
10. Update docs, agent skills, migration guide, and verified var.parts tutorial.
11. Provide a dependency/weights license review and final implementation report.

Final report should include changed files, public and breaking API changes, matching implementation and tradeoffs, AI adapter/config reuse, tests/benchmarks (accuracy, latency, cost, binary size), license findings, limitations, and future work (including optional YOLOv9-E/Florence-2 screen parsing).

Avoid unrelated refactors or an unnecessarily general computer-vision framework. Deliver the smallest credible implementation of **find by intent, interact by reference**.


---

## 18. Authoritative strategy contract and native tree adapter

This section is part of the implementation requirements, not a future suggestion. It supersedes any earlier statement in this file that implies image finding never uses VLM, or that `strategy` is image-only.

### Public `strategy` option

- `auto` (default): select methods appropriate to the locator, using fast structured/local evidence first and VLM only when needed and configured.
- `cv`: classical non-generative computer vision, initially SIFT + descriptors + geometric verification; requires an image locator.
- `vlm`: vision-language model-based grounding for a description or reference image; requires a configured vision-capable provider.
- `tree`: query a real structured UI hierarchy, from browser DOM/accessibility or Roadie's Android UiAutomator / iOS WDA tree adapter; requires a compatible tree and locator.

Do not use `ai`, `feature`, `local`, or `native` as the new canonical public strategy names. Do not build a large registry of speculative strategy names. `strategy` is available across CLI, JS/TS, Python, Java and MCP, using their existing options conventions.

```bash
vibium find 'the Add to Cart button'                   # auto
vibium find image thumbnail.png                        # auto
vibium find image thumbnail.png --strategy cv
vibium find image thumbnail.png --strategy vlm
vibium find 'the Add to Cart button' --strategy vlm
vibium find role button --strategy tree
vibium find css h1
```

```python
vibe.find('the Add to Cart button')
vibe.find(image='thumbnail.png')
vibe.find(image='thumbnail.png', strategy='cv')
vibe.find(image='thumbnail.png', strategy='vlm')
vibe.find(role='button', strategy='tree')
vibe.find(css='h1')
```

JS/TS analogues use options such as `{image: 'thumbnail.png', strategy: 'cv'}`. Keep `find` single-result and `findAll`/`find_all` multi-result; `--all` controls CLI cardinality.

### Exact fallback and error contract

| Locator | Strategy | Behavior | No configured VLM |
| --- | --- | --- | --- |
| `image` | `auto` | CV first; VLM if no verified CV match | Return CV matches or empty if CV completed, without configuration error |
| `image` | `cv` | SIFT-derived local matching only | Works normally |
| `image` | `vlm` | VLM localization against screenshot and supplied reference | Provider/capability error |
| Description | `auto` | Reliable tree/semantic evidence, then VLM when needed | Return deterministically established match, otherwise actionable capability error |
| Description | `vlm` | VLM screenshot grounding | Provider/capability error |
| Description | `tree` | Query real semantic UI hierarchy only | Works if tree available, otherwise tree capability error |
| Description | `cv` | Invalid combination without reference image | Validation error |
| Explicit CSS | `auto` or `tree` on supported web tree | Real CSS selection only | Works normally |
| Explicit CSS | `cv` or `vlm` | Invalid combination | Validation error |

Other explicit structured locators (role, text, label, tag, etc.) retain their existing precise semantics. `tree` is allowed when those locators are supported by the available actual hierarchy. Unsupported combinations are validation errors, not silent changes of search mode. `auto` must not imply every query triggers a model call.

**Do not conflate no-match with failure.** If CV raises a processing error, report it as an error; don't mask it as an empty list. If an image `auto` CV search completes with no verified match and VLM is unconfigured, return no matches, not missing-AI error. If a description cannot be resolved without a VLM and no provider is configured, return an actionable error.

**Migration diagnostics:** `find('h1')`, `find('.submit')`, `find('#login')`, or `find('button.primary')` always mean descriptions after this change. Do not silently reinterpret them as CSS. If required AI is unavailable and a description resembles a legacy CSS selector, suggest the exact syntax for that surface: Python `find(css='h1')`, JS `{css:'h1'}`, CLI `vibium find css h1`, and the existing Java builder/locator equivalent. CSS-like descriptions can also be warned about proactively if warnings don't corrupt machine-readable stdout.

**Privacy and cost:** `auto` image searches can transmit screenshots plus reference images to the configured provider when CV fails; `auto` description searches can transmit screenshots when semantic/tree matching is insufficient. Document this clearly. `cv`, `css`, and other explicit deterministic locator paths must not invoke a model. Expose selected method, whether VLM was invoked, latency and provider/model in optional diagnostics.

### Roadie native tree integration (in-scope adapter)

Roadie owns hardware provisioning and tree retrieval; Vibium owns query semantics, mapped reference allocation, and element interactions. Coordinate the transport contract with the Roadie agent before writing incompatible endpoints or payloads. The separate `roadie-native-tree-spec.md` is the authority for Roadie implementation details.

Consume Roadie's authenticated, capability-discovered tree snapshot endpoint (illustratively `GET /v1/devices/{deviceId}/tree`, subject to existing Roadie conventions). Capture: platform (`android`/`ios`), provider (`uiautomator`/`wda`), snapshot timestamp or generation, screen dimensions/rotation, coordinate space, parent-child relationships where present, stable-enough node locator hints, role, name/text/label, bounds, and nullable properties (enabled/clickable/etc.). Keep platform-specific attributes as optional metadata. Snapshot-local node IDs are **not** Vibium `@eN` refs.

Normalize real tree nodes into Vibium's existing map. Use the same map allocator and session ownership as DOM and vision results. Preserve native-element actions where available, otherwise use safely validated coordinates. Never fabricate HTML IDs/classes/tags from a mobile tree; preserve provenance and distinguish unavailable tree from valid empty tree. Handle rotation, refresh, stale snapshots and navigation.

When WDA is expired/unavailable or Android wired ADB is disconnected: `strategy=tree` returns a named capability error; `strategy=auto` can fall back to screenshot/VLM search, if configured. Do not require WDA or ADB for basic screenshot-based finding. On Android, Roadie will use the **existing wired USB-gadget network** for ADB TCP after provisioning; on iPhone, Roadie will use free-account WDA provisioning with weekly refresh. Vibium must not require either setup to install or start.

### Additional acceptance tests

- `strategy` schema and validation consistent in CLI, MCP, Python, Java and JS/TS.
- Explicit `cv` performs no remote model calls; `auto` image runs CV before invoking optional VLM.
- Image `auto` without provider returns local results or empty, not a configuration error.
- Description `auto` without provider returns deterministic matches where possible; otherwise a useful provider/migration error.
- Native/browser tree matches yield actionable `@eN` references and bounding boxes; stale snapshots do not trigger blind taps.
- Roadie tree unavailable is distinguishable from an empty tree; explicit `tree` fails clearly and `auto` degrades gracefully.
- The existing `run`/`check` model configuration and override precedence is shared, without inventing `VIBIUM_FIND_MODEL` in v1.
- No template matching, mandatory OpenCV, Python, Xcode, YOLO, Appium, or model weights in the standard Go binary.

**Final output:** Implement code, tests, migration guide, documentation, SDK/MCP examples, and the var.parts tutorial. Report exact files, tests, licensing status, open hardware integration assumptions and any deviations from the agreed shared Roadie tree contract.

