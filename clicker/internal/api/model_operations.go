package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vibium/clicker/internal/ai"
	"github.com/vibium/clicker/internal/check"
	runop "github.com/vibium/clicker/internal/run"
)

type modelReply struct {
	result interface{}
	err    error
}

func (s *BrowserSession) replyToModel(id int, result interface{}, err error) bool {
	s.mu.Lock()
	ch := s.modelReplies[id]
	s.mu.Unlock()
	if ch == nil {
		return false
	}
	ch <- modelReply{result, err}
	return true
}

// Recorded commands are handled before session lookup. The existing pipe can
// therefore service an archive without owning or launching a browser.
func (r *Router) handleRecordedCheck(client ClientTransport, raw string) bool {
	var cmd bidiCommand
	if json.Unmarshal([]byte(raw), &cmd) != nil || cmd.Method != check.Method {
		return false
	}
	if _, exists := cmd.Params["record"]; !exists {
		return false
	}
	go func() {
		result, err := recordedCheck(cmd.Params)
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
func recordedCheck(params map[string]interface{}) (ai.Result, error) {
	for k := range params {
		if k != "claim" && k != "record" && !ai.IsOverride(k) {
			return ai.Result{}, fmt.Errorf("record cannot be combined with live session selection or other options")
		}
	}
	claim, ok := params["claim"].(string)
	if !ok {
		return ai.Result{}, fmt.Errorf("claim is required")
	}
	record, ok := params["record"].(string)
	if !ok || record == "" {
		return ai.Result{}, fmt.Errorf("record must be a nonempty path")
	}
	config, err := ai.ConfigFromParams("check", params)
	if err != nil {
		return ai.Result{}, err
	}
	return check.CheckRecord(context.Background(), check.Request{Claim: claim, Record: record, Config: config})
}

func (r *Router) handleCheck(session *BrowserSession, cmd bidiCommand) {
	r.handleModelOperation(session, cmd, false)
}
func (r *Router) handleRun(session *BrowserSession, cmd bidiCommand) {
	r.handleModelOperation(session, cmd, true)
}

func (r *Router) handleModelOperation(session *BrowserSession, cmd bidiCommand, isRun bool) {
	label, method, inputKey, role := "Check", check.Method, "claim", "check"
	if isRun {
		label, method, inputKey, role = "Run", runop.Method, "goal", "run"
	}
	for k := range cmd.Params {
		if k != inputKey && k != "context" && k != "baseURL" && !ai.IsOverride(k) {
			r.sendError(session, cmd.ID, fmt.Errorf("unsupported %s argument", label))
			return
		}
	}
	if v, exists := cmd.Params["context"]; exists {
		if c, ok := v.(string); !ok || c == "" {
			r.sendError(session, cmd.ID, fmt.Errorf("context must be a nonempty page ID"))
			return
		}
	}
	baseSite := ""
	if v, exists := cmd.Params["baseURL"]; exists {
		var ok bool
		if baseSite, ok = v.(string); !ok || baseSite == "" {
			r.sendError(session, cmd.ID, fmt.Errorf("baseURL must be a nonempty site URL"))
			return
		}
	}
	claim, ok := cmd.Params[inputKey].(string)
	if !ok {
		r.sendError(session, cmd.ID, fmt.Errorf("%s is required", inputKey))
		return
	}
	if r.connectURL != "" {
		r.sendError(session, cmd.ID, fmt.Errorf("live %s requires a local browser", label))
		return
	}
	config, err := ai.ConfigFromParams(role, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	var run func(context.Context, ai.ToolExecutor) (ai.RecordedResult, error)
	if isRun {
		req := runop.Request{Goal: claim, BaseSite: baseSite, Config: config}
		if err := req.Validate(); err != nil {
			r.sendError(session, cmd.ID, err)
			return
		}
		run = func(ctx context.Context, tools ai.ToolExecutor) (ai.RecordedResult, error) {
			return runop.Run(ctx, req, tools)
		}
	} else {
		req := check.Request{Claim: claim, BaseSite: baseSite, Config: config}
		if err := req.Validate(); err != nil {
			r.sendError(session, cmd.ID, err)
			return
		}
		run = func(ctx context.Context, tools ai.ToolExecutor) (ai.RecordedResult, error) {
			return check.Check(ctx, req, tools)
		}
	}
	session.modelMu.Lock()
	defer session.modelMu.Unlock()
	session.dispatchMu.Lock()
	defer session.dispatchMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), ai.Timeout)
	defer cancel()
	session.mu.Lock()
	session.modelContext = ctx
	recorder := session.recorder
	session.modelReplies = map[int]chan modelReply{}
	session.mu.Unlock()
	defer func() {
		session.mu.Lock()
		session.modelContext = nil
		session.modelReplies = nil
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
		group = recorder.StartGroup(label + ": " + claim)
		recorder.SetGroupParams(group, map[string]interface{}{"name": label + ": " + claim, "method": method, "modelConfig": config.RecordingMetadata(), inputKey: claim})
	}
	executor := ai.NewModelToolExecutor(&apiModelTools{r: r, session: session, page: page, recorder: recorder}, ai.ToolPolicy{CredentialInput: isRun}, false)
	result, err := run(ctx, executor)
	if group != "" {
		recorder.StopGroup()
		if err != nil {
			recorder.RecordCallOutcome(group, nil, fmt.Errorf("%s execution failed", strings.ToLower(label)))
		} else {
			recorder.RecordCallOutcome(group, result, nil)
			status, summary, evidence := result.RecordingSummary()
			title := label + ": " + claim + " — " + status + ": " + summary
			for _, e := range evidence {
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

// apiModelTools is the pipe-surface ModelDispatcher: the shared policy in
// verifier has already validated the call and applied the navigation,
// scroll, timeout, and password guards; only the routing into the vibium
// handlers and the script probe live here (#571). SDK clients manage their
// own @e refs, so this surface passes refs=false and takes CSS selectors
// only.
type apiModelTools struct {
	r        *Router
	session  *BrowserSession
	page     string
	recorder *Recorder
	nextID   int
}

func (v *apiModelTools) ProbeSecret(ctx context.Context, selector, name string) (bool, error) {
	// CallScript returns strings, so the shared bool-returning probe is
	// wrapped in String().
	script := `(selector) => String((` + ai.SecretProbeJS(PierceQueryJS()) + `)(selector))`
	data, err := CallScript(NewAPISession(v.r, v.session, v.page), v.page, script, []map[string]interface{}{{"type": "string", "value": selector}})
	if err != nil {
		return false, err
	}
	secret, err := parseScriptResult(data)
	if err != nil {
		return false, err
	}
	return secret == "true", nil
}

func (v *apiModelTools) Dispatch(ctx context.Context, name string, args map[string]interface{}) (ai.Observation, error) {
	params := map[string]interface{}{"context": v.page}
	for k, x := range args {
		params[k] = x
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
	ch := make(chan modelReply, 1)
	done := make(chan struct{})
	callID := ""
	v.session.mu.Lock()
	v.session.modelReplies[id] = ch
	v.session.mu.Unlock()
	defer func() { v.session.mu.Lock(); delete(v.session.modelReplies, id); v.session.mu.Unlock() }()
	v.r.dispatch(v.session, bidiCommand{ID: id, Method: method, Params: params, modelDone: done, modelCallID: &callID}, handler)
	// Wait for recording cleanup too; no action goroutine may outlive the parent.
	<-done
	reply := <-ch
	if callID != "" && v.recorder != nil {
		if name == "browser_screenshot" {
			v.recorder.RecordCallOutcome(callID, map[string]interface{}{"observation": "Screenshot captured"}, reply.err)
		} else {
			data, _ := json.Marshal(reply.result)
			v.recorder.RecordCallOutcome(callID, map[string]interface{}{"observation": ai.Clip(string(data))}, reply.err)
		}
	}
	if reply.err != nil {
		return ai.Observation{}, &ai.ActionError{Err: reply.err}
	}
	if name == "browser_screenshot" {
		m, _ := reply.result.(map[string]interface{})
		data, _ := m["data"].(string)
		if len(data) > ai.MaxImage {
			return ai.Observation{Text: "Screenshot exceeds payload limit"}, nil
		}
		return ai.Observation{Text: "Screenshot captured", Image: data}, nil
	}
	data, _ := json.Marshal(reply.result)
	return ai.Observation{Text: ai.Clip(string(data))}, nil
}
