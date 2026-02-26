package proxy

import (
	"encoding/json"
	"fmt"
	"time"
)

// resolveContext extracts the "context" param or returns the first context from getTree.
// It also stores the resolved context on the session for use by trace screenshots.
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
// The fn should be a JS function declaration that returns a string, e.g. "() => document.title".
func (r *Router) evalSimpleScript(session *BrowserSession, context, fn string) (string, error) {
	params := map[string]interface{}{
		"functionDeclaration": fn,
		"target":              map[string]interface{}{"context": context},
		"arguments":           []map[string]interface{}{},
		"awaitPromise":        false,
		"resultOwnership":     "root",
	}

	resp, err := r.sendInternalCommand(session, "script.callFunction", params)
	if err != nil {
		return "", err
	}

	return parseScriptResult(resp)
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
// Expected structure: { "result": { "result": { "type": "string", "value": "..." } } }
func parseScriptResult(resp json.RawMessage) (string, error) {
	var result struct {
		Result struct {
			Result struct {
				Type  string `json:"type"`
				Value string `json:"value,omitempty"`
			} `json:"result"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse script result: %w", err)
	}

	if result.Result.Result.Type == "null" || result.Result.Result.Type == "undefined" {
		return "", fmt.Errorf("script returned %s", result.Result.Result.Type)
	}

	return result.Result.Result.Value, nil
}

// resolveElementRef finds an element and returns its BiDi sharedId (for use with input.setFiles etc.).
// Unlike resolveElement which returns bounding box info, this returns the raw node reference.
func (r *Router) resolveElementRef(session *BrowserSession, context string, ep elementParams) (string, error) {
	script, args := buildRefFindScript(ep)
	deadline := time.Now().Add(ep.Timeout)
	interval := 100 * time.Millisecond

	for {
		params := map[string]interface{}{
			"functionDeclaration": script,
			"target":              map[string]interface{}{"context": context},
			"arguments":           args,
			"awaitPromise":        false,
			"resultOwnership":     "root",
		}

		resp, err := r.sendInternalCommand(session, "script.callFunction", params)
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

// buildRefFindScript builds a JS function that finds an element and returns it directly
// (not JSON-stringified). BiDi will serialize the returned DOM node with a sharedId.
func buildRefFindScript(ep elementParams) (string, []map[string]interface{}) {
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

// elementParams holds extracted parameters for element resolution.
type elementParams struct {
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
}

// extractElementParams extracts element parameters from command params.
func extractElementParams(params map[string]interface{}) elementParams {
	ep := elementParams{
		Timeout: defaultTimeout,
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

	return ep
}

// buildActionFindScript builds a JS function that finds an element (by CSS or semantic selectors),
// supports index for querySelectorAll, scrolls it into view, and returns its bounding box.
func buildActionFindScript(ep elementParams) (string, []map[string]interface{}) {
	hasSemantic := ep.Role != "" || ep.Text != "" || ep.Label != "" || ep.Placeholder != "" ||
		ep.Alt != "" || ep.Title != "" || ep.Testid != "" || ep.Xpath != ""

	if !hasSemantic && ep.Selector != "" {
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
					text: (el.textContent || '').trim().substring(0, 100),
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
func (r *Router) resolveElement(session *BrowserSession, context string, ep elementParams) (*elementInfo, error) {
	script, args := buildActionFindScript(ep)
	return r.waitForElementWithScript(session, context, script, args, ep.Timeout)
}
