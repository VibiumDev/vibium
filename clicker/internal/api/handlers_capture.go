package api

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

// printParams translates vibium:page.pdf options into browsingContext.print
// parameters. Only options the caller set are sent, so everything else keeps
// the browser's own default (portrait, scale 1, 1cm margins, no background,
// letter-size page, all pages, shrink to fit).
func printParams(context string, opts map[string]interface{}) map[string]interface{} {
	p := map[string]interface{}{"context": context}

	if v, ok := opts["landscape"].(bool); ok && v {
		p["orientation"] = "landscape"
	}
	if v, ok := opts["scale"].(float64); ok {
		p["scale"] = v
	}
	if v, ok := opts["background"].(bool); ok {
		p["background"] = v
	}
	if v, ok := opts["shrinkToFit"].(bool); ok {
		p["shrinkToFit"] = v
	}

	margin := map[string]interface{}{}
	for wire, bidi := range map[string]string{
		"marginTop": "top", "marginBottom": "bottom",
		"marginLeft": "left", "marginRight": "right",
	} {
		if v, ok := opts[wire].(float64); ok {
			margin[bidi] = v
		}
	}
	if len(margin) > 0 {
		p["margin"] = margin
	}

	page := map[string]interface{}{}
	if v, ok := opts["pageWidth"].(float64); ok {
		page["width"] = v
	}
	if v, ok := opts["pageHeight"].(float64); ok {
		page["height"] = v
	}
	if len(page) > 0 {
		p["page"] = page
	}

	if v, ok := opts["pageRanges"].([]interface{}); ok && len(v) > 0 {
		p["pageRanges"] = v
	}
	return p
}

// handlePagePDF handles vibium:page.pdf — prints the page to PDF.
// Returns base64-encoded PDF data.
func (r *Router) handlePagePDF(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	resp, err := r.sendInternalCommand(session, "browsingContext.print", printParams(context, cmd.Params))
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

// ---------------------------------------------------------------------------
// Exported standalone capture functions — usable from both proxy and MCP.
// ---------------------------------------------------------------------------

// Screenshot captures a page screenshot and returns base64-encoded PNG data.
func Screenshot(s Session, context string, fullPage bool) (string, error) {
	ssParams := map[string]interface{}{
		"context": context,
	}
	if fullPage {
		ssParams["origin"] = "document"
	}

	resp, err := s.SendBidiCommand("browsingContext.captureScreenshot", ssParams)
	if err != nil {
		return "", err
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		return "", bidiErr
	}

	var ssResult struct {
		Result struct {
			Data string `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &ssResult); err != nil {
		return "", fmt.Errorf("screenshot parse failed: %w", err)
	}
	return ssResult.Result.Data, nil
}

// PrintToPDF prints the page to PDF and returns base64-encoded PDF data.
// opts takes the same keys as vibium:page.pdf (landscape, scale, background,
// marginTop/Bottom/Left/Right, pageWidth, pageHeight, pageRanges,
// shrinkToFit); nil means all defaults.
func PrintToPDF(s Session, context string, opts map[string]interface{}) (string, error) {
	resp, err := s.SendBidiCommand("browsingContext.print", printParams(context, opts))
	if err != nil {
		return "", err
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		return "", bidiErr
	}

	var printResult struct {
		Result struct {
			Data string `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &printResult); err != nil {
		return "", fmt.Errorf("pdf parse failed: %w", err)
	}
	return printResult.Result.Data, nil
}
