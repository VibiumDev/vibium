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

// VerifyCLIOptions controls browser ownership for the CLI only. These options
// are never included in the verifier's inference context or tool permissions.
type VerifyCLIOptions struct {
	LaunchOptions map[string]interface{} `json:"launchOptions,omitempty"`
	KeepOpen      bool                   `json:"keepOpen,omitempty"`
}

// VerifyCLI runs under the daemon command mutex. Checking ownership, launching,
// verifying, finalizing the recording, and closing form one serialized action:
// another command cannot start or borrow a browser between these steps.
func (h *Handlers) VerifyCLI(req verifier.Request, options VerifyCLIOptions) (verifier.Result, error) {
	if err := req.Validate(); err != nil {
		return verifier.Result{}, err
	}
	if req.Record != "" {
		return verifier.Result{}, fmt.Errorf("browser lifecycle options require live verification")
	}
	if h.connectURL != "" {
		return verifier.Result{}, fmt.Errorf("verify requires a local Chrome or Firefox session")
	}
	h.sessionMu.Lock()
	hadBrowser := h.client != nil
	h.sessionMu.Unlock()
	if !hadBrowser && !options.KeepOpen {
		// Verify's recording finalizer runs before this outer defer, even on
		// errors. Close is the same cleanup used by browser_stop, and is safe
		// if launch failed or the browser crashed during verification.
		defer h.Close()
	}
	if _, err := h.browserLaunch(options.LaunchOptions); err != nil {
		return verifier.Result{}, err
	}
	return h.Verify(req)
}

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
	// Reserve the destination before any verifier action. The existing recorder
	// exports without clearing a chunk; a recording owned by Verify stops here.
	if req.Output != "" {
		finish, startErr := h.startVerifyRecording(req.Output)
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
	executor := &verifyTools{h: h, page: page}
	h.verifying = true
	defer func() { h.verifying = false }()
	var group string
	if h.recorder != nil && h.recorder.IsRecording() {
		h.recorder.RegisterSecret(req.Config.APIKey)
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
		if key != "claim" && key != "record" && key != "page" {
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
	config, err := verifier.ConfigFromEnv()
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
	result, err := h.Verify(req)
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

// startVerifyRecording preserves a caller-owned recorder, including its group
// stack, resources, video track, and declared output path. Export the full
// current chunk so snapshot references and earlier evidence remain valid.
func (h *Handlers) startVerifyRecording(path string) (func() error, error) {
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
			return nil, fmt.Errorf("verification output must differ from the active recording's destination")
		}
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("create verification recording: %w", err)
	}
	owned := h.recorder == nil
	if owned {
		_, err = h.browserRecordStart(map[string]interface{}{"name": "verification", "path": path, "snapshots": true})
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
			// before exporting and before Verify closes its browser.
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
			return fmt.Errorf("save verification recording: %w", exportErr)
		}
		return nil
	}, nil
}
