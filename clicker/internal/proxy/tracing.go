package proxy

import (
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// TracingStartOptions configures how tracing behaves.
type TracingStartOptions struct {
	Name        string  `json:"name"`
	Screenshots bool    `json:"screenshots"`
	Snapshots   bool    `json:"snapshots"`
	Sources     bool    `json:"sources"`
	Title       string  `json:"title"`
	Bidi        bool    `json:"bidi"`
	Format      string  `json:"format"`  // "png" or "jpeg" (default "jpeg")
	Quality     float64 `json:"quality"` // 0.0-1.0 for JPEG (default 0.8)
}

// traceEvent is a generic trace event stored as a JSON-friendly map.
type traceEvent = map[string]interface{}

// groupEntry tracks a group's name and callId so StopGroup can emit a matching "after" event.
type groupEntry struct {
	name   string
	callId string
}

// pendingRequest holds a parsed beforeRequestSent event until its response arrives.
type pendingRequest struct {
	context   string
	requestID string
	url       string
	method    string
	headers   []interface{} // raw BiDi header list
	cookies   []interface{}
	headersSize float64
	bodySize    float64
	timestamp   float64 // BiDi timestamp (ms since epoch)
}

// TraceRecorder manages trace recording state for a browser session.
// It collects events, screenshots, and DOM snapshots, then packages
// them into a Playwright-compatible trace zip.
type TraceRecorder struct {
	mu              sync.Mutex
	recording       bool
	options         TracingStartOptions
	events          []traceEvent      // current chunk's trace events
	network         []traceEvent      // current chunk's network events
	resources       map[string][]byte // sha1 hex -> binary data (PNG/HTML)
	groupStack      []groupEntry       // nested group entries (name + callId)
	pendingRequests map[string]*pendingRequest // BiDi request ID -> pending request
	chunkIndex      int
	startTime       int64 // unix ms
	actionCounter   int   // monotonic counter for action/bidi callIds

	// Screenshot goroutine control
	screenshotStop chan struct{}
	screenshotWg   sync.WaitGroup
}

// NewTraceRecorder creates a new trace recorder.
func NewTraceRecorder() *TraceRecorder {
	return &TraceRecorder{
		resources:       make(map[string][]byte),
		pendingRequests: make(map[string]*pendingRequest),
	}
}

// IsRecording returns whether tracing is currently active.
func (t *TraceRecorder) IsRecording() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.recording
}

// Start begins trace recording with the given options.
func (t *TraceRecorder) Start(opts TracingStartOptions) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.recording = true
	t.options = opts
	t.events = nil
	t.network = nil
	t.resources = make(map[string][]byte)
	t.pendingRequests = make(map[string]*pendingRequest)
	t.groupStack = nil
	t.chunkIndex = 0
	t.startTime = time.Now().UnixMilli()

	title := opts.Title
	if title == "" {
		title = opts.Name
	}

	// First event must be context-options (required by Playwright trace viewer)
	t.events = append(t.events, traceEvent{
		"type":          "context-options",
		"browserName":   "chromium",
		"platform":      runtime.GOOS,
		"wallTime":      float64(t.startTime),
		"monotonicTime": float64(t.startTime),
		"title":         title,
		"options":       map[string]interface{}{},
		"sdkLanguage":   "javascript",
		"version":       8,
		"origin":        "library",
	})
}

// Stop stops recording and returns the trace zip data.
func (t *TraceRecorder) Stop() ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.recording {
		return nil, fmt.Errorf("tracing is not started")
	}

	t.recording = false
	return t.buildZipLocked()
}

// StartChunk starts a new chunk within the current trace.
func (t *TraceRecorder) StartChunk(name, title string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.events = nil
	t.network = nil
	t.chunkIndex++

	chunkTitle := title
	if chunkTitle == "" {
		chunkTitle = name
	}

	now := float64(time.Now().UnixMilli())
	t.events = append(t.events, traceEvent{
		"type":          "context-options",
		"browserName":   "chromium",
		"platform":      runtime.GOOS,
		"wallTime":      now,
		"monotonicTime": now,
		"title":         chunkTitle,
		"options":       map[string]interface{}{},
		"sdkLanguage":   "javascript",
		"version":       8,
		"origin":        "library",
	})
}

// StopChunk packages the current chunk into a zip and returns it.
// Tracing remains active for additional chunks.
func (t *TraceRecorder) StopChunk() ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.recording {
		return nil, fmt.Errorf("tracing is not started")
	}

	return t.buildZipLocked()
}

// currentGroupIdLocked returns the callId of the innermost active group, or "".
// Must be called with t.mu held.
func (t *TraceRecorder) currentGroupIdLocked() string {
	if len(t.groupStack) == 0 {
		return ""
	}
	return t.groupStack[len(t.groupStack)-1].callId
}

// StartGroup adds a group-start marker to the trace.
func (t *TraceRecorder) StartGroup(name string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	parentId := t.currentGroupIdLocked()

	t.actionCounter++
	callId := fmt.Sprintf("call@%d", t.actionCounter)
	t.groupStack = append(t.groupStack, groupEntry{name: name, callId: callId})
	now := float64(time.Now().UnixMilli())
	ev := traceEvent{
		"type":      "before",
		"callId":    callId,
		"title":     name,
		"class":     "Tracing",
		"method":    "group",
		"params":    map[string]interface{}{"name": name},
		"wallTime":  now,
		"startTime": now,
	}
	if parentId != "" {
		ev["parentId"] = parentId
	}
	t.events = append(t.events, ev)
}

// StopGroup adds a group-end marker to the trace.
func (t *TraceRecorder) StopGroup() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.groupStack) == 0 {
		return
	}

	entry := t.groupStack[len(t.groupStack)-1]
	t.groupStack = t.groupStack[:len(t.groupStack)-1]

	t.events = append(t.events, traceEvent{
		"type":    "after",
		"callId":  entry.callId,
		"endTime": float64(time.Now().UnixMilli()),
	})
}

// Options returns the current tracing options.
func (t *TraceRecorder) Options() TracingStartOptions {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.options
}

// StoreResource stores binary data (e.g. screenshot JPEG/PNG) in the resources
// map, keyed by its SHA1 hex hash. The data will be written to resources/<sha1>
// in the trace zip.
func (t *TraceRecorder) StoreResource(sha1 string, data []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.resources[sha1] = data
}

// apiNameFromMethod maps a vibium: method to (class, title) for trace display.
func apiNameFromMethod(method string) (string, string) {
	// Strip the "vibium:" prefix
	if len(method) <= 7 || method[:7] != "vibium:" {
		return "Vibium", method
	}
	name := method[7:] // e.g. "click", "page.navigate", "el.text"

	switch {
	// Element interaction: click, dblclick, fill, type, press, clear, check, uncheck, selectOption, hover, focus, dragTo, tap, scrollIntoView, dispatchEvent
	case name == "click" || name == "dblclick" || name == "fill" || name == "type" ||
		name == "press" || name == "clear" || name == "check" || name == "uncheck" ||
		name == "selectOption" || name == "hover" || name == "focus" || name == "dragTo" ||
		name == "tap" || name == "scrollIntoView" || name == "dispatchEvent":
		return "Element", "Element." + name

	// Element finding: find, findAll
	case name == "find" || name == "findAll":
		return "Page", "Page." + name

	// Element state: el.*
	case len(name) > 3 && name[:3] == "el.":
		return "Element", "Element." + name[3:]

	// Page commands: page.*
	case len(name) > 5 && name[:5] == "page.":
		return "Page", "Page." + name[5:]

	// Browser commands: browser.*
	case len(name) > 8 && name[:8] == "browser.":
		return "Browser", "Browser." + name[8:]

	// Context commands: context.*
	case len(name) > 8 && name[:8] == "context.":
		return "BrowserContext", "BrowserContext." + name[8:]

	// Keyboard: keyboard.*
	case len(name) > 9 && name[:9] == "keyboard.":
		return "Page", "Page." + name

	// Mouse: mouse.*
	case len(name) > 6 && name[:6] == "mouse.":
		return "Page", "Page." + name

	// Touch: touch.*
	case len(name) > 6 && name[:6] == "touch.":
		return "Page", "Page." + name

	// Network: network.*
	case len(name) > 8 && name[:8] == "network.":
		return "Network", "Network." + name[8:]

	// Dialog: dialog.*
	case len(name) > 7 && name[:7] == "dialog.":
		return "Dialog", "Dialog." + name[7:]

	// Clock: clock.*
	case len(name) > 6 && name[:6] == "clock.":
		return "Clock", "Clock." + name[6:]

	// Download: download.*
	case len(name) > 9 && name[:9] == "download.":
		return "Download", "Download." + name[9:]

	default:
		return "Vibium", name
	}
}

// NextCallId generates and returns the next call@N id without emitting any event.
// Use this when you need the callId before recording the action (e.g. for snapshots).
func (t *TraceRecorder) NextCallId() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.recording {
		return ""
	}

	t.actionCounter++
	return fmt.Sprintf("call@%d", t.actionCounter)
}

// RecordAction records a vibium command as an action marker in the trace.
// The callId should come from NextCallId(). beforeSnapshot is the snapshot name
// (from AddFrameSnapshot) to link, or "" if none. pageId is a fallback browsing
// context to use when params["context"] is not set.
func (t *TraceRecorder) RecordAction(callId, method string, params map[string]interface{}, beforeSnapshot, pageId string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.recording || callId == "" {
		return
	}

	class, title := apiNameFromMethod(method)
	now := float64(time.Now().UnixMilli())
	ev := traceEvent{
		"type":      "before",
		"callId":    callId,
		"title":     title,
		"class":     class,
		"method":    method,
		"params":    params,
		"wallTime":  now,
		"startTime": now,
	}
	// Add pageId so the viewer can match actions to page screenshots
	if ctx, ok := params["context"].(string); ok && ctx != "" {
		ev["pageId"] = ctx
	} else if pageId != "" {
		ev["pageId"] = pageId
	}
	if beforeSnapshot != "" {
		ev["beforeSnapshot"] = beforeSnapshot
	}
	// Link to parent group for nesting in trace viewer
	if gid := t.currentGroupIdLocked(); gid != "" {
		ev["parentId"] = gid
	}
	t.events = append(t.events, ev)
}

// RecordActionEnd records the end of a vibium command action in the trace.
// The callId must match the value returned by NextCallId(). afterSnapshot is the
// snapshot name (from AddFrameSnapshot) to link, or "" if none. endTime is the
// actual handler completion time (before screenshot captures).
func (t *TraceRecorder) RecordActionEnd(callId, afterSnapshot string, endTime time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.recording || callId == "" {
		return
	}

	ev := traceEvent{
		"type":    "after",
		"callId":  callId,
		"endTime": float64(endTime.UnixMilli()),
	}
	if afterSnapshot != "" {
		ev["afterSnapshot"] = afterSnapshot
	}
	t.events = append(t.events, ev)
}

// RecordBidiCommand records a raw BiDi command sent to the browser (opt-in via bidi: true).
// Returns the callId so the caller can pass it to RecordBidiCommandEnd.
func (t *TraceRecorder) RecordBidiCommand(method string, params map[string]interface{}) string {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.recording {
		return ""
	}

	t.actionCounter++
	callId := fmt.Sprintf("call@%d", t.actionCounter)
	now := float64(time.Now().UnixMilli())
	ev := traceEvent{
		"type":      "before",
		"callId":    callId,
		"title":     method,
		"class":     "BiDi",
		"method":    method,
		"params":    params,
		"wallTime":  now,
		"startTime": now,
	}
	// Link to parent group for nesting in trace viewer
	if gid := t.currentGroupIdLocked(); gid != "" {
		ev["parentId"] = gid
	}
	t.events = append(t.events, ev)
	return callId
}

// RecordBidiCommandEnd records the end of a BiDi command in the trace.
// The callId must match the value returned by the corresponding RecordBidiCommand call.
func (t *TraceRecorder) RecordBidiCommandEnd(callId string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.recording || callId == "" {
		return
	}

	t.events = append(t.events, traceEvent{
		"type":    "after",
		"callId":  callId,
		"endTime": float64(time.Now().UnixMilli()),
	})
}

// AddScreenshot stores a screenshot image (PNG or JPEG) and adds a screencast-frame event.
func (t *TraceRecorder) AddScreenshot(pngData []byte, pageID string, width, height int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.recording {
		return
	}

	hash := sha1Hex(pngData)
	t.resources[hash] = pngData
	t.events = append(t.events, traceEvent{
		"type":      "screencast-frame",
		"pageId":    pageID,
		"sha1":      hash,
		"width":     width,
		"height":    height,
		"timestamp": float64(time.Now().UnixMilli()),
	})
}

// AddFrameSnapshot adds a frame-snapshot event for the Playwright trace viewer.
// snapshotType is "before" or "after"; callId is like "call@1".
// resourceOverrides maps synthetic URLs (e.g. "screenshot://sha1") to resource
// SHA1 hashes so the viewer can resolve them from the zip's resources/ directory.
// Returns the snapshot name (e.g. "before@call@1").
func (t *TraceRecorder) AddFrameSnapshot(callId, snapshotType, pageId, frameURL, doctype string, html interface{}, viewport map[string]interface{}, resourceOverrides []interface{}) string {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.recording {
		return ""
	}

	if resourceOverrides == nil {
		resourceOverrides = []interface{}{}
	}

	snapshotName := snapshotType + "@" + callId
	now := float64(time.Now().UnixMilli())

	t.events = append(t.events, traceEvent{
		"type": "frame-snapshot",
		"snapshot": map[string]interface{}{
			"callId":            callId,
			"snapshotName":      snapshotName,
			"pageId":            pageId,
			"frameId":           pageId,
			"frameUrl":          frameURL,
			"doctype":           doctype,
			"html":              html,
			"viewport":          viewport,
			"timestamp":         now,
			"wallTime":          now,
			"resourceOverrides": resourceOverrides,
			"isMainFrame":       true,
		},
	})

	return snapshotName
}

// RecordBidiEvent records a raw BiDi event from the browser into the trace.
// Network events are correlated by request ID and transformed into
// Playwright-compatible HAR resource-snapshot entries.
func (t *TraceRecorder) RecordBidiEvent(msg string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.recording {
		return
	}

	var bidiEvent struct {
		Method string                 `json:"method"`
		Params map[string]interface{} `json:"params"`
	}
	if err := json.Unmarshal([]byte(msg), &bidiEvent); err != nil {
		return
	}

	// Only record events (not responses)
	if bidiEvent.Method == "" {
		return
	}

	now := float64(time.Now().UnixMilli())

	switch bidiEvent.Method {
	case "network.beforeRequestSent":
		req := parsePendingRequest(bidiEvent.Params)
		if req != nil {
			t.pendingRequests[req.requestID] = req
		}

	case "network.responseCompleted":
		requestID := extractRequestID(bidiEvent.Params)
		pending := t.pendingRequests[requestID]
		if pending == nil {
			// No matching request — write a best-effort entry from response only
			pending = parsePendingRequestFromResponse(bidiEvent.Params)
		} else {
			delete(t.pendingRequests, requestID)
		}
		if pending != nil {
			entry := bidiToHAREntry(pending, bidiEvent.Params, false)
			t.network = append(t.network, entry)
		}

	case "network.fetchError":
		requestID := extractRequestID(bidiEvent.Params)
		pending := t.pendingRequests[requestID]
		if pending == nil {
			pending = parsePendingRequestFromResponse(bidiEvent.Params)
		} else {
			delete(t.pendingRequests, requestID)
		}
		if pending != nil {
			entry := bidiToHAREntry(pending, bidiEvent.Params, true)
			t.network = append(t.network, entry)
		}

	default:
		t.events = append(t.events, traceEvent{
			"type":   "event",
			"method": bidiEvent.Method,
			"params": bidiEvent.Params,
			"time":   now,
			"class":  "BrowserContext",
		})
	}
}

// extractRequestID pulls params.request.request from a BiDi network event.
func extractRequestID(params map[string]interface{}) string {
	req, _ := params["request"].(map[string]interface{})
	if req == nil {
		return ""
	}
	id, _ := req["request"].(string)
	return id
}

// parsePendingRequest extracts request details from a beforeRequestSent event.
func parsePendingRequest(params map[string]interface{}) *pendingRequest {
	req, _ := params["request"].(map[string]interface{})
	if req == nil {
		return nil
	}
	id, _ := req["request"].(string)
	if id == "" {
		return nil
	}
	p := &pendingRequest{
		requestID: id,
	}
	p.url, _ = req["url"].(string)
	p.method, _ = req["method"].(string)
	p.headers, _ = req["headers"].([]interface{})
	p.cookies, _ = req["cookies"].([]interface{})
	p.headersSize = toFloat64(req["headersSize"])
	p.bodySize = toFloat64(req["bodySize"])
	p.context, _ = params["context"].(string)
	p.timestamp = toFloat64(params["timestamp"])
	return p
}

// parsePendingRequestFromResponse creates a minimal pendingRequest from a
// responseCompleted/fetchError event when no matching beforeRequestSent exists.
func parsePendingRequestFromResponse(params map[string]interface{}) *pendingRequest {
	req, _ := params["request"].(map[string]interface{})
	if req == nil {
		return nil
	}
	p := &pendingRequest{}
	p.requestID, _ = req["request"].(string)
	p.url, _ = req["url"].(string)
	p.method, _ = req["method"].(string)
	p.headers, _ = req["headers"].([]interface{})
	p.cookies, _ = req["cookies"].([]interface{})
	p.headersSize = toFloat64(req["headersSize"])
	p.bodySize = toFloat64(req["bodySize"])
	p.context, _ = params["context"].(string)
	p.timestamp = toFloat64(params["timestamp"])
	return p
}

// bidiToHAREntry builds a Playwright resource-snapshot event from a
// correlated BiDi request and response (or fetchError).
func bidiToHAREntry(pending *pendingRequest, responseParams map[string]interface{}, isFetchError bool) traceEvent {
	endTimestamp := toFloat64(responseParams["timestamp"])
	timeDelta := 0.0
	if endTimestamp > 0 && pending.timestamp > 0 {
		timeDelta = endTimestamp - pending.timestamp
	}

	startTime := pending.timestamp
	if startTime == 0 {
		startTime = float64(time.Now().UnixMilli())
	}

	// Build HAR request
	harRequest := map[string]interface{}{
		"method":      pending.method,
		"url":         pending.url,
		"httpVersion": "HTTP/1.1",
		"cookies":     flattenBidiCookies(pending.cookies),
		"headers":     flattenBidiHeaders(pending.headers),
		"queryString": parseQueryString(pending.url),
		"headersSize": pending.headersSize,
		"bodySize":    pending.bodySize,
	}

	// Build HAR response
	harResponse := buildHARResponse(responseParams, isFetchError)

	// Context for _frameref
	context := pending.context
	if c, _ := responseParams["context"].(string); c != "" {
		context = c
	}

	// Build startedDateTime as ISO 8601
	startedDateTime := time.UnixMilli(int64(startTime)).UTC().Format(time.RFC3339Nano)

	entry := map[string]interface{}{
		"startedDateTime": startedDateTime,
		"time":            timeDelta,
		"request":         harRequest,
		"response":        harResponse,
		"cache":           map[string]interface{}{},
		"timings": map[string]interface{}{
			"send":    float64(-1),
			"wait":    timeDelta,
			"receive": float64(-1),
		},
		"_monotonicTime": startTime / 1000.0,
	}
	if context != "" {
		entry["_frameref"] = context
	}

	return traceEvent{
		"type":     "resource-snapshot",
		"snapshot": entry,
	}
}

// buildHARResponse creates the HAR response object from BiDi responseCompleted
// or fetchError params.
func buildHARResponse(params map[string]interface{}, isFetchError bool) map[string]interface{} {
	if isFetchError {
		errorText, _ := params["errorText"].(string)
		return map[string]interface{}{
			"status":      0,
			"statusText":  "",
			"httpVersion": "HTTP/1.1",
			"cookies":     []interface{}{},
			"headers":     []interface{}{},
			"content": map[string]interface{}{
				"size":     float64(0),
				"mimeType": "",
			},
			"redirectURL":  "",
			"headersSize":  float64(-1),
			"bodySize":     float64(0),
			"_failureText": errorText,
		}
	}

	resp, _ := params["response"].(map[string]interface{})
	if resp == nil {
		resp = map[string]interface{}{}
	}

	status := toFloat64(resp["status"])
	statusText, _ := resp["statusText"].(string)
	protocol, _ := resp["protocol"].(string)
	mimeType, _ := resp["mimeType"].(string)
	bytesReceived := toFloat64(resp["bytesReceived"])
	headers, _ := resp["headers"].([]interface{})

	httpVersion := protocolToHTTPVersion(protocol)

	return map[string]interface{}{
		"status":      status,
		"statusText":  statusText,
		"httpVersion": httpVersion,
		"cookies":     []interface{}{},
		"headers":     flattenBidiHeaders(headers),
		"content": map[string]interface{}{
			"size":     bytesReceived,
			"mimeType": mimeType,
		},
		"redirectURL": "",
		"headersSize": float64(-1),
		"bodySize":    bytesReceived,
	}
}

// flattenBidiHeaders converts BiDi header format [{name, value: {type, value}}]
// to HAR format [{name, value}].
func flattenBidiHeaders(headers []interface{}) []interface{} {
	if headers == nil {
		return []interface{}{}
	}
	result := make([]interface{}, 0, len(headers))
	for _, h := range headers {
		hdr, ok := h.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := hdr["name"].(string)
		value := ""
		if v, ok := hdr["value"].(map[string]interface{}); ok {
			value, _ = v["value"].(string)
		} else if v, ok := hdr["value"].(string); ok {
			value = v
		}
		result = append(result, map[string]interface{}{
			"name":  name,
			"value": value,
		})
	}
	return result
}

// flattenBidiCookies converts BiDi cookies to a simple array.
// BiDi cookies are already fairly flat, so we just ensure the result is non-nil.
func flattenBidiCookies(cookies []interface{}) []interface{} {
	if cookies == nil {
		return []interface{}{}
	}
	return cookies
}

// parseQueryString extracts query parameters from a URL as HAR queryString entries.
func parseQueryString(rawURL string) []interface{} {
	u, err := url.Parse(rawURL)
	if err != nil || u.RawQuery == "" {
		return []interface{}{}
	}
	result := []interface{}{}
	for _, pair := range strings.Split(u.RawQuery, "&") {
		parts := strings.SplitN(pair, "=", 2)
		name := parts[0]
		value := ""
		if len(parts) > 1 {
			value = parts[1]
		}
		// URL-decode for readability
		decodedName, err := url.QueryUnescape(name)
		if err == nil {
			name = decodedName
		}
		decodedValue, err := url.QueryUnescape(value)
		if err == nil {
			value = decodedValue
		}
		result = append(result, map[string]interface{}{
			"name":  name,
			"value": value,
		})
	}
	return result
}

// protocolToHTTPVersion maps a BiDi protocol string to an HTTP version string.
func protocolToHTTPVersion(protocol string) string {
	switch protocol {
	case "h2", "h2c":
		return "h2"
	case "h3":
		return "h3"
	case "http/1.0":
		return "HTTP/1.0"
	case "http/1.1", "":
		return "HTTP/1.1"
	default:
		return "HTTP/1.1"
	}
}

// toFloat64 converts a numeric interface{} to float64.
func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}

// StopScreenshots signals the screenshot goroutine to stop and waits for it.
func (t *TraceRecorder) StopScreenshots() {
	t.mu.Lock()
	ch := t.screenshotStop
	t.screenshotStop = nil
	t.mu.Unlock()

	if ch != nil {
		close(ch)
		t.screenshotWg.Wait()
	}
}

// StartScreenshotLoop starts a background goroutine that captures screenshots periodically.
// captureFunc should return (base64-encoded image data, pageID, error).
func (t *TraceRecorder) StartScreenshotLoop(captureFunc func() (string, string, error)) {
	t.mu.Lock()
	t.screenshotStop = make(chan struct{})
	stopCh := t.screenshotStop
	t.mu.Unlock()

	t.screenshotWg.Add(1)
	go func() {
		defer t.screenshotWg.Done()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				b64Data, pageID, err := captureFunc()
				if err != nil || b64Data == "" {
					continue
				}

				imgData, err := decodeBase64(b64Data)
				if err != nil {
					continue
				}

				w, h := imageDimensions(imgData)
				t.AddScreenshot(imgData, pageID, w, h)
			}
		}
	}()
}

// buildZipLocked creates the Playwright-compatible trace zip.
// Must be called with t.mu held.
func (t *TraceRecorder) buildZipLocked() ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// Write trace events: <chunkIndex>-trace.trace
	traceName := fmt.Sprintf("%d-trace.trace", t.chunkIndex)
	tw, err := zw.Create(traceName)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace entry: %w", err)
	}
	for _, event := range t.events {
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}
		tw.Write(data)
		tw.Write([]byte("\n"))
	}

	// Write network events: <chunkIndex>-trace.network
	netName := fmt.Sprintf("%d-trace.network", t.chunkIndex)
	nw, err := zw.Create(netName)
	if err != nil {
		return nil, fmt.Errorf("failed to create network entry: %w", err)
	}
	for _, event := range t.network {
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}
		nw.Write(data)
		nw.Write([]byte("\n"))
	}

	// Write resources: resources/<sha1> (no extension — Playwright looks up by bare sha1)
	for hash, data := range t.resources {
		rw, err := zw.Create("resources/" + hash)
		if err != nil {
			continue
		}
		rw.Write(data)
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close zip: %w", err)
	}

	return buf.Bytes(), nil
}

// sha1Hex returns the lowercase hex-encoded SHA1 hash of data.
func sha1Hex(data []byte) string {
	h := sha1.Sum(data)
	return fmt.Sprintf("%x", h)
}

// decodeBase64 decodes a standard base64 string.
func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// pngDimensions reads width and height from a PNG file's IHDR chunk.
// Returns (0, 0) if the data is not a valid PNG.
func pngDimensions(data []byte) (int, int) {
	// PNG header: 8 bytes signature + 4 bytes chunk length + 4 bytes "IHDR" + 4 bytes width + 4 bytes height
	if len(data) < 24 {
		return 0, 0
	}
	w := int(binary.BigEndian.Uint32(data[16:20]))
	h := int(binary.BigEndian.Uint32(data[20:24]))
	return w, h
}

// jpegDimensions reads width and height from a JPEG file's SOF0 marker.
// Returns (0, 0) if the data is not a valid JPEG.
func jpegDimensions(data []byte) (int, int) {
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		return 0, 0 // not a JPEG
	}
	i := 2
	for i+1 < len(data) {
		if data[i] != 0xFF {
			return 0, 0
		}
		marker := data[i+1]
		i += 2
		// Skip padding bytes (0xFF fill)
		if marker == 0xFF {
			i--
			continue
		}
		// SOI, RST0-RST7 and TEM have no payload
		if marker == 0xD8 || (marker >= 0xD0 && marker <= 0xD7) || marker == 0x01 {
			continue
		}
		// EOI or SOS — stop scanning
		if marker == 0xD9 || marker == 0xDA {
			return 0, 0
		}
		// Read segment length
		if i+2 > len(data) {
			return 0, 0
		}
		segLen := int(binary.BigEndian.Uint16(data[i : i+2]))
		// SOF0 (0xC0) through SOF3 (0xC3) contain dimensions
		if marker >= 0xC0 && marker <= 0xC3 {
			if i+segLen > len(data) || segLen < 7 {
				return 0, 0
			}
			// Offset within segment: 2 (length) + 1 (precision) + 2 (height) + 2 (width)
			h := int(binary.BigEndian.Uint16(data[i+3 : i+5]))
			w := int(binary.BigEndian.Uint16(data[i+5 : i+7]))
			return w, h
		}
		i += segLen
	}
	return 0, 0
}

// imageDimensions detects the image format (PNG or JPEG) and returns width, height.
func imageDimensions(data []byte) (int, int) {
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
		return pngDimensions(data)
	}
	return jpegDimensions(data)
}

// WriteTraceToFile writes trace zip data to a file, creating directories as needed.
func WriteTraceToFile(data []byte, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create trace dir: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}
