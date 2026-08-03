package api

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/vibium/clicker/internal/urlmatch"
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
	s := NewAPISession(r, session, context)
	if err := Navigate(s, context, url, wait); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Capture filmstrip screenshot while page is in its clean post-navigate state,
	// before sendSuccess unblocks the client to send further commands.
	session.mu.Lock()
	recorder := session.recorder
	session.mu.Unlock()
	if recorder != nil && recorder.IsRecording() {
		ps := NewAPISession(r, session, context)
		CaptureRecordingScreenshot(ps, recorder, time.Now())
		atomic.StoreInt32(&session.handlerScreenshot, 1)
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

	s := NewAPISession(r, session, context)
	if err := GoBack(s, context); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// handlePageForward handles vibium:page.forward — navigates forward in history.
func (r *Router) handlePageForward(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	s := NewAPISession(r, session, context)
	if err := GoForward(s, context); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

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
	s := NewAPISession(r, session, context)
	if err := Reload(s, context, wait); err != nil {
		r.sendError(session, cmd.ID, err)
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

	s := NewAPISession(r, session, context)
	url, err := GetURL(s, context)
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

	s := NewAPISession(r, session, context)
	title, err := GetTitle(s, context)
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

	s := NewAPISession(r, session, context)
	content, err := GetContent(s, context)
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
	timeout := DefaultTimeout
	if timeoutMs > 0 {
		timeout = time.Duration(timeoutMs) * time.Millisecond
	}

	s := NewAPISession(r, session, context)
	url, err := WaitForURL(s, context, pattern, timeout)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"url": url})
}

// handlePageWaitForLoad handles vibium:page.waitForLoad — waits until the page reaches a load state.
func (r *Router) handlePageWaitForLoad(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	state, _ := cmd.Params["state"].(string)
	timeoutMs, _ := cmd.Params["timeout"].(float64)
	timeout := DefaultTimeout
	if timeoutMs > 0 {
		timeout = time.Duration(timeoutMs) * time.Millisecond
	}

	s := NewAPISession(r, session, context)
	if err := WaitForLoad(s, context, state, timeout); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// waitForReadyState polls document.readyState until it matches the target state.
func (r *Router) waitForReadyState(session *BrowserSession, context, targetState string, timeout time.Duration) error {
	return WaitForReadyState(NewAPISession(r, session, context), context, targetState, timeout)
}

// ---------------------------------------------------------------------------
// Exported standalone navigation functions — usable from both proxy and MCP.
// ---------------------------------------------------------------------------

// Navigate navigates to a URL and waits for the given load state.
func Navigate(s Session, context, url, wait string) error {
	if wait == "" {
		wait = "complete"
	}

	params := map[string]interface{}{
		"context": context,
		"url":     url,
		"wait":    wait,
	}

	resp, err := s.SendBidiCommand("browsingContext.navigate", params)
	if err != nil {
		return err
	}

	if bidiErr := checkBidiError(resp); bidiErr != nil {
		return bidiErr
	}

	return nil
}

// GoBack navigates back in history.
func GoBack(s Session, context string) error {
	params := map[string]interface{}{
		"context": context,
		"delta":   -1,
	}

	resp, err := s.SendBidiCommand("browsingContext.traverseHistory", params)
	if err != nil {
		return err
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		return bidiErr
	}

	// Wait for page load after traversal
	WaitForReadyState(s, context, "complete", 10*time.Second)

	return nil
}

// GoForward navigates forward in history.
func GoForward(s Session, context string) error {
	params := map[string]interface{}{
		"context": context,
		"delta":   1,
	}

	resp, err := s.SendBidiCommand("browsingContext.traverseHistory", params)
	if err != nil {
		return err
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		return bidiErr
	}

	// Wait for page load after traversal
	WaitForReadyState(s, context, "complete", 10*time.Second)

	return nil
}

// Reload reloads the current page and waits for the given load state.
func Reload(s Session, context, wait string) error {
	if wait == "" {
		wait = "complete"
	}

	params := map[string]interface{}{
		"context": context,
		"wait":    wait,
	}

	resp, err := s.SendBidiCommand("browsingContext.reload", params)
	if err != nil {
		return err
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		return bidiErr
	}

	return nil
}

// GetURL returns the current page URL.
func GetURL(s Session, context string) (string, error) {
	return EvalSimpleScript(s, context, "() => window.location.href")
}

// GetTitle returns the current page title.
func GetTitle(s Session, context string) (string, error) {
	return EvalSimpleScript(s, context, "() => document.title")
}

// GetContent returns the page's full HTML.
func GetContent(s Session, context string) (string, error) {
	return EvalSimpleScript(s, context, "() => document.documentElement.outerHTML")
}

// WaitForURL waits until the URL matches a pattern.
func WaitForURL(s Session, context, pattern string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond

	for {
		url, err := EvalSimpleScript(s, context, "() => window.location.href")
		if err == nil && urlmatch.Loose(pattern, url) {
			return url, nil
		}

		if time.Now().After(deadline) {
			return "", fmt.Errorf("timeout after %s waiting for URL matching '%s'", timeout, pattern)
		}

		time.Sleep(interval)
	}
}

// WaitForLoad waits until the page reaches a given load state.
func WaitForLoad(s Session, context, state string, timeout time.Duration) error {
	if state == "" {
		state = "complete"
	}
	return WaitForReadyState(s, context, state, timeout)
}

// WaitForReadyState polls document.readyState until it matches the target state.
func WaitForReadyState(s Session, context, targetState string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond

	for {
		state, err := EvalSimpleScript(s, context, "() => document.readyState")
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
