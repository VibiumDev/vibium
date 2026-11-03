# Setting Up Vibium MCP in Claude Desktop

This tutorial covers how to configure Vibium as an MCP (Model Context Protocol) server in Claude Desktop for browser automation.

## Prerequisites

- Claude Desktop installed: [claude.ai/download](https://claude.ai/download)
- Vibium installed (`npm install vibium`) or clicker binary built locally

## Configure Vibium MCP

Claude Desktop uses a configuration file to manage MCP servers. You'll need to edit `claude_desktop_config.json`.

### Locate the Configuration File

The configuration file location varies by platform:

- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
- **Linux**: `~/.config/Claude/claude_desktop_config.json`

If the file doesn't exist, create it.

### Option 1: Using npx (Recommended)

Add this to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "vibium": {
      "command": "npx",
      "args": [
        "-y",
        "vibium"
      ]
    }
  }
}
```

### Option 2: Using Local Binary

If you built clicker locally, use the absolute path:

```json
{
  "mcpServers": {
    "vibium": {
      "command": "/absolute/path/to/clicker",
      "args": [
        "mcp"
      ]
    }
  }
}
```

For example, on macOS:

```json
{
  "mcpServers": {
    "vibium": {
      "command": "/Users/yourname/Projects/vibium/clicker/bin/clicker",
      "args": [
        "mcp"
      ]
    }
  }
}
```

### Option 3: Custom Screenshot Directory

By default, screenshots are saved to:
- macOS: `~/Pictures/Vibium/`
- Linux: `~/Pictures/Vibium/`
- Windows: `%USERPROFILE%\Pictures\Vibium\`

To use a different directory:

```json
{
  "mcpServers": {
    "vibium": {
      "command": "/path/to/clicker",
      "args": [
        "mcp",
        "--screenshot-dir",
        "./screenshots"
      ]
    }
  }
}
```

To disable file saving (base64 inline only):

```json
{
  "mcpServers": {
    "vibium": {
      "command": "/path/to/clicker",
      "args": [
        "mcp",
        "--screenshot-dir",
        ""
      ]
    }
  }
}
```

### Verify Installation

After editing the configuration file:

1. **Restart Claude Desktop** completely (quit and reopen)
2. Open Claude Desktop settings
3. Go to **Developer** → **Local MCP servers**
4. You should see `vibium` listed with a green checkmark

If you see an error icon, check the troubleshooting section below.

## How Claude Discovers MCP Tools

When Claude Desktop starts, it connects to each configured MCP server and performs a discovery handshake:

**Step 1: Initialize** - Establish the connection and exchange capabilities

```
→ {"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"capabilities": {}}}
← {"jsonrpc": "2.0", "id": 1, "result": {"capabilities": {"tools": {}}, "serverInfo": {"name": "vibium", "version": "0.1.0"}}}
```

**Step 2: Initialized Notification** - Confirm initialization complete

```
→ {"jsonrpc": "2.0", "method": "notifications/initialized"}
```

**Step 3: List Tools** - Get available tools with their schemas

```
→ {"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}}
← {"jsonrpc": "2.0", "id": 2, "result": {"tools": [
    {"name": "browser_launch", "description": "Launch a new browser session", "inputSchema": {...}},
    {"name": "browser_navigate", "description": "Navigate to a URL", "inputSchema": {...}},
    ...
  ]}}
```

The `inputSchema` (JSON Schema) tells Claude:
- What parameters each tool accepts
- Which parameters are required vs optional
- Parameter types and descriptions

**Important:** Tool discovery happens **on startup**. After modifying the configuration file, you must restart Claude Desktop for changes to take effect.

## Testing the Integration

Once configured, start a new conversation in Claude Desktop and ask:

```
Take a screenshot of https://example.com
```

Claude will use the Vibium MCP tools:
1. `browser_launch` - Start the browser
2. `browser_navigate` - Go to the URL
3. `browser_screenshot` - Capture the page
4. `browser_quit` - Close the browser

You should see a Chrome window open, navigate to the URL, and Claude will respond with the screenshot.

## Available MCP Tools

| Tool | Description |
|------|-------------|
| `browser_launch` | Start a browser session (visible by default) |
| `browser_navigate` | Navigate to a URL |
| `browser_click` | Click an element by CSS selector |
| `browser_type` | Type text into an element |
| `browser_screenshot` | Capture a screenshot (optionally save to file with `--screenshot-dir`) |
| `browser_find` | Find element info (tag, text, bounding box) |
| `browser_quit` | Close the browser session |

## Troubleshooting

### MCP server shows error in Claude Desktop

If you see an error icon next to `vibium` in the MCP servers list:

1. **Check the configuration file syntax** - Ensure the JSON is valid (no trailing commas, proper quotes)
2. **Verify the command path** - If using a local binary, ensure the absolute path is correct
3. **Check Claude Desktop logs** - Look for error messages in:
   - macOS: `~/Library/Logs/Claude/`
   - Windows: `%APPDATA%\Claude\logs\`
   - Linux: `~/.config/Claude/logs/`

### Browser fails to launch

Ensure Chrome for Testing is installed:

```bash
npx -y vibium install
```

Or if using the local binary:

```bash
./clicker/bin/clicker install
```

### Configuration file not found

If the configuration file doesn't exist, create it manually:

**macOS/Linux:**
```bash
mkdir -p ~/Library/Application\ Support/Claude  # macOS
mkdir -p ~/.config/Claude                        # Linux
echo '{"mcpServers":{}}' > ~/Library/Application\ Support/Claude/claude_desktop_config.json  # macOS
echo '{"mcpServers":{}}' > ~/.config/Claude/claude_desktop_config.json                      # Linux
```

**Windows (PowerShell):**
```powershell
New-Item -ItemType Directory -Force -Path "$env:APPDATA\Claude"
Set-Content -Path "$env:APPDATA\Claude\claude_desktop_config.json" -Value '{"mcpServers":{}}'
```

### Test MCP server manually

You can test the MCP server directly to verify it's working:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{}}}' | npx -y vibium
```

Expected response:
```json
{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"vibium","version":"0.1.0"}}}
```

If using a local binary:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{}}}' | ./clicker/bin/clicker mcp
```

### Multiple MCP servers

If you have other MCP servers configured, your configuration file should look like this:

```json
{
  "mcpServers": {
    "vibium": {
      "command": "npx",
      "args": ["-y", "vibium"]
    },
    "other-server": {
      "command": "other-command",
      "args": ["arg1", "arg2"]
    }
  }
}
```
