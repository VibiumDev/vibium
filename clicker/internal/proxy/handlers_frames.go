package proxy

import (
	"encoding/json"
	"fmt"
	"strings"
)

// contextInfo represents a browsing context from getTree.
type contextInfo struct {
	Context  string        `json:"context"`
	URL      string        `json:"url"`
	Children []contextInfo `json:"children"`
}

// collectFrames recursively collects all child frames into a flat list.
func collectFrames(contexts []contextInfo) []map[string]interface{} {
	var frames []map[string]interface{}
	for _, ctx := range contexts {
		frames = append(frames, map[string]interface{}{
			"context": ctx.Context,
			"url":     ctx.URL,
		})
		if len(ctx.Children) > 0 {
			frames = append(frames, collectFrames(ctx.Children)...)
		}
	}
	return frames
}

// resolveFrameNames evaluates window.name in each frame context to populate names.
// Chrome's BiDi getTree doesn't return the iframe name attribute, so we resolve it manually.
func (r *Router) resolveFrameNames(session *BrowserSession, frames []map[string]interface{}) {
	for _, f := range frames {
		ctx, _ := f["context"].(string)
		if ctx == "" {
			continue
		}
		name, err := r.evalSimpleScript(session, ctx, "() => window.name")
		if err == nil {
			f["name"] = name
		} else {
			f["name"] = ""
		}
	}
}

// getFrameTree gets the frame tree for a context and returns flattened child frames.
func (r *Router) getFrameTree(session *BrowserSession, context string) ([]map[string]interface{}, error) {
	resp, err := r.sendInternalCommand(session, "browsingContext.getTree", map[string]interface{}{
		"root": context,
	})
	if err != nil {
		return nil, err
	}

	if err := checkBidiError(resp); err != nil {
		return nil, err
	}

	var result struct {
		Result struct {
			Contexts []contextInfo `json:"contexts"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse getTree response: %w", err)
	}

	var frames []map[string]interface{}
	if len(result.Result.Contexts) > 0 {
		frames = collectFrames(result.Result.Contexts[0].Children)
	}
	if frames == nil {
		frames = []map[string]interface{}{}
	}

	return frames, nil
}

// handlePageFrames handles vibium:page.frames — returns all child frames of a page.
func (r *Router) handlePageFrames(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	frames, err := r.getFrameTree(session, context)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.resolveFrameNames(session, frames)

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"frames": frames})
}

// handlePageFrame handles vibium:page.frame — finds a frame by name or URL substring.
func (r *Router) handlePageFrame(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	nameOrUrl, _ := cmd.Params["nameOrUrl"].(string)
	if nameOrUrl == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("nameOrUrl parameter is required"))
		return
	}

	frames, err := r.getFrameTree(session, context)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.resolveFrameNames(session, frames)

	// Match by name first (exact match)
	for _, f := range frames {
		if name, _ := f["name"].(string); name == nameOrUrl {
			r.sendSuccess(session, cmd.ID, f)
			return
		}
	}

	// Then match by URL substring
	for _, f := range frames {
		if url, _ := f["url"].(string); strings.Contains(url, nameOrUrl) {
			r.sendSuccess(session, cmd.ID, f)
			return
		}
	}

	// No match found — return null
	r.sendSuccess(session, cmd.ID, nil)
}
