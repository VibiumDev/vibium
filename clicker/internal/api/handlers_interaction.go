package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vibium/clicker/internal/bidi"
)

// handleVibiumClick handles the vibium:element.click command with actionability checks.
// Supports index param for elements from findAll().
func (r *Router) handleVibiumClick(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	s := NewAPISession(r, session, context)
	info, err := resolveWithActionability(s, context, ep, ClickChecks)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.captureBeforeSnapshotAfterScroll(session, cmd.Params)
	if err := ClickAtCenter(s, context, info); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"clicked": true})
}

// handleVibiumDblclick handles the vibium:element.dblclick command.
func (r *Router) handleVibiumDblclick(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	s := NewAPISession(r, session, context)
	info, err := resolveWithActionability(s, context, ep, ClickChecks)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.captureBeforeSnapshotAfterScroll(session, cmd.Params)
	if err := DblClickAtCenter(s, context, info); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"dblclicked": true})
}

// handleVibiumFill handles the vibium:element.fill command.
// Uses JS to set the element value, then dispatches input/change events.
func (r *Router) handleVibiumFill(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)
	value, _ := cmd.Params["value"].(string)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	s := NewAPISession(r, session, context)
	if _, err := resolveWithActionability(s, context, ep, FillChecks); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.captureBeforeSnapshotAfterScroll(session, cmd.Params)
	script, args := buildSetValueScript(ep, value)
	resp, err := CallScript(s, context, script, args)
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

// handleVibiumType handles the vibium:element.type command with actionability checks.
// Clicks to focus and types text (does NOT clear first).
func (r *Router) handleVibiumType(session *BrowserSession, cmd bidiCommand) {
	// Extract text-to-type BEFORE ExtractElementParams, since "text" is also
	// a semantic selector param. Remove it from params to avoid collision.
	text, _ := cmd.Params["text"].(string)
	paramsCopy := make(map[string]interface{})
	for k, v := range cmd.Params {
		if k != "text" {
			paramsCopy[k] = v
		}
	}
	ep := ExtractElementParams(paramsCopy)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	s := NewAPISession(r, session, context)
	info, err := resolveWithActionability(s, context, ep, ClickChecks)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.captureBeforeSnapshotAfterScroll(session, cmd.Params)
	if err := ClickAtCenter(s, context, info); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	if err := TypeText(s, context, text); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"typed": true})
}

// handleVibiumPress handles the vibium:element.press command.
// Clicks to focus, then presses a key (supports combos like "Control+a").
func (r *Router) handleVibiumPress(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)
	key, _ := cmd.Params["key"].(string)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	s := NewAPISession(r, session, context)
	info, err := resolveWithActionability(s, context, ep, ClickChecks)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.captureBeforeSnapshotAfterScroll(session, cmd.Params)
	if err := ClickAtCenter(s, context, info); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	if err := PressKey(s, context, key); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"pressed": true})
}

// handleVibiumClear handles the vibium:element.clear command.
// Uses JS to clear the element value.
func (r *Router) handleVibiumClear(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	s := NewAPISession(r, session, context)
	if _, err := resolveWithActionability(s, context, ep, FillChecks); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.captureBeforeSnapshotAfterScroll(session, cmd.Params)
	script, args := buildSetValueScript(ep, "")
	resp, err := CallScript(s, context, script, args)
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

// handleVibiumCheck handles the vibium:element.check command.
// Clicks the checkbox only if it's not already checked.
func (r *Router) handleVibiumCheck(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	s := NewAPISession(r, session, context)
	info, err := resolveWithActionability(s, context, ep, ClickChecks)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.captureBeforeSnapshotAfterScroll(session, cmd.Params)
	checked, err := IsChecked(s, context, ep)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	if !checked {
		if err := ClickAtCenter(s, context, info); err != nil {
			r.sendError(session, cmd.ID, err)
			return
		}
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"checked": true})
}

// handleVibiumUncheck handles the vibium:element.uncheck command.
// Clicks the checkbox only if it's currently checked.
func (r *Router) handleVibiumUncheck(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	s := NewAPISession(r, session, context)
	info, err := resolveWithActionability(s, context, ep, ClickChecks)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.captureBeforeSnapshotAfterScroll(session, cmd.Params)
	checked, err := IsChecked(s, context, ep)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	if checked {
		if err := ClickAtCenter(s, context, info); err != nil {
			r.sendError(session, cmd.ID, err)
			return
		}
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"unchecked": true})
}

// handleVibiumSelectOption handles the vibium:element.selectOption command.
// Sets the value of a <select> element and dispatches a change event.
func (r *Router) handleVibiumSelectOption(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)
	value, _ := cmd.Params["value"].(string)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	s := NewAPISession(r, session, context)
	if _, err := resolveWithActionability(s, context, ep, SelectChecks); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.captureBeforeSnapshotAfterScroll(session, cmd.Params)
	script, args := buildSelectOptionScript(ep, value)
	resp, err := CallScript(s, context, script, args)
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

// handleVibiumHover handles the vibium:element.hover command.
// Moves the mouse pointer to the element's center without clicking.
func (r *Router) handleVibiumHover(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	s := NewAPISession(r, session, context)
	info, err := resolveWithActionability(s, context, ep, HoverChecks)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.captureBeforeSnapshotAfterScroll(session, cmd.Params)
	if err := HoverAtCenter(s, context, info); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"hovered": true})
}

// handleVibiumFocus handles the vibium:element.focus command.
// Runs element.focus() via JavaScript.
func (r *Router) handleVibiumFocus(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	s := NewAPISession(r, session, context)
	if err := FocusElement(s, context, ep); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"focused": true})
}

// handleVibiumDragTo handles the vibium:element.dragTo command.
// Resolves source and target elements, then performs pointer drag.
func (r *Router) handleVibiumDragTo(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Extract target params
	targetParams, ok := cmd.Params["target"].(map[string]interface{})
	if !ok {
		r.sendError(session, cmd.ID, fmt.Errorf("dragTo requires 'target' parameter"))
		return
	}
	targetEp := ExtractElementParams(targetParams)

	s := NewAPISession(r, session, context)
	srcInfo, err := resolveWithActionability(s, context, ep, HoverChecks)
	if err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("source: %w", err))
		return
	}
	r.captureBeforeSnapshotAfterScroll(session, cmd.Params)
	targetInfo, err := resolveWithActionability(s, context, targetEp, HoverChecks)
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

	if _, err := s.SendBidiCommand("input.performActions", dragParams); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"dragged": true})
}

// handleVibiumTap handles the vibium:element.tap command.
// Performs a touch tap at the element's center.
func (r *Router) handleVibiumTap(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	s := NewAPISession(r, session, context)
	info, err := resolveWithActionability(s, context, ep, ClickChecks)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.captureBeforeSnapshotAfterScroll(session, cmd.Params)
	if err := TapAtCenter(s, context, info); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"tapped": true})
}

// handleVibiumScrollIntoView handles the vibium:element.scrollIntoView command.
// Resolves the element (which auto-scrolls it into view).
func (r *Router) handleVibiumScrollIntoView(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	s := NewAPISession(r, session, context)
	if err := ScrollIntoView(s, context, ep); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"scrolled": true})
}

// handleVibiumDispatchEvent handles the vibium:element.dispatchEvent command.
// Dispatches a DOM event on the element via JavaScript.
func (r *Router) handleVibiumDispatchEvent(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)
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

// handleVibiumElSetFiles handles the vibium:element.setFiles command.
// Sets files on an <input type="file"> element using BiDi input.setFiles.
func (r *Router) handleVibiumElSetFiles(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)

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
func (r *Router) clickAtCenter(session *BrowserSession, context string, info *ElementInfo) error {
	return ClickAtCenter(NewAPISession(r, session, context), context, info)
}

// typeText types a string of text using keyboard events.
func (r *Router) typeText(session *BrowserSession, context, text string) error {
	return TypeText(NewAPISession(r, session, context), context, text)
}

// pressKey presses a key or key combo (e.g. "Enter", "Control+a").
func (r *Router) pressKey(session *BrowserSession, context, key string) error {
	return PressKey(NewAPISession(r, session, context), context, key)
}

// ---------------------------------------------------------------------------
// Exported standalone input primitives — usable from both proxy and MCP.
// ---------------------------------------------------------------------------

// ClickAtCenter performs a mouse click at the center of an element.
func ClickAtCenter(s Session, context string, info *ElementInfo) error {
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

	_, err := s.SendBidiCommand("input.performActions", clickParams)
	return err
}

// DblClickAtCenter performs a double-click at the center of an element.
func DblClickAtCenter(s Session, context string, info *ElementInfo) error {
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

	_, err := s.SendBidiCommand("input.performActions", dblclickParams)
	return err
}

// TypeText types a string of text using keyboard events.
func TypeText(s Session, context, text string) error {
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

	_, err := s.SendBidiCommand("input.performActions", typeParams)
	return err
}

// PressKey presses a key or key combo (e.g. "Enter", "Control+a").
func PressKey(s Session, context, key string) error {
	parts := strings.Split(key, "+")
	keyActions := make([]map[string]interface{}, 0)

	if len(parts) == 1 {
		resolved := bidi.ResolveKey(parts[0])
		keyActions = append(keyActions,
			map[string]interface{}{"type": "keyDown", "value": resolved},
			map[string]interface{}{"type": "keyUp", "value": resolved},
		)
	} else {
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

	_, err := s.SendBidiCommand("input.performActions", params)
	return err
}

// isChecked runs JS to check if an element is checked (for checkboxes/radios).
func (r *Router) isChecked(session *BrowserSession, context string, ep ElementParams) (bool, error) {
	return IsChecked(NewAPISession(r, session, context), context, ep)
}

// ---------------------------------------------------------------------------
// Exported standalone composite functions — usable from both proxy and MCP.
// ---------------------------------------------------------------------------

// Click resolves an element with actionability checks and clicks at its center.
func Click(s Session, context string, ep ElementParams) error {
	info, err := resolveWithActionability(s, context, ep, ClickChecks)
	if err != nil {
		return err
	}
	return ClickAtCenter(s, context, info)
}

// DblClick resolves an element with actionability checks and double-clicks at its center.
func DblClick(s Session, context string, ep ElementParams) error {
	info, err := resolveWithActionability(s, context, ep, ClickChecks)
	if err != nil {
		return err
	}
	return DblClickAtCenter(s, context, info)
}

// Hover resolves an element with actionability checks and moves the mouse to its center.
func Hover(s Session, context string, ep ElementParams) error {
	info, err := resolveWithActionability(s, context, ep, HoverChecks)
	if err != nil {
		return err
	}
	return HoverAtCenter(s, context, info)
}

// HoverAtCenter moves the mouse to the center of an element without clicking.
func HoverAtCenter(s Session, context string, info *ElementInfo) error {
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

	_, err := s.SendBidiCommand("input.performActions", hoverParams)
	return err
}

// TapAtCenter performs a touch tap at the center of an element.
func TapAtCenter(s Session, context string, info *ElementInfo) error {
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

	_, err := s.SendBidiCommand("input.performActions", tapParams)
	return err
}

// Fill resolves an element with actionability checks and sets its value via JS.
func Fill(s Session, context string, ep ElementParams, value string) error {
	if _, err := resolveWithActionability(s, context, ep, FillChecks); err != nil {
		return err
	}
	script, args := buildSetValueScript(ep, value)
	resp, err := CallScript(s, context, script, args)
	if err != nil {
		return err
	}
	val, err := parseScriptResult(resp)
	if err != nil {
		return fmt.Errorf("fill failed: %w", err)
	}
	if val != "ok" {
		return fmt.Errorf("fill: %s", val)
	}
	return nil
}

// TypeInto resolves an element with actionability checks, clicks to focus, and types text.
func TypeInto(s Session, context string, ep ElementParams, text string) error {
	info, err := resolveWithActionability(s, context, ep, ClickChecks)
	if err != nil {
		return err
	}
	if err := ClickAtCenter(s, context, info); err != nil {
		return err
	}
	return TypeText(s, context, text)
}

// PressOn resolves an element with actionability checks, clicks to focus, and presses a key.
func PressOn(s Session, context string, ep ElementParams, key string) error {
	info, err := resolveWithActionability(s, context, ep, ClickChecks)
	if err != nil {
		return err
	}
	if err := ClickAtCenter(s, context, info); err != nil {
		return err
	}
	return PressKey(s, context, key)
}

// Check resolves a checkbox with actionability checks and clicks it only if not already checked.
func Check(s Session, context string, ep ElementParams) (bool, error) {
	info, err := resolveWithActionability(s, context, ep, ClickChecks)
	if err != nil {
		return false, err
	}
	checked, err := IsChecked(s, context, ep)
	if err != nil {
		return false, err
	}
	if !checked {
		if err := ClickAtCenter(s, context, info); err != nil {
			return false, err
		}
		return true, nil // was toggled
	}
	return false, nil // already checked
}

// Uncheck resolves a checkbox with actionability checks and clicks it only if currently checked.
func Uncheck(s Session, context string, ep ElementParams) (bool, error) {
	info, err := resolveWithActionability(s, context, ep, ClickChecks)
	if err != nil {
		return false, err
	}
	checked, err := IsChecked(s, context, ep)
	if err != nil {
		return false, err
	}
	if checked {
		if err := ClickAtCenter(s, context, info); err != nil {
			return false, err
		}
		return true, nil // was toggled
	}
	return false, nil // already unchecked
}

// IsChecked checks if a checkbox/radio element is checked.
func IsChecked(s Session, context string, ep ElementParams) (bool, error) {
	script, args := buildIsCheckedScript(ep)
	resp, err := CallScript(s, context, script, args)
	if err != nil {
		return false, err
	}
	val, err := parseScriptResult(resp)
	if err != nil {
		return false, err
	}
	return val == "true", nil
}

// SelectOption resolves a select element with actionability checks and sets its value.
func SelectOption(s Session, context string, ep ElementParams, value string) error {
	if _, err := resolveWithActionability(s, context, ep, SelectChecks); err != nil {
		return err
	}
	script, args := buildSelectOptionScript(ep, value)
	resp, err := CallScript(s, context, script, args)
	if err != nil {
		return err
	}
	val, err := parseScriptResult(resp)
	if err != nil {
		return fmt.Errorf("selectOption failed: %w", err)
	}
	if val != "ok" {
		return fmt.Errorf("selectOption: %s", val)
	}
	return nil
}

// FocusElement resolves an element and focuses it via JS.
func FocusElement(s Session, context string, ep ElementParams) error {
	if _, err := ResolveElement(s, context, ep); err != nil {
		return err
	}
	script, args := buildFocusScript(ep)
	_, err := CallScript(s, context, script, args)
	return err
}

// ScrollIntoView resolves an element with stability check, which auto-scrolls it into view.
func ScrollIntoView(s Session, context string, ep ElementParams) error {
	_, err := resolveWithActionability(s, context, ep, ScrollChecks)
	return err
}

// Tap resolves an element with actionability checks and performs a touch tap at its center.
func Tap(s Session, context string, ep ElementParams) error {
	info, err := resolveWithActionability(s, context, ep, ClickChecks)
	if err != nil {
		return err
	}
	return TapAtCenter(s, context, info)
}

// DragTo resolves source and target elements with actionability checks and drags from one to the other.
func DragTo(s Session, context string, source, target ElementParams) error {
	srcInfo, err := resolveWithActionability(s, context, source, HoverChecks)
	if err != nil {
		return fmt.Errorf("source: %w", err)
	}
	targetInfo, err := resolveWithActionability(s, context, target, HoverChecks)
	if err != nil {
		return fmt.Errorf("target: %w", err)
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

	_, err = s.SendBidiCommand("input.performActions", dragParams)
	return err
}

// ScrollWheel performs a mouse wheel scroll at the given coordinates.
func ScrollWheel(s Session, context string, x, y, deltaX, deltaY int) error {
	scrollParams := map[string]interface{}{
		"context": context,
		"actions": []map[string]interface{}{
			{
				"type": "wheel",
				"id":   "wheel",
				"actions": []map[string]interface{}{
					{
						"type":    "scroll",
						"x":      x,
						"y":      y,
						"deltaX": deltaX,
						"deltaY": deltaY,
					},
				},
			},
		},
	}

	_, err := s.SendBidiCommand("input.performActions", scrollParams)
	return err
}

// --- Script builders for JS-based interactions ---

// buildIsCheckedScript builds a JS function to check if an element is checked.
func buildIsCheckedScript(ep ElementParams) (string, []map[string]interface{}) {
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
func buildSelectOptionScript(ep ElementParams, value string) (string, []map[string]interface{}) {
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
			if (el.tagName !== 'SELECT') return 'not a <select> element';
			// Match by option value, then fall back to the visible label/text so
			// passing "California" (for <option value="CA">California</option>)
			// works. Without this, an unmatched value silently no-ops (issue #140).
			const opts = Array.from(el.options || []);
			const match = opts.find(o => o.value === value)
				|| opts.find(o => ((o.label || o.text || '').trim() === value));
			if (!match) return 'no <option> matches "' + value + '"';
			el.value = match.value;
			el.dispatchEvent(new Event('input', { bubbles: true }));
			el.dispatchEvent(new Event('change', { bubbles: true }));
			return 'ok';
		}
	`
	return script, args
}

// buildSetValueScript builds a JS function to set an element's value and dispatch events.
func buildSetValueScript(ep ElementParams, value string) (string, []map[string]interface{}) {
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
			// Pick the native value setter for the element's ACTUAL type. Calling
			// the HTMLInputElement setter on a <textarea> throws "Illegal
			// invocation" because the setter validates its receiver (issue #117).
			const proto = (el instanceof window.HTMLTextAreaElement)
				? window.HTMLTextAreaElement.prototype
				: (el instanceof window.HTMLInputElement)
					? window.HTMLInputElement.prototype
					: null;
			const nativeSetter = proto
				? Object.getOwnPropertyDescriptor(proto, 'value')?.set
				: null;
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
func buildFocusScript(ep ElementParams) (string, []map[string]interface{}) {
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
func buildDispatchEventScript(ep ElementParams, eventType, initJSON string) (string, []map[string]interface{}) {
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

