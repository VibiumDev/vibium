package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/vibium/clicker/internal/api"
	"github.com/vibium/clicker/internal/verifier"
)

// Verify runs under the daemon mutex or the MCP server's serialized handler.
func (h *Handlers) Verify(req verifier.Request) (result verifier.Result, err error) {
	if err = req.Validate(); err != nil {
		return
	}
	if req.Record != "" {
		return verifier.VerifyRecord(context.Background(), req)
	}
	if h.connectURL != "" || (h.launchedEngine != "chrome" && h.launchedEngine != "firefox") || h.client == nil {
		return result, fmt.Errorf("verify requires an existing local Chrome or Firefox session; run vibium go first")
	}
	if dead, _ := h.client.Dead(); dead {
		return result, fmt.Errorf("verify browser session is no longer usable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), verifier.Timeout)
	defer cancel()
	restore := h.client.SetCommandContext(ctx)
	defer restore()
	page, err := h.newSession().GetContextID()
	if err != nil {
		return result, err
	}
	executor := &verifyTools{h: h, page: page}
	h.verifying = true
	defer func() { h.verifying = false }()
	var group string
	if h.recorder != nil && h.recorder.IsRecording() {
		group = h.recorder.StartGroup("Verify: " + req.Claim)
		h.recorder.SetGroupParams(group, map[string]interface{}{"name": "Verify: " + req.Claim, "method": verifier.Method, "claim": req.Claim})
		defer func() {
			h.recorder.StopGroup()
			if err != nil {
				h.recorder.RecordCallOutcome(group, nil, fmt.Errorf("verification execution failed"))
				return
			}
			h.recorder.RecordCallOutcome(group, result, nil)
			// Existing Record Player displays group titles; retain the verdict and
			// concise evidence there as well as the structured standard after.result.
			title := "Verify: " + req.Claim + " — " + result.Verdict() + ": " + result.Summary
			for _, evidence := range result.Evidence {
				title += " | " + evidence.Summary
			}
			h.recorder.SetGroupTitle(group, title)
		}()
	}
	var adapter verifier.Verifier = &verifier.OpenAI{}
	result, err = adapter.Verify(ctx, req, executor)
	return
}

func (h *Handlers) verifyMCP(args map[string]interface{}) (*ToolsCallResult, error) {
	for key := range args {
		if key != "claim" && key != "record" {
			return nil, fmt.Errorf("unsupported verification argument %s", key)
		}
	}
	claim, ok := args["claim"].(string)
	if !ok {
		return nil, fmt.Errorf("claim is required")
	}
	record := ""
	if v, ok := args["record"]; ok {
		var valid bool
		record, valid = v.(string)
		if !valid || record == "" {
			return nil, fmt.Errorf("record must be a nonempty path")
		}
	}
	config, err := verifier.ConfigFromEnv()
	if err != nil {
		return nil, err
	}
	if record == "" {
		if err := h.ensureBrowser(); err != nil {
			return nil, err
		}
	}
	result, err := h.Verify(verifier.Request{Claim: claim, Record: record, Config: config})
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(result)
	return &ToolsCallResult{Content: []Content{{Type: "text", Text: string(data)}}}, nil
}

type verifyTools struct {
	h    *Handlers
	page string
}

var verifyAllowed = map[string]bool{
	"browser_get_url": true, "browser_map": true, "browser_a11y_tree": true,
	"browser_navigate": true, "browser_reload": true, "browser_click": true,
	"browser_fill": true, "browser_type": true, "browser_press": true,
	"browser_scroll": true, "browser_find": true, "browser_get_text": true,
	"browser_get_value": true, "browser_screenshot": true,
}

func (v *verifyTools) Tools() []verifier.Tool {
	var result []verifier.Tool
	for _, t := range GetToolSchemas() {
		if !verifyAllowed[t.Name] {
			continue
		}
		props := t.InputSchema["properties"].(map[string]interface{})
		// Pin all tools to the existing page; no arbitrary file output or hidden
		// handler parameters may cross this boundary.
		delete(props, "page")
		if t.Name == "browser_screenshot" {
			delete(props, "filename")
			delete(props, "annotate")
			delete(props, "fullPage")
		}
		result = append(result, verifier.Tool{Name: t.Name, Description: t.Description, Parameters: t.InputSchema})
	}
	for _, kind := range []string{"console", "network"} {
		result = append(result, verifier.Tool{Name: "browser_" + kind, Description: "Inspect up to 50 recent " + kind + " observations (newest first) from this page's active live recording. Unavailable if recording is off; absence is not proof of no errors. Headers, cookies and bodies are excluded.", Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "additionalProperties": false}})
	}
	return result
}

func (v *verifyTools) Execute(ctx context.Context, name string, args map[string]interface{}) (verifier.Observation, error) {
	if err := ctx.Err(); err != nil {
		return verifier.Observation{}, err
	}
	var schema map[string]interface{}
	for _, t := range v.Tools() {
		if t.Name == name {
			schema = t.Parameters
			break
		}
	}
	if schema == nil {
		return verifier.Observation{}, fmt.Errorf("disallowed verifier tool")
	}
	props := schema["properties"].(map[string]interface{})
	clean := map[string]interface{}{}
	for key, val := range args {
		prop, ok := props[key].(map[string]interface{})
		if !ok {
			return verifier.Observation{}, fmt.Errorf("disallowed verifier argument %q", key)
		}
		valid := false
		switch prop["type"] {
		case "string":
			_, valid = val.(string)
		case "number", "integer":
			n, ok := val.(float64)
			valid = ok && n >= 0 && n <= 30000
		case "boolean":
			_, valid = val.(bool)
		}
		if !valid {
			return verifier.Observation{}, fmt.Errorf("invalid verifier argument %q", key)
		}
		if enum, ok := prop["enum"].([]string); ok {
			found := false
			for _, e := range enum {
				if e == val {
					found = true
				}
			}
			if !found {
				return verifier.Observation{}, fmt.Errorf("invalid verifier argument %q", key)
			}
		}
		clean[key] = val
	}
	if required, ok := schema["required"].([]string); ok {
		for _, key := range required {
			if val, ok := clean[key]; !ok || val == "" {
				return verifier.Observation{}, fmt.Errorf("missing verifier argument %q", key)
			}
		}
	}
	if name == "browser_navigate" {
		u, err := url.Parse(clean["url"].(string))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
			return verifier.Observation{}, fmt.Errorf("verifier navigation requires an HTTP(S) URL without credentials")
		}
	}
	if name == "browser_scroll" {
		if n, ok := clean["amount"].(float64); ok && n > 10 {
			return verifier.Observation{}, fmt.Errorf("verifier scroll amount exceeds 10")
		}
	}
	if _, ok := props["timeout"]; ok {
		remaining := 5 * time.Second
		if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < remaining {
			remaining = time.Until(deadline)
		}
		if supplied, ok := clean["timeout"].(float64); !ok || supplied > float64(remaining.Milliseconds()) || supplied == 0 {
			clean["timeout"] = float64(remaining.Milliseconds())
		}
	}
	// Never read or type into a password field. This fixed inspection uses the
	// existing BiDi client; it does not expose eval to the model.
	selector, _ := clean["selector"].(string)
	if selector != "" || name == "browser_press" {
		script := `(selector) => { ` + api.PierceQueryJS() + `; let el = selector ? pierceQuery(document, selector) : document.activeElement; while (el && el.shadowRoot && el.shadowRoot.activeElement) el = el.shadowRoot.activeElement; return !!el && (el.type === 'password' || el.autocomplete === 'current-password' || el.autocomplete === 'new-password'); }`
		secret, err := v.h.client.CallFunction(v.page, script, []interface{}{v.h.resolveSelector(selector)})
		if err != nil {
			return verifier.Observation{}, fmt.Errorf("cannot inspect verifier target")
		}
		if secret == true {
			return verifier.Observation{Text: "Password fields are unavailable to the verifier. Return inconclusive if required."}, nil
		}
	}
	clean["page"] = v.page
	result, err := v.h.Call(name, clean)
	if err != nil {
		return verifier.Observation{}, err
	}
	var obs verifier.Observation
	for _, c := range result.Content {
		if c.Type == "text" {
			obs.Text += c.Text + "\n"
		}
		if c.Type == "image" {
			obs.Image = c.Data
			obs.Text += "Screenshot captured."
		}
	}
	obs.Text = verifier.Clip(strings.TrimSpace(obs.Text))
	if len(obs.Image) > verifier.MaxImage {
		obs.Image = ""
		obs.Text = "Screenshot exceeds payload limit; use structured inspection."
	}
	return obs, nil
}

// only the concise observations from browser handlers reach recording; never
// model messages, prompts, provider responses, or configuration.
func verifyRecordedResult(result *ToolsCallResult) interface{} {
	var text []string
	if result != nil {
		for _, c := range result.Content {
			if c.Type == "text" {
				text = append(text, verifier.Clip(c.Text))
			}
		}
	}
	data, _ := json.Marshal(text)
	return map[string]interface{}{"observations": json.RawMessage(data)}
}

func (h *Handlers) verifyBrowserObservations(kind string) (*ToolsCallResult, error) {
	if !h.verifying {
		return nil, fmt.Errorf("observation tool is only available during verification")
	}
	if h.recorder == nil || !h.recorder.IsRecording() {
		return &ToolsCallResult{Content: []Content{{Type: "text", Text: "Unavailable: no live recording is active. Earlier console/network events are not available."}}}, nil
	}
	data, err := json.Marshal(h.recorder.BrowserObservations(kind, h.currentContext()))
	if err != nil {
		return nil, err
	}
	return &ToolsCallResult{Content: []Content{{Type: "text", Text: string(data)}}}, nil
}
