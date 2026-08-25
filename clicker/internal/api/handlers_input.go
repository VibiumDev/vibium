package api

import (
	"fmt"

	"github.com/vibium/clicker/internal/bidi"
)

// --- Page-level keyboard handlers ---

// handleKeyboardPress handles vibium:keyboard.press — presses and releases a key.
// Supports combos like "Control+a".
func (r *Router) handleKeyboardPress(session *BrowserSession, cmd bidiCommand) {
	key, _ := cmd.Params["key"].(string)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	if err := r.pressKey(session, context, key); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"pressed": true})
}

// handleKeyboardDown handles vibium:keyboard.down — presses a key down (no release).
func (r *Router) handleKeyboardDown(session *BrowserSession, cmd bidiCommand) {
	key, _ := cmd.Params["key"].(string)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	resolved := bidi.ResolveKey(key)
	params := map[string]interface{}{
		"context": context,
		"actions": []map[string]interface{}{
			{
				"type": "key",
				"id":   "keyboard",
				"actions": []map[string]interface{}{
					{"type": "keyDown", "value": resolved},
				},
			},
		},
	}

	if _, err := r.sendInternalCommand(session, "input.performActions", params); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"pressed": true})
}

// handleKeyboardUp handles vibium:keyboard.up — releases a key.
func (r *Router) handleKeyboardUp(session *BrowserSession, cmd bidiCommand) {
	key, _ := cmd.Params["key"].(string)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	resolved := bidi.ResolveKey(key)
	params := map[string]interface{}{
		"context": context,
		"actions": []map[string]interface{}{
			{
				"type": "key",
				"id":   "keyboard",
				"actions": []map[string]interface{}{
					{"type": "keyUp", "value": resolved},
				},
			},
		},
	}

	if _, err := r.sendInternalCommand(session, "input.performActions", params); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"released": true})
}

// handleKeyboardType handles vibium:keyboard.type — types a string of text.
func (r *Router) handleKeyboardType(session *BrowserSession, cmd bidiCommand) {
	text, _ := cmd.Params["text"].(string)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	if err := r.typeText(session, context, text); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"typed": true})
}

// --- Page-level mouse handlers ---

// handleMouseClick handles vibium:mouse.click — clicks at (x, y) coordinates.
func (r *Router) handleMouseClick(session *BrowserSession, cmd bidiCommand) {
	x, _ := cmd.Params["x"].(float64)
	y, _ := cmd.Params["y"].(float64)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	clickParams := map[string]interface{}{
		"context": context,
		"actions": []map[string]interface{}{
			{
				"type": "pointer",
				"id":   "mouse",
				"parameters": map[string]interface{}{
					"pointerType": "mouse",
				},
				"actions": []map[string]interface{}{
					{"type": "pointerMove", "x": int(x), "y": int(y), "duration": 0},
					{"type": "pointerDown", "button": 0},
					{"type": "pointerUp", "button": 0},
				},
			},
		},
	}

	if _, err := r.sendInternalCommand(session, "input.performActions", clickParams); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"clicked": true})
}

// handleMouseMove handles vibium:mouse.move — moves mouse to (x, y).
func (r *Router) handleMouseMove(session *BrowserSession, cmd bidiCommand) {
	x, _ := cmd.Params["x"].(float64)
	y, _ := cmd.Params["y"].(float64)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	moveParams := map[string]interface{}{
		"context": context,
		"actions": []map[string]interface{}{
			{
				"type": "pointer",
				"id":   "mouse",
				"parameters": map[string]interface{}{
					"pointerType": "mouse",
				},
				"actions": []map[string]interface{}{
					{"type": "pointerMove", "x": int(x), "y": int(y), "duration": 0},
				},
			},
		},
	}

	if _, err := r.sendInternalCommand(session, "input.performActions", moveParams); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"moved": true})
}

// handleMouseDown handles vibium:mouse.down — presses mouse button down.
func (r *Router) handleMouseDown(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	button := 0
	if b, ok := cmd.Params["button"].(float64); ok {
		button = int(b)
	}

	downParams := map[string]interface{}{
		"context": context,
		"actions": []map[string]interface{}{
			{
				"type": "pointer",
				"id":   "mouse",
				"parameters": map[string]interface{}{
					"pointerType": "mouse",
				},
				"actions": []map[string]interface{}{
					{"type": "pointerDown", "button": button},
				},
			},
		},
	}

	if _, err := r.sendInternalCommand(session, "input.performActions", downParams); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"pressed": true})
}

// handleMouseUp handles vibium:mouse.up — releases mouse button.
func (r *Router) handleMouseUp(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	button := 0
	if b, ok := cmd.Params["button"].(float64); ok {
		button = int(b)
	}

	upParams := map[string]interface{}{
		"context": context,
		"actions": []map[string]interface{}{
			{
				"type": "pointer",
				"id":   "mouse",
				"parameters": map[string]interface{}{
					"pointerType": "mouse",
				},
				"actions": []map[string]interface{}{
					{"type": "pointerUp", "button": button},
				},
			},
		},
	}

	if _, err := r.sendInternalCommand(session, "input.performActions", upParams); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"released": true})
}

// handleMouseWheel handles vibium:mouse.wheel — scrolls with deltaX/deltaY.
func (r *Router) handleMouseWheel(session *BrowserSession, cmd bidiCommand) {
	x, _ := cmd.Params["x"].(float64)
	y, _ := cmd.Params["y"].(float64)
	deltaX, _ := cmd.Params["deltaX"].(float64)
	deltaY, _ := cmd.Params["deltaY"].(float64)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	wheelParams := map[string]interface{}{
		"context": context,
		"actions": []map[string]interface{}{
			{
				"type": "wheel",
				"id":   "wheel",
				"actions": []map[string]interface{}{
					{
						"type":   "scroll",
						"x":      int(x),
						"y":      int(y),
						"deltaX": int(deltaX),
						"deltaY": int(deltaY),
					},
				},
			},
		},
	}

	if _, err := r.sendInternalCommand(session, "input.performActions", wheelParams); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"scrolled": true})
}

// handlePageScroll handles vibium:page.scroll — scrolls the page in a direction.
// Accepts direction (up/down/left/right, default "down"), amount (default 3), optional selector.
func (r *Router) handlePageScroll(session *BrowserSession, cmd bidiCommand) {
	direction := "down"
	if d, ok := cmd.Params["direction"].(string); ok && d != "" {
		direction = d
	}

	amount := 3
	if a, ok := cmd.Params["amount"].(float64); ok {
		amount = int(a)
	}

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Determine scroll target coordinates
	x, y := 0, 0
	if selector, ok := cmd.Params["selector"].(string); ok && selector != "" {
		// Scroll at element center
		info, err := r.resolveElement(session, context, ElementParams{Selector: selector})
		if err != nil {
			r.sendError(session, cmd.ID, err)
			return
		}
		x, y = pointerTarget(info)
	}

	// Map direction to deltas (120 pixels per scroll "notch")
	deltaX, deltaY := 0, 0
	pixels := amount * 120
	switch direction {
	case "down":
		deltaY = pixels
	case "up":
		deltaY = -pixels
	case "right":
		deltaX = pixels
	case "left":
		deltaX = -pixels
	default:
		r.sendError(session, cmd.ID, fmt.Errorf("invalid direction: %q (use up, down, left, right)", direction))
		return
	}

	wheelParams := map[string]interface{}{
		"context": context,
		"actions": []map[string]interface{}{
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
		},
	}

	if _, err := r.sendInternalCommand(session, "input.performActions", wheelParams); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"scrolled": true})
}

// --- Page-level touch handler ---

// handleTouchTap handles vibium:touch.tap — touch tap at (x, y).
func (r *Router) handleTouchTap(session *BrowserSession, cmd bidiCommand) {
	x, _ := cmd.Params["x"].(float64)
	y, _ := cmd.Params["y"].(float64)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	tapParams := map[string]interface{}{
		"context": context,
		"actions": []map[string]interface{}{
			{
				"type": "pointer",
				"id":   "touch",
				"parameters": map[string]interface{}{
					"pointerType": "touch",
				},
				"actions": []map[string]interface{}{
					{"type": "pointerMove", "x": int(x), "y": int(y), "duration": 0},
					{"type": "pointerDown", "button": 0},
					{"type": "pointerUp", "button": 0},
				},
			},
		},
	}

	if _, err := r.sendInternalCommand(session, "input.performActions", tapParams); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"tapped": true})
}
