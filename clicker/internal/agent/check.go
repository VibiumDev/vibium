package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vibium/clicker/internal/ai"
	"github.com/vibium/clicker/internal/api"
	"github.com/vibium/clicker/internal/check"
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
func (h *Handlers) CheckCLI(req check.Request, options OperationCLIOptions) (ai.Result, error) {
	if err := req.Validate(); err != nil {
		return ai.Result{}, err
	}
	if req.Record != "" {
		return ai.Result{}, fmt.Errorf("browser lifecycle options require live verification")
	}
	return withOperationBrowser(h, options, func() (ai.Result, error) { return h.Check(req) })
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
func (h *Handlers) Check(req check.Request) (ai.Result, error) {
	if err := req.Validate(); err != nil {
		return ai.Result{}, err
	}
	if req.Record != "" {
		return check.CheckRecord(context.Background(), req)
	}
	result, err := h.runLiveOperation("Check", check.Method, "claim", req.Claim, req.Output, req.Config, ai.ToolPolicy{}, func(ctx context.Context, tools ai.ToolExecutor) (ai.RecordedResult, error) {
		return check.Check(ctx, req, tools)
	})
	if err != nil {
		return ai.Result{}, err
	}
	return result.(ai.Result), nil
}

func (h *Handlers) runLiveOperation(label, method, inputKey, input, output string, config ai.Config, policy ai.ToolPolicy, run func(context.Context, ai.ToolExecutor) (ai.RecordedResult, error)) (result ai.RecordedResult, err error) {
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
	ctx, cancel := context.WithTimeout(context.Background(), ai.Timeout)
	defer cancel()
	restore := h.client.SetCommandContext(ctx)
	defer restore()
	page, err := h.newSession().GetContextID()
	if err != nil {
		return result, err
	}
	executor := ai.NewModelToolExecutor(&modelTools{h: h, page: page}, policy, true)
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
		if key != "claim" && key != "record" && key != "page" && key != "baseURL" && !ai.IsOverride(key) {
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
	baseSite := ""
	if v, exists := args["baseURL"]; exists {
		var ok bool
		if baseSite, ok = v.(string); !ok || baseSite == "" {
			return nil, fmt.Errorf("baseURL must be a nonempty site URL")
		}
	}
	config, err := ai.ConfigFromParams("check", args)
	if err != nil {
		return nil, err
	}
	req := check.Request{Claim: claim, Record: record, BaseSite: baseSite, Config: config}
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

// modelTools is the agent-side ModelDispatcher: the shared policy in
// verifier has already validated the call and applied the navigation,
// scroll, timeout, and password guards; only the dispatch into the MCP
// handlers and the script probe live here (#571). @e refs resolve through
// resolveSelector, so this surface passes refs=true.
type modelTools struct {
	h    *Handlers
	page string
}

func (v *modelTools) ProbeSecret(ctx context.Context, selector, name string) (bool, error) {
	script := ai.SecretProbeJS(api.PierceQueryJS())
	secret, err := v.h.client.CallFunction(v.page, script, []interface{}{v.h.resolveSelector(selector)})
	if err != nil {
		return false, err
	}
	return secret == true, nil
}

func (v *modelTools) Dispatch(ctx context.Context, name string, args map[string]interface{}) (ai.Observation, error) {
	args["page"] = v.page
	result, err := v.h.Call(name, args)
	if err != nil {
		return ai.Observation{}, &ai.ActionError{Err: err}
	}
	var obs ai.Observation
	for _, c := range result.Content {
		if c.Type == "text" {
			obs.Text += c.Text + "\n"
		}
		if c.Type == "image" {
			obs.Image = c.Data
			obs.Text += "Screenshot captured."
		}
	}
	obs.Text = ai.Clip(strings.TrimSpace(obs.Text))
	if len(obs.Image) > ai.MaxImage {
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
				text = append(text, ai.Clip(c.Text))
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
