package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// handleRecordingStart handles vibium:recording.start — starts recording.
// Options: name, screenshots, snapshots, sources, title.
func (r *Router) handleRecordingStart(session *BrowserSession, cmd bidiCommand) {
	opts := ParseRecordingOptions(cmd.Params)

	// Best-effort viewport query
	viewport := r.queryViewport(session)

	// Create and start the recorder
	recorder := NewRecorder()
	recorder.Start(opts, viewport)
	recorder.StartActionScreenshotWorker(func(context string, actionEnd time.Time) {
		CaptureRecordingScreenshot(NewAPISession(r, session, context), recorder, actionEnd)
	})

	session.mu.Lock()
	session.recorder = recorder
	session.mu.Unlock()

	// Screenshots are queued per-action in dispatch(), then drained on stop.

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// handleRecordingStop handles vibium:recording.stop — stops recording and returns recording data.
// Options: path (file path to save zip).
func (r *Router) handleRecordingStop(session *BrowserSession, cmd bidiCommand) {
	// Wait for any in-flight dispatch() to finish so its after-event is recorded.
	session.dispatchMu.Lock()
	defer session.dispatchMu.Unlock()

	session.mu.Lock()
	recorder := session.recorder
	session.mu.Unlock()

	if recorder == nil {
		r.sendError(session, cmd.ID, fmt.Errorf("recording is not started"))
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
	session.recorder = nil
	session.mu.Unlock()

	// Write to file or return base64
	if path, ok := cmd.Params["path"].(string); ok && path != "" {
		if err := WriteRecordToFile(zipData, path); err != nil {
			r.sendError(session, cmd.ID, fmt.Errorf("failed to write recording: %w", err))
			return
		}
		r.sendSuccess(session, cmd.ID, map[string]interface{}{"path": path})
	} else {
		encoded := base64.StdEncoding.EncodeToString(zipData)
		r.sendSuccess(session, cmd.ID, map[string]interface{}{"data": encoded})
	}
}

// handleRecordingStartChunk handles vibium:recording.startChunk — starts a new recording chunk.
// Options: name, title.
func (r *Router) handleRecordingStartChunk(session *BrowserSession, cmd bidiCommand) {
	// Wait for any in-flight dispatch() to finish so events are properly ordered.
	session.dispatchMu.Lock()
	defer session.dispatchMu.Unlock()

	session.mu.Lock()
	recorder := session.recorder
	session.mu.Unlock()

	if recorder == nil {
		r.sendError(session, cmd.ID, fmt.Errorf("recording is not started"))
		return
	}

	name, _ := cmd.Params["name"].(string)
	title, _ := cmd.Params["title"].(string)

	// Best-effort viewport query
	viewport := r.queryViewport(session)

	recorder.StartChunk(name, title, viewport)
	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// handleRecordingStopChunk handles vibium:recording.stopChunk — stops the current chunk.
// Options: path (file path to save zip).
func (r *Router) handleRecordingStopChunk(session *BrowserSession, cmd bidiCommand) {
	// Wait for any in-flight dispatch() to finish so its after-event is recorded.
	session.dispatchMu.Lock()
	defer session.dispatchMu.Unlock()

	session.mu.Lock()
	recorder := session.recorder
	session.mu.Unlock()

	if recorder == nil {
		r.sendError(session, cmd.ID, fmt.Errorf("recording is not started"))
		return
	}

	zipData, err := recorder.StopChunk()
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	if path, ok := cmd.Params["path"].(string); ok && path != "" {
		if err := WriteRecordToFile(zipData, path); err != nil {
			r.sendError(session, cmd.ID, fmt.Errorf("failed to write recording chunk: %w", err))
			return
		}
		r.sendSuccess(session, cmd.ID, map[string]interface{}{"path": path})
	} else {
		encoded := base64.StdEncoding.EncodeToString(zipData)
		r.sendSuccess(session, cmd.ID, map[string]interface{}{"data": encoded})
	}
}

// handleRecordingStartGroup handles vibium:recording.startGroup — starts a named group in the recording.
func (r *Router) handleRecordingStartGroup(session *BrowserSession, cmd bidiCommand) {
	// Wait for any in-flight dispatch() to finish so events are properly ordered.
	session.dispatchMu.Lock()
	defer session.dispatchMu.Unlock()

	session.mu.Lock()
	recorder := session.recorder
	session.mu.Unlock()

	if recorder == nil {
		r.sendError(session, cmd.ID, fmt.Errorf("recording is not started"))
		return
	}

	name, _ := cmd.Params["name"].(string)
	if name == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("name is required for recording.startGroup"))
		return
	}

	recorder.StartGroup(name)
	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// handleRecordingStopGroup handles vibium:recording.stopGroup — ends the current group.
func (r *Router) handleRecordingStopGroup(session *BrowserSession, cmd bidiCommand) {
	// Wait for any in-flight dispatch() to finish so its after-event is recorded.
	session.dispatchMu.Lock()
	defer session.dispatchMu.Unlock()

	session.mu.Lock()
	recorder := session.recorder
	session.mu.Unlock()

	if recorder == nil {
		r.sendError(session, cmd.ID, fmt.Errorf("recording is not started"))
		return
	}

	recorder.StopGroup()
	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// ScreenshotParams builds the BiDi captureScreenshot params with optional format/quality.
func ScreenshotParams(context string, opts RecordingStartOptions) map[string]interface{} {
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

// captureScreenshotForRecording takes a screenshot via BiDi for the recorder.
// Returns (base64 image data, pageID, error).
func (r *Router) captureScreenshotForRecording(session *BrowserSession, opts RecordingStartOptions) (string, string, error) {
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

	resp, err := r.sendInternalCommand(session, "browsingContext.captureScreenshot", ScreenshotParams(context, opts))
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

// captureBeforeSnapshotAfterScroll captures a before-snapshot for click-like
// actions after the element has been scrolled into view. Called from recording
// handlers between resolveWithActionability and the actual input action.
func (r *Router) captureBeforeSnapshotAfterScroll(session *BrowserSession, params map[string]interface{}) {
	callId, _ := params["_recordCallId"].(string)
	if callId == "" {
		return
	}
	session.mu.Lock()
	recorder := session.recorder
	session.mu.Unlock()
	if recorder == nil || !recorder.IsRecording() {
		return
	}
	if !recorder.Options().Snapshots {
		return
	}
	name := r.captureActionSnapshot(session, recorder, params, callId, "before")
	if name != "" {
		recorder.PatchBeforeSnapshot(callId, name)
	}
}

// captureActionSnapshot captures a screenshot and wraps it as a frame-snapshot
// for the Record Player / Playwright trace viewer. Returns the snapshot name
// (e.g. "before@call@1") or "" on failure.
func (r *Router) captureActionSnapshot(session *BrowserSession, recorder *Recorder, params map[string]interface{}, callId, snapshotType string) string {
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
	resp, err := r.sendInternalCommandWithTimeout(session, "browsingContext.captureScreenshot", ScreenshotParams(context, opts), 2*time.Second)
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
	w, h := ImageDimensions(imgData)

	// Store image in resources for Record Player
	name := recorder.ScreenshotName(context, time.Now())
	recorder.StoreResource(name, imgData)

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
		map[string]interface{}{"url": imgSrc, "sha1": name},
	}

	session.mu.Lock()
	frameURL := session.lastURL
	session.mu.Unlock()

	return recorder.AddFrameSnapshot(callId, snapshotType, context, frameURL, "html", html, viewport, resourceOverrides)
}

// CaptureRecordingScreenshot captures a screenshot via the Session interface
// and adds it to the recorder. This is the shared version used by both the
// proxy dispatch() and MCP Call() paths. The Session's GetContextID() handles
// context resolution (explicit context → lastContext → getTree).
func CaptureRecordingScreenshot(s Session, recorder *Recorder, actionEnd time.Time) {
	if !recorder.Options().Screenshots {
		return
	}

	context, err := s.GetContextID()
	if err != nil {
		return
	}

	opts := recorder.Options()
	resp, err := s.SendBidiCommandWithTimeout("browsingContext.captureScreenshot", ScreenshotParams(context, opts), 5*time.Second)
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

	w, h := ImageDimensions(imgData)
	recorder.AddScreenshot(imgData, context, w, h, actionEnd)
}

// queryViewport queries the browser for the current viewport size.
// Returns nil if the query fails (best-effort).
func (r *Router) queryViewport(session *BrowserSession) map[string]interface{} {
	context, err := r.getContext(session)
	if err != nil {
		return nil
	}
	result, err := r.evalSimpleScript(session, context, "() => window.innerWidth + ',' + window.innerHeight")
	if err != nil {
		return nil
	}
	parts := strings.SplitN(result, ",", 2)
	if len(parts) != 2 {
		return nil
	}
	w, err1 := strconv.Atoi(parts[0])
	h, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return nil
	}
	return map[string]interface{}{"width": w, "height": h}
}
