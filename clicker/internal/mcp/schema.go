package mcp

// GetToolSchemas returns the list of available MCP tools with their schemas.
func GetToolSchemas() []Tool {
	return []Tool{
		{
			Name:        "browser_launch",
			Description: "Launch a new browser session",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"headless": map[string]interface{}{
						"type":        "boolean",
						"description": "Run browser in headless mode (no visible window)",
						"default":     false,
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_navigate",
			Description: "Navigate to a URL in the browser",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "The URL to navigate to",
					},
				},
				"required":             []string{"url"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_click",
			Description: "Click an element by CSS selector. Waits for element to be visible, stable, and enabled.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for the element to click",
					},
				},
				"required":             []string{"selector"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_type",
			Description: "Type text into an element by CSS selector. Waits for element to be visible, stable, enabled, and editable.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for the element to type into",
					},
					"text": map[string]interface{}{
						"type":        "string",
						"description": "The text to type",
					},
				},
				"required":             []string{"selector", "text"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_screenshot",
			Description: "Capture a screenshot of the current page",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"filename": map[string]interface{}{
						"type":        "string",
						"description": "Optional filename to save the screenshot (e.g., screenshot.png)",
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_find",
			Description: "Find an element by CSS selector and return its info (tag, text, bounding box)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for the element to find",
					},
				},
				"required":             []string{"selector"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_quit",
			Description: "Close the browser session",
			InputSchema: map[string]interface{}{
				"type":                 "object",
				"properties":           map[string]interface{}{},
				"additionalProperties": false,
			},
		},
	}
}
