package proxy

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

// handleBrowserClose handles vibium:browser.close — sends success then tears down the session.
func (r *Router) handleBrowserClose(session *BrowserSession, cmd bidiCommand) {
	// Send success before closing so the client receives confirmation
	r.sendSuccess(session, cmd.ID, map[string]interface{}{})

	// Close the session (browser + connections)
	r.sessions.Delete(session.Client.ID)
	r.closeSession(session)
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
