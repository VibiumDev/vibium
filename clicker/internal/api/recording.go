package api

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vibium/clicker/internal/log"
)

// RecordingStartOptions configures how recording behaves.
type RecordingStartOptions struct {
	Name        string  `json:"name"`
	Screenshots bool    `json:"screenshots"`
	Snapshots   bool    `json:"snapshots"`
	Sources     bool    `json:"sources"`
	Title       string  `json:"title"`
	Bidi        bool    `json:"bidi"`
	Format      string  `json:"format"`  // "png" or "jpeg" (default "jpeg")
	Quality     float64 `json:"quality"` // 0.0-1.0 for JPEG (default 0.5)
	Video       VideoOptions
	Path        string // declared delivery path; "" = bytes-only, lost on close
}

// VideoMode selects how the recording treats engine video support.
type VideoMode int

const (
	// VideoAuto records video when the engine supports it; otherwise the
	// recording proceeds and the stop result reports videoUnavailable.
	VideoAuto VideoMode = iota
	// VideoOff disables video capture.
	VideoOff
	// VideoRequired makes recording.start fail if the engine can't deliver.
	VideoRequired
)

// VideoOptions is the recording's video track configuration. Dimensions
// default to the viewport; explicit dimensions that mismatch the window
// aspect are letterboxed by the engine.
type VideoOptions struct {
	Mode      VideoMode
	Width     int
	Height    int
	FrameRate int
	// RemoteKeep (video: {remote: "keep"}) records on a remote browser
	// connection and leaves the file on the remote host: the manifest and
	// stop result carry its path there, and cleanup is the caller's.
	// Ignored on local connections, where the video embeds normally.
	RemoteKeep bool
}

// ParseRecordingOptions extracts RecordingStartOptions from a params map.
// Used by both the proxy (handleRecordingStart) and MCP (browserRecordStart)
// paths so option parsing is defined once.
func ParseRecordingOptions(params map[string]interface{}) RecordingStartOptions {
	var opts RecordingStartOptions
	opts.Screenshots = true // default: screenshots on (opt out with screenshots=false)
	if name, ok := params["name"].(string); ok {
		opts.Name = name
	}
	if title, ok := params["title"].(string); ok {
		opts.Title = title
	}
	if ss, ok := params["screenshots"].(bool); ok {
		opts.Screenshots = ss
	}
	if sn, ok := params["snapshots"].(bool); ok {
		opts.Snapshots = sn
	}
	if src, ok := params["sources"].(bool); ok {
		opts.Sources = src
	}
	if b, ok := params["bidi"].(bool); ok {
		opts.Bidi = b
	}
	// Screenshot format: "jpeg" (default) or "png"
	opts.Format = "jpeg"
	if f, ok := params["format"].(string); ok && (f == "png" || f == "jpeg") {
		opts.Format = f
	}
	opts.Quality = 0.5
	if q, ok := params["quality"].(float64); ok && q >= 0 && q <= 1 {
		opts.Quality = q
	}
	// video: bool | {width, height, frameRate}. An explicit object counts as
	// requiring video: the caller asked for specific output, so a silent
	// no-video recording would be a surprise.
	switch v := params["video"].(type) {
	case bool:
		if v {
			opts.Video.Mode = VideoRequired
		} else {
			opts.Video.Mode = VideoOff
		}
	case map[string]interface{}:
		opts.Video.Mode = VideoRequired
		if w, ok := v["width"].(float64); ok {
			opts.Video.Width = int(w)
		}
		if h, ok := v["height"].(float64); ok {
			opts.Video.Height = int(h)
		}
		if f, ok := v["frameRate"].(float64); ok {
			opts.Video.FrameRate = int(f)
		}
		if r, ok := v["remote"].(string); ok && r == "keep" {
			opts.Video.RemoteKeep = true
		}
	}
	if p, ok := params["path"].(string); ok {
		opts.Path = p
	}
	return opts
}

// recordEvent is a generic recording event stored as a JSON-friendly map.
type recordEvent = map[string]interface{}

// VideoTrack is the engine screencast attached to a recording. The video is
// one continuous session track filming the context active at start.
type VideoTrack struct {
	Context    string
	ID         string // active engine screencast id; "" once stopped
	EnginePath string // file the engine live-muxes; moved into the zip at stop
	// Remote: the engine wrote EnginePath on a remote host. The file is
	// never read, moved, or deleted from here; the manifest records the
	// path and the caller retrieves it.
	Remote    bool
	StartedAt int64 // wall-clock ms at the startScreencast acknowledgement
	// OffsetMs is the video's start relative to the recording's t0, aligning
	// the video timeline with trace event timestamps.
	OffsetMs   float64
	DurationMs int64
	Width      int
	Height     int
	Error      string // engine failure; the zip still delivers, video absent or partial
}

// VideoSummary is one entry of the videos array in the recording.stop result.
type VideoSummary struct {
	Context    string
	DurationMs int64
	Width      int
	Height     int
	// RemotePath is where a remote-keep video lives on the remote host.
	RemotePath string
	Error      string
}

// RecordingSummary is the metadata carried by the recording.stop result.
type RecordingSummary struct {
	Steps            int
	DurationMs       int64
	Videos           []VideoSummary
	VideoUnavailable string
}

// ResultFields renders the summary as wire result fields: steps, durationMs,
// and videos — or videoUnavailable in its place.
func (s RecordingSummary) ResultFields() map[string]interface{} {
	fields := map[string]interface{}{
		"steps":      s.Steps,
		"durationMs": s.DurationMs,
	}
	if s.VideoUnavailable != "" {
		fields["videoUnavailable"] = s.VideoUnavailable
		return fields
	}
	if len(s.Videos) > 0 {
		videos := make([]interface{}, 0, len(s.Videos))
		for _, v := range s.Videos {
			entry := map[string]interface{}{
				"context":    strings.ToLower(v.Context),
				"durationMs": v.DurationMs,
				"width":      v.Width,
				"height":     v.Height,
			}
			if v.RemotePath != "" {
				entry["remotePath"] = v.RemotePath
			}
			if v.Error != "" {
				entry["error"] = v.Error
			}
			videos = append(videos, entry)
		}
		fields["videos"] = videos
	}
	return fields
}

// groupEntry tracks a group's name and callId so StopGroup can emit a matching "after" event.
type groupEntry struct {
	name   string
	callId string
}

// pendingRequest holds a parsed beforeRequestSent event until its response arrives.
type pendingRequest struct {
	context     string
	requestID   string
	url         string
	method      string
	headers     []interface{} // raw BiDi header list
	cookies     []interface{}
	headersSize float64
	bodySize    float64
	timestamp   float64 // BiDi timestamp (ms since epoch)
}

// Recorder manages recording state for a browser session.
// It collects events, screenshots, and DOM snapshots, then packages
// them into a Playwright-compatible trace zip.
type Recorder struct {
	mu              sync.Mutex
	recording       bool
	options         RecordingStartOptions
	events          []recordEvent              // current chunk's recording events
	network         []recordEvent              // current chunk's network events
	resources       map[string][]byte          // resource name -> binary data (JPEG/PNG)
	groupStack      []groupEntry               // nested group entries (name + callId)
	pendingRequests map[string]*pendingRequest // BiDi request ID -> pending request
	chunkIndex      int
	startTime       int64  // unix ms
	monotonicBase   int64  // unix ms at recording start, all monotonic times are relative to this
	contextId       string // unique context ID for this recording session
	actionCounter   int    // monotonic counter for action/bidi callIds

	// Video track (engine screencast folded into the recording)
	video            *VideoTrack
	videoUnavailable string // engine reason video could not start (auto mode)

	// Screenshot goroutine control
	screenshotStop chan struct{}
	screenshotWg   sync.WaitGroup
}

// NewRecorder creates a new recorder.
func NewRecorder() *Recorder {
	return &Recorder{
		resources:       make(map[string][]byte),
		pendingRequests: make(map[string]*pendingRequest),
	}
}

// monotonicNow returns the current time as relative monotonic ms since recording start.
func (t *Recorder) monotonicNow() float64 {
	return float64(time.Now().UnixMilli() - t.monotonicBase)
}

// IsRecording returns whether recording is currently active.
func (t *Recorder) IsRecording() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.recording
}

// Start begins recording with the given options.
// viewport is the browser viewport size (may be nil if unknown).
func (t *Recorder) Start(opts RecordingStartOptions, viewport map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.recording = true
	t.options = opts
	t.events = nil
	t.network = nil
	t.resources = make(map[string][]byte)
	t.pendingRequests = make(map[string]*pendingRequest)
	t.groupStack = nil
	t.video = nil
	t.videoUnavailable = ""
	t.chunkIndex = 0
	t.startTime = time.Now().UnixMilli()
	t.monotonicBase = t.startTime
	t.contextId = fmt.Sprintf("context@%x", t.startTime)

	title := opts.Title
	if title == "" {
		title = opts.Name
	}

	// Build options map
	options := map[string]interface{}{}
	if viewport != nil {
		options["viewport"] = viewport
	}

	// First event must be context-options (required by Playwright trace viewer / Record Player)
	t.events = append(t.events, recordEvent{
		"type":           "context-options",
		"browserName":    "chromium",
		"platform":       runtime.GOOS,
		"wallTime":       float64(t.startTime),
		"monotonicTime":  float64(0),
		"title":          title,
		"contextId":      t.contextId,
		"options":        options,
		"sdkLanguage":    "javascript",
		"version":        8,
		"origin":         "library",
		"libraryName":    "vibium",
		"libraryVersion": Version,
	})
}

// NoteDroppedEvents stamps into the trace that count BiDi events were
// dropped before reaching the recorder, so a recording with holes says so.
func (t *Recorder) NoteDroppedEvents(count uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.recording || count == 0 {
		return
	}

	t.events = append(t.events, recordEvent{
		"type":   "event",
		"method": "vibium.eventsDropped",
		"params": map[string]interface{}{"count": count},
		"time":   t.monotonicNow(),
		"class":  "BrowserContext",
	})
}

// Stop stops recording and returns the recording zip data. The engine-written
// video file, if any, is moved into the zip: read into the archive and then
// deleted.
func (t *Recorder) Stop() ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.recording {
		return nil, fmt.Errorf("recording is not started")
	}

	t.recording = false
	data, err := t.buildZipLocked(true)
	if err == nil && t.video != nil && t.video.EnginePath != "" && !t.video.Remote {
		if rmErr := os.Remove(t.video.EnginePath); rmErr != nil {
			// A leaked engine temp is worth a trace, not a failed stop (#317).
			log.Debug("failed to delete engine video temp", "path", t.video.EnginePath, "error", rmErr)
		}
		t.video.EnginePath = ""
	}
	return data, err
}

// SetVideoTrack attaches the engine screencast to the recording and stamps
// its offset from the recording's t0.
func (t *Recorder) SetVideoTrack(v *VideoTrack) {
	t.mu.Lock()
	defer t.mu.Unlock()
	v.OffsetMs = float64(v.StartedAt - t.startTime)
	t.video = v
}

// SetVideoUnavailable records why video could not start; the recording
// proceeds without it and the stop result carries the reason.
func (t *Recorder) SetVideoUnavailable(reason string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.videoUnavailable = reason
}

// VideoUnavailable returns the engine's reason video could not start, or "".
func (t *Recorder) VideoUnavailable() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.videoUnavailable
}

// ActiveVideo returns a copy of the recording's video track, or nil.
func (t *Recorder) ActiveVideo() *VideoTrack {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.video == nil {
		return nil
	}
	v := *t.video
	return &v
}

// FinishVideo marks the screencast stopped. enginePath, when non-empty,
// is where the engine finalized the file. errMsg records a stop failure;
// the engine path is kept so a partial live-muxed video still delivers.
func (t *Recorder) FinishVideo(enginePath, errMsg string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.video == nil {
		return
	}
	t.video.ID = ""
	t.video.DurationMs = time.Now().UnixMilli() - t.video.StartedAt
	if enginePath != "" {
		t.video.EnginePath = enginePath
	}
	if errMsg != "" {
		t.video.Error = errMsg
	}
}

// RemoveEngineFile deletes the engine-written video temp file, for abandoned
// recordings (superseded, or lost on session close). A remote-keep video is
// never touched: its path names a file on another machine, and deleting it
// here could hit an unrelated local file of the same name.
func (t *Recorder) RemoveEngineFile() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.video != nil && t.video.EnginePath != "" && !t.video.Remote {
		if rmErr := os.Remove(t.video.EnginePath); rmErr != nil {
			log.Debug("failed to delete engine video temp", "path", t.video.EnginePath, "error", rmErr)
		}
		t.video.EnginePath = ""
	}
}

// Summary reports the stop-result metadata: step count, recording duration,
// and the video track's outcome.
func (t *Recorder) Summary() RecordingSummary {
	t.mu.Lock()
	defer t.mu.Unlock()

	s := RecordingSummary{
		DurationMs:       time.Now().UnixMilli() - t.startTime,
		VideoUnavailable: t.videoUnavailable,
	}
	for _, ev := range t.events {
		if ev["type"] == "before" && ev["class"] != "Tracing" {
			s.Steps++
		}
	}
	if t.video != nil {
		vs := VideoSummary{
			Context:    t.video.Context,
			DurationMs: t.video.DurationMs,
			Width:      t.video.Width,
			Height:     t.video.Height,
			Error:      t.video.Error,
		}
		if t.video.Remote {
			vs.RemotePath = t.video.EnginePath
		}
		s.Videos = append(s.Videos, vs)
	}
	return s
}

// StartChunk starts a new chunk within the current recording.
// viewport is the browser viewport size (may be nil if unknown).
func (t *Recorder) StartChunk(name, title string, viewport map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.events = nil
	t.network = nil
	t.chunkIndex++
	t.monotonicBase = time.Now().UnixMilli()

	chunkTitle := title
	if chunkTitle == "" {
		chunkTitle = name
	}

	// Build options map
	options := map[string]interface{}{}
	if viewport != nil {
		options["viewport"] = viewport
	}

	t.events = append(t.events, recordEvent{
		"type":           "context-options",
		"browserName":    "chromium",
		"platform":       runtime.GOOS,
		"wallTime":       float64(t.monotonicBase),
		"monotonicTime":  float64(0),
		"title":          chunkTitle,
		"contextId":      t.contextId,
		"options":        options,
		"sdkLanguage":    "javascript",
		"version":        8,
		"origin":         "library",
		"libraryName":    "vibium",
		"libraryVersion": Version,
	})
}

// StopChunk packages the current chunk into a zip and returns it.
// Recording remains active for additional chunks. Chunk artifacts carry no
// video file — the video is one continuous session track — but their
// manifest records where the chunk falls in it.
func (t *Recorder) StopChunk() ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.recording {
		return nil, fmt.Errorf("recording is not started")
	}

	return t.buildZipLocked(false)
}

// currentGroupIdLocked returns the callId of the innermost active group, or "".
// Must be called with t.mu held.
func (t *Recorder) currentGroupIdLocked() string {
	if len(t.groupStack) == 0 {
		return ""
	}
	return t.groupStack[len(t.groupStack)-1].callId
}

// StartGroup adds a group-start marker to the recording.
func (t *Recorder) StartGroup(name string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	parentId := t.currentGroupIdLocked()

	t.actionCounter++
	callId := fmt.Sprintf("call@%d", t.actionCounter)
	t.groupStack = append(t.groupStack, groupEntry{name: name, callId: callId})
	ev := recordEvent{
		"type":      "before",
		"callId":    callId,
		"title":     name,
		"class":     "Tracing",
		"method":    "tracingGroup",
		"params":    map[string]interface{}{"name": name},
		"startTime": t.monotonicNow(),
	}
	if parentId != "" {
		ev["parentId"] = parentId
	}
	t.events = append(t.events, ev)
}

// StopGroup adds a group-end marker to the recording.
func (t *Recorder) StopGroup() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.groupStack) == 0 {
		return
	}

	entry := t.groupStack[len(t.groupStack)-1]
	t.groupStack = t.groupStack[:len(t.groupStack)-1]

	t.events = append(t.events, recordEvent{
		"type":    "after",
		"callId":  entry.callId,
		"endTime": t.monotonicNow(),
	})
}

// Options returns the current recording options.
func (t *Recorder) Options() RecordingStartOptions {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.options
}

// StoreResource stores binary data (e.g. screenshot JPEG/PNG) in the resources
// map, keyed by name. The data will be written to resources/<name>
// in the recording zip.
func (t *Recorder) StoreResource(name string, data []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.resources[name] = data
}

// ScreenshotName generates a Playwright-compatible resource name for a screenshot.
// Format: page@<lowercase-hex>-<wallTimeMs>.<ext> (e.g. "page@abc123-1773879004791.jpeg")
func (t *Recorder) ScreenshotName(pageID string, ts time.Time) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return fmt.Sprintf("%s-%d.%s", formatPageID(pageID), ts.UnixMilli(), t.imageExtension())
}

// apiNameFromMethod maps a vibium: method to (class, title) for recording display.
func apiNameFromMethod(method string) (string, string) {
	// Strip the "vibium:" prefix
	if len(method) <= 7 || method[:7] != "vibium:" {
		return "Vibium", method
	}
	name := method[7:] // e.g. "element.click", "page.navigate", "element.text"

	switch {
	// Element commands: element.*
	case len(name) > 8 && name[:8] == "element.":
		return "Element", "Element." + name[8:]

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
func (t *Recorder) NextCallId() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.recording {
		return ""
	}

	t.actionCounter++
	return fmt.Sprintf("call@%d", t.actionCounter)
}

// PatchBeforeSnapshot retroactively adds a beforeSnapshot to an already-emitted
// "before" event. This is used by click-like handlers that capture the snapshot
// after scrolling the element into view (via resolveWithActionability) but before
// the actual click/hover/tap action.
func (t *Recorder) PatchBeforeSnapshot(callId, snapshotName string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i := len(t.events) - 1; i >= 0; i-- {
		if t.events[i]["callId"] == callId && t.events[i]["type"] == "before" {
			t.events[i]["beforeSnapshot"] = snapshotName
			return
		}
	}
}

// RecordAction records a vibium command as an action marker in the recording.
// The callId should come from NextCallId(). beforeSnapshot is the snapshot name
// (from AddFrameSnapshot) to link, or "" if none. pageId is a fallback browsing
// context to use when params["context"] is not set.
func (t *Recorder) RecordAction(callId, method string, params map[string]interface{}, beforeSnapshot, pageId string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.recording || callId == "" {
		return
	}

	// Shallow-copy params and lowercase context for recording (don't mutate caller's map)
	recordParams := make(map[string]interface{}, len(params))
	for k, v := range params {
		recordParams[k] = v
	}
	if ctx, ok := recordParams["context"].(string); ok && ctx != "" {
		recordParams["context"] = strings.ToLower(ctx)
	}

	class, title := apiNameFromMethod(method)
	ev := recordEvent{
		"type":      "before",
		"callId":    callId,
		"title":     title,
		"class":     class,
		"method":    method,
		"params":    recordParams,
		"startTime": t.monotonicNow(),
	}
	// Add pageId so the viewer can match actions to page screenshots
	if ctx, ok := recordParams["context"].(string); ok && ctx != "" {
		ev["pageId"] = formatPageID(ctx)
	} else if pageId != "" {
		ev["pageId"] = formatPageID(pageId)
	}
	if beforeSnapshot != "" {
		ev["beforeSnapshot"] = beforeSnapshot
	}
	// Link to parent group for nesting in Record Player
	if gid := t.currentGroupIdLocked(); gid != "" {
		ev["parentId"] = gid
	}
	t.events = append(t.events, ev)
}

// RecordActionEnd records the end of a vibium command action in the recording.
// The callId must match the value returned by NextCallId(). afterSnapshot is the
// snapshot name (from AddFrameSnapshot) to link, or "" if none. endTime is the
// actual handler completion time (before screenshot captures). box is the bounding
// box of the element that was interacted with, or nil for non-element actions.
func (t *Recorder) RecordActionEnd(callId, afterSnapshot string, endTime time.Time, box *BoxInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.recording || callId == "" {
		return
	}

	// Emit a Playwright-compatible "input" event with point and box when an
	// element was resolved. Playwright's trace viewer reads point from this
	// event type (keyed by callId) to render click-dot overlays.
	if box != nil {
		t.events = append(t.events, recordEvent{
			"type":   "input",
			"callId": callId,
			"point": map[string]interface{}{
				"x": box.X + box.Width/2,
				"y": box.Y + box.Height/2,
			},
			"box": map[string]interface{}{
				"x": box.X, "y": box.Y, "width": box.Width, "height": box.Height,
			},
		})
	}

	ev := recordEvent{
		"type":    "after",
		"callId":  callId,
		"endTime": float64(endTime.UnixMilli() - t.monotonicBase),
	}
	if afterSnapshot != "" {
		ev["afterSnapshot"] = afterSnapshot
	}
	t.events = append(t.events, ev)
}

// RecordBidiCommand records a raw BiDi command sent to the browser in the recording (opt-in via bidi: true).
// Returns the callId so the caller can pass it to RecordBidiCommandEnd.
func (t *Recorder) RecordBidiCommand(method string, params map[string]interface{}) string {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.recording {
		return ""
	}

	// Shallow-copy params and lowercase context for recording (don't mutate caller's map)
	recordParams := make(map[string]interface{}, len(params))
	for k, v := range params {
		recordParams[k] = v
	}
	if ctx, ok := recordParams["context"].(string); ok && ctx != "" {
		recordParams["context"] = strings.ToLower(ctx)
	}

	t.actionCounter++
	callId := fmt.Sprintf("call@%d", t.actionCounter)
	ev := recordEvent{
		"type":      "before",
		"callId":    callId,
		"title":     method,
		"class":     "BiDi",
		"method":    method,
		"params":    recordParams,
		"startTime": t.monotonicNow(),
	}
	// Link to parent group for nesting in Record Player
	if gid := t.currentGroupIdLocked(); gid != "" {
		ev["parentId"] = gid
	}
	t.events = append(t.events, ev)
	return callId
}

// RecordBidiCommandEnd records the end of a BiDi command in the recording.
// The callId must match the value returned by the corresponding RecordBidiCommand call.
func (t *Recorder) RecordBidiCommandEnd(callId string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.recording || callId == "" {
		return
	}

	t.events = append(t.events, recordEvent{
		"type":    "after",
		"callId":  callId,
		"endTime": t.monotonicNow(),
	})
}

// AddScreenshot stores a screenshot image (PNG or JPEG) and adds a screencast-frame event.
// If ts is non-zero it is used as the event timestamp; otherwise time.Now() is used.
func (t *Recorder) AddScreenshot(pngData []byte, pageID string, width, height int, ts time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.recording {
		return
	}

	if ts.IsZero() {
		ts = time.Now()
	}

	formattedPageID := formatPageID(pageID)
	name := fmt.Sprintf("%s-%d.%s", formattedPageID, ts.UnixMilli(), t.imageExtension())
	t.resources[name] = pngData
	t.events = append(t.events, recordEvent{
		"type":      "screencast-frame",
		"pageId":    formattedPageID,
		"sha1":      name,
		"width":     width,
		"height":    height,
		"timestamp": float64(ts.UnixMilli() - t.monotonicBase),
	})
}

// AddFrameSnapshot adds a frame-snapshot event for the Record Player / Playwright trace viewer.
// snapshotType is "before" or "after"; callId is like "call@1".
// resourceOverrides maps synthetic URLs (e.g. "screenshot://sha1") to resource
// SHA1 hashes so the viewer can resolve them from the zip's resources/ directory.
// Returns the snapshot name (e.g. "before@call@1").
func (t *Recorder) AddFrameSnapshot(callId, snapshotType, pageId, frameURL, doctype string, html interface{}, viewport map[string]interface{}, resourceOverrides []interface{}) string {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.recording {
		return ""
	}

	if resourceOverrides == nil {
		resourceOverrides = []interface{}{}
	}

	snapshotName := snapshotType + "@" + callId
	now := t.monotonicNow()

	formattedPageId := formatPageID(pageId)
	t.events = append(t.events, recordEvent{
		"type": "frame-snapshot",
		"snapshot": map[string]interface{}{
			"callId":            callId,
			"snapshotName":      snapshotName,
			"pageId":            formattedPageId,
			"frameId":           formattedPageId,
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

// RecordBidiEvent records a raw BiDi event from the browser into the recording.
// Network events are correlated by request ID and transformed into
// Playwright-compatible HAR resource-snapshot entries.
func (t *Recorder) RecordBidiEvent(msg string) {
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
			entry := bidiToHAREntry(pending, bidiEvent.Params, false, t.monotonicBase)
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
			entry := bidiToHAREntry(pending, bidiEvent.Params, true, t.monotonicBase)
			t.network = append(t.network, entry)
		}

	default:
		// Lowercase context in params for consistency
		if ctx, ok := bidiEvent.Params["context"].(string); ok && ctx != "" {
			bidiEvent.Params["context"] = strings.ToLower(ctx)
		}
		t.events = append(t.events, recordEvent{
			"type":   "event",
			"method": bidiEvent.Method,
			"params": bidiEvent.Params,
			"time":   t.monotonicNow(),
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
func bidiToHAREntry(pending *pendingRequest, responseParams map[string]interface{}, isFetchError bool, monotonicBase int64) recordEvent {
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
		"_monotonicTime": startTime - float64(monotonicBase),
	}
	if context != "" {
		entry["_frameref"] = formatPageID(context)
	}

	return recordEvent{
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
func (t *Recorder) StopScreenshots() {
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
func (t *Recorder) StartScreenshotLoop(captureFunc func() (string, string, error)) {
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

				w, h := ImageDimensions(imgData)
				t.AddScreenshot(imgData, pageID, w, h, time.Time{})
			}
		}
	}()
}

// buildZipLocked creates the Playwright-compatible recording zip.
// Must be called with t.mu held. includeVideo embeds the engine-written
// video file (session artifact); chunk artifacts pass false and get a
// videoRange manifest instead.
func (t *Recorder) buildZipLocked(includeVideo bool) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	now := time.Now()

	// createEntry creates a zip entry with the current timestamp.
	createEntry := func(name string) (io.Writer, error) {
		return zw.CreateHeader(&zip.FileHeader{
			Name:     name,
			Method:   zip.Deflate,
			Modified: now,
		})
	}

	// Write trace events
	var traceName string
	if t.chunkIndex == 0 {
		traceName = "trace.trace"
	} else {
		traceName = fmt.Sprintf("%d.trace", t.chunkIndex)
	}
	tw, err := createEntry(traceName)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace entry: %w", err)
	}
	for _, event := range t.events {
		data, err := marshalEvent(event)
		if err != nil {
			continue
		}
		tw.Write(data)
		tw.Write([]byte("\n"))
	}

	// Write network events
	var netName string
	if t.chunkIndex == 0 {
		netName = "trace.network"
	} else {
		netName = fmt.Sprintf("%d.network", t.chunkIndex)
	}
	nw, err := createEntry(netName)
	if err != nil {
		return nil, fmt.Errorf("failed to create network entry: %w", err)
	}
	for _, event := range t.network {
		data, err := marshalEvent(event)
		if err != nil {
			continue
		}
		nw.Write(data)
		nw.Write([]byte("\n"))
	}

	// Write resources: resources/<name> (e.g. resources/page@abc123-1773879004791.jpeg)
	for name, data := range t.resources {
		rw, err := createEntry("resources/" + name)
		if err != nil {
			continue
		}
		rw.Write(data)
	}

	// Video entries are additive to the trace format; existing trace tooling
	// ignores them.
	if t.video != nil {
		if err := t.writeVideoEntriesLocked(zw, now, includeVideo); err != nil {
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close zip: %w", err)
	}

	return buf.Bytes(), nil
}

// writeVideoEntriesLocked writes video/<context>.webm and video/index.json.
// Must be called with t.mu held and t.video non-nil.
func (t *Recorder) writeVideoEntriesLocked(zw *zip.Writer, now time.Time, includeVideo bool) error {
	context := strings.ToLower(t.video.Context)

	entry := map[string]interface{}{"context": context}
	if includeVideo {
		entry["startedAt"] = t.video.StartedAt
		entry["offsetMs"] = t.video.OffsetMs
		entry["width"] = t.video.Width
		entry["height"] = t.video.Height
		entry["mimeType"] = "video/webm"
		if t.video.Error != "" {
			entry["error"] = t.video.Error
		}
		if t.video.Remote {
			// remote: 'keep' — the file lives on the remote host; record
			// where, and leave retrieval to the caller.
			entry["remotePath"] = t.video.EnginePath
		} else if t.video.EnginePath != "" {
			// WebM is already compressed; store it instead of deflating.
			name := "video/" + context + ".webm"
			data, err := os.ReadFile(t.video.EnginePath)
			if err == nil {
				vw, werr := zw.CreateHeader(&zip.FileHeader{
					Name:     name,
					Method:   zip.Store,
					Modified: now,
				})
				if werr != nil {
					return fmt.Errorf("failed to create video entry: %w", werr)
				}
				vw.Write(data)
				entry["file"] = name
			} else if t.video.Error == "" {
				entry["error"] = "video file unreadable: " + err.Error()
			}
		}
	} else {
		// Chunk manifest: where this chunk falls in the session video.
		start := t.monotonicBase - t.video.StartedAt
		if start < 0 {
			start = 0
		}
		entry["videoRange"] = []int64{start, time.Now().UnixMilli() - t.video.StartedAt}
	}

	iw, err := zw.CreateHeader(&zip.FileHeader{
		Name:     "video/index.json",
		Method:   zip.Deflate,
		Modified: now,
	})
	if err != nil {
		return fmt.Errorf("failed to create video index: %w", err)
	}
	index, err := json.Marshal(map[string]interface{}{
		"version": 1,
		"videos":  []interface{}{entry},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal video index: %w", err)
	}
	iw.Write(index)
	return nil
}

// imageExtension returns the file extension for the recording's image format.
func (t *Recorder) imageExtension() string {
	if t.options.Format == "png" {
		return "png"
	}
	return "jpeg"
}

// formatPageID converts a raw browsing context ID to Playwright-compatible
// page ID format: "page@" prefix + lowercase hex.
func formatPageID(contextID string) string {
	return "page@" + strings.ToLower(contextID)
}

// marshalEvent marshals a recordEvent with keys in Playwright-compatible order.
// "type" always comes first, then known fields in priority order, then any remaining keys alphabetically.
func marshalEvent(event recordEvent) ([]byte, error) {
	order := []string{
		// context-options (version before type to match Playwright)
		"version",
		// Common
		"type",
		// context-options (continued)
		"origin", "libraryName", "libraryVersion",
		"browserName", "platform", "wallTime", "monotonicTime",
		"sdkLanguage", "title", "contextId", "options",
		// before/after
		"callId", "startTime", "endTime",
		"class", "method", "pageId", "parentId", "params",
		"beforeSnapshot", "afterSnapshot", "inputSnapshot",
		// screencast-frame
		"sha1", "width", "height", "timestamp", "frameSwapWallTime",
		// input
		"point", "box",
		// frame-snapshot
		"snapshot",
		// event
		"time",
	}

	var buf bytes.Buffer
	buf.WriteByte('{')
	first := true
	written := make(map[string]bool)

	for _, key := range order {
		val, ok := event[key]
		if !ok {
			continue
		}
		if !first {
			buf.WriteByte(',')
		}
		keyJSON, _ := json.Marshal(key)
		valJSON, _ := json.Marshal(val)
		buf.Write(keyJSON)
		buf.WriteByte(':')
		buf.Write(valJSON)
		first = false
		written[key] = true
	}

	// Remaining keys alphabetically
	var remaining []string
	for key := range event {
		if !written[key] {
			remaining = append(remaining, key)
		}
	}
	sort.Strings(remaining)
	for _, key := range remaining {
		if !first {
			buf.WriteByte(',')
		}
		keyJSON, _ := json.Marshal(key)
		valJSON, _ := json.Marshal(event[key])
		buf.Write(keyJSON)
		buf.WriteByte(':')
		buf.Write(valJSON)
		first = false
	}

	buf.WriteByte('}')
	return buf.Bytes(), nil
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

// ImageDimensions detects the image format (PNG or JPEG) and returns width, height.
func ImageDimensions(data []byte) (int, int) {
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
		return pngDimensions(data)
	}
	return jpegDimensions(data)
}

// DefaultRecordPath returns the default recording destination in dir
// ("" = the working directory): <stem>-YYYYMMDD-HHMMSS.zip, where the
// stem is the recording's name, sanitized, or "record". Timestamped so a
// rerun never clobbers the previous artifact; same-second collisions get a
// -2 suffix.
func DefaultRecordPath(dir, name string) string {
	stem := sanitizeRecordStem(name)
	stamp := time.Now().Format("20060102-150405")
	path := filepath.Join(dir, fmt.Sprintf("%s-%s.zip", stem, stamp))
	for n := 2; ; n++ {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path
		}
		path = filepath.Join(dir, fmt.Sprintf("%s-%s-%d.zip", stem, stamp, n))
	}
}

// sanitizeRecordStem makes a recording name safe as a filename stem:
// characters outside [A-Za-z0-9._-] become "-", leading/trailing dashes and
// dots are trimmed (a leading dot would hide the file), and an empty result
// falls back to "record".
func sanitizeRecordStem(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	stem := strings.Trim(b.String(), "-.")
	if stem == "" {
		return "record"
	}
	return stem
}

// WriteRecordToFile writes recording zip data to a file, creating directories as needed.
func WriteRecordToFile(data []byte, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create recording dir: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}
