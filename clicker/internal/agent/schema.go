package agent

import "github.com/vibium/clicker/internal/toolschema"

// The tool schemas live in toolschema so the MCP server, the CLI, and the
// shared model-tool policy in verifier describe one tool the same way
// (#571). These re-exports keep agent as the MCP-facing entry point.

// GetToolSchemas returns the list of available MCP tools with their schemas.
func GetToolSchemas() []Tool { return toolschema.GetToolSchemas() }

const (
	ScrollSelectorDesc = toolschema.ScrollSelectorDesc
	ScrollAmountDesc   = toolschema.ScrollAmountDesc
)
