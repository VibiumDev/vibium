package bidi

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PerformActions executes a sequence of input actions.
func (c *Client) PerformActions(context string, actions []map[string]interface{}) error {
	// If no context provided, get the first one from the tree
	if context == "" {
		tree, err := c.GetTree()
		if err != nil {
			return fmt.Errorf("failed to get browsing context: %w", err)
		}
		if len(tree.Contexts) == 0 {
			return fmt.Errorf("no browsing contexts available")
		}
		context = tree.Contexts[0].Context
	}

	params := map[string]interface{}{
		"context": context,
		"actions": actions,
	}

	_, err := c.SendCommand("input.performActions", params)
	return err
}

// Click performs a mouse click at the specified coordinates.
func (c *Client) Click(context string, x, y float64) error {
	actions := []map[string]interface{}{
		{
			"type": "pointer",
			"id":   "mouse",
			"parameters": map[string]interface{}{
				"pointerType": "mouse",
			},
			"actions": []map[string]interface{}{
				{
					"type":     "pointerMove",
					"x":        int(x),
					"y":        int(y),
					"duration": 0,
				},
				{
					"type":   "pointerDown",
					"button": 0,
				},
				{
					"type":   "pointerUp",
					"button": 0,
				},
			},
		},
	}

	return c.PerformActions(context, actions)
}

// ClickElement finds an element and clicks its center.
func (c *Client) ClickElement(context, selector string) error {
	info, err := c.FindElement(context, selector)
	if err != nil {
		return err
	}

	x, y := info.GetCenter()
	return c.Click(context, x, y)
}

// DoubleClick performs a double-click at the specified coordinates.
func (c *Client) DoubleClick(context string, x, y float64) error {
	actions := []map[string]interface{}{
		{
			"type": "pointer",
			"id":   "mouse",
			"parameters": map[string]interface{}{
				"pointerType": "mouse",
			},
			"actions": []map[string]interface{}{
				{
					"type":     "pointerMove",
					"x":        int(x),
					"y":        int(y),
					"duration": 0,
				},
				{
					"type":   "pointerDown",
					"button": 0,
				},
				{
					"type":   "pointerUp",
					"button": 0,
				},
				{
					"type":   "pointerDown",
					"button": 0,
				},
				{
					"type":   "pointerUp",
					"button": 0,
				},
			},
		},
	}

	return c.PerformActions(context, actions)
}

// MoveMouse moves the mouse to the specified coordinates.
func (c *Client) MoveMouse(context string, x, y float64) error {
	actions := []map[string]interface{}{
		{
			"type": "pointer",
			"id":   "mouse",
			"parameters": map[string]interface{}{
				"pointerType": "mouse",
			},
			"actions": []map[string]interface{}{
				{
					"type":     "pointerMove",
					"x":        int(x),
					"y":        int(y),
					"duration": 0,
				},
			},
		},
	}

	return c.PerformActions(context, actions)
}

// MouseDown presses a mouse button down at the current position.
func (c *Client) MouseDown(context string, button int) error {
	actions := []map[string]interface{}{
		{
			"type": "pointer",
			"id":   "mouse",
			"parameters": map[string]interface{}{
				"pointerType": "mouse",
			},
			"actions": []map[string]interface{}{
				{
					"type":   "pointerDown",
					"button": button,
				},
			},
		},
	}

	return c.PerformActions(context, actions)
}

// MouseUp releases a mouse button at the current position.
func (c *Client) MouseUp(context string, button int) error {
	actions := []map[string]interface{}{
		{
			"type": "pointer",
			"id":   "mouse",
			"parameters": map[string]interface{}{
				"pointerType": "mouse",
			},
			"actions": []map[string]interface{}{
				{
					"type":   "pointerUp",
					"button": button,
				},
			},
		},
	}

	return c.PerformActions(context, actions)
}

// DragElement drags from one element to another using selectors.
func (c *Client) DragElement(context, srcSelector, dstSelector string) error {
	srcInfo, err := c.FindElement(context, srcSelector)
	if err != nil {
		return fmt.Errorf("failed to find source element: %w", err)
	}
	dstInfo, err := c.FindElement(context, dstSelector)
	if err != nil {
		return fmt.Errorf("failed to find target element: %w", err)
	}

	srcX, srcY := srcInfo.GetCenter()
	dstX, dstY := dstInfo.GetCenter()

	actions := []map[string]interface{}{
		{
			"type": "pointer",
			"id":   "mouse",
			"parameters": map[string]interface{}{
				"pointerType": "mouse",
			},
			"actions": []map[string]interface{}{
				{
					"type":     "pointerMove",
					"x":        int(srcX),
					"y":        int(srcY),
					"duration": 0,
				},
				{
					"type":   "pointerDown",
					"button": 0,
				},
				{
					"type":     "pause",
					"duration": 100,
				},
				{
					"type":     "pointerMove",
					"x":        int(dstX),
					"y":        int(dstY),
					"duration": 200,
				},
				{
					"type":   "pointerUp",
					"button": 0,
				},
			},
		},
	}

	return c.PerformActions(context, actions)
}

// SetFiles sets files on an input[type=file] element.
func (c *Client) SetFiles(context, selector string, files []string) error {
	if context == "" {
		tree, err := c.GetTree()
		if err != nil {
			return fmt.Errorf("failed to get browsing context: %w", err)
		}
		if len(tree.Contexts) == 0 {
			return fmt.Errorf("no browsing contexts available")
		}
		context = tree.Contexts[0].Context
	}

	// Find element to get its sharedId
	script := `(selector) => {
		const el = document.querySelector(selector);
		if (!el) return null;
		return el;
	}`

	params := map[string]interface{}{
		"functionDeclaration": script,
		"target":              map[string]interface{}{"context": context},
		"arguments": []map[string]interface{}{
			{"type": "string", "value": selector},
		},
		"awaitPromise":    false,
		"resultOwnership": "root",
	}

	msg, err := c.SendCommand("script.callFunction", params)
	if err != nil {
		return fmt.Errorf("failed to find element: %w", err)
	}

	// Parse to get sharedId
	var callResult struct {
		Result struct {
			Type     string `json:"type"`
			SharedID string `json:"sharedId"`
		} `json:"result"`
	}
	if err := json.Unmarshal(msg.Result, &callResult); err != nil {
		return fmt.Errorf("failed to parse element result: %w", err)
	}

	if callResult.Result.Type == "null" || callResult.Result.SharedID == "" {
		return fmt.Errorf("element not found: %s", selector)
	}

	// Call input.setFiles
	setParams := map[string]interface{}{
		"context": context,
		"element": map[string]interface{}{
			"sharedId": callResult.Result.SharedID,
		},
		"files": files,
	}

	_, err = c.SendCommand("input.setFiles", setParams)
	return err
}

// SetViewport sets the viewport size.
func (c *Client) SetViewport(context string, width, height int, devicePixelRatio float64) error {
	if context == "" {
		tree, err := c.GetTree()
		if err != nil {
			return fmt.Errorf("failed to get browsing context: %w", err)
		}
		if len(tree.Contexts) == 0 {
			return fmt.Errorf("no browsing contexts available")
		}
		context = tree.Contexts[0].Context
	}

	params := map[string]interface{}{
		"context": context,
		"viewport": map[string]interface{}{
			"width":  width,
			"height": height,
		},
	}
	if devicePixelRatio > 0 {
		params["devicePixelRatio"] = devicePixelRatio
	}

	_, err := c.SendCommand("browsingContext.setViewport", params)
	return err
}

// TypeText types a string of text using keyboard events.
func (c *Client) TypeText(context, text string) error {
	// Build key actions for each character
	keyActions := make([]map[string]interface{}, 0, len(text)*2)
	for _, char := range text {
		keyActions = append(keyActions,
			map[string]interface{}{
				"type": "keyDown",
				"value": string(char),
			},
			map[string]interface{}{
				"type": "keyUp",
				"value": string(char),
			},
		)
	}

	actions := []map[string]interface{}{
		{
			"type":    "key",
			"id":      "keyboard",
			"actions": keyActions,
		},
	}

	return c.PerformActions(context, actions)
}

// TypeIntoElement clicks an element and types text into it.
func (c *Client) TypeIntoElement(context, selector, text string) error {
	// Click the element first to focus it
	if err := c.ClickElement(context, selector); err != nil {
		return fmt.Errorf("failed to click element: %w", err)
	}

	// Type the text
	return c.TypeText(context, text)
}

// PressKey presses a single key (for special keys like Enter, Tab, etc).
func (c *Client) PressKey(context, key string) error {
	actions := []map[string]interface{}{
		{
			"type": "key",
			"id":   "keyboard",
			"actions": []map[string]interface{}{
				{
					"type":  "keyDown",
					"value": key,
				},
				{
					"type":  "keyUp",
					"value": key,
				},
			},
		},
	}

	return c.PerformActions(context, actions)
}

// GetElementValue gets the value of an input element.
func (c *Client) GetElementValue(context, selector string) (string, error) {
	// If no context provided, get the first one from the tree
	if context == "" {
		tree, err := c.GetTree()
		if err != nil {
			return "", fmt.Errorf("failed to get browsing context: %w", err)
		}
		if len(tree.Contexts) == 0 {
			return "", fmt.Errorf("no browsing contexts available")
		}
		context = tree.Contexts[0].Context
	}

	result, err := c.Evaluate(context, fmt.Sprintf(`document.querySelector(%q)?.value || ''`, selector))
	if err != nil {
		return "", err
	}

	if result == nil {
		return "", nil
	}

	return fmt.Sprintf("%v", result), nil
}

// ScrollWheel performs a scroll action at the specified coordinates.
func (c *Client) ScrollWheel(context string, x, y, deltaX, deltaY int) error {
	actions := []map[string]interface{}{
		{
			"type": "wheel",
			"id":   "wheel",
			"actions": []map[string]interface{}{
				{
					"type":   "scroll",
					"x":      x,
					"y":      y,
					"deltaX": deltaX,
					"deltaY": deltaY,
				},
			},
		},
	}

	return c.PerformActions(context, actions)
}

// keyMap maps named keys to their WebDriver key codepoints.
var keyMap = map[string]string{
	"Enter":      "\uE006",
	"Tab":        "\uE004",
	"Escape":     "\uE00C",
	"Backspace":  "\uE003",
	"Delete":     "\uE017",
	"ArrowUp":    "\uE013",
	"ArrowDown":  "\uE015",
	"ArrowLeft":  "\uE012",
	"ArrowRight": "\uE014",
	"Home":       "\uE011",
	"End":        "\uE010",
	"PageUp":     "\uE00E",
	"PageDown":   "\uE00F",
	"Insert":     "\uE016",
	"Space":      " ",
	"Control":    "\uE009",
	"Shift":      "\uE008",
	"Alt":        "\uE00A",
	"Meta":       "\uE03D",
	"F1":         "\uE031",
	"F2":         "\uE032",
	"F3":         "\uE033",
	"F4":         "\uE034",
	"F5":         "\uE035",
	"F6":         "\uE036",
	"F7":         "\uE037",
	"F8":         "\uE038",
	"F9":         "\uE039",
	"F10":        "\uE03A",
	"F11":        "\uE03B",
	"F12":        "\uE03C",
}

// ResolveKey resolves a key name to its WebDriver codepoint.
// If the name is not found in the keyMap, it's returned as-is.
func ResolveKey(name string) string {
	if val, ok := keyMap[name]; ok {
		return val
	}
	return name
}

// PressKeyCombo presses a key combination (e.g., "Control+a", "Shift+Enter").
// Modifiers are held down, the key is pressed, then modifiers are released.
func (c *Client) PressKeyCombo(context string, keys string) error {
	parts := strings.Split(keys, "+")
	if len(parts) == 1 {
		// Single key
		return c.PressKey(context, ResolveKey(parts[0]))
	}

	// Multiple keys: modifiers + final key
	keyActions := make([]map[string]interface{}, 0)

	// Press modifiers
	for _, part := range parts[:len(parts)-1] {
		keyActions = append(keyActions, map[string]interface{}{
			"type":  "keyDown",
			"value": ResolveKey(strings.TrimSpace(part)),
		})
	}

	// Press and release the main key
	mainKey := ResolveKey(strings.TrimSpace(parts[len(parts)-1]))
	keyActions = append(keyActions,
		map[string]interface{}{
			"type":  "keyDown",
			"value": mainKey,
		},
		map[string]interface{}{
			"type":  "keyUp",
			"value": mainKey,
		},
	)

	// Release modifiers in reverse order
	for i := len(parts) - 2; i >= 0; i-- {
		keyActions = append(keyActions, map[string]interface{}{
			"type":  "keyUp",
			"value": ResolveKey(strings.TrimSpace(parts[i])),
		})
	}

	actions := []map[string]interface{}{
		{
			"type":    "key",
			"id":      "keyboard",
			"actions": keyActions,
		},
	}

	return c.PerformActions(context, actions)
}
