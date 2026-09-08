package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vibium/clicker/internal/api"
	"github.com/vibium/clicker/internal/verifier"
)

// OperationCLIOptions controls browser ownership for the CLI only. These options
// are never included in the verifier's inference context or tool permissions.
type OperationCLIOptions struct {
	LaunchOptions map[string]interface{} `json:"launchOptions,omitempty"`
	KeepOpen      bool                   `json:"keepOpen,omitempty"`
}

// CheckCLI runs under the daemon command mutex. Checking ownership, launching,
// checking the claim, finalizing the recording, and closing form one serialized action:
// another command cannot start or borrow a browser between these steps.
func (h *Handlers) CheckCLI(req verifier.Request, options OperationCLIOptions) (verifier.Result, error) {
	if err := req.Validate(); err != nil {
		return verifier.Result{}, err
	}
	if req.Record != "" {
		return verifier.Result{}, fmt.Errorf("browser lifecycle options require live verification")
	}
	return withOperationBrowser(h, options, func() (verifier.Result, error) { return h.Check(req) })
}

// One serialized ownership policy for standalone CLI model operations.
func withOperationBrowser[T any](h *Handlers, options OperationCLIOptions, run func() (T, error)) (zero T, err error) {
	if h.connectURL != "" {
		return zero, fmt.Errorf("operation requires a local Chrome or Firefox session")
	}
	h.sessionMu.Lock()
	hadBrowser := h.client != nil
	h.sessionMu.Unlock()
	if !hadBrowser && !options.KeepOpen {
		defer h.Close()
	}
	if _, err := h.browserLaunch(options.LaunchOptions); err != nil {
		return zero, err
	}
	return run()
}

// Check runs under the daemon mutex or the MCP server's serialized handler.
func (h *Handlers) Check(req verifier.Request) (verifier.Result, error) {
	if err := req.Validate(); err != nil {
		return verifier.Result{}, err
	}
	if req.Record != "" {
		return verifier.CheckRecord(context.Background(), req)
	}
	result, err := h.runLiveOperation("Check", verifier.Method, "claim", req.Claim, req.Output, req.Config, verifier.ToolPolicy{}, func(ctx context.Context, tools verifier.ToolExecutor) (verifier.RecordedResult, error) {
		return (&verifier.Model{}).Check(ctx, req, tools)
	})
	if err != nil {
		return verifier.Result{}, err
	}
	return result.(verifier.Result), nil
}

func (h *Handlers) runLiveOperation(label, method, inputKey, input, output string, config verifier.Config, policy verifier.ToolPolicy, run func(context.Context, verifier.ToolExecutor) (verifier.RecordedResult, error)) (result verifier.RecordedResult, err error) {
	if h.connectURL != "" || (h.launchedEngine != "chrome" && h.launchedEngine != "firefox") || h.client == nil {
		return result, fmt.Errorf("operation requires an existing local Chrome or Firefox session; run vibium go first")
	}
	if dead, _ := h.client.Dead(); dead {
		return result, fmt.Errorf("browser session is no longer usable")
	}
	// Reserve the destination before any model action. The existing recorder
	// exports without clearing a chunk; an operation-owned recording stops here.
	if output != "" {
		finish, startErr := h.startOperationRecording(output, label)
		if startErr != nil {
			return result, startErr
		}
		defer func() { err = errors.Join(err, finish()) }()
	}
	ctx, cancel := context.WithTimeout(context.Background(), verifier.Timeout)
	defer cancel()
	restore := h.client.SetCommandContext(ctx)
	defer restore()
	page, err := h.newSession().GetContextID()
	if err != nil {
		return result, err
	}
	executor := &modelTools{h: h, page: page, policy: policy}
	h.modelRunning = true
	defer func() { h.modelRunning = false }()
	var group string
	if h.recorder != nil && h.recorder.IsRecording() {
		h.recorder.RegisterSecret(config.APIKey)
		group = h.recorder.StartGroup(label + ": " + input)
		h.recorder.SetGroupParams(group, map[string]interface{}{"name": label + ": " + input, "method": method, "modelConfig": config.RecordingMetadata(), inputKey: input})
		defer func() {
			h.recorder.StopGroup()
			if err != nil {
				h.recorder.RecordCallOutcome(group, nil, fmt.Errorf("%s execution failed", strings.ToLower(label)))
				return
			}
			h.recorder.RecordCallOutcome(group, result, nil)
			// Existing Record Player displays group titles; retain the verdict and
			// concise evidence there as well as the structured standard after.result.
			status, summary, evidence := result.RecordingSummary()
			title := label + ": " + input + " — " + status + ": " + summary
			for _, item := range evidence {
				title += " | " + item.Summary
			}
			h.recorder.SetGroupTitle(group, title)
		}()
	}
	result, err = run(ctx, executor)
	return
}

func (h *Handlers) checkMCP(args map[string]interface{}) (*ToolsCallResult, error) {
	for key := range args {
		if key != "claim" && key != "record" && key != "page" && !verifier.IsOverride(key) {
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
	page, hasPage := args["page"]
	if hasPage {
		if p, ok := page.(string); !ok || p == "" {
			return nil, fmt.Errorf("page must be a nonempty context ID")
		}
		if record != "" {
			return nil, fmt.Errorf("record and live page selection cannot be combined")
		}
	}
	config, err := verifier.ConfigFromParams("check", args)
	if err != nil {
		return nil, err
	}
	req := verifier.Request{Claim: claim, Record: record, Config: config}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if record == "" {
		if err := h.ensureBrowser(); err != nil {
			return nil, err
		}
	}
	if hasPage {
		if err := h.checkPageOpen(page.(string)); err != nil {
			return nil, err
		}
		h.pageOverride = page.(string)
		defer func() { h.pageOverride = "" }()
	}
	result, err := h.Check(req)
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(result)
	return &ToolsCallResult{Content: []Content{{Type: "text", Text: string(data)}}}, nil
}

type modelTools struct {
	h      *Handlers
	page   string
	policy verifier.ToolPolicy
}

var modelToolAllowlist = map[string]bool{
	"browser_get_url": true, "browser_map": true, "browser_a11y_tree": true,
	"browser_navigate": true, "browser_reload": true, "browser_click": true,
	"browser_fill": true, "browser_type": true, "browser_press": true,
	"browser_scroll": true, "browser_find": true, "browser_get_text": true,
	"browser_get_value": true, "browser_screenshot": true,
}

func (v *modelTools) Tools() []verifier.Tool {
	var result []verifier.Tool
	for _, t := range GetToolSchemas() {
		if !modelToolAllowlist[t.Name] {
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

func (v *modelTools) Execute(ctx context.Context, name string, args map[string]interface{}) (verifier.Observation, error) {
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
		if secret == true && !v.policy.AllowsCredentialInput(name) {
			if v.policy.CredentialInput {
				return verifier.Observation{Text: "Password fields are unavailable for reading. Return not_completed if completion cannot be established without reading them."}, nil
			}
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
func recordedToolResult(result *ToolsCallResult) interface{} {
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

func (h *Handlers) browserObservations(kind string) (*ToolsCallResult, error) {
	if !h.modelRunning {
		return nil, fmt.Errorf("observation tool is only available during Run or Check")
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

// startOperationRecording preserves a caller-owned recorder, including its group
// stack, resources, video track, and declared output path. Export the full
// current chunk so snapshot references and earlier evidence remain valid.
func (h *Handlers) startOperationRecording(path, label string) (func() error, error) {
	if h.recorder != nil {
		// Resolve parent symlinks even when the destination file does not exist yet.
		canonical := func(p string) string {
			absolute, _ := filepath.Abs(p)
			if dir, err := filepath.EvalSymlinks(filepath.Dir(absolute)); err == nil {
				return filepath.Join(dir, filepath.Base(absolute))
			}
			return absolute
		}
		if canonical(path) == canonical(h.recorder.Options().Path) {
			return nil, fmt.Errorf("recording output must differ from the active recording's destination")
		}
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("create %s recording: %w", label, err)
	}
	owned := h.recorder == nil
	if owned {
		_, err = h.browserRecordStart(map[string]interface{}{"name": label, "path": path, "snapshots": true})
		if err != nil {
			f.Close()
			os.Remove(path)
			return nil, err
		}
	}
	return func() error {
		recorder := h.recorder
		if h.client != nil {
			recorder.NoteDroppedEvents(h.client.DroppedEvents() - h.recordDropBase)
		}
		var data []byte
		var exportErr error
		if owned {
			recorder.StopScreenshots()
			// Match ordinary recording.stop: finalize the native screencast
			// before exporting and before the operation closes its browser.
			api.StopRecordingVideo(h.newSession(), recorder)
			data, exportErr = recorder.Stop()
			h.recorder = nil
		} else {
			data, exportErr = recorder.StopChunk()
		}
		if exportErr == nil {
			_, exportErr = f.Write(data)
		}
		exportErr = errors.Join(exportErr, f.Close())
		if exportErr != nil {
			os.Remove(path)
			return fmt.Errorf("save %s recording: %w", label, exportErr)
		}
		return nil
	}, nil
}
