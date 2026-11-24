package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// handlePageRoute handles vibium:page.route — registers a route pattern and
// ensures one beforeRequestSent intercept serves the context. The engine owns
// the pattern matching: blocked request events reach clients annotated with
// vibiumMatchedPatterns (see routeRegistry), so clients dispatch handlers
// without their own glob implementations.
func (r *Router) handlePageRoute(session *BrowserSession, cmd bidiCommand) {
	// Web Bot Auth signing owns the one beforeRequestSent intercept for the
	// session and continues every request itself. A client route would install
	// a second intercept and race that continue (one blocked event, two
	// continues), so the two cannot share a session yet (#28).
	if r.signer != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("page.route is not supported while Web Bot Auth signing is enabled"))
		return
	}
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

	interceptID, needIntercept, err := session.routes.add(context, pattern)
	if err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("invalid pattern: %w", err))
		return
	}

	if needIntercept {
		resp, err := r.sendInternalCommand(session, "network.addIntercept", map[string]interface{}{
			"phases":   []string{"beforeRequestSent"},
			"contexts": []interface{}{context},
		})
		if err == nil {
		}
		if err != nil {
			session.routes.remove(context, pattern)
			r.sendError(session, cmd.ID, err)
			return
		}
		var result struct {
			Result struct {
				Intercept string `json:"intercept"`
			} `json:"result"`
		}
		if err := json.Unmarshal(resp, &result); err != nil {
			session.routes.remove(context, pattern)
			r.sendError(session, cmd.ID, fmt.Errorf("failed to parse addIntercept response: %w", err))
			return
		}
		session.routes.setIntercept(context, result.Result.Intercept)
		interceptID = result.Result.Intercept
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"intercept": interceptID})
}

// handlePageUnroute handles vibium:page.unroute — deregisters a route pattern
// and tears the context's intercept down once no patterns remain. The older
// intercept-id form is still accepted for callers that manage their own
// intercepts.
func (r *Router) handlePageUnroute(session *BrowserSession, cmd bidiCommand) {
	pattern, _ := cmd.Params["pattern"].(string)
	intercept, _ := cmd.Params["intercept"].(string)

	if pattern != "" {
		context, err := r.resolveContext(session, cmd.Params)
		if err != nil {
			r.sendError(session, cmd.ID, err)
			return
		}
		interceptID, empty := session.routes.remove(context, pattern)
		if !empty || interceptID == "" {
			r.sendSuccess(session, cmd.ID, map[string]interface{}{})
			return
		}
		intercept = interceptID
	}
	if intercept == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("pattern or intercept is required"))
		return
	}

	_, err := r.sendInternalCommand(session, "network.removeIntercept", map[string]interface{}{
		"intercept": intercept,
	})
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// handleNetworkContinue handles vibium:network.continue — continues an intercepted request.
// Optional overrides: url, method, headers, body.
func (r *Router) handleNetworkContinue(session *BrowserSession, cmd bidiCommand) {
	request, _ := cmd.Params["request"].(string)
	if request == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("request is required"))
		return
	}

	params := map[string]interface{}{
		"request": request,
	}

	if url, ok := cmd.Params["url"].(string); ok && url != "" {
		params["url"] = url
	}
	if method, ok := cmd.Params["method"].(string); ok && method != "" {
		params["method"] = method
	}
	if body, ok := cmd.Params["body"].(string); ok {
		params["body"] = map[string]interface{}{
			"type":  "string",
			"value": body,
		}
	}

	// Convert headers from {"Name": "Value"} to BiDi format [{name, value: {type, value}}]
	if headers, ok := cmd.Params["headers"].(map[string]interface{}); ok {
		params["headers"] = convertHeadersToBidi(headers)
	}

	_, err := r.sendInternalCommand(session, "network.continueRequest", params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// handleNetworkFulfill handles vibium:network.fulfill — provides a response for an intercepted request.
// Optional: statusCode, headers, body, reasonPhrase.
func (r *Router) handleNetworkFulfill(session *BrowserSession, cmd bidiCommand) {
	request, _ := cmd.Params["request"].(string)
	if request == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("request is required"))
		return
	}

	params := map[string]interface{}{
		"request": request,
	}

	if statusCode, ok := cmd.Params["statusCode"].(float64); ok {
		params["statusCode"] = int(statusCode)
	}
	if reasonPhrase, ok := cmd.Params["reasonPhrase"].(string); ok && reasonPhrase != "" {
		params["reasonPhrase"] = reasonPhrase
	}
	if body, ok := cmd.Params["body"].(string); ok {
		params["body"] = map[string]interface{}{
			"type":  "string",
			"value": body,
		}
	}

	// Convert headers from {"Name": "Value"} to BiDi format
	headers, hasHeaders := cmd.Params["headers"].(map[string]interface{})
	if !hasHeaders {
		headers = map[string]interface{}{}
	}

	// contentType convenience: inject Content-Type if not already present
	if contentType, ok := cmd.Params["contentType"].(string); ok && contentType != "" {
		hasContentType := false
		for name := range headers {
			if strings.EqualFold(name, "content-type") {
				hasContentType = true
				break
			}
		}
		if !hasContentType {
			headers["Content-Type"] = contentType
		}
	}

	if len(headers) > 0 {
		params["headers"] = convertHeadersToBidi(headers)
	}

	_, err := r.sendInternalCommand(session, "network.provideResponse", params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// handleNetworkAbort handles vibium:network.abort — fails an intercepted request.
func (r *Router) handleNetworkAbort(session *BrowserSession, cmd bidiCommand) {
	request, _ := cmd.Params["request"].(string)
	if request == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("request is required"))
		return
	}

	params := map[string]interface{}{
		"request": request,
	}

	_, err := r.sendInternalCommand(session, "network.failRequest", params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// handlePageSetHeaders handles vibium:page.setHeaders: the browser adds the
// headers to every request the context makes. An empty map clears them.
func (r *Router) handlePageSetHeaders(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	headers, ok := cmd.Params["headers"].(map[string]interface{})
	if !ok {
		r.sendError(session, cmd.ID, fmt.Errorf("headers is required"))
		return
	}

	if _, err := r.sendInternalCommand(session, "network.setExtraHeaders", map[string]interface{}{
		"headers":  convertHeadersToBidi(headers),
		"contexts": []interface{}{context},
	}); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// convertHeadersToBidi converts headers from {"Name": "Value"} to BiDi format:
// [{name: "Name", value: {type: "string", value: "Value"}}]
func convertHeadersToBidi(headers map[string]interface{}) []map[string]interface{} {
	bidiHeaders := make([]map[string]interface{}, 0, len(headers))
	for name, val := range headers {
		valStr, _ := val.(string)
		bidiHeaders = append(bidiHeaders, map[string]interface{}{
			"name": name,
			"value": map[string]interface{}{
				"type":  "string",
				"value": valStr,
			},
		})
	}
	return bidiHeaders
}

// registerNetworkCapture handles the registration half of
// vibium:page.captureRequest / captureResponse and returns the handler that
// waits the capture out. The router calls this inline on the client message
// order and dispatches only the returned wait: registered on the dispatch
// goroutine instead, the capture could lose the race against the very next
// client command triggering its traffic, which a short trigger fuse in a
// loaded Firefox run turned into a real timeout. A nil return means the
// command was already answered with an error.
func (r *Router) registerNetworkCapture(session *BrowserSession, cmd bidiCommand, eventMethod, what string) vibiumHandler {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return nil
	}
	pattern, _ := cmd.Params["pattern"].(string)
	if pattern == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("pattern is required"))
		return nil
	}
	timeoutMs := 10000.0
	if t, ok := cmd.Params["timeout"].(float64); ok && t > 0 {
		timeoutMs = t
	}

	id, ch, err := session.captures.register(eventMethod, context, pattern)
	if err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("invalid pattern: %w", err))
		return nil
	}

	return func(session *BrowserSession, cmd bidiCommand) {
		select {
		case params := <-ch:
			var event interface{}
			if err := json.Unmarshal(params, &event); err != nil {
				r.sendError(session, cmd.ID, fmt.Errorf("capture event unparseable: %w", err))
				return
			}
			r.sendSuccess(session, cmd.ID, map[string]interface{}{"event": event})
		case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
			session.captures.cancel(id)
			r.sendError(session, cmd.ID, fmt.Errorf("timeout waiting for %s matching %q", what, pattern))
		case <-session.stopChan:
			session.captures.cancel(id)
		}
	}
}
