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

- Prioritize bug fixes over new features
- Run tests before committing: `make test`
- When adding new command line options to the vibium binary, add a simple example and sample output (or short description)
