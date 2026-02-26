package proxy

import (
	"encoding/json"
	"fmt"
)

// handlePageScreenshot handles vibium:page.screenshot — captures a page screenshot.
// Options: fullPage (boolean), clip ({x, y, width, height}).
func (r *Router) handlePageScreenshot(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	ssParams := map[string]interface{}{
		"context": context,
	}

	// Handle fullPage option: set origin to "document"
	if fullPage, ok := cmd.Params["fullPage"].(bool); ok && fullPage {
		ssParams["origin"] = "document"
	}

	// Handle clip option: {x, y, width, height}
	if clip, ok := cmd.Params["clip"].(map[string]interface{}); ok {
		ssParams["clip"] = map[string]interface{}{
			"type":   "box",
			"x":      clip["x"],
			"y":      clip["y"],
			"width":  clip["width"],
			"height": clip["height"],
		}
	}

	resp, err := r.sendInternalCommand(session, "browsingContext.captureScreenshot", ssParams)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	if bidiErr := checkBidiError(resp); bidiErr != nil {
		r.sendError(session, cmd.ID, bidiErr)
		return
	}

	var ssResult struct {
		Result struct {
			Data string `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &ssResult); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("screenshot parse failed: %w", err))
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"data": ssResult.Result.Data})
}

// handlePagePDF handles vibium:page.pdf — prints the page to PDF.
// Returns base64-encoded PDF data.
func (r *Router) handlePagePDF(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	printParams := map[string]interface{}{
		"context": context,
	}

	resp, err := r.sendInternalCommand(session, "browsingContext.print", printParams)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	if bidiErr := checkBidiError(resp); bidiErr != nil {
		r.sendError(session, cmd.ID, bidiErr)
		return
	}

	var printResult struct {
		Result struct {
			Data string `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &printResult); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("pdf parse failed: %w", err))
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"data": printResult.Result.Data})
}
