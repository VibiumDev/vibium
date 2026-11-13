# How find by intent works

`find` turns a description, a reference image, or an exact locator into
mapped elements with `@eN` refs. This page explains how each is resolved,
what was measured, what is licensed how, and what is deliberately left out.
The user-facing guide is [Vibium Find: Natural Language by Default](../how-to-guides/find-by-description-and-image.md).

## One implementation, every surface

The CLI, the MCP server, and the JavaScript, Python, and Java clients all
reach `FindElements` in `clicker/internal/api/find.go`. The clients and the
MCP tools only encode a locator; they never match or ground anything
themselves.

| Surface | Entry | Notes |
| --- | --- | --- |
| Wire (`vibium:page.find`, `vibium:element.find`, and the `findAll` pair) | `handleVibiumFind` | returns `tag`, `text`, `box`, plus `ref`, `role`, `name`, `source`, `method`, and `selector` for a DOM-backed match |
| MCP (`browser_find`, `browser_find_all`) | `Handlers.find` | text lines `@eN label` plus `structuredContent` with boxes |
| CLI (`vibium find`) | the MCP tools through the daemon | `--json` prints `{"ok":true,"result":[...]}`, always an array |

`ParseLocator` reads every form and applies the strategy table, so an
invalid combination fails the same way everywhere. `ExtractElementParams`
and a pre-dispatch step (`ResolveRefParams`) turn `@eN` into the element's
selector, or attach the region for a coordinate-backed ref, before any
command runs.

## Refs

One `RefTable` per browsing context, shared by `find` and `map`. A selector
seen before keeps its ref; a region at the same box keeps its ref; anything
new gets the next number. The CLI drops a page's table when `go`, `back`,
`forward`, or `reload` runs, because the refs named a document that is gone.
The SDK keeps its tables and relies on the page-state check below, so an
element held across a navigation reports itself stale rather than vanishing.

A ref that has no DOM element (a region in a canvas, a node of a phone's UI
tree, a model's box) supports `click`, `dblclick`, `hover`, `tap`,
`highlight`, and `bounds`. Before acting, `VisualElementInfo` compares the
page's URL, scroll, and viewport with what the match was measured against:
a navigation is stale; a scroll or resize re-runs local image matching when
the ref came from an image, and is stale otherwise. A native-tree ref is
re-read from the tree and follows its node, or is stale when the node is
gone.

## Descriptions

1. `parseDescription` drops articles and filler ("labeled", "that says"),
   maps role words to roles ("button", "link", "search box", "image"), and
   marks words the page tree cannot judge: colors, positions, "icon",
   "logo", size words.
2. `candidatesScript` lists visible controls, headings, images, and labelled
   fields with an accessible name approximation (aria-label, labelledby,
   label[for], alt, placeholder, button value, text, title).
3. `matchDescription` accepts an exact name, then whole-phrase containment,
   then the lone element of a named role. Several equal candidates are an
   `AmbiguousError` for a single find and the full list for `findAll`. A
   description with visual words skips this step unless `strategy=tree`,
   which then matches names only and refuses role-only queries.
4. With `auto` and a configured provider, the local pass polls for two
   seconds, then `ai.Ground` sends one screenshot and the request to the
   model with a forced `report_matches` tool call. Boxes are validated
   (finite, inside the image, not degenerate, deduplicated by IoU, inside
   the scope) and `snapToDOM` swaps a box for the element under its center
   when that element covers it.
5. A bare string that reads like a selector never gets this far: it is
   refused before any wait or model call, with the css form for the
   surface. Only the explicit description option bypasses the guard. This
   is the one concession to the old meaning, and it is a refusal, not a
   reinterpretation: an old `find("h1")` fails fast instead of waiting out
   a timeout or paying for a model request.

The grounding call is bounded: one turn, 90 seconds, no tools other than the
report, no history. It reuses `ai.ResolveConfig` with a new `find` role, so
`VIBIUM_AI_*`, the provider keys, `~/.config/vibium/ai.env`, and the
per-call overrides behave exactly as for `run` and `check`. No
`VIBIUM_FIND_MODEL` exists; the per-call `--model` override covers
evaluation. The HTTP adapter reports provider errors by status and field,
never by body, so a text-only model rejecting the image surfaces as "find
needs a vision-capable model".

## Images

`clicker/internal/sift` is an original, standard-library-only SIFT
implementation (Lowe 2004): Gaussian pyramid, DoG extrema, sub-pixel
refinement, orientation histograms, 128-D descriptors, brute-force k-NN with
a ratio test generalised to repeated instances, and iterative RANSAC on a
2-D similarity transform for multiple occurrences. No template matching
exists anywhere in the path. Its README lists every threshold, the measured
contrast-normalisation study, and the fixture results.

`findByImage` captures the viewport, computes the scale between CSS pixels
and image pixels from the viewport size, runs the matcher, converts boxes
back to CSS pixels, and snaps each to the element under it. `auto` retries
locally for three seconds before consulting the model with the screenshot
and the reference; `cv` polls locally until the timeout and never calls a
model; `vlm` goes straight to the model. The reference bytes stay with the
ref so a scroll can re-detect it.

Only the viewport is searched. A below-the-fold target needs a scroll first.

Measured on this macOS VM (12 vCPU), 96×96 reference, from
`SIFT_REPORT=1 go test ./internal/sift/ -run Report`:

| Screen | p50 | p95 | keypoints ref / screen |
| --- | --- | --- | --- |
| 1280×800 | 82–90 ms | 86–106 ms | 122 / 2079 |
| 2560×1600 | 308–337 ms | 329–364 ms | 122 / 7351 |

Fixture results: exact copy IoU 1.00; 0.5× and 1.5× copies found with
scale within 0.2 %; 25 % occlusion found; three copies return exactly three
disjoint boxes; a same-style lookalike with the reference absent returns
nothing; tiny and flat references are `ErrNoFeatures`; a 2× (high-DPI)
scene is found with scale 2.003. End to end through the daemon on the
`/find-images` fixture page, `vibium find image` answers in about 90 ms. On
a real, photo-heavy shop page at 1280×900 (var.parts, twelve product
photos) the same call took 1.9 s: a screen full of texture yields far more
keypoints than a synthetic fixture, and the doubled octave is paid on the
whole viewport.

Binary size: +175 KB on darwin/arm64, +136 KB on linux/amd64 with `-s -w`,
about 1 % of the binary. `CGO_ENABLED=0` builds for all five release targets.

## Devices (Roadie)

A target that runs no script but answers `roadie:screen.getViewport` is a
device. A description there goes to the native UI tree first, fetched over
the `roadie:tree.get` extension command on the same socket:

```
{ "platform": "android"|"ios", "provider": "uiautomator"|"wda", "generation": 7,
  "screen": {"width": 1080, "height": 2400, "rotation": 0},
  "nodes": [{"id": "n1", "parent": "", "role": "button", "name": "Add to Cart",
             "text": "", "bounds": {"x":100,"y":2000,"width":880,"height":120},
             "enabled": true, "clickable": true, "hints": {}}] }
```

This is Vibium's proposed contract. Roadie does not implement it yet; until
it does, `strategy=tree` on a device is a named capability error and `auto`
falls back to the screenshot and the configured model. Nodes are never
turned into HTML; a tree match is a coordinate-backed ref that is re-read
from the tree before every action.

Image matching works on a device today: Roadie's screenshot is the viewport
at scale 1, and the ref is driven by coordinates.

## Benchmarks

`clicker/cmd/siftbench` compares one reference against one screenshot and
prints matches and timings as JSON; `scripts/siftbench/opencv_reference.py`
is the same contract on OpenCV for offline comparison when `cv2` is
installed (it is not, here, and the script is unverified).

`clicker/cmd/findbench` runs a JSONL manifest of labelled cases through
either backend and reports hit rate (IoU ≥ 0.5), center-in-target rate,
false positives, misses, and p50/p95 latency; `ground` takes the same
provider and model overrides as the CLI, so models can be compared on
identical inputs. It reads `~/.config/vibium/ai.env` like the CLI.

`tests/fixtures/findbench/cases.jsonl` holds eleven labelled cases over two
1280×900 screenshots: the repository's `/find-images` fixture page and the
var.parts shop. Truth boxes came from `vibium find css` on the live pages.
Measured once with OpenAI `gpt-5.6-sol` (reasoning effort none):

| | |
| --- | --- |
| Cases | 11 (10 present, 1 absent) |
| Hits, IoU ≥ 0.5 | 9 of 10 |
| Center in target | 10 of 10 |
| False positives | 0 (the absent "Checkout button" came back empty with a reason) |
| Misses | 0 |
| Latency p50 / p95 | 1.2 s / 2.3 s per call |

The one case under 0.5 IoU is the 40×40 cart icon, which the model boxed
loosely (IoU 0.32) but centered correctly. A first run scored a miss on the
battery-pack photo because the screenshot was taken before the shop's images
had loaded; the manifest screenshot was retaken after `wait fn` on
`document.images`. That is also the operational lesson: `find` captures the
viewport as it is. Token cost is not surfaced by the provider adapters; it
has to be read from the provider's usage dashboard.

Reproduce with `go run ./cmd/findbench ground --manifest
../tests/fixtures/findbench/cases.jsonl` from `clicker/`, with a vision-capable
model configured.

## Licensing

| Component | License | Notes |
| --- | --- | --- |
| `clicker/internal/sift` | Apache-2.0 (this repository) | Original implementation from Lowe 2004. No third-party code, no weights. |
| Go standard library (`image`, `image/png`, `image/jpeg`, `math`) | BSD-3-Clause | Already a dependency |
| Vision models | the provider's terms | Called only through the user's own configured provider and key; nothing is bundled |
| OpenCV (reference script only) | Apache-2.0 | Optional, offline, never imported by the binary |

`go.mod` gained no dependency. No OpenCV, Python, CGO, or model weights are
required or shipped. The OmniParser-style pipeline (YOLOv9-E detector,
Florence-2 captioner) is out of scope; its checkpoints carry their own
licenses and would need a separate review before any future integration.

## Deviations from the specification

- The CLI prints `find --json` inside the existing `{"ok": true, "result":
  ...}` envelope, with `result` the array. The repository's `--json`
  contract applies to every command.
- `find` across the CLI keeps its existing miss behavior: a single `find`
  that matches nothing exits non-zero, `--all` prints `[]` (or "No elements
  found"). That is the "existing no-match convention" the specification
  defers to.
- `tag` combined with another semantic locator is accepted by `find`, but
  the SDK element handed back addresses the match by its resolved selector
  rather than re-running a tag-filtered locator.
- The Roadie tree contract above is a proposal; it was not coordinated with
  a Roadie implementation because none exists in this repository.
- Only the viewport is searched by image matching and grounding; a
  full-page search is future work.
- `click`, `fill`, `count`, `wait`, and the other selector-taking commands
  still take bare CSS, as the specification left them. `vibium find "a"`
  and `vibium click "a"` therefore mean different things. That asymmetry
  was not weighed in the specification and needs its own decision: let
  those commands take descriptions too, or keep find's bare string CSS.

## Future work

- Full-page image search with automatic scrolling.
- Surfacing provider token usage so `findbench` can report cost per
  successful localization.
- More labelled screenshots for `findbench ground`, device screenshots
  among them, and measurements across models.
- An optional screen parser (UI detector plus captioner) to populate a flat
  semantic map for devices without a tree, kept out of the standard binary.
