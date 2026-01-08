# Vibium V2 Roadmap

Features deferred from V1 to keep scope tight. Revisit after V1 ships and we have user feedback.

---

## The Full Vision: Sense → Think → Act

Vibium's architecture follows the classic robotics control loop:

| Layer | Component | Purpose |
|-------|-----------|---------|
| **Sense** | Retina | Chrome extension that observes everything |
| **Think** | Cortex | Memory + navigation planning |
| **Act** | Clicker | Browser automation via BiDi |

**V1 ships Act (Clicker).** V2 adds Sense and Think.

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

## Python Client ✅

**Status:** shipped 2025-12-31

```bash
pip install vibium
```

- [getting started (Python)](docs/tutorials/getting-started-python.md)
- [release update](docs/updates/2025-12-31-python-client.md)

---

## Java Client

**What:** Maven/Gradle dependency with idiomatic Java API.

**Why deferred:** Java ecosystem moves slowly. Enterprise users will want stability we can't guarantee in V1.

**API:**
```java
import com.vibium.Browser;
import com.vibium.Vibe;

Vibe vibe = Browser.launch();
vibe.go("https://example.com");
var el = vibe.find("a");
el.click();
vibe.quit();
```

**When to build:** When enterprise users request it, likely after V1 is proven stable.

**Estimated effort:** 1-2 weeks

---

## Video Recording ✅

**Status:** Implemented

Record browser sessions as MP4 or WebM video files. Requires FFmpeg to be installed.

**CLI:**
```bash
clicker record https://example.com -o recording.mp4 --duration 10 --fps 10
```

**JavaScript API:**
```typescript
await vibe.startRecording({ fps: 10, format: 'mp4' });
// ... perform actions ...
const videoPath = await vibe.stopRecording();
```

**Python API:**
```python
await vibe.start_recording(fps=10, format='mp4')
# ... perform actions ...
video_path = await vibe.stop_recording()
```

**BiDi Extension Commands:**
- `vibium:startRecording` - Start recording with options (fps, format, outputPath)
- `vibium:stopRecording` - Stop recording and return video file path

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

## Network Tracing

**What:** Capture and inspect network requests/responses.

**Why deferred:** BiDi network module is complex. Most agent use cases don't need request inspection.

**Features:**
- Enable/disable network capture
- Log all requests/responses
- HAR export
- Request interception (mock responses)

**When to build:** When users need to debug API calls or mock backends.

**Estimated effort:** 1-2 weeks

---

## Firefox & Edge Support

**What:** Support browsers beyond Chrome.

**Why deferred:** Chrome covers 90%+ of use cases. BiDi implementations vary across browsers.

**When to build:** When users explicitly need Firefox (privacy testing) or Edge (enterprise).

**Estimated effort:** 1 week per browser

---

## Docker & Cloud Deployment

**What:** Official Docker images and Fly.io deployment guides.

**Why deferred:** Local-first is V1 priority. Cloud adds operational complexity.

**Deliverables:**
- Dockerfile.clicker
- docker-compose.yml for full stack
- Fly.io fly.toml and deployment guide
- GPU machine setup for local models

**When to build:** When users want to run agents in CI or production.

**Estimated effort:** 1 week

---

## Priority Order (Tentative)

Based on likely user demand:

1. ~~**Python client**~~ ✅ shipped
2. ~~**Video recording**~~ ✅ shipped
3. **Network tracing** — DevTools parity
4. **Cortex** — If agents need persistent memory
5. **Retina** — If recording human sessions matters
6. **AI locators** — High value but high uncertainty
7. **Java client** — Enterprise demand
8. **Cortex UI** — Nice to have
9. **Firefox/Edge** — Edge cases

---

## Feedback Channels

After V1 ships, track what users actually ask for:
- GitHub issues
- Discord/community feedback
- Usage analytics (opt-in)
- Direct user interviews

Build what's requested, not what we assume is needed.
