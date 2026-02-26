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
					"fullPage": map[string]interface{}{
						"type":        "boolean",
						"description": "Capture the full page (entire document) instead of just the viewport (default: false)",
						"default":     false,
					},
					"annotate": map[string]interface{}{
						"type":        "boolean",
						"description": "Annotate interactive elements with numbered labels (default: false)",
						"default":     false,
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_find",
			Description: "Find an element and return its info (tag, text, bounding box). Use a CSS selector or a semantic locator (role, text, label, placeholder, testid, xpath, alt, title). Combine role with text or other locators to narrow results.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for the element to find",
					},
					"role": map[string]interface{}{
						"type":        "string",
						"description": "ARIA role to match (e.g., \"button\", \"link\", \"textbox\", \"heading\", \"checkbox\")",
					},
					"text": map[string]interface{}{
						"type":        "string",
						"description": "Find element containing this text",
					},
					"label": map[string]interface{}{
						"type":        "string",
						"description": "Find input by associated label text or aria-label",
					},
					"placeholder": map[string]interface{}{
						"type":        "string",
						"description": "Find element by placeholder attribute",
					},
					"testid": map[string]interface{}{
						"type":        "string",
						"description": "Find element by data-testid attribute",
					},
					"xpath": map[string]interface{}{
						"type":        "string",
						"description": "Find element by XPath expression",
					},
					"alt": map[string]interface{}{
						"type":        "string",
						"description": "Find element by alt attribute",
					},
					"title": map[string]interface{}{
						"type":        "string",
						"description": "Find element by title attribute",
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_evaluate",
			Description: "Execute JavaScript in the browser to extract data, query the DOM, or inspect page state. Returns the evaluated result. Use this to get text content, attributes, element data, or any information from the page.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"expression": map[string]interface{}{
						"type":        "string",
						"description": "JavaScript expression to evaluate",
					},
				},
				"required":             []string{"expression"},
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
		{
			Name:        "browser_get_html",
			Description: "Get the HTML content of the page or a specific element",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for a specific element (optional, defaults to full page HTML)",
					},
					"outer": map[string]interface{}{
						"type":        "boolean",
						"description": "Return outerHTML instead of innerHTML (default: false)",
						"default":     false,
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_find_all",
			Description: "Find all elements matching a CSS selector and return their info (tag, text, bounding box)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector to match elements",
					},
					"limit": map[string]interface{}{
						"type":        "number",
						"description": "Maximum number of elements to return (default: 10)",
						"default":     10,
					},
				},
				"required":             []string{"selector"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_wait",
			Description: "Wait for an element to reach a specified state (attached, visible, or hidden)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for the element to wait for",
					},
					"state": map[string]interface{}{
						"type":        "string",
						"description": "State to wait for: \"attached\" (exists in DOM), \"visible\" (visible on page), or \"hidden\" (not found or not visible)",
						"enum":        []string{"attached", "visible", "hidden"},
						"default":     "attached",
					},
					"timeout": map[string]interface{}{
						"type":        "number",
						"description": "Timeout in milliseconds (default: 30000)",
						"default":     30000,
					},
				},
				"required":             []string{"selector"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_new_tab",
			Description: "Open a new browser tab, optionally navigating to a URL",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "URL to navigate to in the new tab (optional)",
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_list_tabs",
			Description: "List all open browser tabs with their URLs",
			InputSchema: map[string]interface{}{
				"type":                 "object",
				"properties":           map[string]interface{}{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_switch_tab",
			Description: "Switch to a browser tab by index or URL substring",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"index": map[string]interface{}{
						"type":        "number",
						"description": "Tab index (0-based) from browser_list_tabs",
					},
					"url": map[string]interface{}{
						"type":        "string",
						"description": "URL substring to match (alternative to index)",
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_close_tab",
			Description: "Close a browser tab by index (default: current tab)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"index": map[string]interface{}{
						"type":        "number",
						"description": "Tab index to close (default: 0, the current tab)",
						"default":     0,
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_hover",
			Description: "Hover over an element by CSS selector",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for the element to hover over",
					},
				},
				"required":             []string{"selector"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_select",
			Description: "Select an option in a <select> element by value",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for the <select> element",
					},
					"value": map[string]interface{}{
						"type":        "string",
						"description": "The value to select",
					},
				},
				"required":             []string{"selector", "value"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_scroll",
			Description: "Scroll the page or a specific element",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"direction": map[string]interface{}{
						"type":        "string",
						"description": "Scroll direction: up, down, left, right (default: down)",
						"enum":        []string{"up", "down", "left", "right"},
						"default":     "down",
					},
					"amount": map[string]interface{}{
						"type":        "number",
						"description": "Number of scroll increments (default: 3)",
						"default":     3,
					},
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for element to scroll to (optional, defaults to viewport center)",
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_keys",
			Description: "Press a key or key combination (e.g., \"Enter\", \"Control+a\", \"Shift+Tab\")",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"keys": map[string]interface{}{
						"type":        "string",
						"description": "Key or key combination to press (e.g., \"Enter\", \"Control+a\", \"Shift+ArrowDown\")",
					},
				},
				"required":             []string{"keys"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_get_text",
			Description: "Get the text content of the page or a specific element",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for a specific element (optional, defaults to full page text)",
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_get_url",
			Description: "Get the current page URL",
			InputSchema: map[string]interface{}{
				"type":                 "object",
				"properties":           map[string]interface{}{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_get_title",
			Description: "Get the current page title",
			InputSchema: map[string]interface{}{
				"type":                 "object",
				"properties":           map[string]interface{}{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_a11y_tree",
			Description: "Get the accessibility tree of the current page. Returns a tree of ARIA roles, names, and states — useful for understanding page structure without visual rendering.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"everything": map[string]interface{}{
						"type":        "boolean",
						"description": "Show all nodes including generic containers. Default: false",
						"default":     false,
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "page_clock_install",
			Description: "Install a fake clock on the page, overriding Date, setTimeout, setInterval, requestAnimationFrame, and performance.now",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"time": map[string]interface{}{
						"type":        "number",
						"description": "Initial time as epoch milliseconds (optional)",
					},
					"timezone": map[string]interface{}{
						"type":        "string",
						"description": "IANA timezone ID to override (e.g. 'America/New_York', 'Europe/London')",
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "page_clock_fast_forward",
			Description: "Jump the fake clock forward by N milliseconds, firing each due timer at most once",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ticks": map[string]interface{}{
						"type":        "number",
						"description": "Number of milliseconds to fast-forward",
					},
				},
				"required":             []string{"ticks"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "page_clock_run_for",
			Description: "Advance the fake clock by N milliseconds, firing all time-related callbacks systematically",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ticks": map[string]interface{}{
						"type":        "number",
						"description": "Number of milliseconds to advance",
					},
				},
				"required":             []string{"ticks"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "page_clock_pause_at",
			Description: "Jump the fake clock to a specific time and pause — no timers fire until resumed or advanced",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"time": map[string]interface{}{
						"type":        "number",
						"description": "Time as epoch milliseconds to pause at",
					},
				},
				"required":             []string{"time"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "page_clock_resume",
			Description: "Resume real-time progression from the current fake clock time",
			InputSchema: map[string]interface{}{
				"type":                 "object",
				"properties":           map[string]interface{}{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "page_clock_set_fixed_time",
			Description: "Freeze Date.now() at a specific value permanently. Timers still run.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"time": map[string]interface{}{
						"type":        "number",
						"description": "Time as epoch milliseconds to freeze at",
					},
				},
				"required":             []string{"time"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "page_clock_set_system_time",
			Description: "Set Date.now() to a specific value without triggering any timers",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"time": map[string]interface{}{
						"type":        "number",
						"description": "Time as epoch milliseconds to set",
					},
				},
				"required":             []string{"time"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "page_clock_set_timezone",
			Description: "Override the browser timezone. Pass an IANA timezone ID (e.g. 'America/New_York'), or empty string to reset to system default",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"timezone": map[string]interface{}{
						"type":        "string",
						"description": "IANA timezone ID (e.g. 'America/New_York', 'Europe/London', 'Asia/Tokyo'). Empty string resets to system default.",
					},
				},
				"required":             []string{"timezone"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_fill",
			Description: "Clear an input field and type new text. Waits for element to be editable, clears existing value, then types. Use this instead of browser_type when you want to replace the field contents.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for the input element",
					},
					"text": map[string]interface{}{
						"type":        "string",
						"description": "The text to fill in",
					},
				},
				"required":             []string{"selector", "text"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_press",
			Description: "Press a key or key combination on a specific element or the focused element. If selector is given, clicks the element first to focus it, then presses the key.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"key": map[string]interface{}{
						"type":        "string",
						"description": "Key or key combination to press (e.g., \"Enter\", \"Control+a\", \"Escape\")",
					},
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for the element to focus before pressing (optional, defaults to currently focused element)",
					},
				},
				"required":             []string{"key"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_back",
			Description: "Navigate back in browser history (like clicking the back button)",
			InputSchema: map[string]interface{}{
				"type":                 "object",
				"properties":           map[string]interface{}{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_forward",
			Description: "Navigate forward in browser history (like clicking the forward button)",
			InputSchema: map[string]interface{}{
				"type":                 "object",
				"properties":           map[string]interface{}{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_reload",
			Description: "Reload the current page. Waits for the page to fully load.",
			InputSchema: map[string]interface{}{
				"type":                 "object",
				"properties":           map[string]interface{}{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_get_value",
			Description: "Get the current value of an input, textarea, or select element",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for the form element",
					},
				},
				"required":             []string{"selector"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_get_attribute",
			Description: "Get the value of an HTML attribute on an element",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for the element",
					},
					"attribute": map[string]interface{}{
						"type":        "string",
						"description": "Attribute name to retrieve (e.g., \"href\", \"src\", \"class\", \"data-id\")",
					},
				},
				"required":             []string{"selector", "attribute"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_is_visible",
			Description: "Check if an element is visible on the page. Returns true/false without throwing errors.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for the element",
					},
				},
				"required":             []string{"selector"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_check",
			Description: "Check a checkbox or radio button. Idempotent — does nothing if already checked.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for the checkbox or radio button",
					},
				},
				"required":             []string{"selector"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_uncheck",
			Description: "Uncheck a checkbox. Idempotent — does nothing if already unchecked.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for the checkbox",
					},
				},
				"required":             []string{"selector"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_scroll_into_view",
			Description: "Scroll an element into view, centering it on screen",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for the element to scroll into view",
					},
				},
				"required":             []string{"selector"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_wait_for_url",
			Description: "Wait until the page URL contains a given substring",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "Substring to match in the URL",
					},
					"timeout": map[string]interface{}{
						"type":        "number",
						"description": "Timeout in milliseconds (default: 30000)",
						"default":     30000,
					},
				},
				"required":             []string{"pattern"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_wait_for_load",
			Description: "Wait until the page reaches the \"complete\" ready state (all resources loaded)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"timeout": map[string]interface{}{
						"type":        "number",
						"description": "Timeout in milliseconds (default: 30000)",
						"default":     30000,
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_sleep",
			Description: "Pause execution for a specified number of milliseconds. Use sparingly — prefer browser_wait or browser_wait_for_url when possible.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ms": map[string]interface{}{
						"type":        "number",
						"description": "Number of milliseconds to sleep (max 30000)",
					},
				},
				"required":             []string{"ms"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_map",
			Description: "Map interactive page elements with @refs for targeting. Returns a list of interactive elements (buttons, links, inputs, etc.) each with a short @ref like @e1, @e2. Use these refs as selectors in other commands (click, fill, etc.).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector to scope element discovery to a subtree (e.g. \"nav\", \"#sidebar\")",
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_diff_map",
			Description: "Compare current page state vs last map. Shows additions (+) and removals (-) since the last browser_map call.",
			InputSchema: map[string]interface{}{
				"type":                 "object",
				"properties":           map[string]interface{}{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_pdf",
			Description: "Save the current page as a PDF file",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"filename": map[string]interface{}{
						"type":        "string",
						"description": "Output filename for the PDF (e.g., page.pdf)",
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_highlight",
			Description: "Highlight an element with a red outline for 3 seconds. Useful for visual debugging.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector or @ref for the element to highlight",
					},
				},
				"required":             []string{"selector"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_dblclick",
			Description: "Double-click an element by CSS selector or @ref",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector or @ref for the element to double-click",
					},
				},
				"required":             []string{"selector"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_focus",
			Description: "Focus an element by CSS selector or @ref",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector or @ref for the element to focus",
					},
				},
				"required":             []string{"selector"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_count",
			Description: "Count the number of elements matching a CSS selector",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector to count matches for",
					},
				},
				"required":             []string{"selector"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_is_enabled",
			Description: "Check if an element is enabled (not disabled). Returns true/false.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector or @ref for the element",
					},
				},
				"required":             []string{"selector"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_is_checked",
			Description: "Check if a checkbox or radio button is checked. Returns true/false.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector or @ref for the checkbox/radio element",
					},
				},
				"required":             []string{"selector"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_wait_for_text",
			Description: "Wait until specific text appears on the page",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text": map[string]interface{}{
						"type":        "string",
						"description": "Text to wait for on the page",
					},
					"timeout": map[string]interface{}{
						"type":        "number",
						"description": "Timeout in milliseconds (default: 30000)",
						"default":     30000,
					},
				},
				"required":             []string{"text"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_wait_for_fn",
			Description: "Wait until a JavaScript expression returns a truthy value",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"expression": map[string]interface{}{
						"type":        "string",
						"description": "JavaScript expression to evaluate (e.g., \"window.ready === true\")",
					},
					"timeout": map[string]interface{}{
						"type":        "number",
						"description": "Timeout in milliseconds (default: 30000)",
						"default":     30000,
					},
				},
				"required":             []string{"expression"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_dialog_accept",
			Description: "Accept a dialog (alert, confirm, prompt). Optionally provide text for prompt dialogs.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text": map[string]interface{}{
						"type":        "string",
						"description": "Text to enter in the prompt dialog (optional)",
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_dialog_dismiss",
			Description: "Dismiss a dialog (cancel/close)",
			InputSchema: map[string]interface{}{
				"type":                 "object",
				"properties":           map[string]interface{}{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_get_cookies",
			Description: "List all cookies for the current page",
			InputSchema: map[string]interface{}{
				"type":                 "object",
				"properties":           map[string]interface{}{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_set_cookie",
			Description: "Set a cookie",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Cookie name",
					},
					"value": map[string]interface{}{
						"type":        "string",
						"description": "Cookie value",
					},
					"domain": map[string]interface{}{
						"type":        "string",
						"description": "Cookie domain (optional, defaults to current page domain)",
					},
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Cookie path (optional, defaults to /)",
					},
				},
				"required":             []string{"name", "value"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_delete_cookies",
			Description: "Delete cookies. If name is given, deletes that cookie. Otherwise deletes all cookies.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Cookie name to delete (optional, omit to delete all)",
					},
				},
				"additionalProperties": false,
			},
		},
		// --- Mouse primitives ---
		{
			Name:        "browser_mouse_move",
			Description: "Move the mouse to specific coordinates",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"x": map[string]interface{}{
						"type":        "number",
						"description": "X coordinate",
					},
					"y": map[string]interface{}{
						"type":        "number",
						"description": "Y coordinate",
					},
				},
				"required":             []string{"x", "y"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_mouse_down",
			Description: "Press a mouse button down at the current position",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"button": map[string]interface{}{
						"type":        "number",
						"description": "Mouse button (0=left, 1=middle, 2=right). Default: 0",
						"default":     0,
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_mouse_up",
			Description: "Release a mouse button at the current position",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"button": map[string]interface{}{
						"type":        "number",
						"description": "Mouse button (0=left, 1=middle, 2=right). Default: 0",
						"default":     0,
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_mouse_click",
			Description: "Click at coordinates or at the current mouse position. If x and y are provided, moves the mouse there first.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"x": map[string]interface{}{
						"type":        "number",
						"description": "X coordinate to click at",
					},
					"y": map[string]interface{}{
						"type":        "number",
						"description": "Y coordinate to click at",
					},
					"button": map[string]interface{}{
						"type":        "number",
						"description": "Mouse button (0=left, 1=middle, 2=right). Default: 0",
						"default":     0,
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_drag",
			Description: "Drag from one element to another",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"source": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector or @ref for the source element",
					},
					"target": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector or @ref for the target element",
					},
				},
				"required":             []string{"source", "target"},
				"additionalProperties": false,
			},
		},
		// --- Emulation ---
		{
			Name:        "browser_set_viewport",
			Description: "Set the browser viewport size",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"width": map[string]interface{}{
						"type":        "number",
						"description": "Viewport width in pixels",
					},
					"height": map[string]interface{}{
						"type":        "number",
						"description": "Viewport height in pixels",
					},
					"devicePixelRatio": map[string]interface{}{
						"type":        "number",
						"description": "Device pixel ratio (optional, e.g., 2 for Retina)",
					},
				},
				"required":             []string{"width", "height"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_get_viewport",
			Description: "Get the current viewport dimensions",
			InputSchema: map[string]interface{}{
				"type":                 "object",
				"properties":           map[string]interface{}{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_get_window",
			Description: "Get the OS browser window dimensions and state",
			InputSchema: map[string]interface{}{
				"type":                 "object",
				"properties":           map[string]interface{}{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_set_window",
			Description: "Set the OS browser window size, position, or state",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"width": map[string]interface{}{
						"type":        "number",
						"description": "Window width in pixels",
					},
					"height": map[string]interface{}{
						"type":        "number",
						"description": "Window height in pixels",
					},
					"x": map[string]interface{}{
						"type":        "number",
						"description": "Window x position in pixels",
					},
					"y": map[string]interface{}{
						"type":        "number",
						"description": "Window y position in pixels",
					},
					"state": map[string]interface{}{
						"type":        "string",
						"description": "Window state: normal, maximized, minimized, or fullscreen",
						"enum":        []string{"normal", "maximized", "minimized", "fullscreen"},
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_emulate_media",
			Description: "Override CSS media features (color scheme, reduced motion, etc.)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"media": map[string]interface{}{
						"type":        "string",
						"description": "Media type: \"screen\" or \"print\"",
						"enum":        []string{"screen", "print"},
					},
					"colorScheme": map[string]interface{}{
						"type":        "string",
						"description": "Color scheme: \"light\", \"dark\", or \"no-preference\"",
						"enum":        []string{"light", "dark", "no-preference"},
					},
					"reducedMotion": map[string]interface{}{
						"type":        "string",
						"description": "Reduced motion: \"reduce\" or \"no-preference\"",
						"enum":        []string{"reduce", "no-preference"},
					},
					"forcedColors": map[string]interface{}{
						"type":        "string",
						"description": "Forced colors: \"active\" or \"none\"",
						"enum":        []string{"active", "none"},
					},
					"contrast": map[string]interface{}{
						"type":        "string",
						"description": "Contrast preference: \"more\", \"less\", or \"no-preference\"",
						"enum":        []string{"more", "less", "no-preference"},
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_set_geolocation",
			Description: "Override the browser geolocation",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"latitude": map[string]interface{}{
						"type":        "number",
						"description": "Latitude (-90 to 90)",
					},
					"longitude": map[string]interface{}{
						"type":        "number",
						"description": "Longitude (-180 to 180)",
					},
					"accuracy": map[string]interface{}{
						"type":        "number",
						"description": "Accuracy in meters (default: 1)",
						"default":     1,
					},
				},
				"required":             []string{"latitude", "longitude"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_set_content",
			Description: "Replace the page HTML content",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"html": map[string]interface{}{
						"type":        "string",
						"description": "HTML content to set",
					},
				},
				"required":             []string{"html"},
				"additionalProperties": false,
			},
		},
		// --- Frames ---
		{
			Name:        "browser_frames",
			Description: "List all child frames (iframes) on the current page",
			InputSchema: map[string]interface{}{
				"type":                 "object",
				"properties":           map[string]interface{}{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_frame",
			Description: "Find a frame by name (exact match) or URL (substring match)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"nameOrUrl": map[string]interface{}{
						"type":        "string",
						"description": "Frame name (exact match) or URL substring to find",
					},
				},
				"required":             []string{"nameOrUrl"},
				"additionalProperties": false,
			},
		},
		// --- Upload ---
		{
			Name:        "browser_upload",
			Description: "Set files on an input[type=file] element",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{
						"type":        "string",
						"description": "CSS selector for the file input element",
					},
					"files": map[string]interface{}{
						"type":        "array",
						"description": "Array of absolute file paths to upload",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
				"required":             []string{"selector", "files"},
				"additionalProperties": false,
			},
		},
		// --- Tracing ---
		{
			Name:        "browser_trace_start",
			Description: "Start recording a browser trace (screenshots and/or HTML snapshots)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Name for the trace (default: \"trace\")",
					},
					"screenshots": map[string]interface{}{
						"type":        "boolean",
						"description": "Capture screenshots periodically (default: false)",
						"default":     false,
					},
					"snapshots": map[string]interface{}{
						"type":        "boolean",
						"description": "Capture HTML snapshots (default: false)",
						"default":     false,
					},
					"bidi": map[string]interface{}{
						"type":        "boolean",
						"description": "Record raw BiDi commands in the trace (default: false)",
						"default":     false,
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_trace_stop",
			Description: "Stop trace recording and save to a ZIP file",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Output file path (default: trace.zip)",
					},
				},
				"additionalProperties": false,
			},
		},
		// --- Storage state ---
		{
			Name:        "browser_storage_state",
			Description: "Export cookies, localStorage, and sessionStorage as JSON",
			InputSchema: map[string]interface{}{
				"type":                 "object",
				"properties":           map[string]interface{}{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "browser_restore_storage",
			Description: "Restore cookies and storage from a JSON state file",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the JSON state file",
					},
				},
				"required":             []string{"path"},
				"additionalProperties": false,
			},
		},
		// --- Downloads ---
		{
			Name:        "browser_download_set_dir",
			Description: "Set the download directory for the browser",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Directory path for downloads",
					},
				},
				"required":             []string{"path"},
				"additionalProperties": false,
			},
		},
	}
}
