---
name: vibe-explore
description: Browser exploration mode — clicks every safe interactive element on a page, classifies what each does, and produces a ranked "what you can do here" report. Use when the user lands on an unfamiliar tool, dashboard, or app and wants to know its capabilities without reading docs. Skips destructive actions (delete/submit/pay/sign-out/send) by default.
---

# Vibium Explore — "What can I do on this site?"

`vibe-explore` is the mode for landing on an unfamiliar UI and asking *what does this thing do*. It clicks every safe interactive element on the page in turn, screenshots before and after, classifies the outcome, then produces a ranked capability map.

This is the inverse of `vibe-check`: instead of executing a known plan against a known UI, you let the page tell you what it offers.

## When to use this skill

- An unfamiliar internal tool or SaaS dashboard — "what can I do here?"
- Onboarding to a new app — capture the surface before reading docs.
- Pre-test exploration — find clickable surface before writing test cases.
- UX audit of a competitor or partner.

## Hard safety rules (always applied unless `--include-destructive`)

The mode skips any element whose visible text or `aria-label` matches a destructive pattern:

- destructive: `delete`, `remove`, `drop`, `destroy`, `clear (all|cache|history)`, `reset`, `wipe`, `purge`, `archive`
- auth: `sign out`, `log out`, `disconnect account`, `revoke`
- payments: `submit`, `send`, `pay`, `charge`, `subscribe`, `confirm purchase`, `place order`, `checkout`, `buy`
- comms: `send (email|message|notification|invite)`, `post`, `publish`, `tweet`, `reply (all)?`
- DB: `truncate`, `migrate`, `seed`

Always skipped:

- form submit buttons (`type=submit` or any `<button>` inside a `<form>`)
- `<a target="_blank">` links (would open new tabs)
- external-origin `<a href>` links (would leave the site)
- duplicate clickables (deduped by visible-text + selector)

## How to run

The skill ships a self-contained bash runner. From the repo root:

```bash
skills/vibe-explore/explore.sh <run-dir> <url> [--max N] [--include-destructive] [--auth-required]
```

| Flag | Default | Meaning |
|---|---|---|
| `--max N` | 30 | Cap on clicks. Hard limit so runs stay bounded. |
| `--include-destructive` | off | Lift the destructive-action filter. **Off by default — safety first.** |
| `--auth-required` | off | Pause for manual login in the headed Chrome window before clicking. |

Example:

```bash
skills/vibe-explore/explore.sh ./run https://example.com/dashboard --max 20
```

## Pipeline

1. **Resolve binary**: try `vibium` on PATH, fall back to `./clicker/bin/vibium`, fall back to `./node_modules/.bin/vibium`.
2. **Daemon start** (idempotent — `vibium daemon start`).
3. **Navigate** to the URL.
4. **Auth pause** if `--auth-required` (operator signs in manually, hits Enter to resume).
5. **Dismiss consent banner** — Usercentrics / OneTrust / Cookiebot / generic "Accept all" / shadow-DOM walk / force-hide as last resort.
6. **Capture baseline** state (URL, title, modal count, body-text hash) + `before.png`.
7. **Enumerate** all `button`, `[role=button]`, `[role=tab]`, `[role=menuitem]`, `[role=link]`, `a[href]` (same-origin), `[onclick]`, `.MuiButtonBase-root`. Apply the safety filter.
8. **Click loop** — for each safe element, capture before-state, click, sleep 1.5s, capture after-state, classify outcome, screenshot, reset to baseline.
9. **Write `EXPLORE.md`** in the run dir.

## Outcome taxonomy

| Outcome | Detected by |
|---|---|
| `navigation` | URL changed (same origin) |
| `external` | URL changed (different origin); reset to baseline |
| `modal` | new visible `[role=dialog]` / `.modal` count > before |
| `inline-disclosure` | URL same, body-text-hash changed (accordion/menu opened) |
| `route-error` | URL ends in `/error` or `/404`, or page contains "page not found" / "something went wrong" / "forbidden" / "unauthorized" |
| `noop` | URL same, body-text-hash same — button does nothing observable |
| `click_failed` | element couldn't be clicked (off-screen / detached / obscured) |

## Reset between clicks

| Outcome | Reset |
|---|---|
| `navigation` / `external` / `route-error` / `click_failed` | `vibium go <baseline>` + 2s sleep |
| `modal` | `vibium keys Escape`; if still up, navigate baseline |
| `inline-disclosure` | click the same element again to collapse; if that fails, navigate baseline |
| `noop` | no reset needed |

## Deliverable: `EXPLORE.md`

Required sections:

1. **Header** — URL, time, total clickables found / safe / clicked / skipped.
2. **Capability map** — table of every clicked element with ordinal, label, outcome, leads-to URL, screenshot links.
3. **Skipped (and why)** — table of every filtered element so the operator can decide if any deserve manual review.
4. **What you can do here** — the deliverable section. Ranked, grouped by intent (not by DOM order).
5. **Coverage gaps** — what the page hints at but the explorer couldn't reach without different auth or input.
6. **State pollution** — what side effects this run created (audit logs, GA pageviews, last-viewed lists). The operator should know.

## What this skill does NOT do

- **No form filling.** If a button reveals an input, the report notes "form revealed: <field labels>" and moves on.
- **No drags, scroll-jacking, or hover-only interactions.** Clicks only.
- **No new tabs.** Single-tab discipline; if a click opens a new tab despite the filter, close it and continue from the baseline tab.
- **No automatic re-login** if a session expires mid-run. Bails with a clear message instead.

## Files in this skill

| File | Purpose |
|---|---|
| `SKILL.md` | This file. |
| `explore.sh` | Bash runner — orchestrates the pipeline. |
| `enumerate-clickables.js` | Finds + safety-filters all clickables on the page. Returns JSON. |
| `probe-state.js` | Captures `{ url, title, modal_count, body_text_hash, error_visible }` before/after each click. |
| `dismiss-consent.js` | 4-strategy consent-banner dismissal: visible-button text-match → vendor-specific (Usercentrics / OneTrust / Cookiebot) → shadow-DOM walk → force-hide-CSS as last resort. |

The JS helpers are pure browser scripts — `vibium eval --stdin` runs them. Only `explore.sh` is platform-dependent (bash + jq).

## Safe-by-default examples

```bash
# Public marketing site, no auth, default cap of 30 clicks
skills/vibe-explore/explore.sh ./run https://example.com

# Internal SaaS — pause for manual login
skills/vibe-explore/explore.sh ./run https://app.example.com --auth-required

# Tighter exploration of a specific section
skills/vibe-explore/explore.sh ./run https://example.com/products --max 15

# Lift the safety filter (use with care — can submit forms, send messages, etc.)
skills/vibe-explore/explore.sh ./run https://staging.example.com --include-destructive
```

## See also

- `skills/vibe-check/SKILL.md` — full Vibium CLI reference for executing known plans.
- `docs/tutorials/getting-started-mcp.md` — for AI-agent integration via MCP.
