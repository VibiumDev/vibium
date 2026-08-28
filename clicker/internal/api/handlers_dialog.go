package api

import (
	"encoding/json"
	"fmt"
	"os"
)

// handleDialogSetPolicy handles vibium:dialog.setPolicy — decides who owns a
// user prompt when it opens in the given context. "dismiss" (the default) has
// the router close the prompt itself; "manual" leaves it open for the client's
// dialog handlers. Client pages flip their context to manual when their first
// dialog handler registers and back when the last one is removed. The context
// is required and taken as given (no resolution round trip), because this
// handler runs inline on the client message order — that ordering is what
// guarantees a dialog cannot open under a policy the client already left.
func (r *Router) handleDialogSetPolicy(session *BrowserSession, cmd bidiCommand) {
	context, _ := cmd.Params["context"].(string)
	if context == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("vibium:dialog.setPolicy requires a context"))
		return
	}
	policy, _ := cmd.Params["policy"].(string)
	session.mu.Lock()
	if session.dialogManual == nil {
		session.dialogManual = make(map[string]struct{})
	}
	switch policy {
	case "manual":
		session.dialogManual[context] = struct{}{}
	case "dismiss":
		delete(session.dialogManual, context)
	default:
		session.mu.Unlock()
		r.sendError(session, cmd.ID, fmt.Errorf("invalid dialog policy %q (want \"dismiss\" or \"manual\")", policy))
		return
	}
	session.mu.Unlock()
	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// autoDismissContext returns the browsing context of a userPromptOpened event
// the router must close itself, or "" when the event is something else or the
// client has taken that context over via vibium:dialog.setPolicy.
func (s *BrowserSession) autoDismissContext(msg string) string {
	var evt struct {
		Method string `json:"method"`
		Params struct {
			Context string `json:"context"`
		} `json:"params"`
	}
	if err := json.Unmarshal([]byte(msg), &evt); err != nil {
		return ""
	}
	if evt.Method != "browsingContext.userPromptOpened" {
		return ""
	}
	s.mu.Lock()
	_, manual := s.dialogManual[evt.Params.Context]
	s.mu.Unlock()
	if manual {
		return ""
	}
	return evt.Params.Context
}

// dismissUnhandledPrompt closes a user prompt the client has not claimed. Runs
// off the routing goroutine so a slow browser cannot stall event delivery. The
// usual failure is benign — the prompt was already closed by a racing command —
// but it is still logged, so a dialog that stays stuck names its cause.
func (r *Router) dismissUnhandledPrompt(session *BrowserSession, context string) {
	resp, err := r.sendInternalCommand(session, "browsingContext.handleUserPrompt", map[string]interface{}{
		"context": context,
		"accept":  false,
	})
	if err == nil {
		err = checkBidiError(resp)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "[router] auto-dismiss of prompt in context %s failed for client %d: %v\n",
			context, session.Client.ID(), err)
	}
}

// handleDialogAccept handles vibium:dialog.accept — accepts a user prompt (alert/confirm/prompt).
func (r *Router) handleDialogAccept(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	params := map[string]interface{}{
		"context": context,
		"accept":  true,
	}

	if userText, ok := cmd.Params["userText"].(string); ok {
		params["userText"] = userText
	}

	resp, err := r.sendInternalCommand(session, "browsingContext.handleUserPrompt", params)
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

// handleDialogDismiss handles vibium:dialog.dismiss — dismisses a user prompt.
func (r *Router) handleDialogDismiss(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	params := map[string]interface{}{
		"context": context,
		"accept":  false,
	}

	resp, err := r.sendInternalCommand(session, "browsingContext.handleUserPrompt", params)
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

// ---------------------------------------------------------------------------
// Exported standalone dialog functions — usable from both proxy and MCP.
// ---------------------------------------------------------------------------

// DialogAccept accepts a user prompt (alert/confirm/prompt).
func DialogAccept(s Session, context, userText string) error {
	params := map[string]interface{}{
		"context": context,
		"accept":  true,
	}
	if userText != "" {
		params["userText"] = userText
	}

	resp, err := s.SendBidiCommand("browsingContext.handleUserPrompt", params)
	if err != nil {
		return err
	}
	return checkBidiError(resp)
}

// DialogDismiss dismisses a user prompt.
func DialogDismiss(s Session, context string) error {
	params := map[string]interface{}{
		"context": context,
		"accept":  false,
	}

	resp, err := s.SendBidiCommand("browsingContext.handleUserPrompt", params)
	if err != nil {
		return err
	}
	return checkBidiError(resp)
}
