package api

import (
	"encoding/json"
	"fmt"
	"sync"
)

// PromptTracker records which browsing contexts have an open user prompt.
//
// The browser is launched with unhandledPromptBehavior "ignore"
// (browser/launcher.go), so an alert/confirm/prompt opened by a click stays
// open, and Chrome answers no script or input command for that context until
// browsingContext.handleUserPrompt arrives. Without this state a doomed command
// just sits in its 60s timeout.
type PromptTracker struct {
	mu   sync.Mutex
	open map[string]string // context -> prompt type ("alert", "confirm", ...)
}

// NewPromptTracker creates an empty tracker.
func NewPromptTracker() *PromptTracker {
	return &PromptTracker{open: make(map[string]string)}
}

// Observe updates the tracker from a raw BiDi event. Anything that is not a
// user-prompt event is ignored, so callers can hand it every event they see.
func (t *PromptTracker) Observe(raw []byte) {
	if t == nil {
		return
	}
	var evt struct {
		Method string `json:"method"`
		Params struct {
			Context string `json:"context"`
			Type    string `json:"type"`
		} `json:"params"`
	}
	if err := json.Unmarshal(raw, &evt); err != nil {
		return
	}
	if evt.Params.Context == "" {
		return
	}

	switch evt.Method {
	case "browsingContext.userPromptOpened":
		promptType := evt.Params.Type
		if promptType == "" {
			promptType = "dialog"
		}
		t.mu.Lock()
		t.open[evt.Params.Context] = promptType
		t.mu.Unlock()
	case "browsingContext.userPromptClosed":
		t.mu.Lock()
		delete(t.open, evt.Params.Context)
		t.mu.Unlock()
	}
}

// OpenPrompt returns the open prompt's type for a context, and whether one is open.
func (t *PromptTracker) OpenPrompt(context string) (string, bool) {
	if t == nil || context == "" {
		return "", false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	promptType, ok := t.open[context]
	return promptType, ok
}

// Clear forgets any prompt recorded for a context. Used when a context goes
// away, so a destroyed context cannot leave a permanent block behind.
func (t *PromptTracker) Clear(context string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	delete(t.open, context)
	t.mu.Unlock()
}

// BlockedByPromptError is returned instead of waiting out the command timeout
// when the target context has an open user prompt.
type BlockedByPromptError struct {
	Context    string
	PromptType string
}

func (e *BlockedByPromptError) Error() string {
	return fmt.Sprintf("context is blocked by an open %s dialog; accept or dismiss it first "+
		"(dialog.accept / dialog.dismiss, or handle it with an onDialog listener)", e.PromptType)
}

// promptSensitiveMethods are the BiDi commands Chrome will not answer while a
// user prompt is open. It is an allowlist rather than a denylist so an
// unrecognised method keeps its current behavior; in particular
// browsingContext.handleUserPrompt must always get through, since it is what
// clears the block.
var promptSensitiveMethods = map[string]bool{
	"script.callFunction":               true,
	"script.evaluate":                   true,
	"input.performActions":              true,
	"input.releaseActions":              true,
	"input.setFiles":                    true,
	"browsingContext.navigate":          true,
	"browsingContext.reload":            true,
	"browsingContext.traverseHistory":   true,
	"browsingContext.captureScreenshot": true,
	"browsingContext.print":             true,
}

// contextFromParams pulls the browsing context a command targets, covering both
// the flat "context" param and script.*'s nested target object.
func contextFromParams(params map[string]interface{}) string {
	if ctx, ok := params["context"].(string); ok && ctx != "" {
		return ctx
	}
	if target, ok := params["target"].(map[string]interface{}); ok {
		if ctx, ok := target["context"].(string); ok {
			return ctx
		}
	}
	return ""
}

// checkPromptBlocked returns a BlockedByPromptError when the command would
// target a context whose user prompt is open.
func checkPromptBlocked(t *PromptTracker, method string, params map[string]interface{}) error {
	if t == nil || !promptSensitiveMethods[method] {
		return nil
	}
	context := contextFromParams(params)
	if context == "" {
		return nil
	}
	if promptType, open := t.OpenPrompt(context); open {
		return &BlockedByPromptError{Context: context, PromptType: promptType}
	}
	return nil
}
