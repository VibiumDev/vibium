package agent

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vibium/clicker/internal/bidi"
	"github.com/vibium/clicker/internal/browser"
	"github.com/vibium/clicker/internal/log"
	"github.com/vibium/clicker/internal/api"
)

// Handlers manages browser session state and executes tool calls.
type Handlers struct {
	launchResult   *browser.LaunchResult
	client         *bidi.Client
	conn           *bidi.Connection
	screenshotDir  string
	headless       bool
	connectURL     string      // remote BiDi WebSocket URL (empty = local browser)
	connectHeaders http.Header // headers for remote WebSocket connection
	refMap         map[string]string // @e1 -> CSS selector
	lastMap        string            // last map output (for diff)
	recorder       *api.Recorder
	downloadDir    string
	lastElementBox *api.BoxInfo // stashed by AgentSession.SetLastElementBox via callback
	activeContext  string         // last page context switched to or created
}

// NewHandlers creates a new Handlers instance.
// screenshotDir specifies where screenshots are saved. If empty, file saving is disabled.
// headless controls whether the browser is launched in headless mode.
func NewHandlers(screenshotDir string, headless bool, connectURL string, connectHeaders http.Header) *Handlers {
	return &Handlers{
		screenshotDir:  screenshotDir,
		headless:       headless,
		connectURL:     connectURL,
		connectHeaders: connectHeaders,
	}
}

// newSession creates an AgentSession that writes element box info back to
// h.lastElementBox so Call() can include it in RecordActionEnd.
func (h *Handlers) newSession() *api.AgentSession {
	s := api.NewAgentSession(h.client)
	s.Context = h.activeContext
	s.OnBoxSet = func(box *api.BoxInfo) {
		h.lastElementBox = box
	}
	return s
}

// Call executes a tool by name with the given arguments.
// When recording is active, it wraps the dispatch with RecordAction/RecordActionEnd
// to produce before/after events (matching the API path), and captures a
// screenshot after each non-recording action completes.
func (h *Handlers) Call(name string, args map[string]interface{}) (*ToolsCallResult, error) {
	log.Debug("tool call", "name", name, "args", args)

	// Inject a synthetic find trace event before selector-based actions
	// so CLI recordings match the JS client's find→action pairs.
	// Skip @e refs — those come from an explicit find the user already ran.
	if h.recorder != nil && h.recorder.IsRecording() && needsFindStep(name) {
		if sel, ok := args["selector"].(string); ok && sel != "" && !strings.HasPrefix(sel, "@e") {
			h.recordFindStep(sel)
		}
	}

	var callId string
	if h.recorder != nil && h.recorder.IsRecording() && !isRecordingCommand(name) {
		callId = h.recorder.NextCallId()
		pageId := h.getContext()
		// Resolve @e1 refs to real selectors so the trace shows meaningful selectors
		recordArgs := h.resolveRefsInArgs(args)
		h.recorder.RecordAction(callId, mcpToolToMethod(name), recordArgs, "", pageId)
		h.lastElementBox = nil
	}

	result, err := h.dispatch(name, args)

	endTime := time.Now()

	// Read and clear the element box stashed by AgentSession.SetLastElementBox
	box := h.lastElementBox
	h.lastElementBox = nil

	// Per-action screenshot: capture after successful non-recording commands
	if err == nil && h.recorder != nil && h.recorder.IsRecording() && !isRecordingCommand(name) {
		api.CaptureRecordingScreenshot(h.newSession(), h.recorder, endTime)
	}

	if callId != "" {
		h.recorder.RecordActionEnd(callId, "", endTime, box)
	}

	return result, err
}

// dispatch routes a tool call to the appropriate handler method.
func (h *Handlers) dispatch(name string, args map[string]interface{}) (*ToolsCallResult, error) {
	switch name {
	case "browser_start":
		return h.browserLaunch(args)
	case "browser_navigate":
		return h.browserNavigate(args)
	case "browser_click":
		return h.browserClick(args)
	case "browser_type":
		return h.browserType(args)
	case "browser_screenshot":
		return h.browserScreenshot(args)
	case "browser_find":
		return h.browserFind(args)
	case "browser_evaluate":
		return h.browserEvaluate(args)
	case "browser_stop":
		return h.browserQuit(args)
	case "browser_get_text":
		return h.browserGetText(args)
	case "browser_get_url":
		return h.browserGetURL(args)
	case "browser_get_title":
		return h.browserGetTitle(args)
	case "browser_get_html":
		return h.browserGetHTML(args)
	case "browser_find_all":
		return h.browserFindAll(args)
	case "browser_wait":
		return h.browserWait(args)
	case "browser_hover":
		return h.browserHover(args)
	case "browser_select":
		return h.browserSelect(args)
	case "browser_scroll":
		return h.browserScroll(args)
	case "browser_keys":
		return h.browserKeys(args)
	case "browser_new_page":
		return h.browserNewPage(args)
	case "browser_list_pages":
		return h.browserListPages(args)
	case "browser_switch_page":
		return h.browserSwitchPage(args)
	case "browser_close_page":
		return h.browserClosePage(args)
	case "browser_a11y_tree":
		return h.browserA11yTree(args)
	case "page_clock_install":
		return h.pageClockInstall(args)
	case "page_clock_fast_forward":
		return h.pageClockFastForward(args)
	case "page_clock_run_for":
		return h.pageClockRunFor(args)
	case "page_clock_pause_at":
		return h.pageClockPauseAt(args)
	case "page_clock_resume":
		return h.pageClockResume(args)
	case "page_clock_set_fixed_time":
		return h.pageClockSetFixedTime(args)
	case "page_clock_set_system_time":
		return h.pageClockSetSystemTime(args)
	case "page_clock_set_timezone":
		return h.pageClockSetTimezone(args)
	case "browser_fill":
		return h.browserFill(args)
	case "browser_press":
		return h.browserPress(args)
	case "browser_back":
		return h.browserBack(args)
	case "browser_forward":
		return h.browserForward(args)
	case "browser_reload":
		return h.browserReload(args)
	case "browser_get_value":
		return h.browserGetValue(args)
	case "browser_get_attribute":
		return h.browserGetAttribute(args)
	case "browser_is_visible":
		return h.browserIsVisible(args)
	case "browser_check":
		return h.browserCheck(args)
	case "browser_uncheck":
		return h.browserUncheck(args)
	case "browser_scroll_into_view":
		return h.browserScrollIntoView(args)
	case "browser_wait_for_url":
		return h.browserWaitForURL(args)
	case "browser_wait_for_load":
		return h.browserWaitForLoad(args)
	case "browser_sleep":
		return h.browserSleep(args)
	case "browser_map":
		return h.browserMap(args)
	case "browser_diff_map":
		return h.browserDiffMap(args)
	case "browser_pdf":
		return h.browserPDF(args)
	case "browser_highlight":
		return h.browserHighlight(args)
	case "browser_dblclick":
		return h.browserDblClick(args)
	case "browser_focus":
		return h.browserFocus(args)
	case "browser_count":
		return h.browserCount(args)
	case "browser_is_enabled":
		return h.browserIsEnabled(args)
	case "browser_is_checked":
		return h.browserIsChecked(args)
	case "browser_wait_for_text":
		return h.browserWaitForText(args)
	case "browser_wait_for_fn":
		return h.browserWaitForFn(args)
	case "browser_dialog_accept":
		return h.browserDialogAccept(args)
	case "browser_dialog_dismiss":
		return h.browserDialogDismiss(args)
	case "browser_get_cookies":
		return h.browserGetCookies(args)
	case "browser_set_cookie":
		return h.browserSetCookie(args)
	case "browser_delete_cookies":
		return h.browserDeleteCookies(args)
	case "browser_mouse_move":
		return h.browserMouseMove(args)
	case "browser_mouse_down":
		return h.browserMouseDown(args)
	case "browser_mouse_up":
		return h.browserMouseUp(args)
	case "browser_mouse_click":
		return h.browserMouseClick(args)
	case "browser_drag":
		return h.browserDrag(args)
	case "browser_set_viewport":
		return h.browserSetViewport(args)
	case "browser_get_viewport":
		return h.browserGetViewport(args)
	case "browser_get_window":
		return h.browserGetWindow(args)
	case "browser_set_window":
		return h.browserSetWindow(args)
	case "browser_emulate_media":
		return h.browserEmulateMedia(args)
	case "browser_set_geolocation":
		return h.browserSetGeolocation(args)
	case "browser_set_content":
		return h.browserSetContent(args)
	case "browser_frames":
		return h.browserFrames(args)
	case "browser_frame":
		return h.browserFrame(args)
	case "browser_upload":
		return h.browserUpload(args)
	case "browser_record_start":
		return h.browserRecordStart(args)
	case "browser_record_stop":
		return h.browserRecordStop(args)
	case "browser_record_start_group":
		return h.browserRecordStartGroup(args)
	case "browser_record_stop_group":
		return h.browserRecordStopGroup(args)
	case "browser_record_start_chunk":
		return h.browserRecordStartChunk(args)
	case "browser_record_stop_chunk":
		return h.browserRecordStopChunk(args)
	case "browser_storage_state":
		return h.browserStorageState(args)
	case "browser_restore_storage":
		return h.browserRestoreStorage(args)
	case "browser_download_set_dir":
		return h.browserDownloadSetDir(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// isRecordingCommand returns true for commands that manage recording state
// (to avoid capturing screenshots of recording operations themselves).
func isRecordingCommand(name string) bool {
	switch name {
	case "browser_record_start", "browser_record_stop",
		"browser_record_start_group", "browser_record_stop_group",
		"browser_record_start_chunk", "browser_record_stop_chunk",
		"browser_screenshot":
		return true
	}
	return false
}

// needsFindStep returns true for commands that take a selector and perform an
// action (not find itself). These get a synthetic find trace event injected
// before dispatch so CLI recordings match the JS client's find→action pairs.
func needsFindStep(name string) bool {
	switch name {
	case "browser_click", "browser_dblclick", "browser_fill", "browser_type",
		"browser_press", "browser_hover", "browser_select",
		"browser_check", "browser_uncheck", "browser_focus",
		"browser_scroll_into_view", "browser_drag",
		"browser_get_text", "browser_get_html", "browser_get_value",
		"browser_get_attribute", "browser_is_visible",
		"browser_is_enabled", "browser_is_checked",
		"browser_upload", "browser_highlight":
		return true
	}
	return false
}

// recordFindStep emits a complete find trace event (before + screenshot + after)
// so that CLI recordings produce the same find→action pairs as the JS client.
func (h *Handlers) recordFindStep(selector string) {
	if h.recorder == nil || !h.recorder.IsRecording() {
		return
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return
	}

	callId := h.recorder.NextCallId()
	pageId := h.getContext()
	params := map[string]interface{}{"selector": selector}
	h.recorder.RecordAction(callId, "vibium:page.find", params, "", pageId)

	// Use the API path's polling find (handles page transitions, scrolls into view)
	script, scriptArgs := api.BuildFindScript(
		map[string]interface{}{"selector": selector}, false,
	)
	info, err := api.WaitForElementWithScript(s, ctx, script, scriptArgs, api.DefaultTimeout)

	endTime := time.Now()

	// Capture screenshot (element is now scrolled into view by the find script)
	api.CaptureRecordingScreenshot(s, h.recorder, endTime)

	var box *api.BoxInfo
	if err == nil && info != nil {
		box = &info.Box
	}
	h.recorder.RecordActionEnd(callId, "", endTime, box)
}

// getContext returns the first browsing context from the browser tree, or "".
func (h *Handlers) getContext() string {
	if h.client == nil {
		return ""
	}
	tree, err := h.client.GetTree()
	if err != nil || len(tree.Contexts) == 0 {
		return ""
	}
	return tree.Contexts[0].Context
}

// queryViewport queries the browser for the current viewport size.
// Returns nil if the query fails (best-effort).
func (h *Handlers) queryViewport() map[string]interface{} {
	context := h.getContext()
	if context == "" {
		return nil
	}
	result, err := api.EvalSimpleScript(h.newSession(), context, "() => window.innerWidth + ',' + window.innerHeight")
	if err != nil {
		return nil
	}
	parts := strings.SplitN(result, ",", 2)
	if len(parts) != 2 {
		return nil
	}
	w, err1 := strconv.Atoi(parts[0])
	h2, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return nil
	}
	return map[string]interface{}{"width": w, "height": h2}
}

// mcpToolToMethod maps an MCP tool name to a vibium: method name so the
// trace viewer shows the same action titles as the API path.
func mcpToolToMethod(name string) string {
	switch name {
	// Navigation
	case "browser_navigate":
		return "vibium:page.navigate"
	case "browser_back":
		return "vibium:page.back"
	case "browser_forward":
		return "vibium:page.forward"
	case "browser_reload":
		return "vibium:page.reload"

	// Element interaction
	case "browser_click":
		return "vibium:element.click"
	case "browser_dblclick":
		return "vibium:element.dblclick"
	case "browser_fill":
		return "vibium:element.fill"
	case "browser_type":
		return "vibium:element.type"
	case "browser_press":
		return "vibium:element.press"
	case "browser_hover":
		return "vibium:element.hover"
	case "browser_select":
		return "vibium:element.selectOption"
	case "browser_check":
		return "vibium:element.check"
	case "browser_uncheck":
		return "vibium:element.uncheck"
	case "browser_focus":
		return "vibium:element.focus"
	case "browser_scroll_into_view":
		return "vibium:element.scrollIntoView"
	case "browser_drag":
		return "vibium:element.dragTo"

	// Keyboard/mouse
	case "browser_keys":
		return "vibium:keyboard.press"
	case "browser_mouse_move":
		return "vibium:mouse.move"
	case "browser_mouse_down":
		return "vibium:mouse.down"
	case "browser_mouse_up":
		return "vibium:mouse.up"
	case "browser_mouse_click":
		return "vibium:mouse.click"
	case "browser_scroll":
		return "vibium:page.scroll"

	// Page queries
	case "browser_find":
		return "vibium:page.find"
	case "browser_find_all":
		return "vibium:page.findAll"
	case "browser_get_text":
		return "vibium:element.text"
	case "browser_get_html":
		return "vibium:element.html"
	case "browser_get_url":
		return "vibium:page.url"
	case "browser_get_title":
		return "vibium:page.title"
	case "browser_get_value":
		return "vibium:element.value"
	case "browser_get_attribute":
		return "vibium:element.attr"
	case "browser_is_visible":
		return "vibium:element.isVisible"
	case "browser_is_enabled":
		return "vibium:element.isEnabled"
	case "browser_is_checked":
		return "vibium:element.isChecked"
	case "browser_count":
		return "vibium:page.findAll"
	case "browser_evaluate":
		return "vibium:page.eval"
	case "browser_screenshot":
		return "vibium:page.screenshot"
	case "browser_pdf":
		return "vibium:page.pdf"
	case "browser_a11y_tree":
		return "vibium:page.a11yTree"

	// Waiting
	case "browser_wait":
		return "vibium:page.waitFor"
	case "browser_wait_for_url":
		return "vibium:page.waitForURL"
	case "browser_wait_for_load":
		return "vibium:page.waitForLoad"
	case "browser_wait_for_text":
		return "vibium:page.wait"
	case "browser_wait_for_fn":
		return "vibium:page.waitForFunction"
	case "browser_sleep":
		return "vibium:page.wait"

	// Pages
	case "browser_new_page":
		return "vibium:browser.newPage"
	case "browser_list_pages":
		return "vibium:browser.pages"
	case "browser_switch_page":
		return "vibium:page.activate"
	case "browser_close_page":
		return "vibium:page.close"

	// Viewport/window
	case "browser_set_viewport":
		return "vibium:page.setViewport"
	case "browser_get_viewport":
		return "vibium:page.viewport"
	case "browser_set_window":
		return "vibium:page.setWindow"
	case "browser_get_window":
		return "vibium:page.window"

	// Cookies/storage
	case "browser_get_cookies":
		return "vibium:context.cookies"
	case "browser_set_cookie":
		return "vibium:context.setCookies"
	case "browser_delete_cookies":
		return "vibium:context.clearCookies"
	case "browser_storage_state":
		return "vibium:context.storage"
	case "browser_restore_storage":
		return "vibium:context.setStorage"

	// Dialog
	case "browser_dialog_accept":
		return "vibium:dialog.accept"
	case "browser_dialog_dismiss":
		return "vibium:dialog.dismiss"

	// Media/content
	case "browser_emulate_media":
		return "vibium:page.emulateMedia"
	case "browser_set_geolocation":
		return "vibium:page.setGeolocation"
	case "browser_set_content":
		return "vibium:page.setContent"

	// Frames
	case "browser_frames":
		return "vibium:page.frames"
	case "browser_frame":
		return "vibium:page.frame"

	// Upload/download
	case "browser_upload":
		return "vibium:element.setFiles"
	case "browser_download_set_dir":
		return "vibium:download.saveAs"

	// Browser lifecycle
	case "browser_start":
		return "vibium:browser.newPage"
	case "browser_stop":
		return "vibium:browser.stop"

	// Map/highlight (vibium-specific)
	case "browser_map":
		return "vibium:page.eval"
	case "browser_diff_map":
		return "vibium:page.eval"
	case "browser_highlight":
		return "vibium:page.eval"

	// Clock
	case "page_clock_install":
		return "vibium:clock.install"
	case "page_clock_fast_forward":
		return "vibium:clock.fastForward"
	case "page_clock_run_for":
		return "vibium:clock.runFor"
	case "page_clock_pause_at":
		return "vibium:clock.pauseAt"
	case "page_clock_resume":
		return "vibium:clock.resume"
	case "page_clock_set_fixed_time":
		return "vibium:clock.setFixedTime"
	case "page_clock_set_system_time":
		return "vibium:clock.setSystemTime"
	case "page_clock_set_timezone":
		return "vibium:clock.setTimezone"

	default:
		return name
	}
}

// Close cleans up any active browser sessions.
func (h *Handlers) Close() {
	// Remote mode: end the BiDi session so chromedriver closes Chrome
	if h.connectURL != "" && h.client != nil {
		h.client.SendCommand("session.end", map[string]interface{}{})
	}
	if h.conn != nil {
		h.conn.Close()
		h.conn = nil
	}
	if h.launchResult != nil {
		h.launchResult.Close()
		h.launchResult = nil
	}
	h.client = nil
}

// browserLaunch launches a new browser session or connects to a remote one.
func (h *Handlers) browserLaunch(args map[string]interface{}) (*ToolsCallResult, error) {
	// If browser is already running, return success (no-op)
	if h.client != nil {
		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: "Browser already running",
			}},
		}, nil
	}

	// Remote browser connect mode
	if h.connectURL != "" {
		conn, client, sessionID, err := bidi.ConnectRemote(h.connectURL, h.connectHeaders)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to remote browser: %w", err)
		}
		h.conn = conn
		h.client = client

		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Connected to remote browser at %s (session %s)", h.connectURL, sessionID),
			}},
		}, nil
	}

	// Parse options — per-call headless overrides the default
	useHeadless := h.headless
	if val, ok := args["headless"].(bool); ok {
		useHeadless = val
	}

	// Launch browser
	launchResult, err := browser.Launch(browser.LaunchOptions{Headless: useHeadless})
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	// Use BiDi connection from launch if available, otherwise connect via WebSocket URL
	var conn *bidi.Connection
	if launchResult.BidiConn != nil {
		conn = launchResult.BidiConn
	} else {
		conn, err = bidi.Connect(launchResult.WebSocketURL)
		if err != nil {
			launchResult.Close()
			return nil, fmt.Errorf("failed to connect to browser: %w", err)
		}
	}

	h.launchResult = launchResult
	h.conn = conn
	h.client = bidi.NewClient(conn)

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Browser launched (headless: %v)", useHeadless),
		}},
	}, nil
}

// browserNavigate navigates to a URL.
func (h *Handlers) browserNavigate(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	url, ok := args["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("url is required")
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.Navigate(s, ctx, url, "complete"); err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Navigated to %s", url),
		}},
	}, nil
}

// browserClick clicks an element.
func (h *Handlers) browserClick(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.Click(s, ctx, elementParamsWithTimeout(selector, args)); err != nil {
		return nil, fmt.Errorf("failed to click: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Clicked element: %s", selector),
		}},
	}, nil
}

// browserType types text into an element.
func (h *Handlers) browserType(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	text, ok := args["text"].(string)
	if !ok {
		return nil, fmt.Errorf("text is required")
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.TypeInto(s, ctx, elementParamsWithTimeout(selector, args), text); err != nil {
		return nil, fmt.Errorf("failed to type: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Typed into element: %s", selector),
		}},
	}, nil
}

// browserScreenshot captures a screenshot.
func (h *Handlers) browserScreenshot(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	fullPage, _ := args["fullPage"].(bool)
	annotate, _ := args["annotate"].(bool)

	// If annotate, run map first to get refs, then inject matching labels
	if annotate {
		if _, err := h.browserMap(map[string]interface{}{}); err != nil {
			return nil, fmt.Errorf("failed to map for annotation: %w", err)
		}

		// Build ordered list of selectors from refMap (@e1, @e2, ...)
		selectors := make([]string, 0, len(h.refMap))
		for i := 1; i <= len(h.refMap); i++ {
			ref := fmt.Sprintf("@e%d", i)
			if sel, ok := h.refMap[ref]; ok {
				selectors = append(selectors, sel)
			}
		}

		// Pass selectors as a JSON string: the BiDi arg serializer has no array
		// case and stringifies a Go slice to "[a b c]", so the script received a
		// string and `document.querySelector("[")` threw, crashing annotation on
		// every page (issue #156). Marshal to JSON and parse it back in the page.
		selectorsJSON, err := json.Marshal(selectors)
		if err != nil {
			return nil, fmt.Errorf("failed to encode annotation selectors: %w", err)
		}
		annotateScript := `(selectorsJSON) => {
			const selectors = JSON.parse(selectorsJSON);
			let count = 0;
			for (let i = 0; i < selectors.length; i++) {
				const el = document.querySelector(selectors[i]);
				if (!el) continue;
				const rect = el.getBoundingClientRect();
				if (rect.width === 0 || rect.height === 0) continue;
				const label = document.createElement('div');
				label.className = '__vibium_annotation';
				label.textContent = i + 1;
				label.style.cssText = 'position:fixed;z-index:2147483647;background:red;color:white;font:bold 11px sans-serif;padding:1px 4px;border-radius:8px;pointer-events:none;line-height:16px;min-width:16px;text-align:center;left:' + (rect.left - 2) + 'px;top:' + (rect.top - 2) + 'px;';
				document.body.appendChild(label);
				count++;
			}
			return JSON.stringify({count: count});
		}`
		if _, err := h.client.CallFunction("", annotateScript, []interface{}{string(selectorsJSON)}); err != nil {
			return nil, fmt.Errorf("failed to annotate: %w", err)
		}
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	base64Data, err := api.Screenshot(s, ctx, fullPage)
	if err != nil {
		return nil, fmt.Errorf("failed to capture screenshot: %w", err)
	}

	// Clean up annotation labels
	if annotate {
		cleanupScript := `() => {
			document.querySelectorAll('.__vibium_annotation').forEach(el => el.remove());
			return 'cleaned';
		}`
		h.client.CallFunction("", cleanupScript, nil)
	}

	// If filename provided, save to file (only if screenshotDir is configured)
	if filename, ok := args["filename"].(string); ok && filename != "" {
		if h.screenshotDir == "" {
			return nil, fmt.Errorf("screenshot file saving is disabled (use --screenshot-dir to enable)")
		}

		// Create directory if it doesn't exist
		if err := os.MkdirAll(h.screenshotDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create screenshot directory: %w", err)
		}

		// Use only the basename to prevent path traversal
		safeName := filepath.Base(filename)
		fullPath := filepath.Join(h.screenshotDir, safeName)

		pngData, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode screenshot: %w", err)
		}
		if err := os.WriteFile(fullPath, pngData, 0644); err != nil {
			return nil, fmt.Errorf("failed to save screenshot: %w", err)
		}
		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Screenshot saved to %s", fullPath),
			}},
		}, nil
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type:     "image",
			Data:     base64Data,
			MimeType: "image/png",
		}},
	}, nil
}

// browserFind finds an element and returns its info.
// Supports CSS selector or semantic locators (text, label, placeholder, testid, xpath, alt, title).
func (h *Handlers) browserFind(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	// Check for semantic locators
	role, _ := args["role"].(string)
	text, _ := args["text"].(string)
	label, _ := args["label"].(string)
	placeholder, _ := args["placeholder"].(string)
	testid, _ := args["testid"].(string)
	xpath, _ := args["xpath"].(string)
	alt, _ := args["alt"].(string)
	title, _ := args["title"].(string)

	hasSemantic := role != "" || text != "" || label != "" || placeholder != "" || testid != "" || xpath != "" || alt != "" || title != ""

	if hasSemantic {
		timeout := api.DefaultTimeout
		if t, ok := argFloat(args, "timeout"); ok {
			timeout = time.Duration(t) * time.Millisecond
		}

		script := findBySemanticScript()
		result, err := pollCallFunction(h, script, []interface{}{role, text, label, placeholder, testid, xpath, alt, title}, timeout)
		if err != nil {
			desc := ""
			for _, pair := range []struct{ k, v string }{
				{"role", role}, {"text", text}, {"label", label}, {"placeholder", placeholder},
				{"testid", testid}, {"xpath", xpath}, {"alt", alt}, {"title", title},
			} {
				if pair.v != "" {
					if desc != "" {
						desc += ", "
					}
					desc += pair.k + "=" + pair.v
				}
			}
			return nil, fmt.Errorf("element not found: %s (timeout %s)", desc, timeout)
		}

		// Parse JSON result
		var found struct {
			Selector string `json:"selector"`
			Label    string `json:"label"`
		}
		if err := json.Unmarshal([]byte(fmt.Sprintf("%v", result)), &found); err != nil {
			return nil, fmt.Errorf("failed to parse find result: %w", err)
		}

		// Store ref in refMap
		h.refMap = make(map[string]string)
		h.refMap["@e1"] = found.Selector

		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("@e1 %s", found.Label),
			}},
		}, nil
	}

	// CSS selector mode
	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector or semantic locator (role, text, label, placeholder, testid, xpath, alt, title) is required")
	}
	selector = h.resolveSelector(selector)

	// Run getLabel in browser to get consistent label format (with scroll-into-view)
	labelScript := `(selector) => {
		` + GetLabelJS() + `
		const el = document.querySelector(selector);
		if (!el) return null;
		if (el.scrollIntoViewIfNeeded) {
			el.scrollIntoViewIfNeeded(true);
		} else {
			el.scrollIntoView({ block: 'center', inline: 'nearest' });
		}
		return getLabel(el);
	}`
	labelResult, err := h.client.CallFunction("", labelScript, []interface{}{selector})
	if err != nil {
		return nil, err
	}

	// Store ref in refMap
	h.refMap = make(map[string]string)
	h.refMap["@e1"] = selector

	labelStr := fmt.Sprintf("%v", labelResult)
	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("@e1 %s", labelStr),
		}},
	}, nil
}

// findBySemanticScript returns the JS function for finding elements by semantic criteria.
// Returns JSON: {"selector":"...","label":"...","tag":"...","text":"...","box":{...}}
func findBySemanticScript() string {
	return `(role, text, label, placeholder, testid, xpath, alt, title) => {
		` + GetSelectorJS() + `
		` + GetLabelJS() + `

		const IMPLICIT_ROLES = {
			A: (el) => el.hasAttribute('href') ? 'link' : '',
			AREA: (el) => el.hasAttribute('href') ? 'link' : '',
			ARTICLE: () => 'article',
			ASIDE: () => 'complementary',
			BUTTON: () => 'button',
			DETAILS: () => 'group',
			DIALOG: () => 'dialog',
			FOOTER: () => 'contentinfo',
			FORM: () => 'form',
			H1: () => 'heading', H2: () => 'heading', H3: () => 'heading',
			H4: () => 'heading', H5: () => 'heading', H6: () => 'heading',
			HEADER: () => 'banner',
			HR: () => 'separator',
			IMG: (el) => el.getAttribute('alt') ? 'img' : 'presentation',
			INPUT: (el) => {
				const t = (el.getAttribute('type') || 'text').toLowerCase();
				const map = {button:'button',checkbox:'checkbox',image:'button',
					number:'spinbutton',radio:'radio',range:'slider',
					reset:'button',search:'searchbox',submit:'button',text:'textbox',
					email:'textbox',tel:'textbox',url:'textbox',password:'textbox'};
				return map[t] || 'textbox';
			},
			LI: () => 'listitem',
			MAIN: () => 'main',
			MENU: () => 'list',
			NAV: () => 'navigation',
			OL: () => 'list',
			OPTION: () => 'option',
			OUTPUT: () => 'status',
			PROGRESS: () => 'progressbar',
			SECTION: () => 'region',
			SELECT: (el) => el.hasAttribute('multiple') ? 'listbox' : 'combobox',
			SUMMARY: () => 'button',
			TABLE: () => 'table',
			TBODY: () => 'rowgroup', THEAD: () => 'rowgroup', TFOOT: () => 'rowgroup',
			TD: () => 'cell',
			TEXTAREA: () => 'textbox',
			TH: () => 'columnheader',
			TR: () => 'row',
			UL: () => 'list',
		};

		function getImplicitRole(el) {
			const explicit = el.getAttribute('role');
			if (explicit) return explicit.toLowerCase();
			const fn = IMPLICIT_ROLES[el.tagName];
			return fn ? fn(el).toLowerCase() : '';
		}

		function getName(el) {
			const ariaLabel = el.getAttribute('aria-label');
			if (ariaLabel) return ariaLabel;
			const labelledBy = el.getAttribute('aria-labelledby');
			if (labelledBy) {
				const parts = labelledBy.split(/\s+/).map(id => {
					const ref = document.getElementById(id);
					return ref ? (ref.textContent || '').trim() : '';
				}).filter(Boolean);
				if (parts.length) return parts.join(' ');
			}
			if (el.id) {
				const assocLabel = document.querySelector('label[for="' + el.id + '"]');
				if (assocLabel) return (assocLabel.textContent || '').trim();
			}
			const ph = el.getAttribute('placeholder');
			if (ph) return ph;
			const altAttr = el.getAttribute('alt');
			if (altAttr) return altAttr;
			const titleAttr = el.getAttribute('title');
			if (titleAttr) return titleAttr;
			return (el.textContent || '').trim();
		}

		let el = null;

		if (role) {
			// Role-based matching: walk all elements, filter by role + other criteria
			const roleLower = role.toLowerCase();
			const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_ELEMENT);
			const found = [];
			let node;
			while (node = walker.nextNode()) {
				if (getImplicitRole(node) !== roleLower) continue;
				// Apply additional filters
				if (text && !(node.textContent || '').trim().includes(text)) continue;
				if (label) {
					const elName = getName(node);
					if (!elName.includes(label)) continue;
				}
				if (placeholder) {
					const ph = node.getAttribute('placeholder');
					if (!ph || !ph.includes(placeholder)) continue;
				}
				if (testid) {
					const tid = node.getAttribute('data-testid');
					if (tid !== testid) continue;
				}
				if (alt) {
					const a = node.getAttribute('alt');
					if (!a || !a.includes(alt)) continue;
				}
				if (title) {
					const t = node.getAttribute('title');
					if (!t || !t.includes(title)) continue;
				}
				found.push(node);
			}
			if (found.length === 0) return null;
			// Pick best: prefer shortest text match if text filter is used
			el = found[0];
			if (text && found.length > 1) {
				let bestLen = (el.textContent || '').length;
				for (let i = 1; i < found.length; i++) {
					const len = (found[i].textContent || '').length;
					if (len < bestLen) { el = found[i]; bestLen = len; }
				}
			}
		} else if (xpath) {
			const xresult = document.evaluate(xpath, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null);
			el = xresult.singleNodeValue;
		} else if (testid) {
			el = document.querySelector('[data-testid="' + testid.replace(/"/g, '\\"') + '"]');
		} else if (placeholder) {
			el = document.querySelector('[placeholder="' + placeholder.replace(/"/g, '\\"') + '"]');
		} else if (alt) {
			el = document.querySelector('[alt="' + alt.replace(/"/g, '\\"') + '"]');
		} else if (title) {
			el = document.querySelector('[title="' + title.replace(/"/g, '\\"') + '"]');
		} else if (label) {
			// Try <label> with for= attribute pointing to an input
			const labels = document.querySelectorAll('label');
			for (const lbl of labels) {
				if (lbl.textContent.trim().includes(label)) {
					if (lbl.htmlFor) {
						el = document.getElementById(lbl.htmlFor);
					} else {
						el = lbl.querySelector('input, textarea, select');
					}
					if (el) break;
				}
			}
			// Fallback: aria-label
			if (!el) {
				el = document.querySelector('[aria-label="' + label.replace(/"/g, '\\"') + '"]');
			}
			// Fallback: aria-labelledby
			if (!el) {
				const all = document.querySelectorAll('[aria-labelledby]');
				for (const candidate of all) {
					const labelId = candidate.getAttribute('aria-labelledby');
					const labelEl = document.getElementById(labelId);
					if (labelEl && labelEl.textContent.trim().includes(label)) {
						el = candidate;
						break;
					}
				}
			}
		} else if (text) {
			// Find leaf elements containing the text
			const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_ELEMENT, {
				acceptNode: (node) => {
					if (node.offsetWidth === 0 && node.offsetHeight === 0) return NodeFilter.FILTER_REJECT;
					const style = window.getComputedStyle(node);
					if (style.display === 'none' || style.visibility === 'hidden') return NodeFilter.FILTER_REJECT;
					return NodeFilter.FILTER_ACCEPT;
				}
			});
			let best = null;
			let bestLen = Infinity;
			let node;
			while (node = walker.nextNode()) {
				const content = node.textContent.trim();
				if (content.includes(text) && content.length < bestLen) {
					// Prefer the most specific (smallest text) match
					best = node;
					bestLen = content.length;
				}
			}
			el = best;
		}

		if (!el) return null;

		if (el.scrollIntoViewIfNeeded) {
			el.scrollIntoViewIfNeeded(true);
		} else {
			el.scrollIntoView({ block: 'center', inline: 'nearest' });
		}

		const rect = el.getBoundingClientRect();
		return JSON.stringify({
			selector: getSelector(el),
			label: getLabel(el),
			tag: el.tagName.toLowerCase(),
			text: (el.textContent || '').trim().substring(0, 100),
			box: { x: Math.round(rect.x), y: Math.round(rect.y), w: Math.round(rect.width), h: Math.round(rect.height) }
		});
	}`
}

// browserEvaluate executes JavaScript code in the browser.
func (h *Handlers) browserEvaluate(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	expression, ok := args["expression"].(string)
	if !ok || expression == "" {
		return nil, fmt.Errorf("expression is required")
	}

	result, err := h.client.Evaluate("", expression)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate: %w", err)
	}

	// Format result as string
	var resultText string
	switch v := result.(type) {
	case string:
		resultText = v
	case nil:
		resultText = "null"
	default:
		b, err := json.Marshal(v)
		if err != nil {
			resultText = fmt.Sprintf("%v", v)
		} else {
			resultText = string(b)
		}
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: resultText,
		}},
	}, nil
}

// browserQuit closes the browser session.
func (h *Handlers) browserQuit(args map[string]interface{}) (*ToolsCallResult, error) {
	if h.client == nil {
		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: "No browser session to close",
			}},
		}, nil
	}

	h.Close()

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: "Browser session closed",
		}},
	}, nil
}

// browserNewPage creates a new browser page.
func (h *Handlers) browserNewPage(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	url, _ := args["url"].(string)

	s := h.newSession()
	contextID, err := api.NewPage(s, url)
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	// Activate and track the new page so subsequent commands target it
	if err := api.SwitchPage(s, contextID); err != nil {
		return nil, fmt.Errorf("failed to activate new page: %w", err)
	}
	h.activeContext = contextID

	msg := "New page opened"
	if url != "" {
		msg = fmt.Sprintf("New page opened and navigated to %s", url)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: msg,
		}},
	}, nil
}

// browserListPages lists all open browser pages.
func (h *Handlers) browserListPages(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	s := h.newSession()
	pages, err := api.ListPages(s)
	if err != nil {
		return nil, fmt.Errorf("failed to get pages: %w", err)
	}

	var text string
	for i, page := range pages {
		text += fmt.Sprintf("[%d] %s\n", i, page.URL)
	}
	if text == "" {
		text = "No pages open"
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: text,
		}},
	}, nil
}

// browserSwitchPage switches to a page by index or URL substring.
func (h *Handlers) browserSwitchPage(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	s := h.newSession()
	pages, err := api.ListPages(s)
	if err != nil {
		return nil, fmt.Errorf("failed to get pages: %w", err)
	}

	var contextID string

	// Try index first
	if idx, ok := argFloat(args, "index"); ok {
		i := int(idx)
		if i < 0 || i >= len(pages) {
			return nil, fmt.Errorf("page index %d out of range (0-%d)", i, len(pages)-1)
		}
		contextID = pages[i].Context
	} else if url, ok := args["url"].(string); ok && url != "" {
		// Search by URL substring
		for _, page := range pages {
			if strings.Contains(page.URL, url) {
				contextID = page.Context
				break
			}
		}
		if contextID == "" {
			return nil, fmt.Errorf("no page matching URL %q", url)
		}
	} else {
		return nil, fmt.Errorf("index or url is required")
	}

	if err := api.SwitchPage(s, contextID); err != nil {
		return nil, err
	}
	h.activeContext = contextID

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Switched to page: %s", contextID),
		}},
	}, nil
}

// browserClosePage closes a page by index (default: current page).
func (h *Handlers) browserClosePage(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	s := h.newSession()
	pages, err := api.ListPages(s)
	if err != nil {
		return nil, fmt.Errorf("failed to get pages: %w", err)
	}

	if len(pages) == 0 {
		return nil, fmt.Errorf("no pages open")
	}

	idx := -1
	if i, ok := argFloat(args, "index"); ok {
		idx = int(i)
	} else if h.activeContext != "" {
		// No index given — default to the active page
		for i, page := range pages {
			if page.Context == h.activeContext {
				idx = i
				break
			}
		}
	}
	if idx < 0 {
		idx = 0 // fall back to first page
	}

	if idx < 0 || idx >= len(pages) {
		return nil, fmt.Errorf("page index %d out of range (0-%d)", idx, len(pages)-1)
	}

	closedContext := pages[idx].Context
	if err := api.ClosePage(s, closedContext); err != nil {
		return nil, err
	}
	if h.activeContext == closedContext {
		h.activeContext = ""
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Closed page %d", idx),
		}},
	}, nil
}

// browserA11yTree returns the accessibility tree of the current page.
func (h *Handlers) browserA11yTree(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	interestingOnly := true
	if val, ok := args["everything"].(bool); ok {
		interestingOnly = !val
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}

	result, err := api.A11yTree(s, ctx, interestingOnly, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get accessibility tree: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: result,
		}},
	}, nil
}


// browserHover moves the mouse over an element.
func (h *Handlers) browserHover(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.Hover(s, ctx, elementParamsWithTimeout(selector, args)); err != nil {
		return nil, fmt.Errorf("failed to hover: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Hovered over element: %s", selector),
		}},
	}, nil
}

// browserSelect selects an option in a <select> element.
func (h *Handlers) browserSelect(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	value, ok := args["value"].(string)
	if !ok || value == "" {
		return nil, fmt.Errorf("value is required")
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.SelectOption(s, ctx, elementParamsWithTimeout(selector, args), value); err != nil {
		return nil, fmt.Errorf("failed to select: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Selected value %q in %s", value, selector),
		}},
	}, nil
}

// browserScroll scrolls the page or an element.
func (h *Handlers) browserScroll(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	direction := "down"
	if d, ok := args["direction"].(string); ok && d != "" {
		direction = d
	}

	amount := 3
	if a, ok := argFloat(args, "amount"); ok {
		amount = int(a)
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}

	// Determine scroll target coordinates
	x, y := 0, 0
	if selector, ok := args["selector"].(string); ok && selector != "" {
		selector = h.resolveSelector(selector)
		info, err := api.ResolveElement(s, ctx, api.ElementParams{Selector: selector})
		if err != nil {
			return nil, err
		}
		x = int(info.Box.X + info.Box.Width/2)
		y = int(info.Box.Y + info.Box.Height/2)
	} else {
		x, y = 400, 300 // Viewport center fallback
	}

	// Map direction to deltas (120 pixels per scroll "notch")
	deltaX, deltaY := 0, 0
	pixels := amount * 120
	switch direction {
	case "down":
		deltaY = pixels
	case "up":
		deltaY = -pixels
	case "right":
		deltaX = pixels
	case "left":
		deltaX = -pixels
	default:
		return nil, fmt.Errorf("invalid direction: %q (use up, down, left, right)", direction)
	}

	if err := api.ScrollWheel(s, ctx, x, y, deltaX, deltaY); err != nil {
		return nil, fmt.Errorf("failed to scroll: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Scrolled %s by %d", direction, amount),
		}},
	}, nil
}

// browserKeys presses a key or key combination.
func (h *Handlers) browserKeys(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	keys, ok := args["keys"].(string)
	if !ok || keys == "" {
		return nil, fmt.Errorf("keys is required")
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.PressKey(s, ctx, keys); err != nil {
		return nil, fmt.Errorf("failed to press keys: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Pressed keys: %s", keys),
		}},
	}, nil
}

// browserGetHTML returns the HTML content of the page or an element.
func (h *Handlers) browserGetHTML(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}

	outer, _ := args["outer"].(bool)

	var html string
	if selector, ok := args["selector"].(string); ok && selector != "" {
		selector = h.resolveSelector(selector)
		ep := api.ElementParams{Selector: selector}
		if outer {
			html, err = api.GetOuterHTML(s, ctx, ep)
		} else {
			html, err = api.GetInnerHTML(s, ctx, ep)
		}
	} else {
		html, err = api.GetContent(s, ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get HTML: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: html,
		}},
	}, nil
}

// browserFindAll finds all elements matching a CSS selector.
func (h *Handlers) browserFindAll(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	limit := 10
	if l, ok := argFloat(args, "limit"); ok {
		limit = int(l)
	}

	// Use JS to find elements and generate selectors + labels
	findAllScript := `(selector, limit) => {
		` + GetSelectorJS() + `
		` + GetLabelJS() + `
		const els = document.querySelectorAll(selector);
		const results = [];
		const n = Math.min(els.length, limit);
		for (let i = 0; i < n; i++) {
			const el = els[i];
			results.push({ selector: getSelector(el), label: getLabel(el) });
		}
		return JSON.stringify(results);
	}`
	result, err := h.client.CallFunction("", findAllScript, []interface{}{selector, limit})
	if err != nil {
		return nil, fmt.Errorf("failed to find elements: %w", err)
	}

	var elements []struct {
		Selector string `json:"selector"`
		Label    string `json:"label"`
	}
	if err := json.Unmarshal([]byte(fmt.Sprintf("%v", result)), &elements); err != nil {
		return nil, fmt.Errorf("failed to parse find-all results: %w", err)
	}

	// Build ref map and output
	h.refMap = make(map[string]string)
	var lines []string
	for i, el := range elements {
		ref := fmt.Sprintf("@e%d", i+1)
		h.refMap[ref] = el.Selector
		lines = append(lines, fmt.Sprintf("%s %s", ref, el.Label))
	}

	text := strings.Join(lines, "\n")
	if text == "" {
		text = "No elements found"
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: text,
		}},
	}, nil
}

// browserWait waits for an element to reach a specified state.
func (h *Handlers) browserWait(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	state := "attached"
	if s, ok := args["state"].(string); ok && s != "" {
		state = s
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}

	ep := api.ElementParams{
		Selector: selector,
		Timeout:  api.DefaultTimeout,
	}
	if t, ok := argFloat(args, "timeout"); ok {
		ep.Timeout = time.Duration(t) * time.Millisecond
	}

	switch state {
	case "attached":
		if _, err := api.ResolveElement(s, ctx, ep); err != nil {
			return nil, err
		}
	case "visible":
		if err := api.WaitForVisible(s, ctx, ep); err != nil {
			return nil, err
		}
	case "hidden":
		if err := api.WaitForHidden(s, ctx, ep); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid state: %q (use \"attached\", \"visible\", or \"hidden\")", state)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Element %q reached state: %s", selector, state),
		}},
	}, nil
}

// browserGetText returns the text content of the page or an element.
func (h *Handlers) browserGetText(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}

	var text string
	if selector, ok := args["selector"].(string); ok && selector != "" {
		selector = h.resolveSelector(selector)
		text, err = api.GetInnerText(s, ctx, api.ElementParams{Selector: selector})
	} else {
		text, err = api.EvalSimpleScript(s, ctx, "() => document.body.innerText")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get text: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: text,
		}},
	}, nil
}

// browserGetURL returns the current page URL.
func (h *Handlers) browserGetURL(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	url, err := api.GetURL(s, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get URL: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: url,
		}},
	}, nil
}

// browserGetTitle returns the current page title.
func (h *Handlers) browserGetTitle(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	title, err := api.GetTitle(s, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get title: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: title,
		}},
	}, nil
}

// pageClockInstall installs a fake clock on the page.
func (h *Handlers) pageClockInstall(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}

	_, err = api.EvalSimpleScript(s, ctx, api.ClockScript)
	if err != nil {
		return nil, fmt.Errorf("failed to install clock: %w", err)
	}

	if timeVal, ok := argFloat(args, "time"); ok {
		script := fmt.Sprintf("() => { window.__vibiumClock.setSystemTime(%v); return 'ok'; }", timeVal)
		if _, err := api.EvalSimpleScript(s, ctx, script); err != nil {
			return nil, fmt.Errorf("failed to set initial time: %w", err)
		}
	}

	if tz, ok := args["timezone"].(string); ok && tz != "" {
		if err := api.SetTimezone(s, ctx, tz); err != nil {
			return nil, fmt.Errorf("failed to set timezone: %w", err)
		}
	}

	return &ToolsCallResult{
		Content: []Content{{Type: "text", Text: "Clock installed"}},
	}, nil
}

// pageClockFastForward fast-forwards the fake clock.
func (h *Handlers) pageClockFastForward(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	ticks, ok := argFloat(args, "ticks")
	if !ok {
		return nil, fmt.Errorf("ticks is required")
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	script := fmt.Sprintf("() => { window.__vibiumClock.fastForward(%v); return 'ok'; }", ticks)
	if _, err := api.EvalSimpleScript(s, ctx, script); err != nil {
		return nil, fmt.Errorf("clock.fastForward failed: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("Fast-forwarded %v ms", ticks)}},
	}, nil
}

// pageClockRunFor advances the fake clock, firing all callbacks.
func (h *Handlers) pageClockRunFor(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	ticks, ok := argFloat(args, "ticks")
	if !ok {
		return nil, fmt.Errorf("ticks is required")
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	script := fmt.Sprintf("() => { window.__vibiumClock.runFor(%v); return 'ok'; }", ticks)
	if _, err := api.EvalSimpleScript(s, ctx, script); err != nil {
		return nil, fmt.Errorf("clock.runFor failed: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("Ran for %v ms", ticks)}},
	}, nil
}

// pageClockPauseAt pauses the fake clock at a specific time.
func (h *Handlers) pageClockPauseAt(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	timeVal, ok := argFloat(args, "time")
	if !ok {
		return nil, fmt.Errorf("time is required")
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	script := fmt.Sprintf("() => { window.__vibiumClock.pauseAt(%v); return 'ok'; }", timeVal)
	if _, err := api.EvalSimpleScript(s, ctx, script); err != nil {
		return nil, fmt.Errorf("clock.pauseAt failed: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("Paused at %v", timeVal)}},
	}, nil
}

// pageClockResume resumes real-time progression.
func (h *Handlers) pageClockResume(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if _, err := api.EvalSimpleScript(s, ctx, "() => { window.__vibiumClock.resume(); return 'ok'; }"); err != nil {
		return nil, fmt.Errorf("clock.resume failed: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{Type: "text", Text: "Clock resumed"}},
	}, nil
}

// pageClockSetFixedTime freezes Date.now() at a value.
func (h *Handlers) pageClockSetFixedTime(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	timeVal, ok := argFloat(args, "time")
	if !ok {
		return nil, fmt.Errorf("time is required")
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	script := fmt.Sprintf("() => { window.__vibiumClock.setFixedTime(%v); return 'ok'; }", timeVal)
	if _, err := api.EvalSimpleScript(s, ctx, script); err != nil {
		return nil, fmt.Errorf("clock.setFixedTime failed: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("Fixed time set to %v", timeVal)}},
	}, nil
}

// pageClockSetSystemTime sets Date.now() without triggering timers.
func (h *Handlers) pageClockSetSystemTime(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	timeVal, ok := argFloat(args, "time")
	if !ok {
		return nil, fmt.Errorf("time is required")
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	script := fmt.Sprintf("() => { window.__vibiumClock.setSystemTime(%v); return 'ok'; }", timeVal)
	if _, err := api.EvalSimpleScript(s, ctx, script); err != nil {
		return nil, fmt.Errorf("clock.setSystemTime failed: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("System time set to %v", timeVal)}},
	}, nil
}

// pageClockSetTimezone overrides or resets the browser timezone.
func (h *Handlers) pageClockSetTimezone(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}

	tz, _ := args["timezone"].(string)

	if tz == "" {
		if err := api.ClearTimezone(s, ctx); err != nil {
			return nil, fmt.Errorf("failed to clear timezone: %w", err)
		}
		return &ToolsCallResult{
			Content: []Content{{Type: "text", Text: "Timezone reset to system default"}},
		}, nil
	}

	if err := api.SetTimezone(s, ctx, tz); err != nil {
		return nil, fmt.Errorf("failed to set timezone: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{Type: "text", Text: fmt.Sprintf("Timezone set to %s", tz)}},
	}, nil
}


// pollCallFunction polls a JS function until it returns a non-null/non-empty result.
func pollCallFunction(h *Handlers, script string, args []interface{}, timeout time.Duration) (interface{}, error) {
	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond

	for {
		result, err := h.client.CallFunction("", script, args)
		if err == nil && result != nil {
			s := fmt.Sprintf("%v", result)
			if s != "" && s != "null" && s != "<nil>" {
				return result, nil
			}
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout after %s", timeout)
		}

		time.Sleep(interval)
	}
}

// browserFill clears an input field and types new text.
func (h *Handlers) browserFill(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	// Distinguish an omitted value from an intentionally empty one: an empty
	// string is a valid value that clears the field, so key off key presence,
	// not emptiness.
	value, hasValue := args["value"].(string)
	if !hasValue {
		// Fall back to "text" for backwards compatibility with MCP clients
		value, hasValue = args["text"].(string)
	}
	if !hasValue {
		return nil, fmt.Errorf("value is required")
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.Fill(s, ctx, elementParamsWithTimeout(selector, args), value); err != nil {
		return nil, fmt.Errorf("failed to fill: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Filled %q into %s", value, selector),
		}},
	}, nil
}

// browserPress presses a key on a specific element or the focused element.
func (h *Handlers) browserPress(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	key, ok := args["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("key is required")
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}

	// If selector given, click to focus first then press key
	if selector, ok := args["selector"].(string); ok && selector != "" {
		selector = h.resolveSelector(selector)
		if err := api.PressOn(s, ctx, api.ElementParams{Selector: selector}, key); err != nil {
			return nil, fmt.Errorf("failed to press key: %w", err)
		}
	} else {
		if err := api.PressKey(s, ctx, key); err != nil {
			return nil, fmt.Errorf("failed to press key: %w", err)
		}
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Pressed %s", key),
		}},
	}, nil
}

// browserBack navigates back in history.
func (h *Handlers) browserBack(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.GoBack(s, ctx); err != nil {
		return nil, fmt.Errorf("failed to go back: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: "Navigated back",
		}},
	}, nil
}

// browserForward navigates forward in history.
func (h *Handlers) browserForward(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.GoForward(s, ctx); err != nil {
		return nil, fmt.Errorf("failed to go forward: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: "Navigated forward",
		}},
	}, nil
}

// browserReload reloads the current page.
func (h *Handlers) browserReload(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.Reload(s, ctx, "complete"); err != nil {
		return nil, fmt.Errorf("failed to reload: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: "Page reloaded",
		}},
	}, nil
}

// browserGetValue gets the current value of a form element.
func (h *Handlers) browserGetValue(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	value, err := api.GetValue(s, ctx, api.ElementParams{Selector: selector})
	if err != nil {
		return nil, fmt.Errorf("failed to get value: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: value,
		}},
	}, nil
}

// browserGetAttribute gets an HTML attribute value from an element.
func (h *Handlers) browserGetAttribute(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	attribute, ok := args["attribute"].(string)
	if !ok || attribute == "" {
		return nil, fmt.Errorf("attribute is required")
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	value, err := api.GetAttribute(s, ctx, api.ElementParams{Selector: selector}, attribute)
	if err != nil {
		return nil, fmt.Errorf("failed to get attribute: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: value,
		}},
	}, nil
}

// browserIsVisible checks if an element is visible on the page.
func (h *Handlers) browserIsVisible(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	visible, err := api.IsVisible(s, ctx, api.ElementParams{Selector: selector})
	if err != nil {
		// Element not found or error — return false, not an error
		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: "false",
			}},
		}, nil
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("%v", visible),
		}},
	}, nil
}

// browserCheck checks a checkbox or radio button (idempotent).
func (h *Handlers) browserCheck(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	toggled, err := api.Check(s, ctx, elementParamsWithTimeout(selector, args))
	if err != nil {
		return nil, fmt.Errorf("failed to check: %w", err)
	}

	msg := fmt.Sprintf("Checked %s", selector)
	if !toggled {
		msg = fmt.Sprintf("Already checked: %s", selector)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: msg,
		}},
	}, nil
}

// browserUncheck unchecks a checkbox (idempotent).
func (h *Handlers) browserUncheck(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	toggled, err := api.Uncheck(s, ctx, elementParamsWithTimeout(selector, args))
	if err != nil {
		return nil, fmt.Errorf("failed to uncheck: %w", err)
	}

	msg := fmt.Sprintf("Unchecked %s", selector)
	if !toggled {
		msg = fmt.Sprintf("Already unchecked: %s", selector)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: msg,
		}},
	}, nil
}

// browserScrollIntoView scrolls an element into view.
func (h *Handlers) browserScrollIntoView(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.ScrollIntoView(s, ctx, elementParamsWithTimeout(selector, args)); err != nil {
		return nil, fmt.Errorf("failed to scroll into view: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Scrolled %s into view", selector),
		}},
	}, nil
}

// browserWaitForURL waits until the page URL contains a pattern.
func (h *Handlers) browserWaitForURL(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	pattern, ok := args["pattern"].(string)
	if !ok || pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}

	timeout := api.DefaultTimeout
	if t, ok := argFloat(args, "timeout"); ok {
		timeout = time.Duration(t) * time.Millisecond
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	url, err := api.WaitForURL(s, ctx, pattern, timeout)
	if err != nil {
		return nil, err
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("URL matches pattern %q: %s", pattern, url),
		}},
	}, nil
}

// browserWaitForLoad waits until document.readyState is "complete".
func (h *Handlers) browserWaitForLoad(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	timeout := api.DefaultTimeout
	if t, ok := argFloat(args, "timeout"); ok {
		timeout = time.Duration(t) * time.Millisecond
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.WaitForLoad(s, ctx, "complete", timeout); err != nil {
		return nil, err
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: "Page loaded (readyState: complete)",
		}},
	}, nil
}

// argFloat reads a numeric tool argument, accepting either a JSON number or a
// numeric string. LLMs frequently stringify numbers (e.g. {"ms":"2000"}), which
// previously failed the strict float64 type assertion (issue #85).
func argFloat(args map[string]interface{}, key string) (float64, bool) {
	switch v := args[key].(type) {
	case float64:
		return v, true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

// elementParamsWithTimeout builds ElementParams for a selector, applying a
// caller-supplied "timeout" arg (milliseconds) when present. A zero/absent
// timeout is normalized to the default by the api layer (withDefaultTimeout).
func elementParamsWithTimeout(selector string, args map[string]interface{}) api.ElementParams {
	ep := api.ElementParams{Selector: selector}
	if t, ok := argFloat(args, "timeout"); ok {
		ep.Timeout = time.Duration(t) * time.Millisecond
	}
	return ep
}

// browserSleep pauses execution for a specified number of milliseconds.
func (h *Handlers) browserSleep(args map[string]interface{}) (*ToolsCallResult, error) {
	ms, ok := argFloat(args, "ms")
	if !ok || ms <= 0 {
		return nil, fmt.Errorf("ms is required and must be positive")
	}

	// Cap at 30 seconds
	if ms > 30000 {
		ms = 30000
	}

	time.Sleep(time.Duration(ms) * time.Millisecond)

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Slept for %v ms", ms),
		}},
	}, nil
}

// ensureBrowser checks that a browser session is active.
// If no browser is running, it auto-launches one (lazy launch).
func (h *Handlers) ensureBrowser() error {
	if h.client == nil {
		_, err := h.browserLaunch(map[string]interface{}{})
		if err != nil {
			return fmt.Errorf("auto-launch failed: %w", err)
		}
	}
	return nil
}

// resolveRefsInArgs returns a copy of args with any @ref selector resolved
// to the real CSS selector, so traces show meaningful selectors.
func (h *Handlers) resolveRefsInArgs(args map[string]interface{}) map[string]interface{} {
	sel, ok := args["selector"].(string)
	if !ok || !strings.HasPrefix(sel, "@e") {
		return args
	}
	resolved := h.resolveSelector(sel)
	if resolved == sel {
		return args
	}
	cp := make(map[string]interface{}, len(args))
	for k, v := range args {
		cp[k] = v
	}
	cp["selector"] = resolved
	return cp
}

// resolveSelector resolves @ref selectors to CSS selectors from the refMap.
func (h *Handlers) resolveSelector(selector string) string {
	if strings.HasPrefix(selector, "@e") {
		if resolved, ok := h.refMap[selector]; ok {
			return resolved
		}
	}
	return selector
}

// GetSelectorJS returns the JS getSelector(el) function body that generates unique CSS selectors.
func GetSelectorJS() string {
	return `function getSelector(el) {
			if (el.id) return '#' + CSS.escape(el.id);
			const parts = [];
			let cur = el;
			while (cur && cur !== document.body && cur !== document.documentElement) {
				let seg = cur.tagName.toLowerCase();
				if (cur.id) {
					parts.unshift('#' + CSS.escape(cur.id));
					break;
				}
				const parent = cur.parentElement;
				if (parent) {
					const siblings = Array.from(parent.children).filter(c => c.tagName === cur.tagName);
					if (siblings.length > 1) {
						const idx = siblings.indexOf(cur) + 1;
						seg += ':nth-of-type(' + idx + ')';
					}
				}
				parts.unshift(seg);
				cur = parent;
			}
			if (parts.length === 0) return el.tagName.toLowerCase();
			if (!parts[0].startsWith('#')) parts.unshift('body');
			return parts.join(' > ');
		}`
}

// GetLabelJS returns the JS getLabel(el) function body that generates descriptive labels.
func GetLabelJS() string {
	return `function getLabel(el) {
			const tag = el.tagName.toLowerCase();
			const type = el.getAttribute('type');
			let desc = '[' + tag;
			if (type) desc += ' type="' + type + '"';
			desc += ']';

			const ariaLabel = el.getAttribute('aria-label');
			if (ariaLabel) return desc + ' "' + ariaLabel.substring(0, 60) + '"';

			const placeholder = el.getAttribute('placeholder');
			if (placeholder) return desc + ' placeholder="' + placeholder.substring(0, 60) + '"';

			const title = el.getAttribute('title');
			if (title) return desc + ' title="' + title.substring(0, 60) + '"';

			const text = (el.textContent || '').trim().substring(0, 60);
			if (text) return desc + ' "' + text + '"';

			const name = el.getAttribute('name');
			if (name) return desc + ' name="' + name + '"';

			const src = el.getAttribute('src');
			if (src) return desc + ' src="' + src.substring(0, 60) + '"';

			return desc;
		}`
}

// mapScript returns the JS function that maps interactive elements with refs.
// When a selector is provided, only elements within the matching subtree are returned.
func mapScript() string {
	return `(scopeSelector) => {
		` + GetSelectorJS() + `
		` + GetLabelJS() + `

		const interactive = 'a[href], button, input, textarea, select, [role="button"], [role="link"], [role="checkbox"], [role="radio"], [role="tab"], [role="menuitem"], [role="switch"], [onclick], [tabindex]:not([tabindex="-1"]), summary, details';

		const root = scopeSelector ? document.querySelector(scopeSelector) : document;
		if (!root) return JSON.stringify([]);
		const els = root.querySelectorAll(interactive);
		const results = [];
		const seen = new Set();

		for (const el of els) {
			const style = window.getComputedStyle(el);
			if (style.display === 'none' || style.visibility === 'hidden' || el.offsetWidth === 0) continue;

			const sel = getSelector(el);
			if (seen.has(sel)) continue;
			seen.add(sel);

			results.push({ selector: sel, label: getLabel(el) });
		}

		return JSON.stringify(results);
	}`
}

// browserMap maps interactive elements with @refs.
func (h *Handlers) browserMap(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	var scopeSelector interface{}
	if sel, ok := args["selector"].(string); ok && sel != "" {
		scopeSelector = sel
	}
	result, err := h.client.CallFunction("", mapScript(), []interface{}{scopeSelector})
	if err != nil {
		return nil, fmt.Errorf("failed to map elements: %w", err)
	}

	resultStr := fmt.Sprintf("%v", result)

	var elements []struct {
		Selector string `json:"selector"`
		Label    string `json:"label"`
	}
	if err := json.Unmarshal([]byte(resultStr), &elements); err != nil {
		return nil, fmt.Errorf("failed to parse map results: %w", err)
	}

	// Build ref map and output
	h.refMap = make(map[string]string)
	var lines []string
	for i, el := range elements {
		ref := fmt.Sprintf("@e%d", i+1)
		h.refMap[ref] = el.Selector
		lines = append(lines, fmt.Sprintf("%s %s", ref, el.Label))
	}

	output := strings.Join(lines, "\n")
	if output == "" {
		output = "No interactive elements found"
	}
	h.lastMap = output

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: output,
		}},
	}, nil
}

// browserDiffMap compares current page state vs last map.
func (h *Handlers) browserDiffMap(args map[string]interface{}) (*ToolsCallResult, error) {
	if h.lastMap == "" {
		return nil, fmt.Errorf("no previous map to diff against — run browser_map first")
	}

	// Get current map
	prevMap := h.lastMap
	_, err := h.browserMap(args)
	if err != nil {
		return nil, err
	}
	currentMap := h.lastMap

	// Simple line-based diff
	prevLines := strings.Split(prevMap, "\n")
	currLines := strings.Split(currentMap, "\n")

	prevSet := make(map[string]bool)
	for _, l := range prevLines {
		prevSet[l] = true
	}
	currSet := make(map[string]bool)
	for _, l := range currLines {
		currSet[l] = true
	}

	var diff []string
	for _, l := range prevLines {
		if !currSet[l] {
			diff = append(diff, "- "+l)
		}
	}
	for _, l := range currLines {
		if !prevSet[l] {
			diff = append(diff, "+ "+l)
		}
	}

	output := strings.Join(diff, "\n")
	if output == "" {
		output = "No changes detected"
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: output,
		}},
	}, nil
}

// browserPDF saves the page as PDF.
func (h *Handlers) browserPDF(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	base64Data, err := api.PrintToPDF(s, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to print PDF: %w", err)
	}

	// If filename provided, save to file
	if filename, ok := args["filename"].(string); ok && filename != "" {
		pdfData, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode PDF: %w", err)
		}
		if err := os.WriteFile(filename, pdfData, 0644); err != nil {
			return nil, fmt.Errorf("failed to save PDF: %w", err)
		}
		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("PDF saved to %s (%d bytes)", filename, len(pdfData)),
			}},
		}, nil
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: base64Data,
		}},
	}, nil
}

// browserHighlight highlights an element with a visual overlay.
func (h *Handlers) browserHighlight(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	script := `(selector) => {
		const el = document.querySelector(selector);
		if (!el) return 'not_found';
		const prev = el.style.cssText;
		el.style.outline = '3px solid red';
		el.style.outlineOffset = '2px';
		el.style.backgroundColor = 'rgba(255,0,0,0.1)';
		setTimeout(() => { el.style.cssText = prev; }, 3000);
		return 'highlighted';
	}`

	result, err := h.client.CallFunction("", script, []interface{}{selector})
	if err != nil {
		return nil, fmt.Errorf("failed to highlight: %w", err)
	}

	if fmt.Sprintf("%v", result) == "not_found" {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Highlighted %s (3 seconds)", selector),
		}},
	}, nil
}

// browserDblClick double-clicks an element.
func (h *Handlers) browserDblClick(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.DblClick(s, ctx, elementParamsWithTimeout(selector, args)); err != nil {
		return nil, fmt.Errorf("failed to double-click: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Double-clicked element: %s", selector),
		}},
	}, nil
}

// browserFocus focuses an element.
func (h *Handlers) browserFocus(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.FocusElement(s, ctx, elementParamsWithTimeout(selector, args)); err != nil {
		return nil, fmt.Errorf("failed to focus: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Focused element: %s", selector),
		}},
	}, nil
}

// browserCount counts matching elements.
func (h *Handlers) browserCount(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	count, err := api.GetCount(s, ctx, selector)
	if err != nil {
		return nil, fmt.Errorf("failed to count: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("%d", count),
		}},
	}, nil
}

// browserIsEnabled checks if an element is enabled.
func (h *Handlers) browserIsEnabled(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	enabled, err := api.IsEnabled(s, ctx, api.ElementParams{Selector: selector})
	if err != nil {
		return nil, fmt.Errorf("failed to check enabled: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("%v", enabled),
		}},
	}, nil
}

// browserIsChecked checks if an element is checked.
func (h *Handlers) browserIsChecked(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	checked, err := api.IsChecked(s, ctx, api.ElementParams{Selector: selector})
	if err != nil {
		return nil, fmt.Errorf("failed to check checked state: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("%v", checked),
		}},
	}, nil
}

// browserWaitForText waits until text appears on the page.
func (h *Handlers) browserWaitForText(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	text, ok := args["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("text is required")
	}

	timeout := api.DefaultTimeout
	if t, ok := argFloat(args, "timeout"); ok {
		timeout = time.Duration(t) * time.Millisecond
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.WaitForText(s, ctx, text, timeout); err != nil {
		return nil, err
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Text %q found on page", text),
		}},
	}, nil
}

// browserWaitForFn waits until a JS expression returns truthy.
func (h *Handlers) browserWaitForFn(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	expression, ok := args["expression"].(string)
	if !ok || expression == "" {
		return nil, fmt.Errorf("expression is required")
	}

	timeout := api.DefaultTimeout
	if t, ok := argFloat(args, "timeout"); ok {
		timeout = time.Duration(t) * time.Millisecond
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	result, err := api.WaitForFunction(s, ctx, expression, timeout)
	if err != nil {
		return nil, err
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Expression returned truthy: %s", result),
		}},
	}, nil
}

// browserDialogAccept accepts a dialog (alert, confirm, prompt).
func (h *Handlers) browserDialogAccept(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	text, _ := args["text"].(string)

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.DialogAccept(s, ctx, text); err != nil {
		return nil, fmt.Errorf("failed to accept dialog: %w", err)
	}

	msg := "Dialog accepted"
	if text != "" {
		msg = fmt.Sprintf("Dialog accepted with text: %q", text)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: msg,
		}},
	}, nil
}

// browserDialogDismiss dismisses a dialog.
func (h *Handlers) browserDialogDismiss(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.DialogDismiss(s, ctx); err != nil {
		return nil, fmt.Errorf("failed to dismiss dialog: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: "Dialog dismissed",
		}},
	}, nil
}

// browserGetCookies returns all cookies.
func (h *Handlers) browserGetCookies(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	cookies, err := api.GetCookies(s, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cookies: %w", err)
	}

	if len(cookies) == 0 {
		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: "No cookies",
			}},
		}, nil
	}

	var lines []string
	for _, c := range cookies {
		lines = append(lines, fmt.Sprintf("%s=%s (domain=%s, path=%s)", c.Name, c.Value, c.Domain, c.Path))
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: strings.Join(lines, "\n"),
		}},
	}, nil
}

// browserSetCookie sets a cookie.
func (h *Handlers) browserSetCookie(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required")
	}

	value, ok := args["value"].(string)
	if !ok {
		return nil, fmt.Errorf("value is required")
	}

	domain, _ := args["domain"].(string)
	path, _ := args["path"].(string)

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.SetCookie(s, ctx, name, value, domain, path); err != nil {
		return nil, fmt.Errorf("failed to set cookie: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Cookie set: %s=%s", name, value),
		}},
	}, nil
}

// browserDeleteCookies deletes cookies.
func (h *Handlers) browserDeleteCookies(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	name, _ := args["name"].(string)

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.DeleteCookies(s, ctx, name); err != nil {
		return nil, fmt.Errorf("failed to delete cookies: %w", err)
	}

	msg := "All cookies deleted"
	if name != "" {
		msg = fmt.Sprintf("Cookie %q deleted", name)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: msg,
		}},
	}, nil
}

// browserMouseMove moves the mouse to coordinates.
func (h *Handlers) browserMouseMove(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	x, ok := argFloat(args, "x")
	if !ok {
		return nil, fmt.Errorf("x is required")
	}
	y, ok := argFloat(args, "y")
	if !ok {
		return nil, fmt.Errorf("y is required")
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.MouseMove(s, ctx, int(x), int(y)); err != nil {
		return nil, fmt.Errorf("failed to move mouse: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Mouse moved to (%d, %d)", int(x), int(y)),
		}},
	}, nil
}

// browserMouseDown presses a mouse button.
func (h *Handlers) browserMouseDown(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	button := 0
	if b, ok := argFloat(args, "button"); ok {
		button = int(b)
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.MouseDown(s, ctx, button); err != nil {
		return nil, fmt.Errorf("failed to press mouse button: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Mouse button %d pressed", button),
		}},
	}, nil
}

// browserMouseUp releases a mouse button.
func (h *Handlers) browserMouseUp(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	button := 0
	if b, ok := argFloat(args, "button"); ok {
		button = int(b)
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.MouseUp(s, ctx, button); err != nil {
		return nil, fmt.Errorf("failed to release mouse button: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Mouse button %d released", button),
		}},
	}, nil
}

// browserMouseClick clicks at coordinates or at the current position.
func (h *Handlers) browserMouseClick(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	button := 0
	if b, ok := argFloat(args, "button"); ok {
		button = int(b)
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}

	x, hasX := argFloat(args, "x")
	y, hasY := argFloat(args, "y")
	if hasX && hasY {
		if err := api.MouseClick(s, ctx, int(x), int(y), button); err != nil {
			return nil, fmt.Errorf("failed to click: %w", err)
		}
	} else {
		// Click at current position (down+up only)
		if err := api.MouseDown(s, ctx, button); err != nil {
			return nil, fmt.Errorf("failed to click: %w", err)
		}
		if err := api.MouseUp(s, ctx, button); err != nil {
			return nil, fmt.Errorf("failed to click: %w", err)
		}
	}

	msg := "Clicked at current position"
	if hasX && hasY {
		msg = fmt.Sprintf("Clicked at (%d, %d)", int(x), int(y))
	}
	if button != 0 {
		msg += fmt.Sprintf(" with button %d", button)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: msg,
		}},
	}, nil
}

// browserDrag drags from one element to another.
func (h *Handlers) browserDrag(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	source, ok := args["source"].(string)
	if !ok || source == "" {
		return nil, fmt.Errorf("source selector is required")
	}
	source = h.resolveSelector(source)

	target, ok := args["target"].(string)
	if !ok || target == "" {
		return nil, fmt.Errorf("target selector is required")
	}
	target = h.resolveSelector(target)

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.DragTo(s, ctx, elementParamsWithTimeout(source, args), elementParamsWithTimeout(target, args)); err != nil {
		return nil, fmt.Errorf("failed to drag: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Dragged %q to %q", source, target),
		}},
	}, nil
}

// browserSetViewport sets the viewport size.
func (h *Handlers) browserSetViewport(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	width, ok := argFloat(args, "width")
	if !ok {
		return nil, fmt.Errorf("width is required")
	}
	height, ok := argFloat(args, "height")
	if !ok {
		return nil, fmt.Errorf("height is required")
	}

	dpr := 0.0
	if d, ok := argFloat(args, "devicePixelRatio"); ok {
		dpr = d
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.SetViewport(s, ctx, int(width), int(height), dpr); err != nil {
		return nil, fmt.Errorf("failed to set viewport: %w", err)
	}

	msg := fmt.Sprintf("Viewport set to %dx%d", int(width), int(height))
	if dpr > 0 {
		msg += fmt.Sprintf(" (DPR: %.1f)", dpr)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: msg,
		}},
	}, nil
}

// browserGetViewport returns the current viewport dimensions.
func (h *Handlers) browserGetViewport(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	result, err := api.EvalSimpleScript(s, ctx, "() => JSON.stringify({width: window.innerWidth, height: window.innerHeight, devicePixelRatio: window.devicePixelRatio})")
	if err != nil {
		return nil, fmt.Errorf("failed to get viewport: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: result,
		}},
	}, nil
}

// browserGetWindow returns the OS browser window state and dimensions.
func (h *Handlers) browserGetWindow(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	s := h.newSession()
	win, err := api.GetWindow(s)
	if err != nil {
		return nil, err
	}

	jsonBytes, _ := json.Marshal(win)
	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: string(jsonBytes),
		}},
	}, nil
}

// browserSetWindow sets the OS browser window size, position, or state.
func (h *Handlers) browserSetWindow(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	if h.launchResult == nil {
		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: "Not supported for remote browsers",
			}},
			IsError: true,
		}, nil
	}

	state, _ := args["state"].(string)
	width, hasWidth := argFloat(args, "width")
	height, hasHeight := argFloat(args, "height")
	x, hasX := argFloat(args, "x")
	y, hasY := argFloat(args, "y")

	opts := api.SetWindowOpts{State: state}
	if hasWidth {
		w := int(width)
		opts.Width = &w
	}
	if hasHeight {
		h := int(height)
		opts.Height = &h
	}
	if hasX {
		xv := int(x)
		opts.X = &xv
	}
	if hasY {
		yv := int(y)
		opts.Y = &yv
	}

	if err := api.SetWindow(h.launchResult.Port, h.launchResult.SessionID, opts); err != nil {
		return nil, err
	}

	msg := "Window updated"
	if state != "" && state != "normal" {
		msg = fmt.Sprintf("Window state set to %s", state)
	} else if hasWidth && hasHeight {
		msg = fmt.Sprintf("Window set to %dx%d", int(width), int(height))
		if hasX && hasY {
			msg += fmt.Sprintf(" at (%d, %d)", int(x), int(y))
		}
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: msg,
		}},
	}, nil
}

// browserEmulateMedia overrides CSS media features.
func (h *Handlers) browserEmulateMedia(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	overrides := map[string]interface{}{}
	for _, key := range []string{"media", "colorScheme", "reducedMotion", "forcedColors", "contrast"} {
		if v, ok := args[key].(string); ok && v != "" {
			overrides[key] = v
		}
	}
	if len(overrides) == 0 {
		return nil, fmt.Errorf("at least one media feature override is required")
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}

	if err := api.EmulateMedia(s, ctx, overrides); err != nil {
		return nil, fmt.Errorf("failed to emulate media: %w", err)
	}

	keys := make([]string, 0, len(overrides))
	for k := range overrides {
		keys = append(keys, k)
	}
	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Media emulation applied: %v", keys),
		}},
	}, nil
}

// browserSetGeolocation overrides the browser geolocation.
func (h *Handlers) browserSetGeolocation(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	latitude, ok := argFloat(args, "latitude")
	if !ok {
		return nil, fmt.Errorf("latitude is required")
	}
	longitude, ok := argFloat(args, "longitude")
	if !ok {
		return nil, fmt.Errorf("longitude is required")
	}

	accuracy := 1.0
	if a, ok := argFloat(args, "accuracy"); ok {
		accuracy = a
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}

	if err := api.SetGeolocation(s, ctx, latitude, longitude, accuracy); err != nil {
		return nil, fmt.Errorf("failed to set geolocation: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Geolocation set to (%f, %f)", latitude, longitude),
		}},
	}, nil
}

// browserSetContent replaces the page HTML content.
func (h *Handlers) browserSetContent(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	html, ok := args["html"].(string)
	if !ok || html == "" {
		return nil, fmt.Errorf("html is required")
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.SetContent(s, ctx, html); err != nil {
		return nil, fmt.Errorf("failed to set content: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Page content set (%d chars)", len(html)),
		}},
	}, nil
}

// browserFrames lists all child frames (iframes) on the page.
func (h *Handlers) browserFrames(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}

	frames, err := api.ListFrames(s, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get frames: %w", err)
	}

	if len(frames) == 0 {
		return &ToolsCallResult{
			Content: []Content{{
				Type: "text",
				Text: "No frames found",
			}},
		}, nil
	}

	framesJSON, _ := json.Marshal(frames)
	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: string(framesJSON),
		}},
	}, nil
}

// browserFrame finds a frame by name or URL substring.
func (h *Handlers) browserFrame(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	nameOrURL, ok := args["nameOrUrl"].(string)
	if !ok || nameOrURL == "" {
		return nil, fmt.Errorf("nameOrUrl is required")
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}

	frame, err := api.FindFrame(s, ctx, nameOrURL)
	if err != nil {
		return nil, fmt.Errorf("failed to find frame: %w", err)
	}
	if frame == nil {
		return nil, fmt.Errorf("no frame matching %q", nameOrURL)
	}

	result, _ := json.Marshal(frame)
	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: string(result),
		}},
	}, nil
}

// browserUpload sets files on an input[type=file] element.
func (h *Handlers) browserUpload(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	selector, ok := args["selector"].(string)
	if !ok || selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	selector = h.resolveSelector(selector)

	filesRaw, ok := args["files"]
	if !ok {
		return nil, fmt.Errorf("files is required")
	}

	var files []string
	switch v := filesRaw.(type) {
	case []interface{}:
		for _, f := range v {
			if s, ok := f.(string); ok {
				files = append(files, s)
			}
		}
	default:
		return nil, fmt.Errorf("files must be an array of strings")
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("at least one file path is required")
	}

	s := h.newSession()
	ctx, err := s.GetContextID()
	if err != nil {
		return nil, err
	}
	if err := api.Upload(s, ctx, elementParamsWithTimeout(selector, args), files); err != nil {
		return nil, fmt.Errorf("failed to set files: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Set %d file(s) on %s", len(files), selector),
		}},
	}, nil
}

// browserRecordStart starts recording.
func (h *Handlers) browserRecordStart(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	if h.recorder != nil {
		return nil, fmt.Errorf("already recording — stop it first")
	}

	opts := api.ParseRecordingOptions(args)
	if opts.Name == "" {
		opts.Name = "record"
	}
	name := opts.Name

	h.recorder = api.NewRecorder()
	viewport := h.queryViewport()
	h.recorder.Start(opts, viewport)

	// Subscribe to events and feed them to the recorder
	h.client.SendCommand("session.subscribe", map[string]interface{}{
		"events": []string{
			"network.beforeRequestSent",
			"network.responseCompleted",
			"network.fetchError",
			"log.entryAdded",
			"browsingContext.userPromptOpened",
			"browsingContext.downloadWillBegin",
			"browsingContext.load",
			"browsingContext.fragmentNavigated",
		},
	})
	h.client.SetEventHandler(func(msg string) {
		h.recorder.RecordBidiEvent(msg)
	})

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Recording %q started (screenshots: %v, snapshots: %v)", name, opts.Screenshots, opts.Snapshots),
		}},
	}, nil
}

// browserRecordStop stops recording and saves to a ZIP file.
func (h *Handlers) browserRecordStop(args map[string]interface{}) (*ToolsCallResult, error) {
	if h.recorder == nil {
		return nil, fmt.Errorf("no recording in progress")
	}

	// Stop forwarding events to the recorder
	if h.client != nil {
		h.client.SetEventHandler(nil)
	}

	// Stop screenshot goroutine before stopping the recorder
	h.recorder.StopScreenshots()

	path, _ := args["path"].(string)
	if path == "" {
		path = "record.zip"
	}

	zipData, err := h.recorder.Stop()
	if err != nil {
		h.recorder = nil
		return nil, fmt.Errorf("failed to stop recording: %w", err)
	}

	if err := api.WriteRecordToFile(zipData, path); err != nil {
		h.recorder = nil
		return nil, fmt.Errorf("failed to write recording: %w", err)
	}

	h.recorder = nil

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Recording saved to %s", path),
		}},
	}, nil
}

// browserRecordStartGroup starts a named group in the recording.
func (h *Handlers) browserRecordStartGroup(args map[string]interface{}) (*ToolsCallResult, error) {
	if h.recorder == nil {
		return nil, fmt.Errorf("no recording in progress")
	}

	name, _ := args["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	h.recorder.StartGroup(name)

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Started group %q", name),
		}},
	}, nil
}

// browserRecordStopGroup ends the current group in the recording.
func (h *Handlers) browserRecordStopGroup(args map[string]interface{}) (*ToolsCallResult, error) {
	if h.recorder == nil {
		return nil, fmt.Errorf("no recording in progress")
	}

	h.recorder.StopGroup()

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: "Stopped group",
		}},
	}, nil
}

// browserRecordStartChunk starts a new chunk within the current recording.
func (h *Handlers) browserRecordStartChunk(args map[string]interface{}) (*ToolsCallResult, error) {
	if h.recorder == nil {
		return nil, fmt.Errorf("no recording in progress")
	}

	name, _ := args["name"].(string)
	title, _ := args["title"].(string)

	viewport := h.queryViewport()
	h.recorder.StartChunk(name, title, viewport)

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: "Started new recording chunk",
		}},
	}, nil
}

// browserRecordStopChunk packages the current chunk into a zip file.
// Recording remains active for additional chunks.
func (h *Handlers) browserRecordStopChunk(args map[string]interface{}) (*ToolsCallResult, error) {
	if h.recorder == nil {
		return nil, fmt.Errorf("no recording in progress")
	}

	path, _ := args["path"].(string)
	if path == "" {
		path = "chunk.zip"
	}

	zipData, err := h.recorder.StopChunk()
	if err != nil {
		return nil, fmt.Errorf("failed to stop chunk: %w", err)
	}

	if err := api.WriteRecordToFile(zipData, path); err != nil {
		return nil, fmt.Errorf("failed to write chunk: %w", err)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Chunk saved to %s", path),
		}},
	}, nil
}

// browserStorageState exports cookies, localStorage, and sessionStorage.
func (h *Handlers) browserStorageState(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	// Get cookies
	cookies, err := h.client.GetCookies("")
	if err != nil {
		return nil, fmt.Errorf("failed to get cookies: %w", err)
	}

	// Get localStorage and sessionStorage
	script := `JSON.stringify({
		origin: location.origin,
		localStorage: (function() {
			var ls = {};
			for (var i = 0; i < localStorage.length; i++) {
				var key = localStorage.key(i);
				ls[key] = localStorage.getItem(key);
			}
			return ls;
		})(),
		sessionStorage: (function() {
			var ss = {};
			for (var i = 0; i < sessionStorage.length; i++) {
				var key = sessionStorage.key(i);
				ss[key] = sessionStorage.getItem(key);
			}
			return ss;
		})()
	})`

	storageResult, err := h.client.Evaluate("", script)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage: %w", err)
	}

	// Build combined state
	state := map[string]interface{}{
		"cookies": cookies,
		"storage": storageResult,
	}

	stateJSON, _ := json.MarshalIndent(state, "", "  ")
	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: string(stateJSON),
		}},
	}, nil
}

// browserRestoreStorage restores cookies and storage from a JSON state.
func (h *Handlers) browserRestoreStorage(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	path, ok := args["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("path is required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state struct {
		Cookies []bidi.Cookie `json:"cookies"`
		Storage json.RawMessage `json:"storage"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	// Restore cookies
	for _, cookie := range state.Cookies {
		if err := h.client.SetCookie("", cookie); err != nil {
			log.Debug("failed to restore cookie", "name", cookie.Name, "error", err)
		}
	}

	// Restore localStorage/sessionStorage if present
	if len(state.Storage) > 0 {
		script := fmt.Sprintf(`(function() {
			var state = %s;
			if (state.localStorage) {
				for (var key in state.localStorage) {
					localStorage.setItem(key, state.localStorage[key]);
				}
			}
			if (state.sessionStorage) {
				for (var key in state.sessionStorage) {
					sessionStorage.setItem(key, state.sessionStorage[key]);
				}
			}
			return 'ok';
		})()`, string(state.Storage))
		h.client.Evaluate("", script)
	}

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Storage state restored from %s (%d cookies)", path, len(state.Cookies)),
		}},
	}, nil
}

// browserDownloadSetDir sets the download directory.
func (h *Handlers) browserDownloadSetDir(args map[string]interface{}) (*ToolsCallResult, error) {
	if err := h.ensureBrowser(); err != nil {
		return nil, err
	}

	dir, ok := args["path"].(string)
	if !ok || dir == "" {
		return nil, fmt.Errorf("path is required")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create download directory: %w", err)
	}

	// Make absolute
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Get the connection to set download behavior
	params := map[string]interface{}{
		"downloadBehavior": map[string]interface{}{
			"type":              "allowed",
			"destinationFolder": absDir,
		},
	}

	if _, err := h.client.SendCommand("browser.setDownloadBehavior", params); err != nil {
		return nil, fmt.Errorf("failed to set download directory: %w", err)
	}

	h.downloadDir = absDir

	return &ToolsCallResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Download directory set to %s", absDir),
		}},
	}, nil
}
