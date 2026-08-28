---
name: vibe-aesthetic
description: Design evaluation mode — walks a page at desktop + mobile, extracts a design-token probe (palette, typography, spacing, archetypal signals), captures section-by-section screenshots, and produces a prompt bundle a host LLM can use to score the page across nine dimensions (visual hierarchy, typography, color, spacing & layout, consistency, accessibility, emotional impact, usability, archetypal coherence). Use when the user has a deployed page and wants to know not just whether it works but whether it lands.
---

# Vibium Aesthetic — design evaluation capture

`vibe-aesthetic` is the **design twin** of `vibe-explore`. Where `vibe-explore` asks *what does this page let me do*, `vibe-aesthetic` asks *what does this page feel like, and where is it weak?*

The skill itself only captures evidence. The evaluation — the actual scoring and findings — is done by the host LLM (Claude, GPT, etc.) reading the artifacts this skill produces against the prompt template shipped at `PROMPT.md`. This split keeps the capture mechanical and reproducible while letting any LLM contribute the judgment.

## When to use this skill

- Pre-launch design review on a deployed site.
- Post-rebrand verification — did the new identity actually land in the rendered page?
- Comparing variants — run twice with the same brief, score the diffs.
- Brand-family coherence check — score sibling sites and see who's lifting / dragging.
- When the team has stopped finding things to fix and the page still "feels off."

Complementary to `vibe-explore` (capability map) and any functional audit (broken-things scan). A page can be functionally clean and aesthetically weak, or vice versa.

## How to run

```bash
skills/vibe-aesthetic/aesthetic.sh <run-dir> <url> [--viewport mobile|desktop|both] [--brief "..."]
```

| Flag | Default | Meaning |
|---|---|---|
| `--viewport mobile\|desktop\|both` | both | Capture viewports. |
| `--quick` | off | Single viewport (desktop), 2 sections only (hero + one mid). ~30s. |
| `--brief "..."` | (none) | One-line brand brief to pass to the evaluator. If absent, the bundle uses `<meta name="description">` + the H1 text as a generated brief. |
| `--brief-from <path>` | (none) | Path to a longer brief file passed alongside `--brief`. |

Example:

```bash
skills/vibe-aesthetic/aesthetic.sh ./run https://example.com/ \
  --brief "Invite-only private aviation atelier"
```

## Pipeline

1. **Resolve binary**: `vibium` on PATH → `./clicker/bin/vibium` → `./node_modules/.bin/vibium`.
2. **Prep daemon**: `prep.sh` enforces `--headless` mode and clears zombie chromedriver / chrome processes (see `Daemon hygiene` below).
3. **Walk per viewport** (`walk.sh`):
   - Set viewport (390×844 dpr 3 mobile, 1280×800 dpr 1 desktop).
   - Navigate + wait for hydration.
   - Capture the design-token probe via `probe.js` → `probes/tokens_<viewport>.json`.
   - Auto-discover section anchors: every `<section id="...">` with non-zero height in DOM order. Fall back to viewport-stepped intervals if no anchors found.
   - Smooth-scroll to each anchor, capture screenshot → `sections/s<NN>_<id>__<viewport>.png`.
   - Always capture top-of-page (`s00_top`) and bottom-of-page (`s99_bottom`).
4. **Render the evaluator prompt**: `aesthetic.sh` interpolates the run's URL, brief, probe JSON, and screenshot filenames into `PROMPT.md`, writing the filled prompt to `<run-dir>/PROMPT.filled.md`.
5. **Hand off** to the host LLM. The skill prints the path; the LLM reads the filled prompt + attaches the screenshots and produces `AESTHETIC.md` in the run dir.

## Daemon hygiene

Zombie `chromedriver` and `chrome-for-testing` processes accumulate from prior failed sessions and block new browser sessions with HTTP 500. When the daemon is started without `--headless` in an SSH or CI context where `DISPLAY` is empty, every capture command produces 0-byte output silently. The `prep.sh` script is the durable fix:

```bash
vibium daemon stop
pkill -9 chromedriver
pkill -9 -f chrome-for-testing
rm -f ~/.cache/vibium/vibium.sock ~/.cache/vibium/vibium.pid
vibium --headless daemon start
```

`aesthetic.sh` calls `prep.sh` automatically. Reach for `prep.sh` standalone whenever any vibium command times out, returns HTTP 500, or produces empty output.

## Run dir layout

```
<run-dir>/
├── RUN.md                    # what ran, params, timing
├── probes/
│   ├── tokens_desktop.json   # palette, typography, spacing, archetypal signals
│   └── tokens_mobile.json
├── sections/
│   ├── s00_top__desktop.png
│   ├── s01_hero__desktop.png
│   ├── s02_<section-id>__desktop.png
│   ├── ...
│   ├── s99_bottom__desktop.png
│   └── (same set per viewport)
├── walk.log                  # per-section landing Y + status
├── PROMPT.filled.md          # the prompt to feed to the host LLM
└── AESTHETIC.md              # the LLM writes this; not produced by the skill itself
```

## The evaluator prompt (`PROMPT.md`)

The shipped prompt asks the host LLM to score the page across nine dimensions:

1. Visual Hierarchy
2. Typography
3. Color
4. Spacing & Layout
5. Consistency
6. Accessibility (visual)
7. Emotional Impact
8. Usability (perceived)
9. Archetypal Coherence

It produces `AESTHETIC.md` with:

- An overall score (1–10)
- Per-dimension scores
- Critical → Low findings, each with section + viewport reference, principle being violated, and a concrete fix in CSS / token / copy form
- A per-section walkthrough
- An archetypal read (what archetype the visual signals vs what the copy claims)
- A stable acceptance bar (defaults: ≥ 8/10 overall, no dimension < 6/10, ≥ 8/10 on archetypal coherence)

The full prompt template is at `skills/vibe-aesthetic/PROMPT.md`. Operators can edit it to retune the rubric (different dimension set, different acceptance bar, different domain register).

## Bulk-extraction guardrail

The skill reads the rendered page and screenshots. It never types, submits, clicks, or extracts data records. The mode is read-only by construction — no auth required for public pages, no destructive surface.

## Files in this skill

| File | Purpose |
|---|---|
| `SKILL.md` | This file. |
| `aesthetic.sh` | Bash runner — orchestrates prep + walk + prompt-render. |
| `walk.sh` | Section-aware walker (desktop + mobile), captures probe + section PNGs. |
| `probe.js` | Design-token probe (palette frequencies, typography, surface tokens, composition counts, meta). |
| `prep.sh` | Daemon hygiene — kills zombies, starts daemon with `--headless`. |
| `PROMPT.md` | The evaluator prompt template fed to the host LLM. |
| `examples/reference-run.md` | Anonymized 8-iteration trajectory (7.4 → 9.5) showing how the rubric scores move over real fixes. |

## See also

- `skills/vibe-check/SKILL.md` — the full Vibium CLI reference. `vibe-aesthetic` builds on it.
- `skills/vibe-recon/SKILL.md` — pre-flight auth-wall mapping (useful if the target site is gated).
- `skills/vibe-explore/SKILL.md` — capability inventory by clicking safe elements.

## Acknowledgements

The dimension rubric draws on the LibreUIUX design + archetypal frameworks. The headless-daemon hygiene lesson and the section-walk pattern come from a real 8-iteration polish loop on a deployed editorial site — that run is anonymized in `examples/reference-run.md` for operators to see what realistic score trajectories look like.
