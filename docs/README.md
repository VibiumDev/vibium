# Docs

Four kinds of docs live here, sorted by what the reader is doing:

- **tutorials/** — the reader is *learning*. A lesson that walks a
  beginner through doing something real, end to end. The doc chooses
  the path; the reader follows it.
- **how-to-guides/** — the reader is *doing*. A recipe for one task,
  for someone who already knows the basics. Starts from a goal, ends
  when the goal works.
- **reference/** — the reader is *looking something up*. Facts, API
  surfaces, option tables. Austere and complete; read in fragments,
  never start to finish.
- **explanation/** — the reader is *understanding*. Why things are
  the way they are: architecture, tradeoffs, background. No steps.

The test for a new doc: which verb is the reader doing — learning,
doing, looking up, or understanding? One doc, one verb. If a doc
needs two verbs, it is two docs.

Everything else here is internal, outside those four: **specs/**
(design docs for unbuilt things), **plans/** (implementation plans),
**updates/** (release notes and progress posts), **trackers/**
(living status pages), and **contributing/** (maintainer and contributor
runbooks).

## Vibium Check and Run

| Reader's goal | Document |
|---------------|----------|
| Run a live check yourself | [Part 1: Check a live website](tutorials/first-check.md) |
| Set up Vibium for Codex | [Use Vibium with Codex](how-to-guides/using-vibium-with-codex.md) |
| Set up Vibium for Claude Code | [Use Vibium with Claude Code](how-to-guides/using-vibium-with-claude-code.md) |
| Connect Claude Desktop Chat to Vibium | [Use Vibium with Claude Desktop](how-to-guides/using-vibium-with-claude-desktop.md) |
| Write and run a check with a coding agent | [Part 2: Check with your coding agent](tutorials/check-with-a-coding-agent.md) |
| Check evidence from a saved run | [Part 3: Check a recording or trace](tutorials/check-from-a-recording.md) |
| Explore the new features | [Introducing Run and Check](updates/2026-09-07-run-and-check.md) |
| Accomplish a live browser goal | [Run](how-to-guides/run.md) |
| Configure models for either operation | [Model providers](reference/model-providers.md) |
| Diagnose browser and AI setup | [Readiness diagnostics](reference/ready.md) |
| Run a check and save evidence | [Check a live browser or saved recording](how-to-guides/check.md) |
| Look up options, configuration, and language APIs | [Check reference](reference/check.md) |
| Understand Run, Check, and their browser architecture | [How Run and Check work](explanation/run-check-webdriver-bidi-architecture.md) |
