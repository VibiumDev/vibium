# Save a screenshot where you want it

`vibium screenshot` writes to your current directory.

```bash
vibium screenshot
# Screenshot saved to /Users/you/project/screenshot.png
```

Name it with `-o`:

```bash
vibium screenshot -o login.png          # ./login.png
vibium screenshot -o shots/login.png    # ./shots/login.png, creating shots/
vibium screenshot -o /tmp/login.png     # exactly there
vibium screenshot -o ~/Desktop/x.png    # your shell expands ~
```

Relative paths resolve against the shell you typed them into, not the daemon's. Missing directories are created for you.

---

## Capture options

```bash
vibium screenshot --full-page -o tall.png   # whole page, not just the viewport
vibium screenshot --annotate -o labeled.png # number the interactive elements
vibium screenshot https://example.com -o home.png  # navigate first, then capture
```

---

## From an AI agent (MCP)

Over MCP there is no working directory the user can see, so screenshots go to a fixed, findable place instead: `~/Pictures/Vibium/` on macOS and Linux, `Pictures\Vibium\` on Windows.

Point it elsewhere when you set up the server:

```bash
claude mcp add vibium -- npx -y vibium mcp --screenshot-dir ./screenshots
```

Or return images inline and never touch the disk:

```bash
claude mcp add vibium -- npx -y vibium mcp --screenshot-dir ""
```

---

## Related

- [MCP setup](../tutorials/getting-started-mcp.md) — configuring the server
- [API reference](../reference/api.md) — `page.screenshot()` in the clients
