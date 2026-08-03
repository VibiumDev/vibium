# Test a web component

Web components hide their internals in a shadow root. `document.querySelector` stops at that boundary, so a plain CSS selector cannot see inside one.

```html
<my-card></my-card>
<!-- shadow root: <p>shadow text</p><button id="b">Save</button> -->
```

```bash
vibium find "my-card p"
# Error: element not found: my-card p
```

Use a pierce combinator instead. Full syntax in the [selector reference](../reference/selectors.md).

---

## Reach inside a component

`>>` crosses one shadow boundary:

```bash
vibium find "my-card >> p"
# → @e1 [p] "shadow text"
```

```js
const p = await vibe.find('my-card >> p');
```

Everything works on the result, not just `find`:

```bash
vibium fill  "my-card >> input" "hello"
vibium click "my-card >> button"
vibium text  "my-card >> p"
vibium is visible "my-card >> button"
```

## Reach through nested components

Component libraries nest hosts inside hosts. `>>>` searches any depth below the host, so you do not have to name every level:

```bash
# <my-card> contains <nested-el>, which has its own shadow root
vibium find "my-card >>> #deep"
```

Chain them when you want to be explicit about the path:

```bash
vibium find "app-shell >> side-nav >> button"
```

Mix them when only part of the path is known — `outer >>> mid >> button` searches deeply for `mid`, then takes exactly one hop into it.

## See what a component exposes

`vibium map` traverses shadow roots on its own — no combinator needed. It is the fastest way to find out what a component actually renders:

```bash
vibium go https://example.com/component-page
vibium map
# @e1 [input]
# @e2 [button] "Save"
# @e3 [button] "Deep Button"
```

Scope it to one component with `--selector`, which accepts combinators too:

```bash
vibium map --selector "my-card >> .content"
```

## Don't know the structure?

Ask the page directly, then write the selector:

```bash
vibium eval "document.querySelector('my-card').shadowRoot.innerHTML"
```

---

## When it doesn't work

**`attachShadow({ mode: 'closed' })`** — closed roots are unreachable by design. Nothing in the browser can query them, vibium included. Test through the component's public API (attributes, properties, events) instead.

**The element is there but the click fails** — check `vibium is visible "my-card >> button"` first. If the component renders behind an overlay, that is a real obscured element, not a shadow problem.

**A selector with `>` stopped working** — `>` (CSS child) and `>>` (pierce) are different. `div > p` is still an ordinary child selector; only `>>` and `>>>` hop boundaries.

**Slotted content** — elements passed into a component with `<slot>` stay in the light DOM. Select them normally, without a combinator:

```html
<my-card><span>slotted</span></my-card>
```
```bash
vibium find "my-card span"       # works — not in the shadow root
```

---

## Related

- [Selector reference](../reference/selectors.md) — full syntax
- [Actionability](../explanation/actionability.md) — what is checked before a click
