package agent

import (
	"context"
	"encoding/json"
	"fmt"

	runop "github.com/vibium/clicker/internal/run"
	"github.com/vibium/clicker/internal/verifier"
)

func (h *Handlers) RunCLI(req runop.Request, options OperationCLIOptions) (runop.Result, error) {
	if err := req.Validate(); err != nil {
		return runop.Result{}, err
	}
	return withOperationBrowser(h, options, func() (runop.Result, error) { return h.Run(req) })
}
func (h *Handlers) Run(req runop.Request) (runop.Result, error) {
	if err := req.Validate(); err != nil {
		return runop.Result{}, err
	}
	result, err := h.runLiveOperation("Run", runop.Method, "goal", req.Goal, req.Output, req.Config, verifier.ToolPolicy{CredentialInput: true}, func(ctx context.Context, tools verifier.ToolExecutor) (verifier.RecordedResult, error) {
		return runop.Run(ctx, req, tools)
	})
	if err != nil {
		return runop.Result{}, err
	}
	return result.(runop.Result), nil
}
func (h *Handlers) runMCP(args map[string]interface{}) (*ToolsCallResult, error) {
	for key := range args {
		if key != "goal" && key != "page" && !verifier.IsOverride(key) {
			return nil, fmt.Errorf("unsupported run argument")
		}
	}
	goal, ok := args["goal"].(string)
	if !ok {
		return nil, fmt.Errorf("goal is required")
	}
	config, err := verifier.ConfigFromParams("run", args)
	if err != nil {
		return nil, err
	}
	req := runop.Request{Goal: goal, Config: config}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if p, exists := args["page"]; exists {
		if text, ok := p.(string); !ok || text == "" {
			return nil, fmt.Errorf("page must be a nonempty context ID")
		}
	}
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}
	if p, exists := args["page"]; exists {
		if err := h.checkPageOpen(p.(string)); err != nil {
			return nil, err
		}
		h.pageOverride = p.(string)
		defer func() { h.pageOverride = "" }()
	}
	result, err := h.Run(req)
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(result)
	return &ToolsCallResult{Content: []Content{{Type: "text", Text: string(data)}}}, nil
}
