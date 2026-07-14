package api

import (
	"encoding/json"
	"fmt"
	"time"
)

// resolveContext extracts the "context" param or returns the first context from getTree.
// It also stores the resolved context on the session for use by recording screenshots.
func (r *Router) resolveContext(session *BrowserSession, params map[string]interface{}) (string, error) {
	if ctx, ok := params["context"].(string); ok && ctx != "" {
		session.mu.Lock()
		session.lastContext = ctx
		session.mu.Unlock()
		return ctx, nil
	}
	ctx, err := r.getContext(session)
	if err != nil {
		return "", err
	}
	session.mu.Lock()
	session.lastContext = ctx
	session.mu.Unlock()
	return ctx, nil
}

// evalSimpleScript runs a no-argument script.callFunction and returns the string result.
func (r *Router) evalSimpleScript(session *BrowserSession, context, fn string) (string, error) {
	return EvalSimpleScript(NewAPISession(r, session, context), context, fn)
}

// checkBidiError checks if a BiDi response is an error and returns it.
// BiDi error responses have: { "type": "error", "error": "...", "message": "..." }
func checkBidiError(resp json.RawMessage) error {
	var errResp struct {
		Type    string `json:"type"`
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(resp, &errResp); err != nil {
		return nil // Can't parse, assume not an error
	}
	if errResp.Type == "error" {
		return fmt.Errorf("%s: %s", errResp.Error, errResp.Message)
	}
	return nil
}

// parseScriptResult parses a BiDi script.callFunction response and returns the string value.
// Success structure:   { "result": { "type": "success",   "result": { "type": "string", "value": "..." } } }
// Exception structure: { "result": { "type": "exception", "exceptionDetails": { "text": "..." } } }
func parseScriptResult(resp json.RawMessage) (string, error) {
	var result struct {
		Result struct {
			Type   string `json:"type"`
			Result struct {
				Type  string `json:"type"`
				Value string `json:"value,omitempty"`
			} `json:"result"`
			ExceptionDetails struct {
				Text string `json:"text"`
			} `json:"exceptionDetails"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse script result: %w", err)
	}

	// A thrown exception has no result value — surface it instead of silently
	// returning an empty string, which previously masked errors such as the
	// "Illegal invocation" crash when filling a <textarea> (issues #117, #111).
	if result.Result.Type == "exception" {
		text := result.Result.ExceptionDetails.Text
		if text == "" {
			text = "script threw an exception"
		}
		return "", fmt.Errorf("%s", text)
	}

	if result.Result.Result.Type == "null" || result.Result.Result.Type == "undefined" {
		return "", fmt.Errorf("script returned %s", result.Result.Result.Type)
	}

	return result.Result.Result.Value, nil
}

// resolveElementRef finds an element and returns its BiDi sharedId.
func (r *Router) resolveElementRef(session *BrowserSession, context string, ep ElementParams) (string, error) {
	return ResolveElementRef(NewAPISession(r, session, context), context, ep)
}

// buildRefFindScript builds a JS function that finds an element and returns it directly
// (not JSON-stringified). BiDi will serialize the returned DOM node with a sharedId.
func buildRefFindScript(ep ElementParams) (string, []map[string]interface{}) {
	if hasSemantic(ep) {
		args := buildElSemanticArgs(ep)
		script := `
			(scope, selector, role, text, label, placeholder, alt, title, testid, xpath, index, hasIndex) => {
				const root = scope ? document.querySelector(scope) : document;
				if (!root) return null;
		` + semanticMatchesHelper() + `
				const found = collectMatches(root, selector, role, text, label, placeholder, alt, title, testid, xpath);
				let el;
				if (hasIndex) {
					el = found[index];
				} else {
					el = pickBest(found, text);
				}
				return el || null;
			}
		`
		return script, args
	}

	args := []map[string]interface{}{
		{"type": "string", "value": ep.Scope},
		{"type": "string", "value": ep.Selector},
		{"type": "number", "value": ep.Index},
		{"type": "boolean", "value": ep.HasIndex},
	}

	script := `
		(scope, selector, index, hasIndex) => {
			const root = scope ? document.querySelector(scope) : document;
			if (!root) return null;
			let el;
			if (hasIndex) {
				const all = root.querySelectorAll(selector);
				el = all[index];
			} else {
				el = root.querySelector(selector);
			}
			return el || null;
		}
	`
	return script, args
}

// ElementParams holds extracted parameters for element resolution.
type ElementParams struct {
	Selector    string
	Index       int
	HasIndex    bool
	Scope       string
	Role        string
	Text        string
	Label       string
	Placeholder string
	Alt         string
	Title       string
	Testid      string
	Xpath       string
	Context     string
	Timeout     time.Duration
	Force       bool
}

// withDefaultTimeout returns ep with a default timeout applied when it is unset
// (zero or negative). Element resolution must auto-wait — a core feature — even
// when callers build ElementParams inline without a timeout (e.g. the CLI/MCP
// daemon handlers). Applied at every polling-find entry point so no caller can
// accidentally disable auto-wait. The protocol path (ExtractElementParams) sets
// 30s explicitly, so this only affects inline callers (issue #173).
func (ep ElementParams) withDefaultTimeout() ElementParams {
	if ep.Timeout <= 0 {
		ep.Timeout = DefaultTimeout
	}
	return ep
}

// ExtractElementParams extracts element parameters from command params.
func ExtractElementParams(params map[string]interface{}) ElementParams {
	ep := ElementParams{
		Timeout: DefaultTimeout,
	}

	ep.Selector, _ = params["selector"].(string)
	ep.Context, _ = params["context"].(string)
	ep.Scope, _ = params["scope"].(string)
	ep.Role, _ = params["role"].(string)
	ep.Text, _ = params["text"].(string)
	ep.Label, _ = params["label"].(string)
	ep.Placeholder, _ = params["placeholder"].(string)
	ep.Alt, _ = params["alt"].(string)
	ep.Title, _ = params["title"].(string)
	ep.Testid, _ = params["testid"].(string)
	ep.Xpath, _ = params["xpath"].(string)

	if idx, ok := params["index"].(float64); ok {
		ep.Index = int(idx)
		ep.HasIndex = true
	}

	if timeoutMs, ok := params["timeout"].(float64); ok && timeoutMs > 0 {
		ep.Timeout = time.Duration(timeoutMs) * time.Millisecond
	}

	if force, ok := params["force"].(bool); ok {
		ep.Force = force
	}

	return ep
}

// hasSemantic returns true if any semantic selector params are set.
func hasSemantic(ep ElementParams) bool {
	return ep.Role != "" || ep.Text != "" || ep.Label != "" || ep.Placeholder != "" ||
		ep.Alt != "" || ep.Title != "" || ep.Testid != "" || ep.Xpath != ""
}

// buildActionFindScript builds a JS function that finds an element (by CSS or semantic selectors),
// supports index for querySelectorAll, scrolls it into view, and returns its bounding box.
func buildActionFindScript(ep ElementParams) (string, []map[string]interface{}) {
	if !hasSemantic(ep) && ep.Selector != "" {
		// CSS path with index support
		args := []map[string]interface{}{
			{"type": "string", "value": ep.Scope},
			{"type": "string", "value": ep.Selector},
			{"type": "number", "value": ep.Index},
			{"type": "boolean", "value": ep.HasIndex},
		}
		script := `
			(scope, selector, index, hasIndex) => {
				const root = scope ? document.querySelector(scope) : document;
				if (!root) return null;
				let el;
				if (hasIndex) {
					const all = root.querySelectorAll(selector);
					el = all[index];
				} else {
					el = root.querySelector(selector);
				}
				if (!el) return null;
				if (el.scrollIntoViewIfNeeded) {
					el.scrollIntoViewIfNeeded(true);
				} else {
					el.scrollIntoView({ block: 'center', inline: 'nearest' });
				}
				const rect = el.getBoundingClientRect();
				return JSON.stringify({
					tag: el.tagName.toLowerCase(),
					text: (el.innerText || '').trim(),
					box: { x: rect.x, y: rect.y, width: rect.width, height: rect.height }
				});
			}
		`
		return script, args
	}

	// Semantic path with index support
	args := []map[string]interface{}{
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

	script := `
		(scope, selector, role, text, label, placeholder, alt, title, testid, xpath, index, hasIndex) => {
			const root = scope ? document.querySelector(scope) : document;
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
			if (el.scrollIntoViewIfNeeded) {
				el.scrollIntoViewIfNeeded(true);
			} else {
				el.scrollIntoView({ block: 'center', inline: 'nearest' });
			}
			const rect = el.getBoundingClientRect();
			return JSON.stringify(toInfo(el));
		}
	`
	return script, args
}

// resolveElement finds an element using the given params, polling until found or timeout.
// It returns the element's info with updated bounding box after scrolling into view.
func (r *Router) resolveElement(session *BrowserSession, context string, ep ElementParams) (*ElementInfo, error) {
	s := NewAPISession(r, session, context)
	return ResolveElement(s, context, ep)
}

// ---------------------------------------------------------------------------
// Exported standalone functions — usable from both proxy and MCP handlers.
// ---------------------------------------------------------------------------

// EvalSimpleScript runs a no-argument script.callFunction via the Session and
// returns the string result.
func EvalSimpleScript(s Session, context, fn string) (string, error) {
	params := map[string]interface{}{
		"functionDeclaration": fn,
		"target":              map[string]interface{}{"context": context},
		"arguments":           []map[string]interface{}{},
		"awaitPromise":        false,
		"resultOwnership":     "root",
	}

	resp, err := s.SendBidiCommand("script.callFunction", params)
	if err != nil {
		return "", err
	}

	return parseScriptResult(resp)
}

// CallScript runs a script.callFunction with arguments via the Session and
// returns the raw response.
func CallScript(s Session, context, fn string, args []map[string]interface{}) (json.RawMessage, error) {
	params := map[string]interface{}{
		"functionDeclaration": fn,
		"target":              map[string]interface{}{"context": context},
		"arguments":           args,
		"awaitPromise":        false,
		"resultOwnership":     "root",
	}

	return s.SendBidiCommand("script.callFunction", params)
}

// ResolveElement finds an element using the given params, polling until found or timeout.
func ResolveElement(s Session, context string, ep ElementParams) (*ElementInfo, error) {
	ep = ep.withDefaultTimeout()
	script, args := buildActionFindScript(ep)
	info, err := WaitForElementWithScript(s, context, script, args, ep.Timeout)
	if err == nil && info != nil {
		s.SetLastElementBox(&info.Box)
	}
	return info, err
}

// ResolveElementRef finds an element and returns its BiDi sharedId.
func ResolveElementRef(s Session, context string, ep ElementParams) (string, error) {
	ep = ep.withDefaultTimeout()
	script, args := buildRefFindScript(ep)
	deadline := time.Now().Add(ep.Timeout)
	interval := 100 * time.Millisecond

	for {
		resp, err := CallScript(s, context, script, args)
		if err == nil {
			var result struct {
				Result struct {
					Result struct {
						Type     string `json:"type"`
						SharedID string `json:"sharedId"`
					} `json:"result"`
				} `json:"result"`
			}
			if err := json.Unmarshal(resp, &result); err == nil {
				if result.Result.Result.Type == "node" && result.Result.Result.SharedID != "" {
					return result.Result.Result.SharedID, nil
				}
			}
		}

		if time.Now().After(deadline) {
			return "", fmt.Errorf("timeout waiting for element: not found")
		}

		time.Sleep(interval)
	}
}

// WaitForElementWithScript polls until an element is found using a custom script.
func WaitForElementWithScript(s Session, context, script string, args []map[string]interface{}, timeout time.Duration) (*ElementInfo, error) {
	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond

	desc := describeSelector(args)

	for {
		resp, err := CallScript(s, context, script, args)
		if err == nil {
			var result struct {
				Result struct {
					Result struct {
						Type  string `json:"type"`
						Value string `json:"value,omitempty"`
					} `json:"result"`
				} `json:"result"`
			}
			if err := json.Unmarshal(resp, &result); err == nil {
				if result.Result.Result.Type == "string" && result.Result.Result.Value != "" {
					var info ElementInfo
					if err := json.Unmarshal([]byte(result.Result.Result.Value), &info); err == nil {
						return &info, nil
					}
				}
			}
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout after %s waiting for '%s': element not found", timeout, desc)
		}

		time.Sleep(interval)
	}
}
