---
name: vibe-screenshot-all
description: Walk every known frontend route of a SPA and capture one PNG per route. No JSON probe, no clicking — pure visual catalog. Use when you need a contact sheet of an app's UI surface for a stakeholder review, design comp comparison, or visual regression baseline.
---

# Vibium Screenshot-All — visual catalog of every route

`vibe-screenshot-all` produces the visual catalog: one screenshot per route, plus a contact-sheet HTML that puts them all on one page with URL labels.

The deliverable answers: **"show me what every page in this app looks like."**

## When to use this skill

- Stakeholder review — "here's every page in our app, look it over."
- Design comp comparison — set side-by-side with mockups.
- Visual regression baseline — snapshot now, re-run after a deploy, diff manually.
- Quick UX audit before a redesign.

## How to run

```bash
skills/vibe-screenshot-all/screenshot-all.sh <run-dir> <url> [--max-routes N] [--auth-required]
```

| Flag | Default | Meaning |
|---|---|---|
| `--max-routes N` | 50 | Cap on routes screenshotted. |
| `--auth-required` | off | Pause for manual login before walking. |

## Pipeline

1. **Resolve binary**.
2. **Daemon start** (idempotent).
3. **Navigate** to the URL.
4. **Auth pause** if `--auth-required`.
5. **Discover routes**: pull the SPA bundle, extract React Router `path:"…"` patterns. Filter out parameterized routes (`/order/:id` etc.) — those need a real ID.
6. **Cap at `--max-routes`**.
7. **Walk and screenshot** — one PNG per route, named `<slug>.png`.
8. **Build `contact-sheet.html`** — a single-page HTML with all screenshots inline, labeled by route + landed-URL.

## Deliverable

```
run/
├── contact-sheet.html       # open in a browser to see all screenshots
├── screens/
│   ├── home.png
│   ├── orders.png
│   └── …
├── route-list.txt           # which routes were attempted
└── walk.log                 # per-route landing URL + status
```

## What this skill does NOT do

- **No DOM probe.** If you need structural data, use `skills/vibe-inventory`.
- **No clicking.** If you need to know what each button does, use `skills/vibe-explore`.
- **No diff.** If you want to compare against a previous run, use `skills/vibe-diff`.
- **No parameterized routes.** Routes with `:id` patterns are skipped — provide concrete URLs another way.

## Files in this skill

| File | Purpose |
|---|---|
| `SKILL.md` | This file. |
| `screenshot-all.sh` | Bash runner. |

## See also

- `skills/vibe-inventory/SKILL.md` — for the structural map (routes + APIs + permissions).
- `skills/vibe-diff/SKILL.md` — for periodic re-runs and drift detection.
