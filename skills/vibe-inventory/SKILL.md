---
name: vibe-inventory
description: Walk every visible route of a SPA, capture per-route status (200 / /error / 404), screenshot each, and grep the JS bundle for backend API endpoints + role/permission strings. Produces an INVENTORY.md feature map. Use when handed a tool you need to understand structurally — what routes exist, what each does, what the backend API surface looks like, what the role model is.
---

# Vibium Inventory — full feature map of a SPA

`vibe-inventory` produces the structural map of a single-page app: every route declared by the bundle, walked end-to-end, classified by per-route status, screenshotted, with a backend API surface extracted from the bundle by grep.

The deliverable answers: **"what does this app do, how is it organized, and what's the backend it talks to?"**

## When to use this skill

- New tool or dashboard — capture routes + API surface in one shot.
- Pre-rebuild scoping — produce an inventory the new app's team can work from.
- Permission audit — surface every role / capability string declared in the bundle.
- Drift detection — periodic snapshots compared via `skills/vibe-diff` to catch new routes/endpoints.

## How to run

```bash
skills/vibe-inventory/inventory.sh <run-dir> <url> [--max-routes N] [--auth-required]
```

| Flag | Default | Meaning |
|---|---|---|
| `--max-routes N` | 30 | Cap on routes walked. Hard limit so runs stay bounded. |
| `--auth-required` | off | Pause for manual login in the headed Chrome window before walking. |

## Pipeline

1. **Resolve binary**.
2. **Daemon start** (idempotent).
3. **Navigate** to the URL.
4. **Auth pause** if `--auth-required` (operator signs in, hits Enter).
5. **Bundle grep**: pull the SPA's main JS bundle, extract React Router `path:"…"` patterns and `/api/*` endpoint references. Also extract permission tokens (`Word:Word:Word` patterns like `Portal:View:Order`) and role names.
6. **Build route candidate list** from bundle paths + sidebar/nav links visible in the DOM.
7. **Walk each route** (capped at `--max-routes`), capturing per-route: landed URL, page title, headings, tables, MUI DataGrids, inputs, buttons, error state, body-text excerpt, screenshot.
8. **Classify**: `accessible` / `gated` (URL ends in `/error`) / `dev-leak` (Page not found from a bundle-declared path).
9. **Write `INVENTORY.md`** in the run dir.

## Bulk-extraction guardrail

For routes that show data tables, capture **first page only** (≤10 rows). Bulk pagination is not in scope; use the app's own export flow for full data dumps.

## Deliverable: `INVENTORY.md`

Required sections:

1. **Header** — URL, time, role/identity if visible.
2. **Stack & external dependencies** — auth, hosting, embedded SDKs (PowerBI, OpenCV, Auth0, etc.) inferred from CSP + bundle.
3. **Auth flow** — provider chain.
4. **Role & permission model** — every `Word:Word:Word` pattern + every role name found in the bundle.
5. **Sidebar / navigation tree** — top-level sections + children.
6. **Backend API surface** — `/api/*` endpoints, grouped by top-level path segment.
7. **Feature modules** — each module with route + key endpoints.
8. **Known issues / gaps** — `/error` redirects, 404s on bundle-declared paths, permission boundaries hit.
9. **Data-model hints** — entity relationships inferred from endpoint names.
10. **Concrete artifacts on disk** — file inventory of the run dir.

## Reference run shape

```
run/
├── INVENTORY.md
├── app.bundle.js                # snapshotted JS bundle for grep
├── api-endpoints.txt            # one /api/* path per line, sorted unique
├── frontend-route-patterns.txt  # one React Router path per line
├── api-domains.txt              # api endpoints grouped by top-level segment with counts
├── walk.log                     # per-route landing URL + status
├── routes/
│   ├── home.json                # JSON probe of the home route
│   ├── home.png                 # screenshot of the home route
│   └── …                        # per-route, slug-named
└── inventory.log
```

## Files in this skill

| File | Purpose |
|---|---|
| `SKILL.md` | This file. |
| `inventory.sh` | Bash runner — orchestrates pipeline. |
| `bundle-grep.sh` | Pulls bundle + extracts routes, endpoints, permission strings. |
| `walk.sh` | Walks a list of routes, JSON probe + screenshot per route. |
| `probe.js` | Standard DOM probe (URL, title, headings, tables, MuiDataGrid, inputs, buttons, body text). |

## See also

- `skills/vibe-recon/SKILL.md` — for the auth-wall + edge map without login.
- `skills/vibe-explore/SKILL.md` — for what-the-page-actually-lets-you-do (clicks elements rather than walking declared routes).
- `skills/vibe-diff/SKILL.md` — for periodic re-inventories to catch drift.
