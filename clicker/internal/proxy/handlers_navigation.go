package proxy

import (
	"fmt"
	"strings"
	"time"
)

// handlePageNavigate handles vibium:page.navigate — navigates to a URL.
func (r *Router) handlePageNavigate(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	url, _ := cmd.Params["url"].(string)
	if url == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("url is required"))
		return
	}

	wait, _ := cmd.Params["wait"].(string)
	if wait == "" {
		wait = "complete"
	}

	params := map[string]interface{}{
		"context": context,
		"url":     url,
		"wait":    wait,
	}

	resp, err := r.sendInternalCommand(session, "browsingContext.navigate", params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	if bidiErr := checkBidiError(resp); bidiErr != nil {
		r.sendError(session, cmd.ID, bidiErr)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"url": url})
}

// handlePageBack handles vibium:page.back — navigates back in history.
func (r *Router) handlePageBack(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	params := map[string]interface{}{
		"context": context,
		"delta":   -1,
	}

	resp, err := r.sendInternalCommand(session, "browsingContext.traverseHistory", params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		r.sendError(session, cmd.ID, bidiErr)
		return
	}

	// Wait for page load after traversal
	r.waitForReadyState(session, context, "complete", 10*time.Second)

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// handlePageForward handles vibium:page.forward — navigates forward in history.
func (r *Router) handlePageForward(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	params := map[string]interface{}{
		"context": context,
		"delta":   1,
	}

	resp, err := r.sendInternalCommand(session, "browsingContext.traverseHistory", params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		r.sendError(session, cmd.ID, bidiErr)
		return
	}

	// Wait for page load after traversal
	r.waitForReadyState(session, context, "complete", 10*time.Second)

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// handlePageReload handles vibium:page.reload — reloads the current page.
func (r *Router) handlePageReload(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	wait, _ := cmd.Params["wait"].(string)
	if wait == "" {
		wait = "complete"
	}

	params := map[string]interface{}{
		"context": context,
		"wait":    wait,
	}

	resp, err := r.sendInternalCommand(session, "browsingContext.reload", params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		r.sendError(session, cmd.ID, bidiErr)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// handlePageURL handles vibium:page.url — returns the current page URL.
func (r *Router) handlePageURL(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	url, err := r.evalSimpleScript(session, context, "() => window.location.href")
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"url": url})
}

// handlePageTitle handles vibium:page.title — returns the current page title.
func (r *Router) handlePageTitle(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	title, err := r.evalSimpleScript(session, context, "() => document.title")
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"title": title})
}

// handlePageContent handles vibium:page.content — returns the page's full HTML.
func (r *Router) handlePageContent(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	content, err := r.evalSimpleScript(session, context, "() => document.documentElement.outerHTML")
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"content": content})
}

// handlePageWaitForURL handles vibium:page.waitForURL — waits until the URL matches a pattern.
func (r *Router) handlePageWaitForURL(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	pattern, _ := cmd.Params["pattern"].(string)
	if pattern == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("pattern is required"))
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
		url, err := r.evalSimpleScript(session, context, "() => window.location.href")
		if err == nil && matchesPattern(url, pattern) {
			r.sendSuccess(session, cmd.ID, map[string]interface{}{"url": url})
			return
		}

		if time.Now().After(deadline) {
			r.sendError(session, cmd.ID, fmt.Errorf("timeout after %s waiting for URL matching '%s'", timeout, pattern))
			return
		}

		time.Sleep(interval)
	}
}

// handlePageWaitForLoad handles vibium:page.waitForLoad — waits until the page reaches a load state.
func (r *Router) handlePageWaitForLoad(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	state, _ := cmd.Params["state"].(string)
	if state == "" {
		state = "complete"
	}

	timeoutMs, _ := cmd.Params["timeout"].(float64)
	timeout := defaultTimeout
	if timeoutMs > 0 {
		timeout = time.Duration(timeoutMs) * time.Millisecond
	}

	if err := r.waitForReadyState(session, context, state, timeout); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// waitForReadyState polls document.readyState until it matches the target state.
func (r *Router) waitForReadyState(session *BrowserSession, context, targetState string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond

	for {
		state, err := r.evalSimpleScript(session, context, "() => document.readyState")
		if err == nil && readyStateReached(state, targetState) {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout after %s waiting for readyState '%s'", timeout, targetState)
		}

		time.Sleep(interval)
	}
}

// readyStateReached checks if the current readyState meets or exceeds the target.
// Order: loading < interactive < complete
func readyStateReached(current, target string) bool {
	states := map[string]int{"loading": 0, "interactive": 1, "complete": 2}
	c, ok1 := states[current]
	t, ok2 := states[target]
	if !ok1 || !ok2 {
		return current == target
	}
	return c >= t
}

// matchesPattern checks if a URL matches a pattern.
// Supports simple string containment and glob-like patterns with *.
func matchesPattern(url, pattern string) bool {
	// Exact match
	if url == pattern {
		return true
	}

	// Simple glob: if pattern has *, do basic wildcard matching
	if strings.Contains(pattern, "*") {
		return globMatch(url, pattern)
	}

	// Substring match
	return strings.Contains(url, pattern)
}

// globMatch performs simple glob matching where * matches any characters.
func globMatch(s, pattern string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) == 0 {
		return true
	}

	pos := 0
	for i, part := range parts {
		if part == "" {
			continue
		}
		idx := strings.Index(s[pos:], part)
		if idx < 0 {
			return false
		}
		if i == 0 && idx != 0 {
			// First part must match at start if pattern doesn't start with *
			return false
		}
		pos += idx + len(part)
	}

	// If pattern doesn't end with *, the last part must match at the end
	lastPart := parts[len(parts)-1]
	if lastPart != "" && !strings.HasSuffix(s, lastPart) {
		return false
	}

	return true
}
