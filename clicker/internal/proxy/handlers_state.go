package proxy

import (
	"encoding/json"
	"fmt"
	"time"
)

// handleVibiumElText handles vibium:el.text — returns element.textContent.
func (r *Router) handleVibiumElText(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	script, args := buildElStateScript(ep, `(el.textContent || '').trim()`)
	val, err := r.evalElementScript(session, context, script, args)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{"text": val})
}

// handleVibiumElInnerText handles vibium:el.innerText — returns element.innerText.
func (r *Router) handleVibiumElInnerText(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	script, args := buildElStateScript(ep, `(el.innerText || '').trim()`)
	val, err := r.evalElementScript(session, context, script, args)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{"text": val})
}

// handleVibiumElHTML handles vibium:el.html — returns element.innerHTML.
func (r *Router) handleVibiumElHTML(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	script, args := buildElStateScript(ep, `el.innerHTML`)
	val, err := r.evalElementScript(session, context, script, args)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{"html": val})
}

// handleVibiumElValue handles vibium:el.value — returns element.value (for inputs).
func (r *Router) handleVibiumElValue(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	script, args := buildElStateScript(ep, `el.value || ''`)
	val, err := r.evalElementScript(session, context, script, args)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{"value": val})
}

// handleVibiumElAttr handles vibium:el.attr — returns element.getAttribute(name).
func (r *Router) handleVibiumElAttr(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)
	name, _ := cmd.Params["name"].(string)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	args := buildElBaseArgs(ep)
	args = append(args, map[string]interface{}{"type": "string", "value": name})
	script := `
		(scope, selector, index, hasIndex, name) => {
			const root = scope ? document.querySelector(scope) : document;
			if (!root) return JSON.stringify({error: 'root not found'});
			let el;
			if (hasIndex) {
				el = root.querySelectorAll(selector)[index];
			} else {
				el = root.querySelector(selector);
			}
			if (!el) return JSON.stringify({error: 'element not found'});
			const v = el.getAttribute(name);
			return JSON.stringify({value: v});
		}
	`

	resp, err := r.sendInternalCommand(session, "script.callFunction", map[string]interface{}{
		"functionDeclaration": script,
		"target":              map[string]interface{}{"context": context},
		"arguments":           args,
		"awaitPromise":        false,
		"resultOwnership":     "root",
	})
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	val, err := parseScriptResult(resp)
	if err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("attr failed: %w", err))
		return
	}

	var result struct {
		Value *string `json:"value"`
		Error string  `json:"error"`
	}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("attr parse failed: %w", err))
		return
	}
	if result.Error != "" {
		r.sendError(session, cmd.ID, fmt.Errorf("attr: %s", result.Error))
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{"value": result.Value})
}

// handleVibiumElBounds handles vibium:el.bounds — returns getBoundingClientRect().
func (r *Router) handleVibiumElBounds(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	script, args := buildElJSONScript(ep, `
		const rect = el.getBoundingClientRect();
		return JSON.stringify({x: rect.x, y: rect.y, width: rect.width, height: rect.height});
	`)

	resp, err := r.sendInternalCommand(session, "script.callFunction", map[string]interface{}{
		"functionDeclaration": script,
		"target":              map[string]interface{}{"context": context},
		"arguments":           args,
		"awaitPromise":        false,
		"resultOwnership":     "root",
	})
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	val, err := parseScriptResult(resp)
	if err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("bounds failed: %w", err))
		return
	}

	var box boxInfo
	if err := json.Unmarshal([]byte(val), &box); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("bounds parse failed: %w", err))
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{
		"x": box.X, "y": box.Y, "width": box.Width, "height": box.Height,
	})
}

// handleVibiumElIsVisible handles vibium:el.isVisible — checks computed visibility.
func (r *Router) handleVibiumElIsVisible(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	script, args := buildElBoolScript(ep, `
		const style = window.getComputedStyle(el);
		if (style.display === 'none') return false;
		if (style.visibility === 'hidden') return false;
		if (parseFloat(style.opacity) === 0) return false;
		const rect = el.getBoundingClientRect();
		return rect.width > 0 && rect.height > 0;
	`)

	visible, err := r.evalBoolScript(session, context, script, args)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{"visible": visible})
}

// handleVibiumElIsHidden handles vibium:el.isHidden — inverse of isVisible.
func (r *Router) handleVibiumElIsHidden(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	script, args := buildElBoolScript(ep, `
		const style = window.getComputedStyle(el);
		if (style.display === 'none') return true;
		if (style.visibility === 'hidden') return true;
		if (parseFloat(style.opacity) === 0) return true;
		const rect = el.getBoundingClientRect();
		return rect.width === 0 || rect.height === 0;
	`)

	hidden, err := r.evalBoolScript(session, context, script, args)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{"hidden": hidden})
}

// handleVibiumElIsEnabled handles vibium:el.isEnabled — checks !element.disabled.
func (r *Router) handleVibiumElIsEnabled(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	script, args := buildElBoolScript(ep, `return !el.disabled;`)
	enabled, err := r.evalBoolScript(session, context, script, args)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{"enabled": enabled})
}

// handleVibiumElIsChecked handles vibium:el.isChecked — returns element.checked.
func (r *Router) handleVibiumElIsChecked(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	script, args := buildElBoolScript(ep, `return !!el.checked;`)
	checked, err := r.evalBoolScript(session, context, script, args)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{"checked": checked})
}

// handleVibiumElIsEditable handles vibium:el.isEditable — not disabled and not readonly.
func (r *Router) handleVibiumElIsEditable(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	script, args := buildElBoolScript(ep, `return !el.disabled && !el.readOnly;`)
	editable, err := r.evalBoolScript(session, context, script, args)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{"editable": editable})
}

// handleVibiumElEval handles vibium:el.eval — runs script.callFunction with element.
func (r *Router) handleVibiumElEval(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)
	fn, _ := cmd.Params["fn"].(string)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	args := buildElBaseArgs(ep)
	args = append(args, map[string]interface{}{"type": "string", "value": fn})
	script := `
		(scope, selector, index, hasIndex, fn) => {
			const root = scope ? document.querySelector(scope) : document;
			if (!root) return JSON.stringify({error: 'root not found'});
			let el;
			if (hasIndex) {
				el = root.querySelectorAll(selector)[index];
			} else {
				el = root.querySelector(selector);
			}
			if (!el) return JSON.stringify({error: 'element not found'});
			const userFn = new Function('el', fn);
			const result = userFn(el);
			return JSON.stringify({value: result});
		}
	`

	resp, err := r.sendInternalCommand(session, "script.callFunction", map[string]interface{}{
		"functionDeclaration": script,
		"target":              map[string]interface{}{"context": context},
		"arguments":           args,
		"awaitPromise":        false,
		"resultOwnership":     "root",
	})
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	val, err := parseScriptResult(resp)
	if err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("eval failed: %w", err))
		return
	}

	var result struct {
		Value interface{} `json:"value"`
		Error string      `json:"error"`
	}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("eval parse failed: %w", err))
		return
	}
	if result.Error != "" {
		r.sendError(session, cmd.ID, fmt.Errorf("eval: %s", result.Error))
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{"value": result.Value})
}

// handleVibiumElScreenshot handles vibium:el.screenshot — captures element screenshot.
func (r *Router) handleVibiumElScreenshot(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Resolve element to get bounding box (also scrolls into view)
	info, err := r.resolveElement(session, context, ep)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Use browsingContext.captureScreenshot with clip
	clipParams := map[string]interface{}{
		"context": context,
		"clip": map[string]interface{}{
			"type":   "box",
			"x":      info.Box.X,
			"y":      info.Box.Y,
			"width":  info.Box.Width,
			"height": info.Box.Height,
		},
	}

	resp, err := r.sendInternalCommand(session, "browsingContext.captureScreenshot", clipParams)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	if bidiErr := checkBidiError(resp); bidiErr != nil {
		r.sendError(session, cmd.ID, bidiErr)
		return
	}

	var ssResult struct {
		Result struct {
			Data string `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &ssResult); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("screenshot parse failed: %w", err))
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"data": ssResult.Result.Data})
}

// handleVibiumElWaitFor handles vibium:el.waitFor — waits for element state.
// Supported states: "visible", "hidden", "attached", "detached".
func (r *Router) handleVibiumElWaitFor(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)
	state, _ := cmd.Params["state"].(string)
	if state == "" {
		state = "visible"
	}
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	deadline := time.Now().Add(ep.Timeout)
	interval := 100 * time.Millisecond

	for {
		var met bool
		var checkErr error

		switch state {
		case "attached":
			// Element exists in DOM
			_, findErr := r.resolveElementNoWait(session, context, ep)
			met = findErr == nil
		case "detached":
			// Element does NOT exist in DOM
			_, findErr := r.resolveElementNoWait(session, context, ep)
			met = findErr != nil
		case "visible":
			info, findErr := r.resolveElementNoWait(session, context, ep)
			if findErr == nil {
				script, args := buildElBoolScript(ep, `
					const style = window.getComputedStyle(el);
					if (style.display === 'none') return false;
					if (style.visibility === 'hidden') return false;
					if (parseFloat(style.opacity) === 0) return false;
					const rect = el.getBoundingClientRect();
					return rect.width > 0 && rect.height > 0;
				`)
				_ = info
				met, checkErr = r.evalBoolScript(session, context, script, args)
			}
		case "hidden":
			_, findErr := r.resolveElementNoWait(session, context, ep)
			if findErr != nil {
				// Element not found = hidden
				met = true
			} else {
				script, args := buildElBoolScript(ep, `
					const style = window.getComputedStyle(el);
					if (style.display === 'none') return true;
					if (style.visibility === 'hidden') return true;
					if (parseFloat(style.opacity) === 0) return true;
					const rect = el.getBoundingClientRect();
					return rect.width === 0 || rect.height === 0;
				`)
				met, checkErr = r.evalBoolScript(session, context, script, args)
			}
		default:
			r.sendError(session, cmd.ID, fmt.Errorf("unknown state: %s (expected visible, hidden, attached, detached)", state))
			return
		}

		if checkErr == nil && met {
			r.sendSuccess(session, cmd.ID, map[string]interface{}{"state": state})
			return
		}

		if time.Now().After(deadline) {
			r.sendError(session, cmd.ID, fmt.Errorf("timeout waiting for element to be %s", state))
			return
		}

		time.Sleep(interval)
	}
}

// --- Page-level waiting handlers ---

// handlePageWaitFor handles vibium:page.waitFor — waits for a selector to appear.
func (r *Router) handlePageWaitFor(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Use the standard element resolution with polling
	info, err := r.resolveElement(session, context, ep)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{
		"tag":  info.Tag,
		"text": info.Text,
		"box": map[string]interface{}{
			"x":      info.Box.X,
			"y":      info.Box.Y,
			"width":  info.Box.Width,
			"height": info.Box.Height,
		},
	})
}

// handlePageWait handles vibium:page.wait — client-side delay.
func (r *Router) handlePageWait(session *BrowserSession, cmd bidiCommand) {
	ms, _ := cmd.Params["ms"].(float64)
	if ms <= 0 {
		r.sendSuccess(session, cmd.ID, map[string]interface{}{"waited": true})
		return
	}

	time.Sleep(time.Duration(ms) * time.Millisecond)
	r.sendSuccess(session, cmd.ID, map[string]interface{}{"waited": true})
}

// handlePageWaitForFunction handles vibium:page.waitForFunction — polls script.evaluate.
func (r *Router) handlePageWaitForFunction(session *BrowserSession, cmd bidiCommand) {
	fn, _ := cmd.Params["fn"].(string)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	timeoutMs, _ := cmd.Params["timeout"].(float64)
	timeout := defaultTimeout
	if timeoutMs > 0 {
		timeout = time.Duration(timeoutMs) * time.Millisecond
	}

	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond

	for {
		resp, err := r.sendInternalCommand(session, "script.callFunction", map[string]interface{}{
			"functionDeclaration": fn,
			"target":              map[string]interface{}{"context": context},
			"arguments":           []map[string]interface{}{},
			"awaitPromise":        true,
			"resultOwnership":     "root",
		})
		if err == nil {
			var result struct {
				Result struct {
					Result struct {
						Type  string      `json:"type"`
						Value interface{} `json:"value"`
					} `json:"result"`
				} `json:"result"`
			}
			if err := json.Unmarshal(resp, &result); err == nil {
				// Truthy check: non-null, non-undefined, non-false, non-zero, non-empty-string
				res := result.Result.Result
				truthy := false
				switch res.Type {
				case "boolean":
					truthy = res.Value == true
				case "number":
					if v, ok := res.Value.(float64); ok {
						truthy = v != 0
					}
				case "string":
					if v, ok := res.Value.(string); ok {
						truthy = v != ""
					}
				case "null", "undefined":
					truthy = false
				default:
					truthy = res.Value != nil
				}
				if truthy {
					r.sendSuccess(session, cmd.ID, map[string]interface{}{"value": res.Value})
					return
				}
			}
		}

		if time.Now().After(deadline) {
			r.sendError(session, cmd.ID, fmt.Errorf("timeout waiting for function to return truthy"))
			return
		}

		time.Sleep(interval)
	}
}

// --- Script builder helpers for state queries ---

// buildElBaseArgs returns the standard [scope, selector, index, hasIndex] args.
func buildElBaseArgs(ep elementParams) []map[string]interface{} {
	return []map[string]interface{}{
		{"type": "string", "value": ep.Scope},
		{"type": "string", "value": ep.Selector},
		{"type": "number", "value": ep.Index},
		{"type": "boolean", "value": ep.HasIndex},
	}
}

// buildElStateScript builds a script that finds an element and evaluates an expression.
// The expression receives `el` as the found element and should return a string.
func buildElStateScript(ep elementParams, expr string) (string, []map[string]interface{}) {
	args := buildElBaseArgs(ep)
	script := fmt.Sprintf(`
		(scope, selector, index, hasIndex) => {
			const root = scope ? document.querySelector(scope) : document;
			if (!root) return null;
			let el;
			if (hasIndex) {
				el = root.querySelectorAll(selector)[index];
			} else {
				el = root.querySelector(selector);
			}
			if (!el) return null;
			return %s;
		}
	`, expr)
	return script, args
}

// buildElBoolScript builds a script that finds an element and evaluates a boolean expression.
// The body receives `el` and should use `return true/false;`.
func buildElBoolScript(ep elementParams, body string) (string, []map[string]interface{}) {
	args := buildElBaseArgs(ep)
	script := fmt.Sprintf(`
		(scope, selector, index, hasIndex) => {
			const root = scope ? document.querySelector(scope) : document;
			if (!root) return 'error:root not found';
			let el;
			if (hasIndex) {
				el = root.querySelectorAll(selector)[index];
			} else {
				el = root.querySelector(selector);
			}
			if (!el) return 'error:element not found';
			const _check = (el) => { %s };
			return _check(el) ? 'true' : 'false';
		}
	`, body)
	return script, args
}

// buildElJSONScript builds a script that finds an element and returns JSON.
// The body receives `el` and should use `return JSON.stringify(...)`.
func buildElJSONScript(ep elementParams, body string) (string, []map[string]interface{}) {
	args := buildElBaseArgs(ep)
	script := fmt.Sprintf(`
		(scope, selector, index, hasIndex) => {
			const root = scope ? document.querySelector(scope) : document;
			if (!root) return JSON.stringify({error: 'root not found'});
			let el;
			if (hasIndex) {
				el = root.querySelectorAll(selector)[index];
			} else {
				el = root.querySelector(selector);
			}
			if (!el) return JSON.stringify({error: 'element not found'});
			%s
		}
	`, body)
	return script, args
}

// evalElementScript runs a state script and returns the string result.
func (r *Router) evalElementScript(session *BrowserSession, context, script string, args []map[string]interface{}) (string, error) {
	resp, err := r.sendInternalCommand(session, "script.callFunction", map[string]interface{}{
		"functionDeclaration": script,
		"target":              map[string]interface{}{"context": context},
		"arguments":           args,
		"awaitPromise":        false,
		"resultOwnership":     "root",
	})
	if err != nil {
		return "", err
	}

	val, err := parseScriptResult(resp)
	if err != nil {
		return "", fmt.Errorf("element not found")
	}
	return val, nil
}

// evalBoolScript runs a boolean script and parses the "true"/"false" result.
func (r *Router) evalBoolScript(session *BrowserSession, context, script string, args []map[string]interface{}) (bool, error) {
	resp, err := r.sendInternalCommand(session, "script.callFunction", map[string]interface{}{
		"functionDeclaration": script,
		"target":              map[string]interface{}{"context": context},
		"arguments":           args,
		"awaitPromise":        false,
		"resultOwnership":     "root",
	})
	if err != nil {
		return false, err
	}

	val, err := parseScriptResult(resp)
	if err != nil {
		return false, fmt.Errorf("element not found")
	}

	if len(val) > 6 && val[:6] == "error:" {
		return false, fmt.Errorf(val[6:])
	}

	return val == "true", nil
}

// resolveElementNoWait tries to find an element immediately without polling.
func (r *Router) resolveElementNoWait(session *BrowserSession, context string, ep elementParams) (*elementInfo, error) {
	script, args := buildActionFindScript(ep)

	params := map[string]interface{}{
		"functionDeclaration": script,
		"target":              map[string]interface{}{"context": context},
		"arguments":           args,
		"awaitPromise":        false,
		"resultOwnership":     "root",
	}

	resp, err := r.sendInternalCommand(session, "script.callFunction", params)
	if err != nil {
		return nil, err
	}

	var result struct {
		Result struct {
			Result struct {
				Type  string `json:"type"`
				Value string `json:"value,omitempty"`
			} `json:"result"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse result: %w", err)
	}
	if result.Result.Result.Type != "string" || result.Result.Result.Value == "" {
		return nil, fmt.Errorf("element not found")
	}

	var info elementInfo
	if err := json.Unmarshal([]byte(result.Result.Result.Value), &info); err != nil {
		return nil, fmt.Errorf("failed to parse element info: %w", err)
	}
	return &info, nil
}

// --- Page-level evaluation handlers ---

// handlePageEval handles vibium:page.eval — evaluates a JS expression and returns the result.
func (r *Router) handlePageEval(session *BrowserSession, cmd bidiCommand) {
	expression, _ := cmd.Params["expression"].(string)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	resp, err := r.sendInternalCommand(session, "script.evaluate", map[string]interface{}{
		"expression":      expression,
		"target":          map[string]interface{}{"context": context},
		"awaitPromise":    true,
		"resultOwnership": "none",
	})
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	if bidiErr := checkBidiError(resp); bidiErr != nil {
		r.sendError(session, cmd.ID, bidiErr)
		return
	}

	value, err := deserializeScriptResult(resp)
	if err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("eval failed: %w", err))
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"value": value})
}

// handlePageEvalHandle handles vibium:page.evalHandle — evaluates JS and returns a handle ID.
func (r *Router) handlePageEvalHandle(session *BrowserSession, cmd bidiCommand) {
	expression, _ := cmd.Params["expression"].(string)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	resp, err := r.sendInternalCommand(session, "script.evaluate", map[string]interface{}{
		"expression":      expression,
		"target":          map[string]interface{}{"context": context},
		"awaitPromise":    true,
		"resultOwnership": "root",
	})
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	if bidiErr := checkBidiError(resp); bidiErr != nil {
		r.sendError(session, cmd.ID, bidiErr)
		return
	}

	// Extract the handle from the response
	var result struct {
		Result struct {
			Result struct {
				Handle string `json:"handle"`
			} `json:"result"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("evalHandle parse failed: %w", err))
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"handle": result.Result.Result.Handle})
}

// handlePageAddScript handles vibium:page.addScript — injects a <script> tag.
// Accepts "url" (for external script) or "content" (for inline JS).
func (r *Router) handlePageAddScript(session *BrowserSession, cmd bidiCommand) {
	url, _ := cmd.Params["url"].(string)
	content, _ := cmd.Params["content"].(string)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	var script string
	if url != "" {
		script = `(url) => {
			return new Promise((resolve, reject) => {
				const s = document.createElement('script');
				s.src = url;
				s.onload = () => resolve('ok');
				s.onerror = () => reject(new Error('failed to load script'));
				document.head.appendChild(s);
			});
		}`
	} else {
		script = `(content) => {
			const s = document.createElement('script');
			s.textContent = content;
			document.head.appendChild(s);
			return 'ok';
		}`
	}

	arg := url
	if arg == "" {
		arg = content
	}

	params := map[string]interface{}{
		"functionDeclaration": script,
		"target":              map[string]interface{}{"context": context},
		"arguments": []map[string]interface{}{
			{"type": "string", "value": arg},
		},
		"awaitPromise":    true,
		"resultOwnership": "root",
	}

	if _, err := r.sendInternalCommand(session, "script.callFunction", params); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"added": true})
}

// handlePageAddStyle handles vibium:page.addStyle — injects a <style> or <link> tag.
// Accepts "url" (for external stylesheet) or "content" (for inline CSS).
func (r *Router) handlePageAddStyle(session *BrowserSession, cmd bidiCommand) {
	url, _ := cmd.Params["url"].(string)
	content, _ := cmd.Params["content"].(string)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	var script string
	if url != "" {
		script = `(url) => {
			return new Promise((resolve, reject) => {
				const link = document.createElement('link');
				link.rel = 'stylesheet';
				link.href = url;
				link.onload = () => resolve('ok');
				link.onerror = () => reject(new Error('failed to load stylesheet'));
				document.head.appendChild(link);
			});
		}`
	} else {
		script = `(content) => {
			const s = document.createElement('style');
			s.textContent = content;
			document.head.appendChild(s);
			return 'ok';
		}`
	}

	arg := url
	if arg == "" {
		arg = content
	}

	params := map[string]interface{}{
		"functionDeclaration": script,
		"target":              map[string]interface{}{"context": context},
		"arguments": []map[string]interface{}{
			{"type": "string", "value": arg},
		},
		"awaitPromise":    true,
		"resultOwnership": "root",
	}

	if _, err := r.sendInternalCommand(session, "script.callFunction", params); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"added": true})
}

// handlePageExpose handles vibium:page.expose — injects a named function via preload script.
// Simplified: injects a function that stores calls (no callback channel).
func (r *Router) handlePageExpose(session *BrowserSession, cmd bidiCommand) {
	name, _ := cmd.Params["name"].(string)
	fn, _ := cmd.Params["fn"].(string)

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Inject the function into the page via script.callFunction
	script := `(name, fn) => {
		window[name] = new Function('return ' + fn)();
		return 'ok';
	}`

	params := map[string]interface{}{
		"functionDeclaration": script,
		"target":              map[string]interface{}{"context": context},
		"arguments": []map[string]interface{}{
			{"type": "string", "value": name},
			{"type": "string", "value": fn},
		},
		"awaitPromise":    false,
		"resultOwnership": "root",
	}

	if _, err := r.sendInternalCommand(session, "script.callFunction", params); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"exposed": true})
}

// deserializeScriptResult extracts a usable value from a BiDi script result.
// Handles primitives (string, number, boolean, null, undefined) and objects/arrays.
func deserializeScriptResult(resp json.RawMessage) (interface{}, error) {
	var result struct {
		Result struct {
			Result struct {
				Type   string      `json:"type"`
				Value  interface{} `json:"value"`
				Handle string      `json:"handle,omitempty"`
			} `json:"result"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse script result: %w", err)
	}

	r := result.Result.Result
	switch r.Type {
	case "null", "undefined":
		return nil, nil
	case "string", "number", "boolean":
		return r.Value, nil
	case "array":
		// BiDi returns arrays as {type: "array", value: [{type, value}, ...]}
		if items, ok := r.Value.([]interface{}); ok {
			out := make([]interface{}, len(items))
			for i, item := range items {
				if m, ok := item.(map[string]interface{}); ok {
					out[i] = m["value"]
				} else {
					out[i] = item
				}
			}
			return out, nil
		}
		return r.Value, nil
	case "object":
		// BiDi returns objects as {type: "object", value: [[key, {type, value}], ...]}
		if pairs, ok := r.Value.([]interface{}); ok {
			out := make(map[string]interface{})
			for _, pair := range pairs {
				if kv, ok := pair.([]interface{}); ok && len(kv) == 2 {
					key, _ := kv[0].(string)
					if m, ok := kv[1].(map[string]interface{}); ok {
						out[key] = m["value"]
					}
				}
			}
			return out, nil
		}
		return r.Value, nil
	default:
		return r.Value, nil
	}
}
