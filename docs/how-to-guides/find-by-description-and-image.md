# Vibium Find: Natural Language by Default

`find` locates an element by what it is, not only by how the page is built.
A bare string is a description. Exact locators are explicit. A reference
image is matched where it appears on screen. Every match becomes an `@eN`
ref that `click`, `tap`, `highlight`, and the SDK element methods accept.

**Find by intent. Interact by reference.**

## The breaking change

A bare string used to be a CSS selector. It is now a description.

```python
# Before: positional CSS
vibe.find("h1")
vibe.find(".submit")
vibe.find("#login")

# After: explicit CSS
vibe.find(css="h1")
vibe.find(css=".submit")
vibe.find(css="#login")

# New: description by default
vibe.find("the main heading")
vibe.find("the submit button")
```

```javascript
// Before                      // After
vibe.find('h1')                vibe.find({ css: 'h1' })
vibe.find('.submit')           vibe.find({ css: '.submit' })
vibe.findAll('.row')           vibe.findAll({ css: '.row' })
                               vibe.find('the submit button')
```

```java
// Before                      // After
page.find("h1")                page.find(new SelectorOptions().css("h1"))
page.findAll(".row")           page.findAll(new SelectorOptions().css(".row"))
                               page.find("the submit button")
```

```sh
# Before                       # After
vibium find "h1"               vibium find css h1
vibium find ".submit"          vibium find css .submit
vibium find "#login"           vibium find css '#login'
                               vibium find "the main heading"
```

The migration is a spelling change. The meaning is not the same: a
description is resolved against the page's names and roles, and when they
are not enough, against a screenshot.

A bare string that reads like a selector (`h1`, `.submit`, `#login`,
`nav > a`) is refused at once, before any wait and before any model call,
with the css form for your surface. It is never reinterpreted as CSS, and it
never costs a model request. An element genuinely named "h1" is reachable
with the explicit option: `vibium find description "h1"`,
`find({ description: 'h1' })`, `find(description="h1")`. The MCP tool's
`selector` field and the SDK `selector` key keep meaning CSS.

Only `find` changed. `click "<selector>"`, `fill`, `wait`, `count`, and the
other selector-taking commands still take CSS.

## Locators

| Form | CLI | JavaScript | Python | Java |
| --- | --- | --- | --- | --- |
| Description | `find "the Add to Cart button"` | `find('the Add to Cart button')` | `find("the Add to Cart button")` | `find("the Add to Cart button")` |
| CSS | `find css "button.primary"` | `find({ css: 'button.primary' })` | `find(css="button.primary")` | `new SelectorOptions().css("button.primary")` |
| Tag | `find tag h1` | `find({ tag: 'h1' })` | `find(tag="h1")` | `.tag("h1")` |
| Role + text | `find role button --name "Add to Cart"` | `find({ role: 'button', text: 'Add to Cart' })` | `find(role="button", text="Add to Cart")` | `.role("button").text("Add to Cart")` |
| Accessible name | `find role link --name "More"` | `find({ role: 'link', name: 'More' })` | `find(role="link", name="More")` | `.role("link").name("More")` |
| Image | `find image thumbnail.png` | `find({ image: 'thumbnail.png' })` | `find(image="thumbnail.png")` | `.image(Path.of("thumbnail.png"))` |

`text`, `label`, `placeholder`, `testid`, `xpath`, `alt`, and `title` keep
their meanings and still compose with `role` and `css`. `name` filters by the
accessible name a person reads the element by (text, label, alt, or
aria-label); it is what `find role ... --name` has always meant. A description and an
image each stand alone; combining either with another locator is an error.

`findAll`, `find_all`, and `--all` take the same forms and return every
match. JSON output from the CLI is always an array, empty when nothing
matched:

```sh
$ vibium find image thumbnail.png --all --json
{"ok":true,"result":[{"ref":"@e5","role":"image","tag":"img","box":{"x":20,"y":112,"width":96,"height":96},"source":"image",...}]}
```

## How a description is resolved

1. The page's names and roles are read: controls, headings, images, and
   labelled fields. An exact name wins; a single element whose name contains
   the phrase, or the lone element of a named role, is accepted. Words the
   tree cannot judge, such as colors, positions, and "icon", skip this step.
2. If that is not enough and an AI provider is configured, the screenshot
   goes to a vision model, which answers with bounding boxes. Boxes are
   checked against the image, snapped to the element under them when one
   covers them, and registered as refs.
3. With no provider, the miss is reported with what would have happened
   next, and with the css form when the description reads like a selector.

The model call is one bounded turn. It receives one screenshot and the
request, reports boxes, and never acts.

## How an image is resolved

1. The viewport is captured and the reference is matched with SIFT features
   and a geometric check, locally, in the binary. Every verified occurrence
   is returned, deduplicated, with its box. Nothing leaves the machine.
2. When nothing verifies and a provider is configured, the screenshot and
   the reference go to the model.
3. Without a provider, no match is an empty list (or a miss for the
   single form), never a configuration error.

Only what is on screen is searched. Scroll first when the target is below
the fold. A flat or tiny reference that yields too few features is reported
as such; use a larger or more detailed crop.

## Strategy

`--strategy` (`strategy=` in the SDKs) chooses the method:

| Value | Description locator | Image locator | css, tag, role, ... |
| --- | --- | --- | --- |
| `auto` (default) | tree first, then model if configured | local first, then model if configured | exact |
| `cv` | error | local only; never calls a model | error |
| `vlm` | model only; needs a provider | model only; needs a provider | error |
| `tree` | page tree only | error | exact |

## Privacy and cost

`auto` can send a screenshot to the configured provider: for a description,
when the tree cannot answer; for an image, when local matching finds
nothing. The reference image goes with it. Each call is billed by the
provider like a `vibium check` turn. `cv`, `css`, `tag`, and the semantic
locators never call a model. `--details` shows which method answered and
whether a model was called.

## AI settings

Description and image fallback use the same settings as `vibium run` and
`vibium check`: `VIBIUM_AI_PROVIDER`, `VIBIUM_AI_MODEL`,
`VIBIUM_AI_BASE_URL`, `VIBIUM_AI_REASONING_EFFORT`, the provider's API key,
and `~/.config/vibium/ai.env`. The model must accept images. Override them
for one call with `--provider`, `--model`, `--ai-base-url`, and
`--reasoning-effort` on the CLI, `provider`/`model`/`aiBaseURL`/
`reasoningEffort` in the SDK options, or the same fields on the MCP tools.
There is no separate find model setting.

## Refs

`find` and `map` share one table per page. The same element keeps its ref
across calls; new matches take the next number. A match with no element of
its own, such as a region inside a canvas or a phone screen, is driven by
coordinates: `click`, `dblclick`, `hover`, `tap`, `highlight`, and `bounds`
work; `fill`, `type`, and the other DOM commands refuse it. Before a
coordinate click the page is checked: a navigation makes the ref stale, and
a scroll or resize re-detects an image match or, for a model match, reports
the ref stale. Run `find` again.

## Diagnostics

`--details` adds the method (`css`, `semantic`, `tree`, `sift`, `vlm`),
whether a model was called, the provider and model, elapsed time, and the
raw scores. A SIFT inlier count, a detector score, and a model's confidence
are different quantities. None is a probability.
