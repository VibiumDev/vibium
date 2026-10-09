package ai

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/vibium/clicker/internal/toolschema"
)

// This file is the one tool policy for Check and Run models (#571). Both
// surfaces — the SDK clients over the pipe (internal/api) and the CLI/MCP
// daemon (internal/agent) — expose the same tools with the same schemas,
// argument validation, navigation and password guards, and timeout capping.
// Each surface supplies only a ModelDispatcher: the final call into its own
// browser plumbing, and the script probe the password guard needs.

// modelToolAllowlist is the subset of MCP tools a Check/Run model may call.
// Everything else — session lifecycle, recording control, file output —
// stays out of the model's reach on every surface.
var modelToolAllowlist = map[string]bool{
	"browser_get_url": true, "browser_map": true, "browser_a11y_tree": true,
	"browser_navigate": true, "browser_reload": true, "browser_click": true,
	"browser_fill": true, "browser_type": true, "browser_press": true,
	"browser_scroll": true, "browser_find": true, "browser_get_text": true,
	"browser_get_value": true, "browser_screenshot": true,
}

// scrollMaxAmount caps one scroll call; the model repeats the call to go
// further (#619).
const scrollMaxAmount = 10

// defaultActionTimeout bounds one browser action unless the run deadline is
// nearer.
const defaultActionTimeout = 5 * time.Second

// ModelTools derives the model-facing tool schemas from the MCP schemas in
// toolschema, so the two cannot drift: every tool is pinned to the existing
// page, screenshots lose file output and annotation, and scroll declares its
// per-call cap. refs says whether the surface resolves @e refs from
// browser_find; a surface without them tells the model so in each
// selector-taking tool, instead of letting it find out from an error.
func ModelTools(refs bool) []Tool {
	var result []Tool
	for _, t := range toolschema.GetToolSchemas() {
		if !modelToolAllowlist[t.Name] {
			continue
		}
		props := t.InputSchema["properties"].(map[string]interface{})
		// Pin all tools to the existing page; no arbitrary file output or
		// hidden handler parameters may cross this boundary.
		delete(props, "page")
		if t.Name == "browser_screenshot" {
			delete(props, "filename")
			delete(props, "annotate")
			delete(props, "fullPage")
		}
		// Say the scroll cap in the schema the model gets, instead of
		// letting it find out from the error (#619).
		if t.Name == "browser_scroll" {
			amount := props["amount"].(map[string]interface{})
			amount["description"] = toolschema.ScrollAmountDesc + " (default: 3, maximum: 10 per call; repeat the call to scroll further)"
			amount["maximum"] = scrollMaxAmount
		}
		desc := t.Description
		if _, ok := props["selector"]; ok && !refs {
			desc += " Use CSS selectors returned by browser_map; no @refs."
		}
		result = append(result, Tool{Name: t.Name, Description: desc, Parameters: t.InputSchema})
	}
	for _, kind := range []string{"console", "network"} {
		result = append(result, Tool{Name: "browser_" + kind, Description: "Inspect up to 50 recent " + kind + " observations (newest first) from this page's active live recording. Unavailable if recording is off; absence is not proof of no errors. Headers, cookies and bodies are excluded.", Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "additionalProperties": false}})
	}
	return result
}

// ModelDispatcher is the surface-specific remainder of a model tool call:
// the final dispatch into the surface's browser plumbing, and the script
// probe the password guard needs. Everything before it — allowlist, schema
// validation, guards, timeout capping — is shared in ModelToolExecutor.
type ModelDispatcher interface {
	// Dispatch performs a validated tool call. args never contains a key
	// outside the tool's schema.
	Dispatch(ctx context.Context, name string, args map[string]interface{}) (Observation, error)
	// ProbeSecret reports whether the call's target is a password field:
	// the element selector addresses, or the focused element when selector
	// is empty (browser_press without a selector). Implementations run
	// SecretProbeJS through their own script plumbing.
	ProbeSecret(ctx context.Context, selector, name string) (bool, error)
}

// SecretProbeJS is the password-field inspection behind the credential
// guard, shared so the surfaces cannot drift on what counts as a secret
// target. pierceJS is the caller's pierceQuery helper source (shadow-DOM
// aware lookup); the probe walks nested shadow roots for the active
// element the same way. It does not expose eval to the model.
func SecretProbeJS(pierceJS string) string {
	return `(selector) => { ` + pierceJS + `; let el = selector ? pierceQuery(document, selector) : document.activeElement; while (el && el.shadowRoot && el.shadowRoot.activeElement) el = el.shadowRoot.activeElement; return !!el && (el.type === 'password' || el.autocomplete === 'current-password' || el.autocomplete === 'new-password'); }`
}

// ModelToolExecutor enforces the shared tool policy and hands validated
// calls to the surface's dispatcher. It is the ToolExecutor both Check and
// Run use on every surface.
type ModelToolExecutor struct {
	policy     ToolPolicy
	dispatcher ModelDispatcher
	tools      []Tool
}

func NewModelToolExecutor(d ModelDispatcher, policy ToolPolicy, refs bool) *ModelToolExecutor {
	return &ModelToolExecutor{policy: policy, dispatcher: d, tools: ModelTools(refs)}
}

func (e *ModelToolExecutor) Tools() []Tool { return e.tools }

func (e *ModelToolExecutor) Execute(ctx context.Context, name string, args map[string]interface{}) (Observation, error) {
	if err := ctx.Err(); err != nil {
		return Observation{}, err
	}
	var schema map[string]interface{}
	for _, t := range e.tools {
		if t.Name == name {
			schema = t.Parameters
			break
		}
	}
	if schema == nil {
		return Observation{}, fmt.Errorf("disallowed verifier tool")
	}
	props := schema["properties"].(map[string]interface{})
	clean := map[string]interface{}{}
	for key, val := range args {
		prop, ok := props[key].(map[string]interface{})
		if !ok {
			return Observation{}, fmt.Errorf("disallowed verifier argument %q", key)
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
			return Observation{}, fmt.Errorf("invalid verifier argument %q", key)
		}
		if enum, ok := prop["enum"].([]string); ok {
			found := false
			for _, option := range enum {
				if option == val {
					found = true
				}
			}
			if !found {
				return Observation{}, fmt.Errorf("invalid verifier argument %q", key)
			}
		}
		clean[key] = val
	}
	if required, ok := schema["required"].([]string); ok {
		for _, key := range required {
			if val, ok := clean[key]; !ok || val == "" {
				return Observation{}, fmt.Errorf("missing verifier argument %q", key)
			}
		}
	}
	if name == "browser_navigate" {
		u, err := url.Parse(clean["url"].(string))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
			return Observation{}, fmt.Errorf("verifier navigation requires an HTTP(S) URL without credentials")
		}
	}
	if name == "browser_scroll" {
		if n, ok := clean["amount"].(float64); ok && n > scrollMaxAmount {
			// Model-correctable, so a tool result rather than a fatal error:
			// the model retries with a smaller amount instead of the whole
			// check aborting (#619).
			return Observation{}, &ActionError{Err: fmt.Errorf("scroll amount %v exceeds the maximum of %d; scroll again with a smaller amount, repeating the call if needed", n, scrollMaxAmount)}
		}
	}
	if _, ok := props["timeout"]; ok {
		remaining := defaultActionTimeout
		if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < remaining {
			remaining = time.Until(deadline)
		}
		if supplied, ok := clean["timeout"].(float64); !ok || supplied > float64(remaining.Milliseconds()) || supplied == 0 {
			clean["timeout"] = float64(remaining.Milliseconds())
		}
	}
	// Never read or type into a password field.
	selector, _ := clean["selector"].(string)
	if selector != "" || name == "browser_press" {
		secret, err := e.dispatcher.ProbeSecret(ctx, selector, name)
		if err != nil {
			// Usually a model-supplied selector the engine rejects; recoverable.
			return Observation{}, &ActionError{Err: fmt.Errorf("cannot inspect verifier target")}
		}
		if secret && !e.policy.AllowsCredentialInput(name) {
			if e.policy.CredentialInput {
				return Observation{Text: "Password fields are unavailable for reading. Return not_completed if completion cannot be established without reading them."}, nil
			}
			return Observation{Text: "Password fields are unavailable to the verifier. Return inconclusive if required."}, nil
		}
	}
	return e.dispatcher.Dispatch(ctx, name, clean)
}
