# Reference run — 8-iteration polish loop on an editorial site

This is an anonymized trajectory from the first real-world run of the
aesthetic capture pattern, distilled to show operators (a) realistic score
movement under code-only fixes, (b) where the photography ceiling lands,
and (c) the durable lessons crystallized into this skill.

The site under audit: a deployed Vercel landing page for an invitation-only
service in the luxury / editorial register. Stack was Next.js + Tailwind +
custom design tokens. The brief was specific (one-line invitation-only
positioning) and the page had real founder copy already in place.

## Score trajectory

| Iter | Score | Δ | Notable changes |
|------|-------|---|-----------------|
| 0 (baseline) | 7.4 | — | First audit, before any iteration |
| 1 | 7.7 | +0.3 | Copy refactor, nav, favicon, new section added |
| 2 | 8.1 | +0.4 | H1 line-height fix, accent on hero eyebrows, warm overlay, footer vocabulary, og:image, placeholder contrast |
| 3 | 8.5 | +0.4 | Section chapter overlays, accent on all eyebrows, card surface removed, personal-voice asides |
| 4 | 8.7 | +0.2 | Blend-mode color, items-start chapter alignment, footer eyebrows brass |
| 5 | 8.7 | +0.0 | Grain texture, hero fade, mobile image cap — **code ceiling confirmed** |
| 6 | 9.2 | +0.5 | **Photography** wired in (real imagery, color-graded) |
| 7 | 9.4 | +0.2 | Substitute photo replaced with authentic source |
| 8 | 9.5 | +0.1 | Photo re-grade for palette harmony, mobile crop fix |

**Total improvement: +2.1 from baseline.**

## What moved the score and by how much

### Code-only ceiling at 8.7 (iters 1–5)

Five iterations of pure code work — typography token fixes, accent palette
expansion from ~5% to ~12% of rendered surface, footer copy vocabulary,
mobile button consistency, surface removal, og:image, contrast bumps,
texture additions — collectively moved the score from 7.4 to 8.7 and then
plateaued. The synthesis engine correctly identified at iter 5 that further
code work was diminishing returns.

**Takeaway**: code can carry a page about 1.3 points off a 7.4 baseline.
Beyond that, the gating dependency is usually photography, real
illustration, or videography. Don't keep prescribing micro-CSS fixes once
the plateau is identified.

### Photography breakthrough at 9.2 (iter 6)

A single iteration with real imagery jumped the score 0.5 points. The
mechanism: emotional impact rose from 8.5 to 9.5, archetypal coherence
from 7.5 to 8.5, and the visual field of the hero section transitioned
from 60% unoccupied near-black to a fully-occupied editorial photograph
that signaled operator legitimacy.

**Takeaway**: if the page can carry a hero image, ship it before iterating
further on color or spacing. The image moves dimensions code cannot.

### Diminishing returns at 9.5 (iters 7–8)

Substitute-photo replacement (+0.2) and photo color regrade (+0.1) closed
the remaining gap to ~9.5. The agent reported the realistic ceiling
against the nine-dimension rubric is approximately 9.8 — 10/10 is
structurally unreachable because the rubric always reserves at least one
fractional finding for "what could be better."

## Durable lessons crystallized into this skill

### 1. Daemon must run headless under SSH

The first eight runs all silently produced 0-byte screenshot files because
the vibium daemon was started without `--headless` in an SSH session where
`DISPLAY` was empty. Browser sessions failed with HTTP 500 and every
capture helper logged success while writing nothing.

**Fix**: `prep.sh` enforces `--headless` at daemon start. Reach for it
whenever a capture command times out, returns HTTP 500, or produces empty
output.

### 2. Accent-color percentage is the single most movable archetypal lever

For sites in the warm-luxury register (Sage / Ruler crossover), the
percentage of rendered surface in the accent color is the lever that
shifts the archetypal read. Moving warm accent from 5–6% to 12–15% of
rendered pixels shifted the read from Ruler to Sage by approximately one
full archetype unit, with no other change.

**How to apply**: read `palette.top` from the probe. Sum the rendered
percentages of brass / warm / accent colors. If the design system says
15% target and the rendered ratio is 5%, the finding is "accent
under-applied at render" with a specific fix (which eyebrow, divider,
hover state should switch to accent).

### 3. Photography is the consistent ceiling

Across multiple iterations, the agent repeatedly flagged placeholder
imagery as the dominant aesthetic gap. Without photography in motion, the
loop hit a hard ceiling around 8.0–8.2 for any site that needs imagery.

**How to apply**: when the agent's iter-N report says "the remaining gap
is photography-dependent," believe it. Either source the photography or
accept the ceiling. Don't run another iteration expecting code fixes to
move the score.

### 4. Plateau detection matters

Iter 5 returned the same score as iter 4 (8.7 → 8.7). Without plateau
detection, the operator would have run more code-only iterations chasing
diminishing returns. The right move is to name the plateau, identify the
gating dependency, and either resolve it (photography) or stop.

**How to apply**: if two consecutive iterations move the score by less
than 0.1, stop iterating code and either ship at the current score or
unblock the non-code dependency.

## Pre-launch blockers (not aesthetic)

A reminder, not a finding: aesthetic score is orthogonal to
functional readiness. The reference run identified several launch blockers
that did not affect score:

- Placeholder form action that would silently fail in production
- A button not wired to its handler
- A passphrase hard-coded in client JS
- Robots blocked
- Domain decision still open

None of these moved the aesthetic score but all had to be resolved before
public flip. Keep an `audit` pass separate from `aesthetic` for this
reason.

## What this skill produces vs what it doesn't

This skill captures evidence and renders an evaluator prompt. The host LLM
produces the scoring, findings, and per-section walkthrough. Operators
opting for an autonomous loop (audit → fix → deploy → re-audit) build that
on top of this capture as a separate workflow — the loop is opinionated
about deploy commands and file editing in a way that doesn't generalize
cleanly upstream, so it is intentionally not part of this skill.
