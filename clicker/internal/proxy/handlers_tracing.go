package proxy

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// handleTracingStart handles vibium:tracing.start — starts trace recording.
// Options: name, screenshots, snapshots, sources, title.
func (r *Router) handleTracingStart(session *BrowserSession, cmd bidiCommand) {
	// Parse options
	var opts TracingStartOptions
	if name, ok := cmd.Params["name"].(string); ok {
		opts.Name = name
	}
	if title, ok := cmd.Params["title"].(string); ok {
		opts.Title = title
	}
	if ss, ok := cmd.Params["screenshots"].(bool); ok {
		opts.Screenshots = ss
	}
	if sn, ok := cmd.Params["snapshots"].(bool); ok {
		opts.Snapshots = sn
	}
	if src, ok := cmd.Params["sources"].(bool); ok {
		opts.Sources = src
	}
	if b, ok := cmd.Params["bidi"].(bool); ok {
		opts.Bidi = b
	}
	// Screenshot format: "jpeg" (default) or "png"
	opts.Format = "jpeg"
	if f, ok := cmd.Params["format"].(string); ok && (f == "png" || f == "jpeg") {
		opts.Format = f
	}
	opts.Quality = 0.8
	if q, ok := cmd.Params["quality"].(float64); ok && q >= 0 && q <= 1 {
		opts.Quality = q
	}

	// Create and start the trace recorder
	recorder := NewTraceRecorder()
	recorder.Start(opts)

	session.mu.Lock()
	session.traceRecorder = recorder
	session.mu.Unlock()

	// Screenshots are captured per-action in dispatch(), not via a background loop.

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// handleTracingStop handles vibium:tracing.stop — stops recording and returns trace data.
// Options: path (file path to save zip).
func (r *Router) handleTracingStop(session *BrowserSession, cmd bidiCommand) {
	session.mu.Lock()
	recorder := session.traceRecorder
	session.mu.Unlock()

	if recorder == nil {
		r.sendError(session, cmd.ID, fmt.Errorf("tracing is not started"))
		return
	}

	// Stop recording and get zip data
	zipData, err := recorder.Stop()
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Clear the recorder from the session
	session.mu.Lock()
	session.traceRecorder = nil
	session.mu.Unlock()

	// Write to file or return base64
	if path, ok := cmd.Params["path"].(string); ok && path != "" {
		if err := WriteTraceToFile(zipData, path); err != nil {
			r.sendError(session, cmd.ID, fmt.Errorf("failed to write trace: %w", err))
			return
		}
		r.sendSuccess(session, cmd.ID, map[string]interface{}{"path": path})
	} else {
		encoded := base64.StdEncoding.EncodeToString(zipData)
		r.sendSuccess(session, cmd.ID, map[string]interface{}{"data": encoded})
	}
}

// handleTracingStartChunk handles vibium:tracing.startChunk — starts a new trace chunk.
// Options: name, title.
func (r *Router) handleTracingStartChunk(session *BrowserSession, cmd bidiCommand) {
	session.mu.Lock()
	recorder := session.traceRecorder
	session.mu.Unlock()

	if recorder == nil {
		r.sendError(session, cmd.ID, fmt.Errorf("tracing is not started"))
		return
	}

	name, _ := cmd.Params["name"].(string)
	title, _ := cmd.Params["title"].(string)

	recorder.StartChunk(name, title)
	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// handleTracingStopChunk handles vibium:tracing.stopChunk — stops the current chunk.
// Options: path (file path to save zip).
func (r *Router) handleTracingStopChunk(session *BrowserSession, cmd bidiCommand) {
	session.mu.Lock()
	recorder := session.traceRecorder
	session.mu.Unlock()

	if recorder == nil {
		r.sendError(session, cmd.ID, fmt.Errorf("tracing is not started"))
		return
	}

	zipData, err := recorder.StopChunk()
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	if path, ok := cmd.Params["path"].(string); ok && path != "" {
		if err := WriteTraceToFile(zipData, path); err != nil {
			r.sendError(session, cmd.ID, fmt.Errorf("failed to write trace chunk: %w", err))
			return
		}
		r.sendSuccess(session, cmd.ID, map[string]interface{}{"path": path})
	} else {
		encoded := base64.StdEncoding.EncodeToString(zipData)
		r.sendSuccess(session, cmd.ID, map[string]interface{}{"data": encoded})
	}
}

// handleTracingStartGroup handles vibium:tracing.startGroup — starts a named group in the trace.
func (r *Router) handleTracingStartGroup(session *BrowserSession, cmd bidiCommand) {
	session.mu.Lock()
	recorder := session.traceRecorder
	session.mu.Unlock()

	if recorder == nil {
		r.sendError(session, cmd.ID, fmt.Errorf("tracing is not started"))
		return
	}

	name, _ := cmd.Params["name"].(string)
	if name == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("name is required for tracing.startGroup"))
		return
	}

	recorder.StartGroup(name)
	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// handleTracingStopGroup handles vibium:tracing.stopGroup — ends the current group.
func (r *Router) handleTracingStopGroup(session *BrowserSession, cmd bidiCommand) {
	session.mu.Lock()
	recorder := session.traceRecorder
	session.mu.Unlock()

	if recorder == nil {
		r.sendError(session, cmd.ID, fmt.Errorf("tracing is not started"))
		return
	}

	recorder.StopGroup()
	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// screenshotParams builds the BiDi captureScreenshot params with optional format/quality.
func screenshotParams(context string, opts TracingStartOptions) map[string]interface{} {
	params := map[string]interface{}{"context": context}
	if opts.Format == "jpeg" {
		f := map[string]interface{}{"type": "image/jpeg"}
		if opts.Quality > 0 {
			f["quality"] = opts.Quality
		}
		params["format"] = f
	}
	return params
}

// captureScreenshotForTrace takes a screenshot via BiDi for the trace recorder.
// Returns (base64 image data, pageID, error).
func (r *Router) captureScreenshotForTrace(session *BrowserSession, opts TracingStartOptions) (string, string, error) {
	// Check session is still alive and get last known context
	session.mu.Lock()
	closed := session.closed
	context := session.lastContext
	session.mu.Unlock()
	if closed {
		return "", "", fmt.Errorf("session closed")
	}

	// Fall back to getContext if no lastContext
	if context == "" {
		var err error
		context, err = r.getContext(session)
		if err != nil {
			return "", "", err
		}
	}

	resp, err := r.sendInternalCommand(session, "browsingContext.captureScreenshot", screenshotParams(context, opts))
	if err != nil {
		return "", "", err
	}

	if bidiErr := checkBidiError(resp); bidiErr != nil {
		return "", "", bidiErr
	}

	var ssResult struct {
		Result struct {
			Data string `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &ssResult); err != nil {
		return "", "", fmt.Errorf("screenshot parse failed: %w", err)
	}

	return ssResult.Result.Data, context, nil
}

// captureActionSnapshot captures a screenshot and wraps it as a frame-snapshot
// for the Playwright trace viewer. Returns the snapshot name (e.g. "before@call@1")
// or "" on failure.
func (r *Router) captureActionSnapshot(session *BrowserSession, recorder *TraceRecorder, params map[string]interface{}, callId, snapshotType string) string {
	session.mu.Lock()
	closed := session.closed
	session.mu.Unlock()
	if closed {
		return ""
	}

	// Resolve browsing context from params or session
	context, _ := params["context"].(string)
	if context == "" {
		session.mu.Lock()
		context = session.lastContext
		session.mu.Unlock()
	}
	if context == "" {
		var err error
		context, err = r.getContext(session)
		if err != nil {
			return ""
		}
	}

	// Capture screenshot via native BiDi command (no JS execution)
	opts := recorder.Options()
	resp, err := r.sendInternalCommandWithTimeout(session, "browsingContext.captureScreenshot", screenshotParams(context, opts), 2*time.Second)
	if err != nil {
		return ""
	}

	if bidiErr := checkBidiError(resp); bidiErr != nil {
		return ""
	}

	var ssResult struct {
		Result struct {
			Data string `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &ssResult); err != nil {
		return ""
	}

	if ssResult.Result.Data == "" {
		return ""
	}

	// Decode image and compute dimensions (handles both PNG and JPEG)
	imgData, err := decodeBase64(ssResult.Result.Data)
	if err != nil {
		return ""
	}
	w, h := imageDimensions(imgData)

	// Store image in resources for Vibium viewer
	hash := sha1Hex(imgData)
	recorder.StoreResource(hash, imgData)

	// Inline data URI for Playwright compat (its service worker only intercepts HTTP(S))
	mimeType := "image/jpeg"
	if opts.Format == "png" {
		mimeType = "image/png"
	}
	imgSrc := "data:" + mimeType + ";base64," + ssResult.Result.Data

	// Build minimal HTML with inline screenshot
	html := []interface{}{
		"HTML", map[string]interface{}{},
		[]interface{}{"HEAD", map[string]interface{}{}},
		[]interface{}{
			"BODY", map[string]interface{}{"style": "margin:0;overflow:hidden"},
			[]interface{}{
				"IMG", map[string]interface{}{
					"src":   imgSrc,
					"style": "width:100%",
				},
			},
		},
	}

	viewport := map[string]interface{}{
		"width":  w,
		"height": h,
	}

	resourceOverrides := []interface{}{
		map[string]interface{}{"url": imgSrc, "sha1": hash},
	}

	session.mu.Lock()
	frameURL := session.lastURL
	session.mu.Unlock()

	return recorder.AddFrameSnapshot(callId, snapshotType, context, frameURL, "html", html, viewport, resourceOverrides)
}

// capturePostActionScreenshot captures a screenshot after an action handler completes.
// It extracts the browsing context from the command's own params so each concurrent
// goroutine resolves its own context independently — no cross-goroutine race.
func (r *Router) capturePostActionScreenshot(session *BrowserSession, recorder *TraceRecorder, params map[string]interface{}) {
	session.mu.Lock()
	closed := session.closed
	session.mu.Unlock()
	if closed {
		return
	}

	// 1. Try context from the command's own params
	context, _ := params["context"].(string)

	// 2. Fall back to session's lastContext (set by resolveContext)
	if context == "" {
		session.mu.Lock()
		context = session.lastContext
		session.mu.Unlock()
	}

	// 3. Fall back to getContext
	if context == "" {
		var err error
		context, err = r.getContext(session)
		if err != nil {
			return
		}
	}

	opts := recorder.Options()
	resp, err := r.sendInternalCommandWithTimeout(session, "browsingContext.captureScreenshot", screenshotParams(context, opts), 5*time.Second)
	if err != nil {
		return
	}

	if bidiErr := checkBidiError(resp); bidiErr != nil {
		return
	}

	var ssResult struct {
		Result struct {
			Data string `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &ssResult); err != nil {
		return
	}

	imgData, err := decodeBase64(ssResult.Result.Data)
	if err != nil {
		return
	}

	w, h := imageDimensions(imgData)
	recorder.AddScreenshot(imgData, context, w, h)
}
