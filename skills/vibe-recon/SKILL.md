---
name: vibe-recon
description: Map a web app's auth wall and edge surface without logging in. Captures HTTP headers, follows redirects to identity providers (Auth0, Microsoft Entra, Cloudflare Access, etc.), saves the login DOM, screenshots the landing page, and pulls any SPA bundle for grep-based route discovery. Use when handed an unfamiliar URL and asked "what is this and how does auth work?"
---

# Vibium Recon — auth-wall + edge mapping (no login)

`vibe-recon` is the read-only "look but don't touch" mode. Hand it a URL; it returns a one-page map of the app's auth provider chain, edge headers, and any SPA route patterns that grep can extract from the JS bundle. **No login required.**

## When to use this skill

- New to a SaaS / internal tool — what's it built on, how does sign-in work?
- Pre-pentest scoping — capture the auth chain before deciding whether to test inside or outside.
- Pre-`vibe-inventory` — confirm there's something to inventory before logging in.
- Compliance / security audit — document the auth provider, third-party scripts, CSP.

## How to run

```bash
skills/vibe-recon/recon.sh <run-dir> <url>
```

Example:

```bash
skills/vibe-recon/recon.sh ./run https://example.com/
```

## Pipeline

1. **Resolve binary**: `vibium` on PATH → `./clicker/bin/vibium` → `./node_modules/.bin/vibium`.
2. **Edge curl**: `curl -sSI -L <url>` — capture redirect chain, server header, CSP, security headers.
3. **Daemon start** (idempotent).
4. **Navigate** — follow whatever redirects the browser would. Record final URL + title.
5. **DOM probe**: capture form fields on the landing page (login form? marketing splash? error page?).
6. **Bundle grep**: pull the largest `<script src="…">` from `<head>`. If it's a Vite/CRA SPA bundle, extract React Router `path:"…"` strings and `/api/*` endpoint references.
7. **Screenshot**: `01-landing.png`.
8. **Write `RECON.md`** in the run dir.

## Deliverable: `RECON.md`

Required sections:

1. **Header** — URL, time, redirect chain length, final landed URL.
2. **Edge** — server header, security headers (HSTS, CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy), CDN, any unusual headers.
3. **Auth flow** — provider detected (Auth0 / Microsoft Entra / Cloudflare Access / Okta / Google / custom), client_id / tenant if visible in the redirect URL, OAuth scopes if visible.
4. **Stack inferred** — from the bundle URL pattern, CSP origins, embedded SDK references (PowerBI, Auth0, OpenCV, Stripe, etc.).
5. **DOM at landing** — form fields, buttons, links visible without auth.
6. **Bundle grep** — frontend route patterns + `/api/*` endpoints if found, or "N/A — not a SPA bundle".
7. **What this means / next steps** — what the operator should do if they want to go deeper.

## What this skill does NOT do

- **No login attempts.** The whole point is to map the wall, not pass it.
- **No clicking** beyond the initial navigation. Every observation comes from passive load.
- **No data extraction.** The tool is for surface mapping; if you need data, that's a separate skill.

## Files in this skill

| File | Purpose |
|---|---|
| `SKILL.md` | This file. |
| `recon.sh` | Bash runner — orchestrates curl + navigate + probe + bundle-grep + screenshot. |
| `bundle-grep.sh` | Pulls the SPA's main JS bundle and extracts route + API patterns. |
| `probe.js` | Standard DOM probe (URL, title, forms, inputs, buttons, error state). |

## See also

- `skills/vibe-check/SKILL.md` — full Vibium CLI reference for executing known plans.
- `skills/vibe-inventory/SKILL.md` — once past the auth wall, walk every route the SPA declares.
- `skills/vibe-explore/SKILL.md` — once past the auth wall, click every safe element.
