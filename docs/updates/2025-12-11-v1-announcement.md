# v1 announcement

it's time to actually ship.

starting back in september, i did a few podcasts and interviews about vibium. the response was overwhelming. 9000+ new linkedin connections, 1300+ completed surveys, 1000+ mailing list, 20+ user "listening session" video calls, a sold-out testguild irl in chicago, blog posts, medium articles, reddit threads, a vibium subreddit (!), youtube videos, tiktoks (!), "vibium conf", "vibium certified". some excited, some skeptical, some rightfully calling out that there wasn't actual software to use yet.

fair. i heard you.

**here's what's happening: vibium v1 ships by christmas.**

not a demo site. real software you can `npm install` and use.

---

## what's in v1

- **clicker**: a go binary that launches chrome, speaks webdriver bidi, and exposes an mcp server. one binary, ~10mb, handles everything.

- **js/ts client**: `npm install vibium`. async and sync apis. playwright-level dx.

- **mcp server**: so claude code (or any mcp-compatible agent) can drive a browser out of the box.

that's it. that's v1. browser automation without the drama.

---

## why vibium

there are dozens of "ai-powered browser" tools now. so why this one?

the selenium ecosystem is massive: millions of tests, thousands of companies, decades of investment. but there's no obvious bridge to the ai future. many have moved to playwright — and for good reason: it's fast, easy to use, has popular features like auto-waiting, integrated video recording, and a ton of other batteries included.

vibium takes the same approach. batteries included. great dx. but built for where the industry is going: ai agents that need to drive browsers.

when i did those interviews in september, the response wasn't just "cool idea." it was relief. the community trusts us to build this bridge because we built the last two: selenium in 2004, appium in 2012.

community and ecosystem are the moat.

---

## what's not in v1 (but is planned)

vibium's full vision follows the classic robotics loop: sense → think → act.

- **retina** (sense): chrome extension that records everything happening in the browser
- **cortex** (think): memory layer that builds a map of the app and plans navigation
- **clicker** (act): the part that actually drives the browser ← this is v1

also deferred:
- python / java clients
- ai-powered locators: `vibe.do("click login")`, `vibe.check()` (i still want a t-shirt with "vibe.check()" on it, though. high priority.)
- video recording

i wrote a v2 roadmap. but v2 comes after v1 ships and we learn what people actually need.

---

## why the scope cut

in september, i described a big vision: sense-think-act for ai agents. a browser extension that watches everything. a memory layer that builds app maps. navigation graphs. embeddings. even a globally distributed network of idle devices. it is... a lot.

that vision is real. but i've been building software long enough to know: ship the simplest thing that could possibly work.

v1 is act: llm → mcp → browser.

sense and think can wait until act works.

---

thank you to everyone who's reached out the past few months, who wrote about vibium, who were patient while i figured out what to actually build first.

shipping by christmas. let's go.

✨🎅🎄🎁✨

\- 🤗 hugs

#vibium

---

*december 11, 2025*
