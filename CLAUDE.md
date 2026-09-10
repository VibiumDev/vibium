# CLAUDE.md

Browser automation for AI agents and humans.

## Key Docs

- ROADMAP.md — Future features (not yet prioritized)
- docs/reference/WebDriver-Bidi-Spec.md — BiDirectional WebDriver Protocol spec

## Current Goal

V1 shipped. Focus on fixing critical bugs and issues reported by users.

See open issues: https://github.com/VibiumDev/vibium/issues

## Tech Stack

- Go (vibium binary)
- TypeScript (JS client)
- Python (Python client)
- WebDriver BiDi protocol
- MCP server (stdio)

## Design Philosophy

Optimize for first-time user/developer joy. Defaults should create an "aha!" moment:
- Browser visible by default (see what the AI is doing)
- Screenshots save to a sensible location automatically
- Zero config needed to get started

Power users can override defaults (headless mode, custom paths, etc.) when needed.

## Rules

- Client library launch code must not shell out to vibium subcommands other than `pipe` — `pipe` ensures the browser is installed itself (#312). User-facing CLI install passthroughs are the one exception.
- Prioritize bug fixes over new features
- Run tests before committing: `make test`
- Always fix flaky tests immediately when they show up — never dismiss them as "pre-existing"
- The browser under test is a parameter, never a literal in a test. Suites read `ENGINE` from tests/helpers.js (`VIBIUM_ENGINE`, threaded through the Makefile as `ENGINE=`), and CI runs each suite once per engine job. Naming an engine inside a case in the default suite runs it in the chrome job, which installs no other browser. Adding an engine should be a new job, not a new branch in a test.
- When adding new command line options to the vibium binary, add a simple example and sample output (or short description)
- A CLI command whose positional values can be negative or numeric (coordinates, offsets, durations) must be built with `lateParse` (cmd/clicker/helpers.go). Never set `DisableFlagParsing` by hand; the flag drift test rejects it.
