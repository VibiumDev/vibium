---
name: vibe-diff
description: Compare two `vibe-inventory` snapshots and surface what changed — new routes, removed routes, new API endpoints, version bumps, permission tokens added or dropped. Use when periodically re-inventorying an app to catch drift, especially during a rebuild where you want to mirror upstream changes.
---

# Vibium Diff — what changed between two inventory snapshots

`vibe-diff` takes two `skills/vibe-inventory` run directories and produces a one-page diff: routes added, routes removed, API endpoints added, API endpoints removed, version-string change, permission-token delta.

The deliverable answers: **"what changed in this app since the last time I looked?"**

## When to use this skill

- Rebuilding a tool — re-run weekly during the rebuild to catch new functionality the original team ships in the meantime.
- Audit cadence — monthly snapshot to track what a vendor is shipping.
- Pre-deploy verification — diff a staging snapshot against production to confirm the change set.

## How to run

```bash
skills/vibe-diff/diff.sh <old-run-dir> <new-run-dir> [--out DIFF.md]
```

Example:

```bash
skills/vibe-inventory/inventory.sh ./run-2026-04-15 https://example.com/
# … two weeks pass …
skills/vibe-inventory/inventory.sh ./run-2026-04-30 https://example.com/
skills/vibe-diff/diff.sh ./run-2026-04-15 ./run-2026-04-30
```

## Pipeline

1. **Read both run dirs**: each must have at minimum `api-endpoints.txt` and `frontend-route-patterns.txt` produced by `skills/vibe-inventory`.
2. **Compute deltas** with `comm` / `diff`:
   - frontend routes added / removed
   - api endpoints added / removed (grouped by top-level path segment)
   - permissions added / removed (from `permissions.txt`)
3. **Detect version bump** — grep both `app.bundle.js` snapshots for `"v\d+\.\d+\.\d+"` and report any change.
4. **Bundle size delta** — bytes diff between `app.bundle.js` files.
5. **Write `DIFF.md`** in the new run dir (or to `--out`).

## Deliverable: `DIFF.md`

Required sections:

1. **Summary line** — `+12 endpoints, -2 routes, version 1.1.334 → 1.2.0, bundle +120 KB`.
2. **New endpoints** — table by API domain.
3. **Removed endpoints** — table by API domain.
4. **New routes** — list.
5. **Removed routes** — list.
6. **Permission tokens added / removed**.
7. **Version + bundle-size delta**.
8. **Triage notes** — operator-actionable notes ("the new `/api/Order/Cancel` suggests a feature shipped — check if it's user-facing").

## Files in this skill

| File | Purpose |
|---|---|
| `SKILL.md` | This file. |
| `diff.sh` | Bash runner — pure file-based diff, no browser, no daemon. |

## See also

- `skills/vibe-inventory/SKILL.md` — produces the inputs `vibe-diff` consumes.
- `skills/vibe-recon/SKILL.md` — for the auth-wall + edge map.
