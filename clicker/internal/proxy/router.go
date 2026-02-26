package proxy

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vibium/clicker/internal/bidi"
	"github.com/vibium/clicker/internal/browser"
)

// Default timeout for actionability checks
const defaultTimeout = 30 * time.Second

// BrowserSession represents a browser session connected to a client.
type BrowserSession struct {
	LaunchResult *browser.LaunchResult
	BidiConn     *bidi.Connection
	BidiClient   *bidi.Client
	Client       *ClientConn
	mu           sync.Mutex
	closed       bool
	stopChan     chan struct{}

	// Internal command tracking for vibium: extension commands
	internalCmds   map[int]chan json.RawMessage // id -> response channel
	internalCmdsMu sync.Mutex
	nextInternalID int

	// WebSocket monitoring state
	wsPreloadScriptID string // "" if not installed
	wsSubscribed      bool   // whether script.message is subscribed

	// Download support
	downloadDir string // temp dir for downloads, cleaned up on close

	// Clock support
	clockPreloadScriptID string // "" if not installed

	// Tracing support
	traceRecorder      *TraceRecorder
	lastContext        string // last browsing context resolved by a command
	lastURL            string // last known page URL, updated from load/navigation events
	screenshotInFlight int32  // atomic; 1 = screenshot capture in progress
}

// BiDi command structure for parsing incoming messages
type bidiCommand struct {
	ID     int                    `json:"id"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

// BiDi response structure for sending responses (follows WebDriver BiDi spec)
type bidiResponse struct {
	ID      int         `json:"id"`
	Type    string      `json:"type"` // "success" or "error"
	Result  interface{} `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// Router manages browser sessions for connected clients.
type Router struct {
	sessions sync.Map // map[uint64]*BrowserSession (client ID -> session)
	headless bool
}

// NewRouter creates a new router.
func NewRouter(headless bool) *Router {
	return &Router{
		headless: headless,
	}
}

// OnClientConnect is called when a new client connects.
// It launches a browser and establishes a BiDi connection.
func (r *Router) OnClientConnect(client *ClientConn) {
	fmt.Printf("[router] Launching browser for client %d...\n", client.ID)

	// Launch browser
	launchResult, err := browser.Launch(browser.LaunchOptions{
		Headless: r.headless,
	})
	if err != nil {
		fmt.Printf("[router] Failed to launch browser for client %d: %v\n", client.ID, err)
		client.Send(fmt.Sprintf(`{"error":{"code":-32000,"message":"Failed to launch browser: %s"}}`, err.Error()))
		client.Close()
		return
	}

	fmt.Printf("[router] Browser launched for client %d, WebSocket: %s\n", client.ID, launchResult.WebSocketURL)

	// Connect to browser BiDi WebSocket
	bidiConn, err := bidi.Connect(launchResult.WebSocketURL)
	if err != nil {
		fmt.Printf("[router] Failed to connect to browser BiDi for client %d: %v\n", client.ID, err)
		launchResult.Close()
		client.Send(fmt.Sprintf(`{"error":{"code":-32000,"message":"Failed to connect to browser: %s"}}`, err.Error()))
		client.Close()
		return
	}

	fmt.Printf("[router] BiDi connection established for client %d\n", client.ID)

	// Create a BiDi client for handling custom commands
	bidiClient := bidi.NewClient(bidiConn)

	session := &BrowserSession{
		LaunchResult:   launchResult,
		BidiConn:       bidiConn,
		BidiClient:     bidiClient,
		Client:         client,
		stopChan:       make(chan struct{}),
		internalCmds:   make(map[int]chan json.RawMessage),
		nextInternalID: 1000000, // Start at high number to avoid collision with client IDs
	}

	r.sessions.Store(client.ID, session)

	// Start routing messages from browser to client
	go r.routeBrowserToClient(session)

	// Subscribe to events for onPage/onPopup, network interception, dialog handling, and downloads.
	// Events are forwarded to the JS client by routeBrowserToClient.
	go func() {
		r.sendInternalCommand(session, "session.subscribe", map[string]interface{}{
			"events": []string{
				"browsingContext.contextCreated",
				"network.beforeRequestSent",
				"network.responseCompleted",
				"browsingContext.userPromptOpened",
				"log.entryAdded",
				"browsingContext.downloadWillBegin",
				"browsingContext.downloadEnd",
				"browsingContext.load",
				"browsingContext.fragmentNavigated",
			},
		})
		r.setupDownloads(session)
	}()
}

// vibiumHandler is the signature for vibium: extension command handlers.
type vibiumHandler func(*BrowserSession, bidiCommand)

// isClickLikeAction returns true for actions where the before-state is most
// meaningful (e.g. seeing the element about to be clicked). Only a before-
// snapshot is captured for these. All other actions capture an after-snapshot
// to show the result (e.g. filled text, navigated page).
func isClickLikeAction(method string) bool {
	switch method {
	case "vibium:click", "vibium:dblclick", "vibium:hover", "vibium:tap",
		"vibium:check", "vibium:uncheck", "vibium:focus",
		"vibium:scrollIntoView", "vibium:dragTo", "vibium:dispatchEvent",
		"vibium:mouse.click", "vibium:mouse.move", "vibium:mouse.down", "vibium:mouse.up",
		"vibium:touch.tap":
		return true
	}
	return false
}

// dispatch wraps a vibium handler with automatic action tracing.
func (r *Router) dispatch(session *BrowserSession, cmd bidiCommand, handler vibiumHandler) {
	go func() {
		session.mu.Lock()
		recorder := session.traceRecorder
		session.mu.Unlock()

		var callId string
		clickLike := isClickLikeAction(cmd.Method)

		if recorder != nil && recorder.IsRecording() {
			callId = recorder.NextCallId()
			opts := recorder.Options()

			// Click-like actions capture a before-snapshot (what's about to be clicked).
			// Fill-like actions skip the before-snapshot and capture after instead.
			var beforeSnapshot string
			if opts.Snapshots && clickLike {
				beforeSnapshot = r.captureActionSnapshot(session, recorder, cmd.Params, callId, "before")
			}

			session.mu.Lock()
			pageId := session.lastContext
			session.mu.Unlock()

			recorder.RecordAction(callId, cmd.Method, cmd.Params, beforeSnapshot, pageId)
		}

		handler(session, cmd)

		// Capture endTime immediately after handler returns, before screenshot captures
		endTime := time.Now()

		if recorder != nil && recorder.IsRecording() {
			opts := recorder.Options()

			// Fill-like actions capture an after-snapshot (the result of the action).
			// Click-like actions already captured their snapshot before the handler.
			var afterSnapshot string
			if opts.Snapshots && !clickLike {
				afterSnapshot = r.captureActionSnapshot(session, recorder, cmd.Params, callId, "after")
			}

			// Filmstrip screenshot (CAS-guarded to avoid flooding Chrome)
			if opts.Screenshots && atomic.CompareAndSwapInt32(&session.screenshotInFlight, 0, 1) {
				r.capturePostActionScreenshot(session, recorder, cmd.Params)
				atomic.StoreInt32(&session.screenshotInFlight, 0)
			}

			recorder.RecordActionEnd(callId, afterSnapshot, endTime)
		}
	}()
}

// OnClientMessage is called when a message is received from a client.
// It handles custom vibium: extension commands or forwards to the browser.
func (r *Router) OnClientMessage(client *ClientConn, msg string) {
	sessionVal, ok := r.sessions.Load(client.ID)
	if !ok {
		fmt.Printf("[router] No session for client %d\n", client.ID)
		return
	}

	session := sessionVal.(*BrowserSession)

	session.mu.Lock()
	if session.closed {
		session.mu.Unlock()
		return
	}
	session.mu.Unlock()

	// Parse the command to check for custom vibium: extension methods
	var cmd bidiCommand
	if err := json.Unmarshal([]byte(msg), &cmd); err != nil {
		// Can't parse, forward as-is
		if err := session.BidiConn.Send(msg); err != nil {
			fmt.Printf("[router] Failed to send to browser for client %d: %v\n", client.ID, err)
		}
		return
	}

	// Handle vibium: extension commands (per WebDriver BiDi spec for extensions)
	switch cmd.Method {
	// Element interaction commands
	case "vibium:click":
		r.dispatch(session, cmd, r.handleVibiumClick)
		return
	case "vibium:dblclick":
		r.dispatch(session, cmd, r.handleVibiumDblclick)
		return
	case "vibium:fill":
		r.dispatch(session, cmd, r.handleVibiumFill)
		return
	case "vibium:type":
		r.dispatch(session, cmd, r.handleVibiumType)
		return
	case "vibium:press":
		r.dispatch(session, cmd, r.handleVibiumPress)
		return
	case "vibium:clear":
		r.dispatch(session, cmd, r.handleVibiumClear)
		return
	case "vibium:check":
		r.dispatch(session, cmd, r.handleVibiumCheck)
		return
	case "vibium:uncheck":
		r.dispatch(session, cmd, r.handleVibiumUncheck)
		return
	case "vibium:selectOption":
		r.dispatch(session, cmd, r.handleVibiumSelectOption)
		return
	case "vibium:hover":
		r.dispatch(session, cmd, r.handleVibiumHover)
		return
	case "vibium:focus":
		r.dispatch(session, cmd, r.handleVibiumFocus)
		return
	case "vibium:dragTo":
		r.dispatch(session, cmd, r.handleVibiumDragTo)
		return
	case "vibium:tap":
		r.dispatch(session, cmd, r.handleVibiumTap)
		return
	case "vibium:scrollIntoView":
		r.dispatch(session, cmd, r.handleVibiumScrollIntoView)
		return
	case "vibium:dispatchEvent":
		r.dispatch(session, cmd, r.handleVibiumDispatchEvent)
		return

	// Element finding commands
	case "vibium:find":
		r.dispatch(session, cmd, r.handleVibiumFind)
		return
	case "vibium:findAll":
		r.dispatch(session, cmd, r.handleVibiumFindAll)
		return

	// Element state commands
	case "vibium:el.text":
		r.dispatch(session, cmd, r.handleVibiumElText)
		return
	case "vibium:el.innerText":
		r.dispatch(session, cmd, r.handleVibiumElInnerText)
		return
	case "vibium:el.html":
		r.dispatch(session, cmd, r.handleVibiumElHTML)
		return
	case "vibium:el.value":
		r.dispatch(session, cmd, r.handleVibiumElValue)
		return
	case "vibium:el.attr":
		r.dispatch(session, cmd, r.handleVibiumElAttr)
		return
	case "vibium:el.bounds":
		r.dispatch(session, cmd, r.handleVibiumElBounds)
		return
	case "vibium:el.isVisible":
		r.dispatch(session, cmd, r.handleVibiumElIsVisible)
		return
	case "vibium:el.isHidden":
		r.dispatch(session, cmd, r.handleVibiumElIsHidden)
		return
	case "vibium:el.isEnabled":
		r.dispatch(session, cmd, r.handleVibiumElIsEnabled)
		return
	case "vibium:el.isChecked":
		r.dispatch(session, cmd, r.handleVibiumElIsChecked)
		return
	case "vibium:el.isEditable":
		r.dispatch(session, cmd, r.handleVibiumElIsEditable)
		return
	case "vibium:el.eval":
		r.dispatch(session, cmd, r.handleVibiumElEval)
		return
	case "vibium:el.screenshot":
		r.dispatch(session, cmd, r.handleVibiumElScreenshot)
		return
	case "vibium:el.waitFor":
		r.dispatch(session, cmd, r.handleVibiumElWaitFor)
		return

	// Page-level input commands
	case "vibium:keyboard.press":
		r.dispatch(session, cmd, r.handleKeyboardPress)
		return
	case "vibium:keyboard.down":
		r.dispatch(session, cmd, r.handleKeyboardDown)
		return
	case "vibium:keyboard.up":
		r.dispatch(session, cmd, r.handleKeyboardUp)
		return
	case "vibium:keyboard.type":
		r.dispatch(session, cmd, r.handleKeyboardType)
		return
	case "vibium:mouse.click":
		r.dispatch(session, cmd, r.handleMouseClick)
		return
	case "vibium:mouse.move":
		r.dispatch(session, cmd, r.handleMouseMove)
		return
	case "vibium:mouse.down":
		r.dispatch(session, cmd, r.handleMouseDown)
		return
	case "vibium:mouse.up":
		r.dispatch(session, cmd, r.handleMouseUp)
		return
	case "vibium:mouse.wheel":
		r.dispatch(session, cmd, r.handleMouseWheel)
		return
	case "vibium:touch.tap":
		r.dispatch(session, cmd, r.handleTouchTap)
		return

	// Page-level capture commands
	case "vibium:page.screenshot":
		r.dispatch(session, cmd, r.handlePageScreenshot)
		return
	case "vibium:page.pdf":
		r.dispatch(session, cmd, r.handlePagePDF)
		return

	// Page-level evaluation commands
	case "vibium:page.eval":
		r.dispatch(session, cmd, r.handlePageEval)
		return
	case "vibium:page.evalHandle":
		r.dispatch(session, cmd, r.handlePageEvalHandle)
		return
	case "vibium:page.addScript":
		r.dispatch(session, cmd, r.handlePageAddScript)
		return
	case "vibium:page.addStyle":
		r.dispatch(session, cmd, r.handlePageAddStyle)
		return
	case "vibium:page.expose":
		r.dispatch(session, cmd, r.handlePageExpose)
		return

	// Page-level waiting commands
	case "vibium:page.waitFor":
		r.dispatch(session, cmd, r.handlePageWaitFor)
		return
	case "vibium:page.wait":
		r.dispatch(session, cmd, r.handlePageWait)
		return
	case "vibium:page.waitForFunction":
		r.dispatch(session, cmd, r.handlePageWaitForFunction)
		return

	// Navigation commands
	case "vibium:page.navigate":
		r.dispatch(session, cmd, r.handlePageNavigate)
		return
	case "vibium:page.back":
		r.dispatch(session, cmd, r.handlePageBack)
		return
	case "vibium:page.forward":
		r.dispatch(session, cmd, r.handlePageForward)
		return
	case "vibium:page.reload":
		r.dispatch(session, cmd, r.handlePageReload)
		return
	case "vibium:page.url":
		r.dispatch(session, cmd, r.handlePageURL)
		return
	case "vibium:page.title":
		r.dispatch(session, cmd, r.handlePageTitle)
		return
	case "vibium:page.content":
		r.dispatch(session, cmd, r.handlePageContent)
		return
	case "vibium:page.waitForURL":
		r.dispatch(session, cmd, r.handlePageWaitForURL)
		return
	case "vibium:page.waitForLoad":
		r.dispatch(session, cmd, r.handlePageWaitForLoad)
		return

	// Page & context lifecycle commands
	case "vibium:browser.page":
		r.dispatch(session, cmd, r.handleBrowserPage)
		return
	case "vibium:browser.newPage":
		r.dispatch(session, cmd, r.handleBrowserNewPage)
		return
	case "vibium:browser.newContext":
		r.dispatch(session, cmd, r.handleBrowserNewContext)
		return
	case "vibium:context.newPage":
		r.dispatch(session, cmd, r.handleContextNewPage)
		return
	case "vibium:browser.pages":
		r.dispatch(session, cmd, r.handleBrowserPages)
		return
	case "vibium:context.close":
		r.dispatch(session, cmd, r.handleContextClose)
		return

	// Cookie & storage commands
	case "vibium:context.cookies":
		r.dispatch(session, cmd, r.handleContextCookies)
		return
	case "vibium:context.setCookies":
		r.dispatch(session, cmd, r.handleContextSetCookies)
		return
	case "vibium:context.clearCookies":
		r.dispatch(session, cmd, r.handleContextClearCookies)
		return
	case "vibium:context.storageState":
		r.dispatch(session, cmd, r.handleContextStorageState)
		return
	case "vibium:context.addInitScript":
		r.dispatch(session, cmd, r.handleContextAddInitScript)
		return

	// Frame commands
	case "vibium:page.frames":
		r.dispatch(session, cmd, r.handlePageFrames)
		return
	case "vibium:page.frame":
		r.dispatch(session, cmd, r.handlePageFrame)
		return

	// Emulation commands
	case "vibium:page.setViewport":
		r.dispatch(session, cmd, r.handlePageSetViewport)
		return
	case "vibium:page.viewport":
		r.dispatch(session, cmd, r.handlePageViewport)
		return
	case "vibium:page.emulateMedia":
		r.dispatch(session, cmd, r.handlePageEmulateMedia)
		return
	case "vibium:page.setContent":
		r.dispatch(session, cmd, r.handlePageSetContent)
		return
	case "vibium:page.setGeolocation":
		r.dispatch(session, cmd, r.handlePageSetGeolocation)
		return
	case "vibium:page.setWindow":
		r.dispatch(session, cmd, r.handlePageSetWindow)
		return
	case "vibium:page.window":
		r.dispatch(session, cmd, r.handlePageWindow)
		return

	// Accessibility commands
	case "vibium:page.a11yTree":
		r.dispatch(session, cmd, r.handleVibiumPageA11yTree)
		return
	case "vibium:el.role":
		r.dispatch(session, cmd, r.handleVibiumElRole)
		return
	case "vibium:el.label":
		r.dispatch(session, cmd, r.handleVibiumElLabel)
		return

	case "vibium:browser.close":
		r.dispatch(session, cmd, r.handleBrowserClose)
		return
	case "vibium:page.activate":
		r.dispatch(session, cmd, r.handlePageActivate)
		return
	case "vibium:page.close":
		r.dispatch(session, cmd, r.handlePageClose)
		return

	// Network interception commands
	case "vibium:page.route":
		r.dispatch(session, cmd, r.handlePageRoute)
		return
	case "vibium:page.unroute":
		r.dispatch(session, cmd, r.handlePageUnroute)
		return
	case "vibium:network.continue":
		r.dispatch(session, cmd, r.handleNetworkContinue)
		return
	case "vibium:network.fulfill":
		r.dispatch(session, cmd, r.handleNetworkFulfill)
		return
	case "vibium:network.abort":
		r.dispatch(session, cmd, r.handleNetworkAbort)
		return
	case "vibium:page.setHeaders":
		r.dispatch(session, cmd, r.handlePageSetHeaders)
		return

	// Dialog commands
	case "vibium:dialog.accept":
		r.dispatch(session, cmd, r.handleDialogAccept)
		return
	case "vibium:dialog.dismiss":
		r.dispatch(session, cmd, r.handleDialogDismiss)
		return

	// WebSocket monitoring
	case "vibium:page.onWebSocket":
		r.dispatch(session, cmd, r.handlePageOnWebSocket)
		return

	// Download & file commands
	case "vibium:download.saveAs":
		r.dispatch(session, cmd, r.handleDownloadSaveAs)
		return
	case "vibium:el.setFiles":
		r.dispatch(session, cmd, r.handleVibiumElSetFiles)
		return

	// Tracing commands (not traced — they control tracing itself)
	case "vibium:tracing.start":
		go r.handleTracingStart(session, cmd)
		return
	case "vibium:tracing.stop":
		go r.handleTracingStop(session, cmd)
		return
	case "vibium:tracing.startChunk":
		go r.handleTracingStartChunk(session, cmd)
		return
	case "vibium:tracing.stopChunk":
		go r.handleTracingStopChunk(session, cmd)
		return
	case "vibium:tracing.startGroup":
		go r.handleTracingStartGroup(session, cmd)
		return
	case "vibium:tracing.stopGroup":
		go r.handleTracingStopGroup(session, cmd)
		return

	// Clock commands
	case "vibium:clock.install":
		r.dispatch(session, cmd, r.handleClockInstall)
		return
	case "vibium:clock.fastForward":
		r.dispatch(session, cmd, r.handleClockFastForward)
		return
	case "vibium:clock.runFor":
		r.dispatch(session, cmd, r.handleClockRunFor)
		return
	case "vibium:clock.pauseAt":
		r.dispatch(session, cmd, r.handleClockPauseAt)
		return
	case "vibium:clock.resume":
		r.dispatch(session, cmd, r.handleClockResume)
		return
	case "vibium:clock.setFixedTime":
		r.dispatch(session, cmd, r.handleClockSetFixedTime)
		return
	case "vibium:clock.setSystemTime":
		r.dispatch(session, cmd, r.handleClockSetSystemTime)
		return
	case "vibium:clock.setTimezone":
		r.dispatch(session, cmd, r.handleClockSetTimezone)
		return
	}

	// Forward standard BiDi commands to browser
	if err := session.BidiConn.Send(msg); err != nil {
		fmt.Printf("[router] Failed to send to browser for client %d: %v\n", client.ID, err)
	}
}

// getContext retrieves the first browsing context.
func (r *Router) getContext(session *BrowserSession) (string, error) {
	resp, err := r.sendInternalCommand(session, "browsingContext.getTree", map[string]interface{}{})
	if err != nil {
		return "", err
	}

	var result struct {
		Result struct {
			Contexts []struct {
				Context string `json:"context"`
			} `json:"contexts"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse getTree response: %w", err)
	}
	if len(result.Result.Contexts) == 0 {
		return "", fmt.Errorf("no browsing contexts available")
	}
	return result.Result.Contexts[0].Context, nil
}

// sendSuccess sends a successful response to the client.
func (r *Router) sendSuccess(session *BrowserSession, id int, result interface{}) {
	resp := bidiResponse{ID: id, Type: "success", Result: result}
	data, _ := json.Marshal(resp)
	session.Client.Send(string(data))
}

// sendError sends an error response to the client (follows WebDriver BiDi spec).
func (r *Router) sendError(session *BrowserSession, id int, err error) {
	resp := bidiResponse{
		ID:      id,
		Type:    "error",
		Error:   "timeout",
		Message: err.Error(),
	}
	data, _ := json.Marshal(resp)
	session.Client.Send(string(data))
}

// OnClientDisconnect is called when a client disconnects.
// It closes the browser session.
func (r *Router) OnClientDisconnect(client *ClientConn) {
	sessionVal, ok := r.sessions.LoadAndDelete(client.ID)
	if !ok {
		return
	}

	session := sessionVal.(*BrowserSession)
	r.closeSession(session)
}

// routeBrowserToClient reads messages from the browser and forwards them to the client.
func (r *Router) routeBrowserToClient(session *BrowserSession) {
	for {
		select {
		case <-session.stopChan:
			return
		default:
		}

		msg, err := session.BidiConn.Receive()
		if err != nil {
			session.mu.Lock()
			closed := session.closed
			session.mu.Unlock()

			if !closed {
				fmt.Printf("[router] Browser connection closed for client %d: %v\n", session.Client.ID, err)
				// Browser died, close the client
				session.Client.Close()
			}
			return
		}

		// Check if this is a response to an internal command
		var resp struct {
			ID int `json:"id"`
		}
		if err := json.Unmarshal([]byte(msg), &resp); err == nil && resp.ID > 0 {
			session.internalCmdsMu.Lock()
			ch, isInternal := session.internalCmds[resp.ID]
			session.internalCmdsMu.Unlock()

			if isInternal {
				// Route to internal handler
				ch <- json.RawMessage(msg)
				continue
			}

			// Drop late responses from timed-out internal commands —
			// never forward these to the client (they'd be unrecognized).
			if resp.ID >= 1000000 {
				continue
			}
		}

		// Track page URL from load/navigation events (zero extra BiDi round-trips)
		var bidiEvent struct {
			Method string `json:"method"`
			Params struct {
				URL string `json:"url"`
			} `json:"params"`
		}
		if json.Unmarshal([]byte(msg), &bidiEvent) == nil {
			if bidiEvent.Params.URL != "" && (bidiEvent.Method == "browsingContext.load" || bidiEvent.Method == "browsingContext.fragmentNavigated") {
				session.mu.Lock()
				session.lastURL = bidiEvent.Params.URL
				session.mu.Unlock()
			}
		}

		// Record event for tracing (non-blocking)
		session.mu.Lock()
		recorder := session.traceRecorder
		session.mu.Unlock()
		if recorder != nil && recorder.IsRecording() {
			recorder.RecordBidiEvent(msg)
		}

		// Check for WebSocket channel events (intercept, don't forward raw script.message)
		if r.isWsChannelEvent(session, msg) {
			continue
		}

		// Forward message to client
		if err := session.Client.Send(msg); err != nil {
			fmt.Printf("[router] Failed to send to client %d: %v\n", session.Client.ID, err)
			return
		}
	}
}

// sendInternalCommand sends a BiDi command and waits for the response (60s timeout).
func (r *Router) sendInternalCommand(session *BrowserSession, method string, params map[string]interface{}) (json.RawMessage, error) {
	return r.sendInternalCommandWithTimeout(session, method, params, 60*time.Second)
}

// sendInternalCommandWithTimeout sends a BiDi command and waits for the response with a custom timeout.
func (r *Router) sendInternalCommandWithTimeout(session *BrowserSession, method string, params map[string]interface{}, timeout time.Duration) (json.RawMessage, error) {
	// Record BiDi command in trace (opt-in via bidi: true)
	session.mu.Lock()
	recorder := session.traceRecorder
	session.mu.Unlock()
	if recorder != nil && recorder.IsRecording() && recorder.Options().Bidi {
		callId := recorder.RecordBidiCommand(method, params)
		defer recorder.RecordBidiCommandEnd(callId)
	}

	session.internalCmdsMu.Lock()
	id := session.nextInternalID
	session.nextInternalID++
	ch := make(chan json.RawMessage, 1)
	session.internalCmds[id] = ch
	session.internalCmdsMu.Unlock()

	defer func() {
		session.internalCmdsMu.Lock()
		delete(session.internalCmds, id)
		session.internalCmdsMu.Unlock()
	}()

	// Send the command
	cmd := map[string]interface{}{
		"id":     id,
		"method": method,
		"params": params,
	}
	cmdBytes, _ := json.Marshal(cmd)
	if err := session.BidiConn.Send(string(cmdBytes)); err != nil {
		return nil, err
	}

	// Wait for response (with timeout)
	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for response to %s", method)
	case <-session.stopChan:
		return nil, fmt.Errorf("session closed")
	}
}

// closeSession closes a browser session and cleans up resources.
func (r *Router) closeSession(session *BrowserSession) {
	session.mu.Lock()
	if session.closed {
		session.mu.Unlock()
		return
	}
	session.closed = true
	session.mu.Unlock()

	fmt.Printf("[router] Closing browser session for client %d\n", session.Client.ID)

	// Signal the routing goroutine to stop
	close(session.stopChan)

	// Close BiDi connection
	if session.BidiConn != nil {
		session.BidiConn.Close()
	}

	// Stop any active trace recorder
	if session.traceRecorder != nil {
		session.traceRecorder.StopScreenshots()
	}

	// Clean up download temp dir
	if session.downloadDir != "" {
		os.RemoveAll(session.downloadDir)
	}

	// Close browser
	if session.LaunchResult != nil {
		session.LaunchResult.Close()
	}

	fmt.Printf("[router] Browser session closed for client %d\n", session.Client.ID)
}

// CloseAll closes all browser sessions.
func (r *Router) CloseAll() {
	r.sessions.Range(func(key, value interface{}) bool {
		session := value.(*BrowserSession)
		r.closeSession(session)
		r.sessions.Delete(key)
		return true
	})
}
