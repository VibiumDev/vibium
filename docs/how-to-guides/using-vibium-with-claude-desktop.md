# Use Vibium with Claude Desktop

Connect Claude Desktop's **Chat tab** to Vibium's local MCP server to control
a browser. For CLI skills in the **Code tab**, follow
[Claude Code setup](using-vibium-with-claude-code.md).

## 1. Install Vibium

With Node.js and npm installed, run in a terminal:

```bash
npm install -g vibium
vibium ready browser
```

Readiness checks installed browser files without launching them. Follow its
installation guidance if anything is missing.

## 2. Connect the MCP server

In Claude Desktop, open **Settings → Developer → Edit Config**. Add the
`vibium` entry below to `mcpServers`, preserving any existing servers and
other settings:

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

Save, fully quit Claude Desktop, and reopen it. Confirm Vibium's tools appear
in its connectors/tools list. This uses Claude Desktop's
[local MCP configuration](https://modelcontextprotocol.io/docs/develop/connect-local-servers).

If `npx` cannot be found, use its absolute path. For a development build, set
`command` to the absolute path of the native `vibium` binary (`vibium.exe` on
Windows) and set `args` to `["mcp"]`. Use forward slashes or escaped backslashes
in Windows JSON paths.

## 3. Ask Claude to use Vibium

Send this in Chat:

```text
Use Vibium's browser tools to open https://example.com, take a screenshot,
and summarize the page. Close the browser session you started afterward.
```

A browser opens when Claude starts the session. Direct browser tools need no
separate Vibium AI configuration. This setup exposes MCP tools; it does not
install `/browser` or `/check` slash skills.

## Optional: Run and Check

`vibium_run` carries out a browser goal with a configured model;
`vibium_check` independently assesses a claim. Both currently require the
development binary and [Vibium AI settings](../reference/model-providers.md).

Add an `env` object alongside `command` and `args` in the `vibium` server
entry. Copy your configured provider/model settings and credential into it;
for example:

```json
{
  "VIBIUM_AI_PROVIDER": "openai",
  "VIBIUM_AI_MODEL": "your-model",
  "OPENAI_API_KEY": "your-api-key"
}
```

Include any endpoint or reasoning-effort settings your model requires.
Keep real credentials in this private local configuration, not in chat or
project files. Restart Claude Desktop after changing it.

Vibium does not load `ai.env` automatically. Sourcing it in a separate terminal
does not update an already-running Desktop app or its MCP server.

Then ask:

```text
Use Vibium to open https://var.parts, then call vibium_check with the
claim "https://var.parts is up". Report its verdict and evidence.
Close the browser session you started afterward.
```

For recording controls and other options, see the
[MCP guide](../tutorials/getting-started-mcp.md) and
[Check's MCP reference](../reference/check.md#mcp).
