package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/vibium/clicker/internal/verifier"
)

type verifyReply struct {
	result interface{}
	err    error
}

func (s *BrowserSession) replyToVerifier(id int, result interface{}, err error) bool {
	s.mu.Lock()
	ch := s.verifyReplies[id]
	s.mu.Unlock()
	if ch == nil {
		return false
	}
	ch <- verifyReply{result, err}
	return true
}

// Recorded commands are handled before session lookup. The existing pipe can
// therefore service an archive without owning or launching a browser.
func (r *Router) handleRecordedVerify(client ClientTransport, raw string) bool {
	var cmd bidiCommand
	if json.Unmarshal([]byte(raw), &cmd) != nil || cmd.Method != verifier.Method {
		return false
	}
	if _, exists := cmd.Params["record"]; !exists {
		return false
	}
	go func() {
		result, err := recordedVerify(cmd.Params)
		response := bidiResponse{ID: cmd.ID, Type: "success", Result: result}
		if err != nil {
			response.Type = "error"
			response.Error = "error"
			response.Message = err.Error()
			response.Result = nil
		}
		data, _ := json.Marshal(response)
		client.Send(string(data))
	}()
	return true
}
func recordedVerify(params map[string]interface{}) (verifier.Result, error) {
	for k := range params {
		if k != "claim" && k != "record" {
			return verifier.Result{}, fmt.Errorf("record cannot be combined with live session selection or other options")
		}
	}
	claim, ok := params["claim"].(string)
	if !ok {
		return verifier.Result{}, fmt.Errorf("claim is required")
	}
	record, ok := params["record"].(string)
	if !ok || record == "" {
		return verifier.Result{}, fmt.Errorf("record must be a nonempty path")
	}
	config, err := verifier.ConfigFromEnv()
	if err != nil {
		return verifier.Result{}, err
	}
	return verifier.VerifyRecord(context.Background(), verifier.Request{Claim: claim, Record: record, Config: config})
}

func (r *Router) handleVerify(session *BrowserSession, cmd bidiCommand) {
	for k := range cmd.Params {
		if k != "claim" && k != "context" {
			r.sendError(session, cmd.ID, fmt.Errorf("unsupported verification argument"))
			return
		}
	}
	if v, exists := cmd.Params["context"]; exists {
		if c, ok := v.(string); !ok || c == "" {
			r.sendError(session, cmd.ID, fmt.Errorf("context must be a nonempty page ID"))
			return
		}
	}
	claim, ok := cmd.Params["claim"].(string)
	if !ok {
		r.sendError(session, cmd.ID, fmt.Errorf("claim is required"))
		return
	}
	if r.connectURL != "" {
		r.sendError(session, cmd.ID, fmt.Errorf("live verification requires a local browser"))
		return
	}
	config, err := verifier.ConfigFromEnv()
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	req := verifier.Request{Claim: claim, Config: config}
	if err := req.Validate(); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	session.verifyMu.Lock()
	defer session.verifyMu.Unlock()
	session.dispatchMu.Lock()
	defer session.dispatchMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), verifier.Timeout)
	defer cancel()
	session.mu.Lock()
	session.verifyContext = ctx
	recorder := session.recorder
	session.verifyReplies = map[int]chan verifyReply{}
	session.mu.Unlock()
	defer func() {
		session.mu.Lock()
		session.verifyContext = nil
		session.verifyReplies = nil
		session.mu.Unlock()
	}()
	page, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	// A Page instance explicitly pins its context; a Browser uses the active one.
	var group string
	if recorder != nil && recorder.IsRecording() {
		recorder.RegisterSecret(config.APIKey)
		group = recorder.StartGroup("Verify: " + claim)
		recorder.SetGroupParams(group, map[string]interface{}{"name": "Verify: " + claim, "method": verifier.Method, "claim": claim})
	}
	executor := &apiVerifyTools{r: r, session: session, page: page, recorder: recorder}
	result, err := (&verifier.OpenAI{}).Verify(ctx, req, executor)
	if group != "" {
		recorder.StopGroup()
		if err != nil {
			recorder.RecordCallOutcome(group, nil, fmt.Errorf("verification execution failed"))
		} else {
			recorder.RecordCallOutcome(group, result, nil)
			title := "Verify: " + claim + " — " + result.Verdict() + ": " + result.Summary
			for _, e := range result.Evidence {
				title += " | " + e.Summary
			}
			recorder.SetGroupTitle(group, title)
		}
	}
	if err != nil {
		r.sendError(session, cmd.ID, err)
	} else {
		r.sendSuccess(session, cmd.ID, result)
	}
}

type apiVerifyTools struct {
	r        *Router
	session  *BrowserSession
	page     string
	recorder *Recorder
	nextID   int
}

func (v *apiVerifyTools) Tools() []verifier.Tool {
	var out []verifier.Tool
	for _, name := range []string{"get_url", "map", "a11y_tree", "navigate", "reload", "click", "fill", "type", "press", "scroll", "find", "get_text", "get_value", "screenshot", "console", "network"} {
		props := map[string]interface{}{}
		required := []string{}
		fields := []string{}
		switch name {
		case "click", "get_value", "find":
			fields = []string{"selector"}
		case "fill", "type":
			fields = []string{"selector", "text"}
		case "press":
			fields = []string{"selector", "key"}
		case "navigate":
			fields = []string{"url"}
		}
		for _, field := range fields {
			props[field] = map[string]interface{}{"type": "string"}
			required = append(required, field)
		}
		if name == "get_text" || name == "map" {
			props["selector"] = map[string]interface{}{"type": "string"}
		}
		if name == "scroll" {
			props["direction"] = map[string]interface{}{"type": "string", "enum": []string{"up", "down", "left", "right"}}
			props["amount"] = map[string]interface{}{"type": "number", "minimum": 0, "maximum": 10}
		}
		out = append(out, verifier.Tool{Name: "browser_" + name, Description: "Use existing Vibium " + name + " on the pinned page. Use CSS selectors returned by browser_map; no @refs. Console/network require an active recording.", Parameters: map[string]interface{}{"type": "object", "properties": props, "required": required, "additionalProperties": false}})
	}
	return out
}
func (v *apiVerifyTools) Execute(ctx context.Context, name string, args map[string]interface{}) (verifier.Observation, error) {
	if err := ctx.Err(); err != nil {
		return verifier.Observation{}, err
	}
	var schema map[string]interface{}
	for _, t := range v.Tools() {
		if t.Name == name {
			schema = t.Parameters
		}
	}
	if schema == nil {
		return verifier.Observation{}, fmt.Errorf("disallowed verifier tool")
	}
	props := schema["properties"].(map[string]interface{})
	params := map[string]interface{}{"context": v.page, "timeout": float64(5000)}
	for k, x := range args {
		p, ok := props[k].(map[string]interface{})
		if !ok {
			return verifier.Observation{}, fmt.Errorf("disallowed verifier argument")
		}
		if p["type"] == "string" {
			if _, ok := x.(string); !ok {
				return verifier.Observation{}, fmt.Errorf("invalid verifier argument")
			}
		} else {
			n, ok := x.(float64)
			if !ok || n < 0 || n > 10 {
				return verifier.Observation{}, fmt.Errorf("invalid scroll amount")
			}
		}
		params[k] = x
	}
	for _, k := range schema["required"].([]string) {
		if s, ok := params[k].(string); !ok || s == "" {
			return verifier.Observation{}, fmt.Errorf("missing verifier argument")
		}
	}
	if name == "browser_navigate" {
		u, err := url.Parse(params["url"].(string))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
			return verifier.Observation{}, fmt.Errorf("verifier navigation requires HTTP(S) without credentials")
		}
	}
	if name == "browser_scroll" {
		if d, ok := params["direction"]; ok && d != "up" && d != "down" && d != "left" && d != "right" {
			return verifier.Observation{}, fmt.Errorf("invalid scroll direction")
		}
	}
	if selector, ok := params["selector"].(string); ok && selector != "" {
		script := `(selector) => { ` + PierceQueryJS() + `; const el = pierceQuery(document,selector); return String(!!el && (el.type === 'password' || el.autocomplete === 'current-password' || el.autocomplete === 'new-password')); }`
		data, err := CallScript(NewAPISession(v.r, v.session, v.page), v.page, script, []map[string]interface{}{{"type": "string", "value": selector}})
		if err != nil {
			return verifier.Observation{}, err
		}
		secret, err := parseScriptResult(data)
		if err != nil {
			return verifier.Observation{}, err
		}
		if secret == "true" {
			return verifier.Observation{Text: "Password fields are unavailable; return inconclusive if required."}, nil
		}
	}
	handler := v.r.handlePageURL
	method := "vibium:page.url"
	switch name {
	case "browser_map":
		method = "vibium:page.map"
		handler = func(s *BrowserSession, c bidiCommand) {
			selector, _ := c.Params["selector"].(string)
			data, err := CallScript(NewAPISession(v.r, s, v.page), v.page, MapScript(), []map[string]interface{}{{"type": "string", "value": selector}})
			if err != nil {
				v.r.sendError(s, c.ID, err)
				return
			}
			text, err := parseScriptResult(data)
			if err != nil {
				v.r.sendError(s, c.ID, err)
				return
			}
			v.r.sendSuccess(s, c.ID, map[string]interface{}{"elements": text})
		}
	case "browser_a11y_tree":
		method = "vibium:page.a11yTree"
		handler = v.r.handleVibiumPageA11yTree
	case "browser_navigate":
		method = "vibium:page.navigate"
		handler = v.r.handlePageNavigate
	case "browser_reload":
		method = "vibium:page.reload"
		handler = v.r.handlePageReload
	case "browser_click":
		method = "vibium:element.click"
		handler = v.r.handleVibiumClick
	case "browser_fill":
		method = "vibium:element.fill"
		handler = v.r.handleVibiumFill
		params["value"] = params["text"]
		delete(params, "text")
	case "browser_type":
		method = "vibium:element.type"
		handler = v.r.handleVibiumType
	case "browser_press":
		method = "vibium:element.press"
		handler = v.r.handleVibiumPress
	case "browser_scroll":
		method = "vibium:page.scroll"
		handler = v.r.handlePageScroll
	case "browser_find":
		method = "vibium:page.find"
		handler = v.r.handleVibiumFind
	case "browser_get_text":
		method = "vibium:element.text"
		handler = v.r.handleVibiumElText
		if params["selector"] == nil || params["selector"] == "" {
			params["selector"] = "body"
		}
	case "browser_get_value":
		method = "vibium:element.value"
		handler = v.r.handleVibiumElValue
	case "browser_screenshot":
		method = "vibium:page.screenshot"
		handler = v.r.handlePageScreenshot
	case "browser_console", "browser_network":
		method = "vibium:page." + strings.TrimPrefix(name, "browser_")
		handler = func(s *BrowserSession, c bidiCommand) {
			if v.recorder == nil || !v.recorder.IsRecording() {
				v.r.sendSuccess(s, c.ID, map[string]interface{}{"unavailable": "No active recording; absence is not proof of no errors"})
				return
			}
			v.r.sendSuccess(s, c.ID, v.recorder.BrowserObservations(strings.TrimPrefix(name, "browser_"), v.page))
		}
	}
	v.nextID--
	id := v.nextID
	ch := make(chan verifyReply, 1)
	done := make(chan struct{})
	callID := ""
	v.session.mu.Lock()
	v.session.verifyReplies[id] = ch
	v.session.mu.Unlock()
	defer func() { v.session.mu.Lock(); delete(v.session.verifyReplies, id); v.session.mu.Unlock() }()
	v.r.dispatch(v.session, bidiCommand{ID: id, Method: method, Params: params, verifyDone: done, verifyCallID: &callID}, handler)
	// Wait for recording cleanup too; no action goroutine may outlive the parent.
	<-done
	reply := <-ch
	if callID != "" && v.recorder != nil {
		if name == "browser_screenshot" {
			v.recorder.RecordCallOutcome(callID, map[string]interface{}{"observation": "Screenshot captured"}, reply.err)
		} else {
			data, _ := json.Marshal(reply.result)
			v.recorder.RecordCallOutcome(callID, map[string]interface{}{"observation": verifier.Clip(string(data))}, reply.err)
		}
	}
	if reply.err != nil {
		return verifier.Observation{}, reply.err
	}
	if name == "browser_screenshot" {
		m, _ := reply.result.(map[string]interface{})
		data, _ := m["data"].(string)
		if len(data) > verifier.MaxImage {
			return verifier.Observation{Text: "Screenshot exceeds payload limit"}, nil
		}
		return verifier.Observation{Text: "Screenshot captured", Image: data}, nil
	}
	data, _ := json.Marshal(reply.result)
	return verifier.Observation{Text: verifier.Clip(string(data))}, nil
}
