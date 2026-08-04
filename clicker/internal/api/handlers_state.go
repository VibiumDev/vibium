package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vibium/clicker/internal/bidi"
	"github.com/vibium/clicker/internal/log"
)

// handleVibiumElText handles vibium:element.text — returns element.textContent.
func (r *Router) handleVibiumElText(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	val, err := GetText(NewAPISession(r, session, context), context, ep)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{"text": val})
}

// handleVibiumElHighlight handles vibium:element.highlight — briefly outlines an
// element so a human watching the browser can see which element was targeted.
func (r *Router) handleVibiumElHighlight(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	script, args := buildElStateScript(ep, `(() => {
		const prevOutline = el.style.outline;
		const prevOffset = el.style.outlineOffset;
		el.style.outline = '2px solid #ff2d95';
		el.style.outlineOffset = '1px';
		setTimeout(() => { el.style.outline = prevOutline; el.style.outlineOffset = prevOffset; }, 2000);
		return 'ok';
	})()`)
	val, err := r.evalElementScript(session, context, script, args)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	if val != "ok" {
		r.sendError(session, cmd.ID, fmt.Errorf("highlight: %s", val))
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{"highlighted": true})
}

// handleVibiumElInnerText handles vibium:element.innerText — returns element.innerText.
func (r *Router) handleVibiumElInnerText(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	val, err := GetInnerText(NewAPISession(r, session, context), context, ep)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{"text": val})
}

// handleVibiumElHTML handles vibium:element.html — returns element.innerHTML.
func (r *Router) handleVibiumElHTML(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)
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

// handleVibiumElValue handles vibium:element.value — returns element.value (for inputs).
func (r *Router) handleVibiumElValue(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)
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

// handleVibiumElAttr handles vibium:element.attr — returns element.getAttribute(name).
func (r *Router) handleVibiumElAttr(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)
	name, _ := cmd.Params["name"].(string)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	var args []map[string]interface{}
	var script string

	if hasSemantic(ep) {
		args = buildElSemanticArgs(ep)
		args = append(args, map[string]interface{}{"type": "string", "value": name})
		script = `
			(scope, selector, role, text, label, placeholder, alt, title, testid, xpath, index, hasIndex, name) => {
				` + PierceQueryJS() + `
			const root = scope ? pierceQuery(document, scope) : document;
				if (!root) return JSON.stringify({error: 'root not found'});
		` + semanticMatchesHelper() + `
				const found = collectMatches(root, selector, role, text, label, placeholder, alt, title, testid, xpath);
				let el;
				if (hasIndex) {
					el = found[index];
				} else {
					el = pickBest(found, text);
				}
				if (!el) return JSON.stringify({error: 'element not found'});
				const v = el.getAttribute(name);
				return JSON.stringify({value: v});
			}
		`
	} else {
		args = buildElBaseArgs(ep)
		args = append(args, map[string]interface{}{"type": "string", "value": name})
		script = `
			(scope, selector, index, hasIndex, name) => {
				` + PierceQueryJS() + `
			const root = scope ? pierceQuery(document, scope) : document;
				if (!root) return JSON.stringify({error: 'root not found'});
				let el;
				if (hasIndex) {
					el = pierceQueryAll(root, selector)[index];
				} else {
					el = pierceQuery(root, selector);
				}
				if (!el) return JSON.stringify({error: 'element not found'});
				const v = el.getAttribute(name);
				return JSON.stringify({value: v});
			}
		`
	}

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

// handleVibiumElBounds handles vibium:element.bounds — returns getBoundingClientRect().
func (r *Router) handleVibiumElBounds(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)
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

	var box BoxInfo
	if err := json.Unmarshal([]byte(val), &box); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("bounds parse failed: %w", err))
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{
		"x": box.X, "y": box.Y, "width": box.Width, "height": box.Height,
	})
}

// handleVibiumElIsVisible handles vibium:element.isVisible — checks computed visibility.
func (r *Router) handleVibiumElIsVisible(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)
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

// handleVibiumElIsHidden handles vibium:element.isHidden — inverse of isVisible.
func (r *Router) handleVibiumElIsHidden(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)
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

// handleVibiumElIsEnabled handles vibium:element.isEnabled — checks !element.disabled.
func (r *Router) handleVibiumElIsEnabled(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)
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

// handleVibiumElIsChecked handles vibium:element.isChecked — returns element.checked.
func (r *Router) handleVibiumElIsChecked(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Share IsChecked rather than a second `!!el.checked` script, so this and
	// check/uncheck agree on what counts as a checkable element.
	checked, err := IsChecked(NewAPISession(r, session, context), context, ep)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{"checked": checked})
}

// handleVibiumElIsEditable handles vibium:element.isEditable — not disabled and not readonly.
func (r *Router) handleVibiumElIsEditable(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Same question as the fill path and `is actionable`, so the same predicate:
	// !disabled && !readOnly alone called a <div> editable.
	script, args := buildElBoolScript(ep, `return `+EditablePredicateJS+`;`)
	editable, err := r.evalBoolScript(session, context, script, args)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{"editable": editable})
}

// handleVibiumElScreenshot handles vibium:element.screenshot — captures element screenshot.
func (r *Router) handleVibiumElScreenshot(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)
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

// handleVibiumElWaitFor handles vibium:element.waitFor — waits for element state.
// Supported states: "visible", "hidden", "attached", "detached".
func (r *Router) handleVibiumElWaitFor(session *BrowserSession, cmd bidiCommand) {
	ep := ExtractElementParams(cmd.Params)
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
	ep := ExtractElementParams(cmd.Params)
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
	timeout := DefaultTimeout
	if timeoutMs > 0 {
		timeout = time.Duration(timeoutMs) * time.Millisecond
	}

	// Callers pass either a full function ("() => cond", "function(){...}") or a
	// bare boolean expression ("document.readyState === 'complete'"). A bare
	// expression is not a valid functionDeclaration, so BiDi threw every poll and
	// the call always timed out (issues #123, #131, #145). Wrap uniformly: evaluate
	// the operand, and invoke it if it turned out to be a function.
	wrapped := fmt.Sprintf(
		"() => { const __vibiumPred = (%s); return (typeof __vibiumPred === 'function') ? __vibiumPred() : __vibiumPred; }",
		strings.TrimRight(strings.TrimSpace(fn), ";"),
	)

	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond
	lastErr := ""

	for {
		resp, err := r.sendInternalCommand(session, "script.callFunction", map[string]interface{}{
			"functionDeclaration": wrapped,
			"target":              map[string]interface{}{"context": context},
			"arguments":           []map[string]interface{}{},
			"awaitPromise":        true,
			"resultOwnership":     "root",
		})
		if err == nil {
			sr, parseErr := bidi.ParseScriptResponse(resp)
			var se *bidi.ScriptException
			switch {
			case errors.As(parseErr, &se):
				// Keep polling — the expression may reference state that does
				// not exist yet — but remember why for the timeout message.
				lastErr = se.Text
			case parseErr == nil:
				// Truthy check: non-null, non-undefined, non-false, non-zero, non-empty-string
				res := sr.Result
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
			if lastErr != "" {
				r.sendError(session, cmd.ID, fmt.Errorf("timeout waiting for function to return truthy (last error: %s)", lastErr))
			} else {
				r.sendError(session, cmd.ID, fmt.Errorf("timeout waiting for function to return truthy"))
			}
			return
		}

		time.Sleep(interval)
	}
}

// --- Script builder helpers for state queries ---

// buildElBaseArgs returns the standard [scope, selector, index, hasIndex] args.
func buildElBaseArgs(ep ElementParams) []map[string]interface{} {
	return []map[string]interface{}{
		{"type": "string", "value": ep.Scope},
		{"type": "string", "value": ep.Selector},
		{"type": "number", "value": ep.Index},
		{"type": "boolean", "value": ep.HasIndex},
	}
}

// buildElSemanticArgs returns the 12-arg list for semantic element resolution.
func buildElSemanticArgs(ep ElementParams) []map[string]interface{} {
	return []map[string]interface{}{
		{"type": "string", "value": ep.Scope},
		{"type": "string", "value": ep.Selector},
		{"type": "string", "value": ep.Role},
		{"type": "string", "value": ep.Text},
		{"type": "string", "value": ep.Label},
		{"type": "string", "value": ep.Placeholder},
		{"type": "string", "value": ep.Alt},
		{"type": "string", "value": ep.Title},
		{"type": "string", "value": ep.Testid},
		{"type": "string", "value": ep.Xpath},
		{"type": "number", "value": ep.Index},
		{"type": "boolean", "value": ep.HasIndex},
	}
}

// buildElStateScript builds a script that finds an element and evaluates an expression.
// The expression receives `el` as the found element and should return a string.
func buildElStateScript(ep ElementParams, expr string) (string, []map[string]interface{}) {
	if hasSemantic(ep) {
		args := buildElSemanticArgs(ep)
		script := fmt.Sprintf(`
			(scope, selector, role, text, label, placeholder, alt, title, testid, xpath, index, hasIndex) => {
				`+PierceQueryJS()+`
			const root = scope ? pierceQuery(document, scope) : document;
				if (!root) return null;
		`+semanticMatchesHelper()+`
				const found = collectMatches(root, selector, role, text, label, placeholder, alt, title, testid, xpath);
				let el;
				if (hasIndex) {
					el = found[index];
				} else {
					el = pickBest(found, text);
				}
				if (!el) return null;
				return %s;
			}
		`, expr)
		return script, args
	}

	args := buildElBaseArgs(ep)
	script := fmt.Sprintf(`
		(scope, selector, index, hasIndex) => {
			`+PierceQueryJS()+`
			const root = scope ? pierceQuery(document, scope) : document;
			if (!root) return null;
			let el;
			if (hasIndex) {
				el = pierceQueryAll(root, selector)[index];
			} else {
				el = pierceQuery(root, selector);
			}
			if (!el) return null;
			return %s;
		}
	`, expr)
	return script, args
}

// buildElBoolScript builds a script that finds an element and evaluates a boolean expression.
// The body receives `el` and should use `return true/false;`.
func buildElBoolScript(ep ElementParams, body string) (string, []map[string]interface{}) {
	if hasSemantic(ep) {
		args := buildElSemanticArgs(ep)
		script := fmt.Sprintf(`
			(scope, selector, role, text, label, placeholder, alt, title, testid, xpath, index, hasIndex) => {
				`+PierceQueryJS()+`
			const root = scope ? pierceQuery(document, scope) : document;
				if (!root) return 'error:root not found';
		`+semanticMatchesHelper()+`
				const found = collectMatches(root, selector, role, text, label, placeholder, alt, title, testid, xpath);
				let el;
				if (hasIndex) {
					el = found[index];
				} else {
					el = pickBest(found, text);
				}
				if (!el) return 'error:element not found';
				const _check = (el) => { %s };
				return _check(el) ? 'true' : 'false';
			}
		`, body)
		return script, args
	}

	args := buildElBaseArgs(ep)
	script := fmt.Sprintf(`
		(scope, selector, index, hasIndex) => {
			`+PierceQueryJS()+`
			const root = scope ? pierceQuery(document, scope) : document;
			if (!root) return 'error:root not found';
			let el;
			if (hasIndex) {
				el = pierceQueryAll(root, selector)[index];
			} else {
				el = pierceQuery(root, selector);
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
func buildElJSONScript(ep ElementParams, body string) (string, []map[string]interface{}) {
	if hasSemantic(ep) {
		args := buildElSemanticArgs(ep)
		script := fmt.Sprintf(`
			(scope, selector, role, text, label, placeholder, alt, title, testid, xpath, index, hasIndex) => {
				`+PierceQueryJS()+`
			const root = scope ? pierceQuery(document, scope) : document;
				if (!root) return JSON.stringify({error: 'root not found'});
		`+semanticMatchesHelper()+`
				const found = collectMatches(root, selector, role, text, label, placeholder, alt, title, testid, xpath);
				let el;
				if (hasIndex) {
					el = found[index];
				} else {
					el = pickBest(found, text);
				}
				if (!el) return JSON.stringify({error: 'element not found'});
				%s
			}
		`, body)
		return script, args
	}

	args := buildElBaseArgs(ep)
	script := fmt.Sprintf(`
		(scope, selector, index, hasIndex) => {
			`+PierceQueryJS()+`
			const root = scope ? pierceQuery(document, scope) : document;
			if (!root) return JSON.stringify({error: 'root not found'});
			let el;
			if (hasIndex) {
				el = pierceQueryAll(root, selector)[index];
			} else {
				el = pierceQuery(root, selector);
			}
			if (!el) return JSON.stringify({error: 'element not found'});
			%s
		}
	`, body)
	return script, args
}

// evalElementScript runs a state script and returns the string result.
func (r *Router) evalElementScript(session *BrowserSession, context, script string, args []map[string]interface{}) (string, error) {
	return EvalElementScript(NewAPISession(r, session, context), context, script, args)
}

// evalBoolScript runs a boolean script and parses the "true"/"false" result.
func (r *Router) evalBoolScript(session *BrowserSession, context, script string, args []map[string]interface{}) (bool, error) {
	return EvalBoolScript(NewAPISession(r, session, context), context, script, args)
}

// ---------------------------------------------------------------------------
// Exported standalone state helpers — usable from both proxy and MCP.
// ---------------------------------------------------------------------------

// EvalElementScript runs a state script via the Session and returns the string result.
func EvalElementScript(s Session, context, script string, args []map[string]interface{}) (string, error) {
	resp, err := CallScript(s, context, script, args)
	if err != nil {
		return "", err
	}

	val, err := parseScriptResult(resp)
	if err != nil {
		return "", fmt.Errorf("element not found")
	}
	return val, nil
}

// EvalBoolScript runs a boolean script via the Session and parses the "true"/"false" result.
func EvalBoolScript(s Session, context, script string, args []map[string]interface{}) (bool, error) {
	resp, err := CallScript(s, context, script, args)
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
func (r *Router) resolveElementNoWait(session *BrowserSession, context string, ep ElementParams) (*ElementInfo, error) {
	return ResolveElementNoWait(NewAPISession(r, session, context), context, ep)
}

// ---------------------------------------------------------------------------
// Exported standalone state query functions — usable from both proxy and MCP.
// ---------------------------------------------------------------------------

// GetText returns the element's textContent, trimmed. GetInnerText is the
// rendered-text variant; keeping both on innerText made them the same command.
func GetText(s Session, context string, ep ElementParams) (string, error) {
	script, args := buildElStateScript(ep, `(el.textContent || '').trim()`)
	return EvalElementScript(s, context, script, args)
}

// GetInnerText returns the innerText of an element.
func GetInnerText(s Session, context string, ep ElementParams) (string, error) {
	script, args := buildElStateScript(ep, `(el.innerText || '').trim()`)
	return EvalElementScript(s, context, script, args)
}

// GetInnerHTML returns the innerHTML of an element.
func GetInnerHTML(s Session, context string, ep ElementParams) (string, error) {
	script, args := buildElStateScript(ep, `el.innerHTML`)
	return EvalElementScript(s, context, script, args)
}

// GetOuterHTML returns the outerHTML of an element.
func GetOuterHTML(s Session, context string, ep ElementParams) (string, error) {
	script, args := buildElStateScript(ep, `el.outerHTML`)
	return EvalElementScript(s, context, script, args)
}

// GetValue returns the value property of a form element.
func GetValue(s Session, context string, ep ElementParams) (string, error) {
	script, args := buildElStateScript(ep, `el.value || ''`)
	return EvalElementScript(s, context, script, args)
}

// GetAttribute returns the value of an HTML attribute, or nil when the
// attribute is absent. A present-but-empty attribute (<button disabled>) and a
// missing one are different things, and a plain string cannot tell them apart.
func GetAttribute(s Session, context string, ep ElementParams, name string) (*string, error) {
	var args []map[string]interface{}
	var script string

	if hasSemantic(ep) {
		args = buildElSemanticArgs(ep)
		args = append(args, map[string]interface{}{"type": "string", "value": name})
		script = `
			(scope, selector, role, text, label, placeholder, alt, title, testid, xpath, index, hasIndex, name) => {
				` + PierceQueryJS() + `
			const root = scope ? pierceQuery(document, scope) : document;
				if (!root) return null;
		` + semanticMatchesHelper() + `
				const found = collectMatches(root, selector, role, text, label, placeholder, alt, title, testid, xpath);
				let el;
				if (hasIndex) {
					el = found[index];
				} else {
					el = pickBest(found, text);
				}
				if (!el) return null;
				const v = el.getAttribute(name);
				return JSON.stringify({value: v});
			}
		`
	} else {
		args = buildElBaseArgs(ep)
		args = append(args, map[string]interface{}{"type": "string", "value": name})
		script = `
			(scope, selector, index, hasIndex, name) => {
				` + PierceQueryJS() + `
			const root = scope ? pierceQuery(document, scope) : document;
				if (!root) return null;
				let el;
				if (hasIndex) {
					el = pierceQueryAll(root, selector)[index];
				} else {
					el = pierceQuery(root, selector);
				}
				if (!el) return null;
				const v = el.getAttribute(name);
				return JSON.stringify({value: v});
			}
		`
	}

	val, err := EvalElementScript(s, context, script, args)
	if err != nil {
		return nil, err
	}

	var out struct {
		Value *string `json:"value"`
	}
	if err := json.Unmarshal([]byte(val), &out); err != nil {
		return nil, fmt.Errorf("attr parse failed: %w", err)
	}
	return out.Value, nil
}

// IsVisible checks if an element is visible (not hidden, not zero-size).
func IsVisible(s Session, context string, ep ElementParams) (bool, error) {
	script, args := buildElBoolScript(ep, `
		const style = window.getComputedStyle(el);
		if (style.display === 'none') return false;
		if (style.visibility === 'hidden') return false;
		if (parseFloat(style.opacity) === 0) return false;
		const rect = el.getBoundingClientRect();
		return rect.width > 0 && rect.height > 0;
	`)
	return EvalBoolScript(s, context, script, args)
}

// IsEnabled checks if an element is enabled (!disabled).
func IsEnabled(s Session, context string, ep ElementParams) (bool, error) {
	script, args := buildElBoolScript(ep, `return !el.disabled;`)
	return EvalBoolScript(s, context, script, args)
}

// GetCount counts elements matching a CSS selector.
func GetCount(s Session, context, selector string) (int, error) {
	// Return the count as a string: parseScriptResult unmarshals the BiDi result
	// value into a Go string field, so a bare numeric result fails with "cannot
	// unmarshal number ... into ... string" (issue #149).
	expr := fmt.Sprintf(`() => String(document.querySelectorAll(%q).length)`, selector)
	val, err := EvalSimpleScript(s, context, expr)
	if err != nil {
		return 0, err
	}
	var count int
	if _, err := fmt.Sscanf(val, "%d", &count); err != nil {
		return 0, fmt.Errorf("failed to parse count: %w", err)
	}
	return count, nil
}

// WaitForText waits until the page body contains the given text.
func WaitForText(s Session, context, text string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond

	for {
		pageText, err := EvalSimpleScript(s, context, "() => document.body.innerText")
		if err == nil && strings.Contains(pageText, text) {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for text %q to appear", text)
		}

		time.Sleep(interval)
	}
}

// WaitForFunction waits until a JS expression returns a truthy value.
func WaitForFunction(s Session, context, expression string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond

	for {
		val, err := EvalSimpleScript(s, context, fmt.Sprintf("() => { const r = %s; return r ? String(r) : ''; }", expression))
		if err == nil && val != "" {
			return val, nil
		}

		if time.Now().After(deadline) {
			return "", fmt.Errorf("timeout waiting for expression to return truthy: %s", expression)
		}

		time.Sleep(interval)
	}
}

// ResolveElementNoWait tries to find an element immediately without polling.
func ResolveElementNoWait(s Session, context string, ep ElementParams) (*ElementInfo, error) {
	script, args := buildActionFindScript(ep)

	resp, err := CallScript(s, context, script, args)
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

	var info ElementInfo
	if err := json.Unmarshal([]byte(result.Result.Result.Value), &info); err != nil {
		return nil, fmt.Errorf("failed to parse element info: %w", err)
	}
	return &info, nil
}

// WaitForVisible polls until the element exists and is visible, or times out.
func WaitForVisible(s Session, context string, ep ElementParams) error {
	ep = ep.withDefaultTimeout()
	deadline := time.Now().Add(ep.Timeout)
	interval := 100 * time.Millisecond

	for {
		_, err := ResolveElementNoWait(s, context, ep)
		if err == nil {
			visible, vErr := IsVisible(s, context, ep)
			if vErr == nil && visible {
				return nil
			}
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout after %s: element not visible", ep.Timeout)
		}
		time.Sleep(interval)
	}
}

// WaitForHidden polls until the element is either not found or not visible.
func WaitForHidden(s Session, context string, ep ElementParams) error {
	ep = ep.withDefaultTimeout()
	deadline := time.Now().Add(ep.Timeout)
	interval := 100 * time.Millisecond

	for {
		_, err := ResolveElementNoWait(s, context, ep)
		if err != nil {
			return nil // not found = hidden
		}

		visible, vErr := IsVisible(s, context, ep)
		if vErr != nil || !visible {
			return nil // not visible = hidden
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout after %s: element still visible", ep.Timeout)
		}
		time.Sleep(interval)
	}
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

// handlePageExpose handles vibium:page.expose — defines a named function on
// window and keeps it there across navigations.
//
// It used to inject only into the current document with script.callFunction, so
// the function vanished on the next navigation even though the point of
// exposing one is that the page can call it whenever it loads (#135). A preload
// script runs on every new document, which is what makes it stick; the same
// source is also run against the current document so the function is usable
// immediately, without waiting for a navigation.
func (r *Router) handlePageExpose(session *BrowserSession, cmd bidiCommand) {
	name, _ := cmd.Params["name"].(string)
	fn, _ := cmd.Params["fn"].(string)
	if name == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("name is required"))
		return
	}
	if fn == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("fn is required"))
		return
	}

	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// json.Marshal gives a correctly escaped JS string literal for the name.
	// fn is source, so it is embedded directly, parenthesised so a function
	// expression parses. Building the source here rather than passing it as an
	// argument also avoids the Function constructor, which a page's CSP can
	// block. addPreloadScript takes only channels as arguments, so a preload
	// script has to carry its values in the declaration anyway.
	nameLit, err := json.Marshal(name)
	if err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("invalid name: %w", err))
		return
	}
	declaration := fmt.Sprintf("() => { window[%s] = (%s); }", nameLit, fn)

	// Replace any previous script for this name rather than stacking another.
	session.mu.Lock()
	previous := session.exposedPreloadIDs[name]
	session.mu.Unlock()
	if previous != "" {
		if _, err := r.sendInternalCommand(session, "script.removePreloadScript", map[string]interface{}{
			"script": previous,
		}); err != nil {
			log.Debug("failed to remove previous exposed function", "name", name, "error", err)
		}
	}

	resp, err := r.sendInternalCommand(session, "script.addPreloadScript", map[string]interface{}{
		"functionDeclaration": declaration,
	})
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		r.sendError(session, cmd.ID, bidiErr)
		return
	}

	var added struct {
		Result struct {
			Script string `json:"script"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &added); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("failed to parse addPreloadScript response: %w", err))
		return
	}
	session.mu.Lock()
	session.exposedPreloadIDs[name] = added.Result.Script
	session.mu.Unlock()

	// The preload script only runs for documents loaded from now on, so define
	// it in the document that is already open too.
	if _, err := r.sendInternalCommand(session, "script.callFunction", map[string]interface{}{
		"functionDeclaration": declaration,
		"target":              map[string]interface{}{"context": context},
		"awaitPromise":        false,
	}); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"exposed": true})
}

// deserializeScriptResult extracts a usable value from a BiDi script result.
// Handles primitives (string, number, boolean, null, undefined) and nested
// objects/arrays. A thrown exception is surfaced as an error rather than null.
func deserializeScriptResult(resp json.RawMessage) (interface{}, error) {
	sr, err := bidi.ParseScriptResponse(resp)
	if err != nil {
		return nil, err
	}
	return bidi.ConvertRemoteValue(sr.Result), nil
}
