# AI-Native Methods

> **Status: Draft.** This document describes proposed behavior. It does
> not reflect what is currently implemented in vibium. Do not rely on it
> as documentation of existing functionality.

Vibium's signature differentiators. No Playwright or Selenium equivalent exists.

See also: [ROADMAP.md — AI-Powered Locators](../../ROADMAP.md#ai-powered-locators)

---

## `page.check(claim, options?)` — AI-Powered Verification

```javascript
// Plain English assertions — the thing nobody else can do
await vibe.check('the shopping cart icon shows 0 items')
await vibe.check('user is logged in')
await vibe.check('prices are sorted low to high')
await vibe.check('the form shows a validation error for email')
await vibe.check('dark mode is active')

// With selector hint to narrow scope
await vibe.check('shows 3 results', { near: '.search-results' })
await vibe.check('the total is correct', { near: '#checkout-summary' })

// Structured result
const result = await vibe.check('the dashboard loaded successfully')
// {
//   passed: true,
//   reason: "Dashboard shows welcome message and 3 widget panels",
//   screenshot: Buffer,
//   confidence: 0.95
// }

// Use in test assertions
const { passed } = await vibe.check('no error messages visible')
assert(passed)
```

**Implementation:** screenshot → multimodal LLM → structured response. Optionally augmented with DOM snapshot / a11y tree for precision.

**Options:**
- `near` — CSS selector to constrain the visual/DOM region
- `timeout` — max wait time (default: 5s, retries until claim passes or timeout)
- `screenshot` — include screenshot in result (default: true)
- `model` — override default AI model

---

## `page.do(action, options?)` — AI-Powered Action

```javascript
// Natural language actions when you don't know the exact selectors
await vibe.do('log in with username "admin" and password "secret"')
await vibe.do('add the first item to cart')
await vibe.do('close the cookie consent banner')
await vibe.do('navigate to the settings page')

// With constraints
await vibe.do('fill out the shipping form', {
  data: { name: 'Jane Doe', address: '123 Main St', zip: '60601' }
})

// Structured result
const result = await vibe.do('click the submit button')
// {
//   done: true,
//   steps: ['Found submit button with text "Submit Order"', 'Clicked button'],
//   screenshot: Buffer
// }
```

**Implementation:** screenshot + DOM snapshot → LLM plans actions → executes via Vibium's own API (find, click, fill, etc.) → verifies result.

**The key insight:** `page.do()` uses Vibium's own deterministic API under the hood. It's AI planning, not AI puppeteering. The actions it takes are the same `find()`, `click()`, `fill()` commands a human would write.

---

## Philosophy

Traditional test:
```javascript
await vibe.find('testid=cart-count').text() // "0"
```

Vibium test:
```javascript
await vibe.check('cart is empty')
```

The deterministic API is for when you know what you're looking for. The AI methods are for when you want to describe intent. Both coexist — `page.check()` doesn't replace `el.text()`, it complements it.
