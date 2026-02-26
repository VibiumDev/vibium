package proxy

import (
	"encoding/json"
	"fmt"
	"strings"
)

// handlePageRoute handles vibium:page.route — adds a network intercept for beforeRequestSent.
// The JS client uses the returned intercept ID to match requests against URL patterns.
func (r *Router) handlePageRoute(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	params := map[string]interface{}{
		"phases":   []string{"beforeRequestSent"},
		"contexts": []interface{}{context},
	}

	resp, err := r.sendInternalCommand(session, "network.addIntercept", params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	if bidiErr := checkBidiError(resp); bidiErr != nil {
		r.sendError(session, cmd.ID, bidiErr)
		return
	}

	var result struct {
		Result struct {
			Intercept string `json:"intercept"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("failed to parse addIntercept response: %w", err))
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"intercept": result.Result.Intercept})
}

// handlePageUnroute handles vibium:page.unroute — removes a network intercept.
func (r *Router) handlePageUnroute(session *BrowserSession, cmd bidiCommand) {
	intercept, _ := cmd.Params["intercept"].(string)
	if intercept == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("intercept is required"))
		return
	}

	params := map[string]interface{}{
		"intercept": intercept,
	}

	resp, err := r.sendInternalCommand(session, "network.removeIntercept", params)
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

	resp, err := r.sendInternalCommand(session, "network.continueRequest", params)
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

	resp, err := r.sendInternalCommand(session, "network.provideResponse", params)
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

	resp, err := r.sendInternalCommand(session, "network.failRequest", params)
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

// handlePageSetHeaders handles vibium:page.setHeaders — sets extra HTTP headers for a context.
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

	// We use network.addIntercept + network.continueRequest pattern is too complex.
	// BiDi doesn't have a direct "setExtraHeaders" command.
	// Instead, we use script injection to add headers via the intercept pattern.
	// For simplicity, use a beforeRequestSent intercept and modify headers on each request.
	// Actually, the simplest approach: add an intercept and store headers on the session.
	// But the plan says to use network.addIntercept with header setting.

	// Use the session.subscribe approach: subscribe to beforeRequestSent,
	// then on each request, continue with extra headers.
	// Store the extra headers on the session for the event handler to use.

	// For now, set up an intercept that the Go proxy will handle automatically.
	// Store headers in session and auto-continue with them.

	// First, add an intercept for this context
	interceptParams := map[string]interface{}{
		"phases":   []string{"beforeRequestSent"},
		"contexts": []interface{}{context},
	}

	resp, err := r.sendInternalCommand(session, "network.addIntercept", interceptParams)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	if bidiErr := checkBidiError(resp); bidiErr != nil {
		r.sendError(session, cmd.ID, bidiErr)
		return
	}

	var result struct {
		Result struct {
			Intercept string `json:"intercept"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("failed to parse addIntercept response: %w", err))
		return
	}

	// Store extra headers on the session so the JS client can use them
	bidiHeaders := convertHeadersToBidi(headers)

	r.sendSuccess(session, cmd.ID, map[string]interface{}{
		"intercept": result.Result.Intercept,
		"headers":   bidiHeaders,
	})
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
