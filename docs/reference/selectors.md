# Selectors

How vibium turns a selector into an element. Three kinds: CSS, pierce combinators for shadow DOM, and semantic locators.

For which commands accept a selector, see [API Reference](api.md).

---

## CSS

Any CSS selector the browser accepts.

```bash
vibium find "button.primary"
vibium find "#login input[type=password]"
vibium click "nav a:nth-child(2)"
```

```js
await vibe.find('button.primary');
```

CSS selectors do not cross shadow boundaries. Use a pierce combinator for that.

---

## Pierce combinators

Shadow roots are invisible to `querySelector`, so a component's internals need an explicit hop.

| Combinator | Meaning |
|---|---|
| `>>` | Cross **one** shadow boundary — the host's own shadow root |
| `>>>` | Cross **any depth** of nested shadow roots below the host |

```bash
# <my-card> has a shadow root containing <p>shadow text</p>
vibium find "my-card >> p"

# <my-card> contains <nested-el>, which has its own shadow root
vibium find "my-card >>> #deep"
```

```js
const p   = await vibe.find('my-card >> p');
const btn = await vibe.find('outer-host >>> button');
const all = await vibe.findAll('my-card >> li');
```

The syntax matches Playwright (`>>`) and Selenium 4 (`>>>`).

### Chaining

Each combinator hops from the elements matched so far, so they can be chained:

```
app-shell >> side-nav >> button
```

Combinators may be mixed — `outer >>> mid >> button` searches deeply for `mid`, then takes exactly one hop into it.

### Where they work

Anywhere a selector is accepted — `find`, `findAll`, `click`, `fill`, `text`, `is`, and `map --selector`.

```bash
vibium fill "my-form >> #email" "a@b.com"
vibium click "my-form >> button"
vibium text "my-card >> p"
vibium map --selector "my-card >> .content"
```

### Notes

- A selector with no `>>` is passed straight to `querySelectorAll`, so ordinary CSS is unaffected.
- Only **open** shadow roots are reachable. `attachShadow({ mode: 'closed' })` hides its contents from any script, vibium included.
- Whitespace is optional: `my-card>>p` and `my-card >> p` are the same.
- `>` (CSS child) and `>>` (pierce) are distinct — `div > p` stays a normal CSS child selector.
- `vibium map` traverses shadow roots automatically; you do not need a combinator to see component internals in its output.

---

## Semantic locators

Find by what an element *is* rather than where it sits. Available as `find` subcommands on the CLI and as an options object in the clients.

| Locator | Matches |
|---|---|
| `role` | ARIA role, explicit or implicit (`<button>` → `button`) |
| `text` | Visible text content, substring match |
| `label` | Accessible name — `aria-label`, `aria-labelledby`, `<label for>`, or an input's `value` for submit/reset/button |
| `placeholder` | `placeholder` attribute |
| `alt` | `alt` attribute |
| `title` | `title` attribute |
| `testid` | `data-testid` attribute |
| `xpath` | XPath expression |

```bash
vibium find role button
vibium find role button --name "Log in"
vibium find text "Sign In"
vibium find testid "submit-btn"
```

```js
await vibe.find({ role: 'button', label: 'Log in' });
await vibe.find({ testid: 'submit-btn' });
```

Locators combine — `{ role: 'button', text: 'Save' }` requires both. When several elements match, the one with the shortest text wins, so an exact match beats a container that happens to include it.

---

## Scope

`--scope` (CLI) and `scope` (clients) restrict a search to a subtree. It accepts pierce combinators too.

```bash
vibium find "a" --scope "#sidebar"
vibium map --scope "my-card >> .content"
```

Element objects scope automatically — `element.find(...)` searches within that element.
