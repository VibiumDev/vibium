package api

import (
	"encoding/json"
	"fmt"
)

// handleBrowserPage handles vibium:browser.page — returns the first (default) browsing context.
func (r *Router) handleBrowserPage(session *BrowserSession, cmd bidiCommand) {
	resp, err := r.sendInternalCommand(session, "browsingContext.getTree", map[string]interface{}{})
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	var result struct {
		Result struct {
			Contexts []struct {
				Context     string `json:"context"`
				UserContext string `json:"userContext"`
			} `json:"contexts"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("failed to parse getTree response: %w", err))
		return
	}
	if len(result.Result.Contexts) == 0 {
		r.sendError(session, cmd.ID, fmt.Errorf("no browsing contexts available"))
		return
	}

	ctx := result.Result.Contexts[0]
	r.sendSuccess(session, cmd.ID, map[string]interface{}{
		"context":     ctx.Context,
		"userContext": ctx.UserContext,
	})
}

// handleBrowserNewPage handles vibium:browser.newPage — creates a new tab.
func (r *Router) handleBrowserNewPage(session *BrowserSession, cmd bidiCommand) {
	params := map[string]interface{}{
		"type": "tab",
	}

	userContext := "default"
	// Optionally create in a specific user context
	if uc, ok := cmd.Params["userContext"].(string); ok && uc != "" {
		params["userContext"] = uc
		userContext = uc
	}

	resp, err := r.sendInternalCommand(session, "browsingContext.create", params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	context, err := parseContextFromCreate(resp)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{
		"context":     context,
		"userContext": userContext,
	})
}

// handleBrowserNewContext handles vibium:browser.newContext — creates a new user context (incognito-like).
func (r *Router) handleBrowserNewContext(session *BrowserSession, cmd bidiCommand) {
	resp, err := r.sendInternalCommand(session, "browser.createUserContext", map[string]interface{}{})
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	var result struct {
		Result struct {
			UserContext string `json:"userContext"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("failed to parse createUserContext response: %w", err))
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"userContext": result.Result.UserContext})
}

// handleContextNewPage handles vibium:context.newPage — creates a new tab in a user context.
func (r *Router) handleContextNewPage(session *BrowserSession, cmd bidiCommand) {
	userContext, _ := cmd.Params["userContext"].(string)
	if userContext == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("userContext is required"))
		return
	}

	params := map[string]interface{}{
		"type":        "tab",
		"userContext": userContext,
	}

	resp, err := r.sendInternalCommand(session, "browsingContext.create", params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	context, err := parseContextFromCreate(resp)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{
		"context":     context,
		"userContext": userContext,
	})
}

// handleBrowserPages handles vibium:browser.pages — returns all browsing contexts.
func (r *Router) handleBrowserPages(session *BrowserSession, cmd bidiCommand) {
	resp, err := r.sendInternalCommand(session, "browsingContext.getTree", map[string]interface{}{})
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	var result struct {
		Result struct {
			Contexts []struct {
				Context     string `json:"context"`
				URL         string `json:"url"`
				UserContext string `json:"userContext"`
			} `json:"contexts"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("failed to parse getTree response: %w", err))
		return
	}

	pages := make([]map[string]interface{}, 0, len(result.Result.Contexts))
	for _, ctx := range result.Result.Contexts {
		pages = append(pages, map[string]interface{}{
			"context":     ctx.Context,
			"url":         ctx.URL,
			"userContext": ctx.UserContext,
		})
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"pages": pages})
}

// handleContextClose handles vibium:context.close — closes a user context and all its pages.
func (r *Router) handleContextClose(session *BrowserSession, cmd bidiCommand) {
	userContext, _ := cmd.Params["userContext"].(string)
	if userContext == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("userContext is required"))
		return
	}

	params := map[string]interface{}{
		"userContext": userContext,
	}

	if _, err := r.sendInternalCommand(session, "browser.removeUserContext", params); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// handleBrowserStop handles vibium:browser.stop — stops the browser, then confirms to the client.
// The response is sent AFTER cleanup so the client knows Chrome is fully terminated
// before it sends SIGTERM to the server process.
func (r *Router) handleBrowserStop(session *BrowserSession, cmd bidiCommand) {
	// Close the session (browser + connections) — kills chromedriver + Chrome
	r.sessions.Delete(session.Client.ID())
	r.closeSession(session, false)

	// Send success after closing so the client knows cleanup is done
	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// handlePageActivate handles vibium:page.activate — brings a tab to the foreground.
func (r *Router) handlePageActivate(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	params := map[string]interface{}{
		"context": context,
	}

	if _, err := r.sendInternalCommand(session, "browsingContext.activate", params); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// handlePageClose handles vibium:page.close — closes a specific browsing context (tab).
func (r *Router) handlePageClose(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	params := map[string]interface{}{
		"context": context,
	}

	if _, err := r.sendInternalCommand(session, "browsingContext.close", params); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// ---------------------------------------------------------------------------
// Exported standalone lifecycle functions — usable from both proxy and MCP.
// ---------------------------------------------------------------------------

// PageInfo holds information about a browsing context (page).
type PageInfo struct {
	Context     string `json:"context"`
	URL         string `json:"url"`
	UserContext string `json:"userContext"`
}

// NewPage creates a new page and returns its context ID.
func NewPage(s Session, url string) (string, error) {
	params := map[string]interface{}{
		"type": "tab",
	}
	resp, err := s.SendBidiCommand("browsingContext.create", params)
	if err != nil {
		return "", err
	}
	context, err := parseContextFromCreate(resp)
	if err != nil {
		return "", err
	}
	if url != "" {
		if err := Navigate(s, context, url, "complete"); err != nil {
			return context, err
		}
	}
	return context, nil
}

// ListPages returns all browsing contexts (pages).
func ListPages(s Session) ([]PageInfo, error) {
	resp, err := s.SendBidiCommand("browsingContext.getTree", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	var result struct {
		Result struct {
			Contexts []PageInfo `json:"contexts"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse getTree response: %w", err)
	}
	return result.Result.Contexts, nil
}

// SwitchPage activates a browsing context (page).
func SwitchPage(s Session, contextID string) error {
	_, err := s.SendBidiCommand("browsingContext.activate", map[string]interface{}{
		"context": contextID,
	})
	return err
}

// ClosePage closes a browsing context (page).
func ClosePage(s Session, contextID string) error {
	_, err := s.SendBidiCommand("browsingContext.close", map[string]interface{}{
		"context": contextID,
	})
	return err
}

// SetViewport sets the viewport size of a browsing context.
func SetViewport(s Session, context string, width, height int, dpr float64) error {
	viewport := map[string]interface{}{
		"width":  width,
		"height": height,
	}
	params := map[string]interface{}{
		"context":  context,
		"viewport": viewport,
	}
	if dpr > 0 {
		params["devicePixelRatio"] = dpr
	}
	resp, err := s.SendBidiCommand("browsingContext.setViewport", params)
	if err != nil {
		return err
	}
	return checkBidiError(resp)
}

// SetContent sets the page HTML content.
func SetContent(s Session, context, html string) error {
	script := `(html) => {
		document.open();
		document.write(html);
		document.close();
		return 'ok';
	}`
	args := []map[string]interface{}{
		{"type": "string", "value": html},
	}
	_, err := CallScript(s, context, script, args)
	return err
}

// RequireFileInput errors unless the element resolves to an <input type="file">.
//
// It deliberately does not go through actionability: file inputs are routinely
// display:none, and a visibility check would reject the very elements upload is
// most often used on.
func RequireFileInput(s Session, context string, ep ElementParams) error {
	script, args := buildElStateScript(ep,
		`(el.tagName === 'INPUT' && (el.type || '').toLowerCase() === 'file') ? 'ok' : 'not an <input type="file"> element'`)
	resp, err := CallScript(s, context, script, args)
	if err != nil {
		return err
	}
	val, err := parseScriptResult(resp)
	if err != nil {
		return err
	}
	if val != "ok" {
		return fmt.Errorf("upload: %s", val)
	}
	return nil
}

// Upload sets files on an <input type="file"> element.
func Upload(s Session, context string, ep ElementParams, files []string) error {
	if err := RequireFileInput(s, context, ep); err != nil {
		return err
	}

	sharedID, err := ResolveElementRef(s, context, ep)
	if err != nil {
		return err
	}

	_, err = s.SendBidiCommand("input.setFiles", map[string]interface{}{
		"context": context,
		"element": map[string]interface{}{
			"sharedId": sharedID,
		},
		"files": files,
	})
	return err
}

// MouseMove moves the mouse to the given coordinates.
func MouseMove(s Session, context string, x, y int) error {
	params := map[string]interface{}{
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
	_, err := s.SendBidiCommand("input.performActions", params)
	return err
}

// MouseDown presses a mouse button.
func MouseDown(s Session, context string, button int) error {
	params := map[string]interface{}{
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
	_, err := s.SendBidiCommand("input.performActions", params)
	return err
}

// MouseUp releases a mouse button.
func MouseUp(s Session, context string, button int) error {
	params := map[string]interface{}{
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
	_, err := s.SendBidiCommand("input.performActions", params)
	return err
}

// MouseClick performs a click (move + down + up) at the given coordinates.
func MouseClick(s Session, context string, x, y, button int) error {
	var pointerActions []map[string]interface{}

	pointerActions = append(pointerActions,
		map[string]interface{}{"type": "pointerMove", "x": x, "y": y, "duration": 0},
		map[string]interface{}{"type": "pointerDown", "button": button},
		map[string]interface{}{"type": "pointerUp", "button": button},
	)

	params := map[string]interface{}{
		"context": context,
		"actions": []map[string]interface{}{
			{
				"type": "pointer",
				"id":   "mouse",
				"parameters": map[string]interface{}{
					"pointerType": "mouse",
				},
				"actions": pointerActions,
			},
		},
	}
	_, err := s.SendBidiCommand("input.performActions", params)
	return err
}

// parseContextFromCreate extracts the context ID from a browsingContext.create response.
func parseContextFromCreate(resp json.RawMessage) (string, error) {
	var result struct {
		Result struct {
			Context string `json:"context"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse create response: %w", err)
	}
	if result.Result.Context == "" {
		return "", fmt.Errorf("no context in create response")
	}
	return result.Result.Context, nil
}
