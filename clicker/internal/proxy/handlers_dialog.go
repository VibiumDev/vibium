package proxy

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
