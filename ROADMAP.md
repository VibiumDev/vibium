# Vibium Roadmap

Future features — not yet prioritized. Revisit based on user feedback.

---

## The Full Vision: Sense → Think → Act

Vibium's architecture follows the classic robotics control loop:

| Layer | Component | Purpose |
|-------|-----------|---------|
| **Sense** | Retina | Chrome extension that observes everything |
| **Think** | Cortex | Memory + navigation planning |
| **Act** | Vibium | Browser automation via BiDi |

---

## Cortex — Think Layer

**What:** SQLite-backed datastore that builds an "app map" of the application.

**Why deferred:** Complex infrastructure that may be YAGNI. Agents using Claude Code have conversation context — unclear if persistent navigation graphs add value over just replaying actions.

**Components:**
- SQLite database with schema for pages, actions, sessions
- sqlite-vec integration for embeddings (via CGO or pure Go alternative)
- REST API for data ingestion (JSONL)
- Graph builder and Dijkstra pathfinding
- MCP server with tools: page_info, find_element, find_path, search, history

**When to build:** When users report that agents are:
- Repeatedly rediscovering the same flows
- Losing context across sessions
- Unable to plan multi-step navigation

**Estimated effort:** 2-3 weeks

---

## Retina — Sense Layer

**What:** Chrome extension that passively records all browser activity regardless of what's driving it.

**Why deferred:** Requires Cortex to send data to. Also, MCP screenshot tool may provide enough observability for V1 use cases.

**Components:**
- Chrome Manifest V3 extension
- Content script with click/keypress/navigation listeners
- DOM snapshot capture
- Screenshot capture via background script
- JSONL formatting and Cortex sender
- Popup UI for recording control

**When to build:** When users need to:
- Record human sessions for replay
- Debug what happened during agent runs
- Train models on interaction data

**Estimated effort:** 1-2 weeks

---

## .NET Client

**What:** NuGet package with idiomatic C# API.

**Community implementation:** https://github.com/webdriverbidi-net/vibium-net (by @jimevans)

**Status:** Community project exists. Not officially supported yet, but we hope to include an official .NET client in the future.

**When to build:** When demand warrants official support.

---

## Video Recording

**Status: Shipped** as a recording track: `recording.start({ video: true })` on every surface (JS, Python, Java, CLI, MCP), with the video landing inside the recording zip. Uses the WebDriver BiDi `browsingContext.startScreencast` command, so the browser encodes the video itself and no FFmpeg is needed. Works on Firefox 154+; Chrome has not implemented the BiDi command yet and will work when it does. See `docs/how-to-guides/record-video.md`.

---

## AI-Powered Locators

**What:** Natural language element finding and actions.

```typescript
await vibe.do("click the login button");
await vibe.check("verify the dashboard loaded");
const el = await vibe.find("the blue submit button");
```

**Why deferred:** This is the hardest problem. Requires:
- Vision model integration (which model? where does it run?)
- Latency management (vision calls are slow)
- Cost management (vision calls are expensive)
- Fallback strategies when AI fails

**Open questions:**
- Local model (Qwen-VL) vs API (Claude vision)?
- Screenshot → model → coordinates, or DOM → model → selector?
- How to handle ambiguity ("the button" when there are 5)?
- Caching/memoization of element locations?

**When to build:** After V1, with dedicated research spike. This could be a V2 headline feature or a separate product.

**Estimated effort:** 3-6 weeks (high uncertainty)

---

## Cortex UI

**What:** Web-based visualization of the app map.

**Why deferred:** Depends on Cortex existing. Also unclear if visualization adds value vs just MCP queries.

**Features:**
- Graph view of pages and flows
- Test result display
- Live execution viewer
- Embedded chat for test generation

**Prototype:** https://vibium-cortex.lovable.app/?dataset=view-action-sample

**When to build:** After Cortex, if users struggle to understand app maps via MCP alone.

**Estimated effort:** 2-3 weeks

---

---

## Firefox, Edge, Safari, and Brave Support

**Status: Firefox shipped.** `vibium install --engine firefox`, then `--engine firefox` or `VIBIUM_ENGINE=firefox`. See `docs/how-to-guides/using-firefox.md`.

**What:** Support browsers beyond Chrome and Firefox.

**Why deferred:** Chrome covers 90%+ of use cases. BiDi implementations vary across browsers.

**When to build:** When users explicitly need Edge, Safari, or Brave.

**Estimated effort:** 1 week per browser

---

## Docker & Cloud Deployment

**What:** Official Docker images and Fly.io deployment guides.

**Why deferred:** Local-first is V1 priority. Cloud adds operational complexity.

**Deliverables:**
- Dockerfile.vibium
- docker-compose.yml for full stack
- Fly.io fly.toml and deployment guide
- GPU machine setup for local models

**When to build:** When users want to run agents in CI or production.

**Estimated effort:** 1 week

---

## Priority Order (Tentative)

Based on likely user demand:

1. **More browsers**
2. **Video recording** — Debugging value, moderate effort
5. **Retina** — If recording human sessions matters
6. **Cortex** — If agents need persistent memory
7. **AI locators** — High value but high uncertainty
8. **Cortex UI** — Nice to have

---

## Feedback Channels

After V1 ships, track what users actually ask for:
- GitHub issues
- Discord/community feedback
- Usage analytics (opt-in)
- Direct user interviews

Build what's requested, not what we assume is needed.
