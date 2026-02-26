package proxy

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vibium/clicker/internal/bidi"
)

// handleVibiumClick handles the vibium:click command with actionability checks.
// Supports index param for elements from findAll().
func (r *Router) handleVibiumClick(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	info, err := r.resolveElement(session, context, ep)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	if err := r.clickAtCenter(session, context, info); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"clicked": true})
}

// handleVibiumDblclick handles the vibium:dblclick command.
func (r *Router) handleVibiumDblclick(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	info, err := r.resolveElement(session, context, ep)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	x := int(info.Box.X + info.Box.Width/2)
	y := int(info.Box.Y + info.Box.Height/2)

	dblclickParams := map[string]interface{}{
		"context": context,
		"actions": []map[string]interface{}{
			{
				"type": "pointer",
				"id":   "mouse",
				"parameters": map[string]interface{}{
					"pointerType": "mouse",
				},
				"actions": []map[string]interface{}{
					{"type": "pointerMove", "x": x, "y": y, "duration": 0},
					{"type": "pointerDown", "button": 0},
					{"type": "pointerUp", "button": 0},
					{"type": "pointerDown", "button": 0},
					{"type": "pointerUp", "button": 0},
				},
			},
		},
	}

	if _, err := r.sendInternalCommand(session, "input.performActions", dblclickParams); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"dblclicked": true})
}

// handleVibiumFill handles the vibium:fill command.
// Uses JS to set the element value, then dispatches input/change events.
func (r *Router) handleVibiumFill(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)
	value, _ := cmd.Params["value"].(string)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Resolve element to ensure it exists
	if _, err := r.resolveElement(session, context, ep); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Use JS to set value and dispatch events (most reliable cross-platform)
	script, args := buildSetValueScript(ep, value)

	params := map[string]interface{}{
		"functionDeclaration": script,
		"target":              map[string]interface{}{"context": context},
		"arguments":           args,
		"awaitPromise":        false,
		"resultOwnership":     "root",
	}

	resp, err := r.sendInternalCommand(session, "script.callFunction", params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	val, err := parseScriptResult(resp)
	if err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("fill failed: %w", err))
		return
	}

	if val != "ok" {
		r.sendError(session, cmd.ID, fmt.Errorf("fill: %s", val))
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"filled": true})
}

// handleVibiumType handles the vibium:type command with actionability checks.
// Clicks to focus and types text (does NOT clear first).
func (r *Router) handleVibiumType(session *BrowserSession, cmd bidiCommand) {
	// Extract text-to-type BEFORE extractElementParams, since "text" is also
	// a semantic selector param. Remove it from params to avoid collision.
	text, _ := cmd.Params["text"].(string)
	paramsCopy := make(map[string]interface{})
	for k, v := range cmd.Params {
		if k != "text" {
			paramsCopy[k] = v
		}
	}
	ep := extractElementParams(paramsCopy)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	info, err := r.resolveElement(session, context, ep)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Click to focus
	if err := r.clickAtCenter(session, context, info); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Type the text
	if err := r.typeText(session, context, text); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"typed": true})
}

// handleVibiumPress handles the vibium:press command.
// Clicks to focus, then presses a key (supports combos like "Control+a").
func (r *Router) handleVibiumPress(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)
	key, _ := cmd.Params["key"].(string)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	info, err := r.resolveElement(session, context, ep)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Click to focus
	if err := r.clickAtCenter(session, context, info); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Press the key (supports combos)
	if err := r.pressKey(session, context, key); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"pressed": true})
}

// handleVibiumClear handles the vibium:clear command.
// Uses JS to clear the element value.
func (r *Router) handleVibiumClear(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Resolve element to ensure it exists
	if _, err := r.resolveElement(session, context, ep); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Use JS to set value to empty string
	script, args := buildSetValueScript(ep, "")

	params := map[string]interface{}{
		"functionDeclaration": script,
		"target":              map[string]interface{}{"context": context},
		"arguments":           args,
		"awaitPromise":        false,
		"resultOwnership":     "root",
	}

	resp, err := r.sendInternalCommand(session, "script.callFunction", params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	val, err := parseScriptResult(resp)
	if err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("clear failed: %w", err))
		return
	}

	if val != "ok" {
		r.sendError(session, cmd.ID, fmt.Errorf("clear: %s", val))
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"cleared": true})
}

// handleVibiumCheck handles the vibium:check command.
// Clicks the checkbox only if it's not already checked.
func (r *Router) handleVibiumCheck(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	info, err := r.resolveElement(session, context, ep)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Run JS to check if already checked
	checked, err := r.isChecked(session, context, ep)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	if !checked {
		if err := r.clickAtCenter(session, context, info); err != nil {
			r.sendError(session, cmd.ID, err)
			return
		}
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"checked": true})
}

// handleVibiumUncheck handles the vibium:uncheck command.
// Clicks the checkbox only if it's currently checked.
func (r *Router) handleVibiumUncheck(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	info, err := r.resolveElement(session, context, ep)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Run JS to check if checked
	checked, err := r.isChecked(session, context, ep)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	if checked {
		if err := r.clickAtCenter(session, context, info); err != nil {
			r.sendError(session, cmd.ID, err)
			return
		}
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"unchecked": true})
}

// handleVibiumSelectOption handles the vibium:selectOption command.
// Sets the value of a <select> element and dispatches a change event.
func (r *Router) handleVibiumSelectOption(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)
	value, _ := cmd.Params["value"].(string)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Resolve element first to ensure it exists
	if _, err := r.resolveElement(session, context, ep); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Build a script to find and set value
	script, args := buildSelectOptionScript(ep, value)

	params := map[string]interface{}{
		"functionDeclaration": script,
		"target":              map[string]interface{}{"context": context},
		"arguments":           args,
		"awaitPromise":        false,
		"resultOwnership":     "root",
	}

	resp, err := r.sendInternalCommand(session, "script.callFunction", params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	val, err := parseScriptResult(resp)
	if err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("selectOption failed: %w", err))
		return
	}

	if val != "ok" {
		r.sendError(session, cmd.ID, fmt.Errorf("selectOption: %s", val))
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"selected": true})
}

// handleVibiumHover handles the vibium:hover command.
// Moves the mouse pointer to the element's center without clicking.
func (r *Router) handleVibiumHover(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	info, err := r.resolveElement(session, context, ep)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	x := int(info.Box.X + info.Box.Width/2)
	y := int(info.Box.Y + info.Box.Height/2)

	hoverParams := map[string]interface{}{
		"context": context,
		"actions": []map[string]interface{}{
			{
				"type": "pointer",
				"id":   "mouse",
				"parameters": map[string]interface{}{
					"pointerType": "mouse",
				},
				"actions": []map[string]interface{}{
					{"type": "pointerMove", "x": x, "y": y, "duration": 0},
				},
			},
		},
	}

	if _, err := r.sendInternalCommand(session, "input.performActions", hoverParams); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"hovered": true})
}

// handleVibiumFocus handles the vibium:focus command.
// Runs element.focus() via JavaScript.
func (r *Router) handleVibiumFocus(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Resolve element to confirm it exists
	if _, err := r.resolveElement(session, context, ep); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Run JS to focus
	script, args := buildFocusScript(ep)

	params := map[string]interface{}{
		"functionDeclaration": script,
		"target":              map[string]interface{}{"context": context},
		"arguments":           args,
		"awaitPromise":        false,
		"resultOwnership":     "root",
	}

	if _, err := r.sendInternalCommand(session, "script.callFunction", params); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"focused": true})
}

// handleVibiumDragTo handles the vibium:dragTo command.
// Resolves source and target elements, then performs pointer drag.
func (r *Router) handleVibiumDragTo(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Resolve source element
	srcInfo, err := r.resolveElement(session, context, ep)
	if err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("source: %w", err))
		return
	}

	// Extract target params
	targetParams, ok := cmd.Params["target"].(map[string]interface{})
	if !ok {
		r.sendError(session, cmd.ID, fmt.Errorf("dragTo requires 'target' parameter"))
		return
	}

	targetEp := extractElementParams(targetParams)
	targetInfo, err := r.resolveElement(session, context, targetEp)
	if err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("target: %w", err))
		return
	}

	srcX := int(srcInfo.Box.X + srcInfo.Box.Width/2)
	srcY := int(srcInfo.Box.Y + srcInfo.Box.Height/2)
	dstX := int(targetInfo.Box.X + targetInfo.Box.Width/2)
	dstY := int(targetInfo.Box.Y + targetInfo.Box.Height/2)

	dragParams := map[string]interface{}{
		"context": context,
		"actions": []map[string]interface{}{
			{
				"type": "pointer",
				"id":   "mouse",
				"parameters": map[string]interface{}{
					"pointerType": "mouse",
				},
				"actions": []map[string]interface{}{
					{"type": "pointerMove", "x": srcX, "y": srcY, "duration": 0},
					{"type": "pointerDown", "button": 0},
					{"type": "pause", "duration": 100},
					{"type": "pointerMove", "x": dstX, "y": dstY, "duration": 200},
					{"type": "pointerUp", "button": 0},
				},
			},
		},
	}

	if _, err := r.sendInternalCommand(session, "input.performActions", dragParams); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"dragged": true})
}

// handleVibiumTap handles the vibium:tap command.
// Performs a touch tap at the element's center.
func (r *Router) handleVibiumTap(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	info, err := r.resolveElement(session, context, ep)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	x := int(info.Box.X + info.Box.Width/2)
	y := int(info.Box.Y + info.Box.Height/2)

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
					{"type": "pointerMove", "x": x, "y": y, "duration": 0},
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

// handleVibiumScrollIntoView handles the vibium:scrollIntoView command.
// Resolves the element (which auto-scrolls it into view).
func (r *Router) handleVibiumScrollIntoView(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// resolveElement already scrolls into view
	if _, err := r.resolveElement(session, context, ep); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"scrolled": true})
}

// handleVibiumDispatchEvent handles the vibium:dispatchEvent command.
// Dispatches a DOM event on the element via JavaScript.
func (r *Router) handleVibiumDispatchEvent(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)
	eventType, _ := cmd.Params["eventType"].(string)
	eventInit, _ := cmd.Params["eventInit"].(map[string]interface{})

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Resolve element to confirm it exists
	if _, err := r.resolveElement(session, context, ep); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Build event init JSON
	initJSON := "{}"
	if eventInit != nil {
		initBytes, _ := json.Marshal(eventInit)
		initJSON = string(initBytes)
	}

	// Build dispatch script
	script, args := buildDispatchEventScript(ep, eventType, initJSON)

	params := map[string]interface{}{
		"functionDeclaration": script,
		"target":              map[string]interface{}{"context": context},
		"arguments":           args,
		"awaitPromise":        false,
		"resultOwnership":     "root",
	}

	if _, err := r.sendInternalCommand(session, "script.callFunction", params); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"dispatched": true})
}

// handleVibiumElSetFiles handles the vibium:el.setFiles command.
// Sets files on an <input type="file"> element using BiDi input.setFiles.
func (r *Router) handleVibiumElSetFiles(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Extract files array
	filesRaw, ok := cmd.Params["files"]
	if !ok {
		r.sendError(session, cmd.ID, fmt.Errorf("el.setFiles requires 'files' parameter"))
		return
	}
	filesArr, ok := filesRaw.([]interface{})
	if !ok {
		r.sendError(session, cmd.ID, fmt.Errorf("el.setFiles: 'files' must be an array"))
		return
	}
	files := make([]string, len(filesArr))
	for i, f := range filesArr {
		s, ok := f.(string)
		if !ok {
			r.sendError(session, cmd.ID, fmt.Errorf("el.setFiles: each file must be a string"))
			return
		}
		files[i] = s
	}

	// Resolve the element to get its BiDi sharedId
	sharedID, err := r.resolveElementRef(session, context, ep)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Call input.setFiles
	_, err = r.sendInternalCommand(session, "input.setFiles", map[string]interface{}{
		"context": context,
		"element": map[string]interface{}{
			"sharedId": sharedID,
		},
		"files": files,
	})
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"set": true})
}

// --- Helper methods for interaction handlers ---

// clickAtCenter performs a mouse click at the center of an element.
func (r *Router) clickAtCenter(session *BrowserSession, context string, info *elementInfo) error {
	x := int(info.Box.X + info.Box.Width/2)
	y := int(info.Box.Y + info.Box.Height/2)

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
					{"type": "pointerMove", "x": x, "y": y, "duration": 0},
					{"type": "pointerDown", "button": 0},
					{"type": "pointerUp", "button": 0},
				},
			},
		},
	}

	_, err := r.sendInternalCommand(session, "input.performActions", clickParams)
	return err
}

// typeText types a string of text using keyboard events.
func (r *Router) typeText(session *BrowserSession, context, text string) error {
	keyActions := make([]map[string]interface{}, 0, len(text)*2)
	for _, char := range text {
		keyActions = append(keyActions,
			map[string]interface{}{"type": "keyDown", "value": string(char)},
			map[string]interface{}{"type": "keyUp", "value": string(char)},
		)
	}

	typeParams := map[string]interface{}{
		"context": context,
		"actions": []map[string]interface{}{
			{
				"type":    "key",
				"id":      "keyboard",
				"actions": keyActions,
			},
		},
	}

	_, err := r.sendInternalCommand(session, "input.performActions", typeParams)
	return err
}

// pressKey presses a key or key combo (e.g. "Enter", "Control+a").
func (r *Router) pressKey(session *BrowserSession, context, key string) error {
	parts := strings.Split(key, "+")
	keyActions := make([]map[string]interface{}, 0)

	if len(parts) == 1 {
		// Single key
		resolved := bidi.ResolveKey(parts[0])
		keyActions = append(keyActions,
			map[string]interface{}{"type": "keyDown", "value": resolved},
			map[string]interface{}{"type": "keyUp", "value": resolved},
		)
	} else {
		// Key combo: press modifiers, press+release main key, release modifiers
		for _, part := range parts[:len(parts)-1] {
			keyActions = append(keyActions, map[string]interface{}{
				"type":  "keyDown",
				"value": bidi.ResolveKey(strings.TrimSpace(part)),
			})
		}

		mainKey := bidi.ResolveKey(strings.TrimSpace(parts[len(parts)-1]))
		keyActions = append(keyActions,
			map[string]interface{}{"type": "keyDown", "value": mainKey},
			map[string]interface{}{"type": "keyUp", "value": mainKey},
		)

		for i := len(parts) - 2; i >= 0; i-- {
			keyActions = append(keyActions, map[string]interface{}{
				"type":  "keyUp",
				"value": bidi.ResolveKey(strings.TrimSpace(parts[i])),
			})
		}
	}

	params := map[string]interface{}{
		"context": context,
		"actions": []map[string]interface{}{
			{
				"type":    "key",
				"id":      "keyboard",
				"actions": keyActions,
			},
		},
	}

	_, err := r.sendInternalCommand(session, "input.performActions", params)
	return err
}

// isChecked runs JS to check if an element is checked (for checkboxes/radios).
func (r *Router) isChecked(session *BrowserSession, context string, ep elementParams) (bool, error) {
	script, args := buildIsCheckedScript(ep)

	params := map[string]interface{}{
		"functionDeclaration": script,
		"target":              map[string]interface{}{"context": context},
		"arguments":           args,
		"awaitPromise":        false,
		"resultOwnership":     "root",
	}

	resp, err := r.sendInternalCommand(session, "script.callFunction", params)
	if err != nil {
		return false, err
	}

	val, err := parseScriptResult(resp)
	if err != nil {
		return false, err
	}

	return val == "true", nil
}

// --- Script builders for JS-based interactions ---

// buildIsCheckedScript builds a JS function to check if an element is checked.
func buildIsCheckedScript(ep elementParams) (string, []map[string]interface{}) {
	args := []map[string]interface{}{
		{"type": "string", "value": ep.Scope},
		{"type": "string", "value": ep.Selector},
		{"type": "number", "value": ep.Index},
		{"type": "boolean", "value": ep.HasIndex},
	}

	script := `
		(scope, selector, index, hasIndex) => {
			const root = scope ? document.querySelector(scope) : document;
			if (!root) return 'false';
			let el;
			if (hasIndex) {
				const all = root.querySelectorAll(selector);
				el = all[index];
			} else {
				el = root.querySelector(selector);
			}
			if (!el) return 'false';
			return el.checked ? 'true' : 'false';
		}
	`
	return script, args
}

// buildSelectOptionScript builds a JS function to set a select element's value.
func buildSelectOptionScript(ep elementParams, value string) (string, []map[string]interface{}) {
	args := []map[string]interface{}{
		{"type": "string", "value": ep.Scope},
		{"type": "string", "value": ep.Selector},
		{"type": "number", "value": ep.Index},
		{"type": "boolean", "value": ep.HasIndex},
		{"type": "string", "value": value},
	}

	script := `
		(scope, selector, index, hasIndex, value) => {
			const root = scope ? document.querySelector(scope) : document;
			if (!root) return 'element not found';
			let el;
			if (hasIndex) {
				const all = root.querySelectorAll(selector);
				el = all[index];
			} else {
				el = root.querySelector(selector);
			}
			if (!el) return 'element not found';
			el.value = value;
			el.dispatchEvent(new Event('input', { bubbles: true }));
			el.dispatchEvent(new Event('change', { bubbles: true }));
			return 'ok';
		}
	`
	return script, args
}

// buildSetValueScript builds a JS function to set an element's value and dispatch events.
func buildSetValueScript(ep elementParams, value string) (string, []map[string]interface{}) {
	args := []map[string]interface{}{
		{"type": "string", "value": ep.Scope},
		{"type": "string", "value": ep.Selector},
		{"type": "number", "value": ep.Index},
		{"type": "boolean", "value": ep.HasIndex},
		{"type": "string", "value": value},
	}

	script := `
		(scope, selector, index, hasIndex, value) => {
			const root = scope ? document.querySelector(scope) : document;
			if (!root) return 'element not found';
			let el;
			if (hasIndex) {
				const all = root.querySelectorAll(selector);
				el = all[index];
			} else {
				el = root.querySelector(selector);
			}
			if (!el) return 'element not found';
			el.focus();
			const nativeSetter = Object.getOwnPropertyDescriptor(
				window.HTMLInputElement.prototype, 'value'
			)?.set || Object.getOwnPropertyDescriptor(
				window.HTMLTextAreaElement.prototype, 'value'
			)?.set;
			if (nativeSetter) {
				nativeSetter.call(el, value);
			} else {
				el.value = value;
			}
			el.dispatchEvent(new Event('input', { bubbles: true }));
			el.dispatchEvent(new Event('change', { bubbles: true }));
			return 'ok';
		}
	`
	return script, args
}

// buildFocusScript builds a JS function to focus an element.
func buildFocusScript(ep elementParams) (string, []map[string]interface{}) {
	args := []map[string]interface{}{
		{"type": "string", "value": ep.Scope},
		{"type": "string", "value": ep.Selector},
		{"type": "number", "value": ep.Index},
		{"type": "boolean", "value": ep.HasIndex},
	}

	script := `
		(scope, selector, index, hasIndex) => {
			const root = scope ? document.querySelector(scope) : document;
			if (!root) return 'not found';
			let el;
			if (hasIndex) {
				const all = root.querySelectorAll(selector);
				el = all[index];
			} else {
				el = root.querySelector(selector);
			}
			if (!el) return 'not found';
			el.focus();
			return 'ok';
		}
	`
	return script, args
}

// buildDispatchEventScript builds a JS function to dispatch an event on an element.
func buildDispatchEventScript(ep elementParams, eventType, initJSON string) (string, []map[string]interface{}) {
	args := []map[string]interface{}{
		{"type": "string", "value": ep.Scope},
		{"type": "string", "value": ep.Selector},
		{"type": "number", "value": ep.Index},
		{"type": "boolean", "value": ep.HasIndex},
		{"type": "string", "value": eventType},
		{"type": "string", "value": initJSON},
	}

	script := `
		(scope, selector, index, hasIndex, eventType, initJSON) => {
			const root = scope ? document.querySelector(scope) : document;
			if (!root) return 'not found';
			let el;
			if (hasIndex) {
				const all = root.querySelectorAll(selector);
				el = all[index];
			} else {
				el = root.querySelector(selector);
			}
			if (!el) return 'not found';
			const init = JSON.parse(initJSON);
			el.dispatchEvent(new Event(eventType, init));
			return 'ok';
		}
	`
	return script, args
}

