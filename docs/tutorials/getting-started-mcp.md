# Getting Started with Vibium MCP

Let AI control your browser. This guide shows you how to add Vibium to your AI coding assistant.

---

## What You'll Get

After setup, you can ask your AI assistant things like:
- "Take a screenshot of https://example.com"
- "Go to Hacker News and find the top story"
- "Fill out this form and click submit"

The AI will control a real browser to do it. Vibium exposes dozens of browser automation tools — just describe what you want in plain English.

---

## Prerequisites

Install one of the supported AI coding assistants:

- **Claude Code:** [claude.ai/download](https://claude.ai/download)
- **Gemini CLI:** [github.com/google-gemini/gemini-cli](https://github.com/google-gemini/gemini-cli)

---

## Quick Setup

**Claude Code:**
```bash
claude mcp add vibium -- npx -y vibium mcp
```

**Gemini CLI:**
```bash
gemini mcp add vibium npx -y vibium mcp
```

That's it. Chrome downloads automatically on first use.

---

## Try It

Restart your AI assistant, then ask:

```
Take a screenshot of https://example.com
```

You'll see:
1. A Chrome window open
2. The page load
3. The AI respond with the screenshot

Screenshots are saved to `~/Pictures/Vibium/` (macOS/Linux) or `Pictures\Vibium\` (Windows).

---

## Options

### Custom screenshot directory

```bash
# Claude Code
claude mcp add vibium -- npx -y vibium mcp --screenshot-dir ./screenshots

# Gemini CLI
gemini mcp add vibium npx -y vibium mcp --screenshot-dir ./screenshots
```

To disable file saving (base64 inline only), pass an empty string:

```bash
claude mcp add vibium -- npx -y vibium mcp --screenshot-dir ""
```

### Local binary instead of npx

If you built vibium locally, point to the binary directly:

```bash
# Claude Code
claude mcp add vibium -- /path/to/vibium mcp

# Gemini CLI
gemini mcp add vibium /path/to/vibium mcp
```

### Manual JSON config (Gemini CLI)

Instead of the `gemini mcp add` command, you can edit `~/.gemini/settings.json` directly:

```json
{
  "mcpServers": {
    "vibium": {
      "command": "npx",
      "args": ["-y", "vibium", "mcp"]
    }
  }
}
```

For project-specific config, create `.gemini/settings.json` in your project directory instead.

### Headless mode

Run the browser without a visible window:

```bash
claude mcp add vibium -- npx -y vibium mcp --headless
```

### Remove Vibium

```bash
claude mcp remove vibium    # Claude Code
gemini mcp remove vibium    # Gemini CLI
```

---

## Troubleshooting

### Verify the MCP server works

Run the server directly to check for errors:

```bash
npx -y vibium mcp
```

You should see the server start and wait for input. Press Ctrl+C to exit.

### Chrome fails to download or launch

The first time you run Vibium, it downloads Chrome for Testing automatically. If this fails:

```bash
npx -y vibium install
```

On macOS, if you see a Gatekeeper warning about chromedriver, run:

```bash
xattr -cr "$(npx -y vibium which chromedriver)"
```

### Changes not taking effect

Tool discovery happens **on startup**. After adding or modifying an MCP server, you must start a new session for changes to take effect.

### Test MCP server manually

Send JSON-RPC messages directly to verify the server responds:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{}}}' | npx -y vibium mcp
```

You should see a JSON response with `serverInfo` and `capabilities`.

---

## Next Steps

**Use the JS API directly:**
See [Getting Started Tutorial](getting-started-js.md) for programmatic control with JavaScript/TypeScript.

**Use the Python API:**
See [Getting Started with Python](getting-started-python.md) for programmatic control with Python.

**Learn more about MCP:**
[Model Context Protocol docs](https://modelcontextprotocol.io)
