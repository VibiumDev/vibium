package proxy

import (
	"encoding/json"
	"fmt"
	"sync"
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
	proxy    string
}

// NewRouter creates a new router.
func NewRouter(headless bool, proxy string) *Router {
	return &Router{
		headless: headless,
		proxy:    proxy,
	}
}

// OnClientConnect is called when a new client connects.
// It launches a browser and establishes a BiDi connection.
func (r *Router) OnClientConnect(client *ClientConn) {
	fmt.Printf("[router] Launching browser for client %d...\n", client.ID)

	// Launch browser
	launchResult, err := browser.Launch(browser.LaunchOptions{
		Headless: r.headless,
		Proxy:    r.proxy,
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
	case "vibium:click":
		r.handleVibiumClick(session, cmd)
		return
	case "vibium:type":
		r.handleVibiumType(session, cmd)
		return
	case "vibium:find":
		r.handleVibiumFind(session, cmd)
		return
	}

	// Forward standard BiDi commands to browser
	if err := session.BidiConn.Send(msg); err != nil {
		fmt.Printf("[router] Failed to send to browser for client %d: %v\n", client.ID, err)
	}
}

// handleVibiumClick handles the vibium:click command with actionability checks.
func (r *Router) handleVibiumClick(session *BrowserSession, cmd bidiCommand) {
	selector, _ := cmd.Params["selector"].(string)
	context, _ := cmd.Params["context"].(string)
	timeoutMs, _ := cmd.Params["timeout"].(float64)

	timeout := defaultTimeout
	if timeoutMs > 0 {
		timeout = time.Duration(timeoutMs) * time.Millisecond
	}

	// Get context if not provided
	if context == "" {
		ctx, err := r.getContext(session)
		if err != nil {
			r.sendError(session, cmd.ID, err)
			return
		}
		context = ctx
	}

	// Wait for element and get its position
	info, err := r.waitForElement(session, context, selector, timeout)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Perform the click at element center
	x := int(info.Box.X + info.Box.Width/2)
	y := int(info.Box.Y + info.Box.Height/2)

	clickParams := map[string]interface{}{
		"context": context,
		"actions": []map[string]interface{}{
			{
				"type": "pointer",
				"id":   "mouse",
				"parameters": map[string]interface{}{
					"pointerType": "mouse",
				},
				"actions": []map[string]interface{}{
					{"type": "pointerMove", "x": x, "y": y, "duration": 0},
					{"type": "pointerDown", "button": 0},
					{"type": "pointerUp", "button": 0},
				},
			},
		},
	}

	if _, err := r.sendInternalCommand(session, "input.performActions", clickParams); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"clicked": true})
}

// handleVibiumType handles the vibium:type command with actionability checks.
func (r *Router) handleVibiumType(session *BrowserSession, cmd bidiCommand) {
	selector, _ := cmd.Params["selector"].(string)
	context, _ := cmd.Params["context"].(string)
	text, _ := cmd.Params["text"].(string)
	timeoutMs, _ := cmd.Params["timeout"].(float64)

	timeout := defaultTimeout
	if timeoutMs > 0 {
		timeout = time.Duration(timeoutMs) * time.Millisecond
	}

	// Get context if not provided
	if context == "" {
		ctx, err := r.getContext(session)
		if err != nil {
			r.sendError(session, cmd.ID, err)
			return
		}
		context = ctx
	}

	// Wait for element and get its position
	info, err := r.waitForElement(session, context, selector, timeout)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Click to focus first
	x := int(info.Box.X + info.Box.Width/2)
	y := int(info.Box.Y + info.Box.Height/2)

	clickParams := map[string]interface{}{
		"context": context,
		"actions": []map[string]interface{}{
			{
				"type": "pointer",
				"id":   "mouse",
				"parameters": map[string]interface{}{
					"pointerType": "mouse",
				},
				"actions": []map[string]interface{}{
					{"type": "pointerMove", "x": x, "y": y, "duration": 0},
					{"type": "pointerDown", "button": 0},
					{"type": "pointerUp", "button": 0},
				},
			},
		},
	}

	if _, err := r.sendInternalCommand(session, "input.performActions", clickParams); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Build key actions for typing
	keyActions := make([]map[string]interface{}, 0, len(text)*2)
	for _, char := range text {
		keyActions = append(keyActions,
			map[string]interface{}{"type": "keyDown", "value": string(char)},
			map[string]interface{}{"type": "keyUp", "value": string(char)},
		)
	}

	typeParams := map[string]interface{}{
		"context": context,
		"actions": []map[string]interface{}{
			{
				"type":    "key",
				"id":      "keyboard",
				"actions": keyActions,
			},
		},
	}

	if _, err := r.sendInternalCommand(session, "input.performActions", typeParams); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"typed": true})
}

// handleVibiumFind handles the vibium:find command with wait-for-selector.
func (r *Router) handleVibiumFind(session *BrowserSession, cmd bidiCommand) {
	selector, _ := cmd.Params["selector"].(string)
	context, _ := cmd.Params["context"].(string)
	timeoutMs, _ := cmd.Params["timeout"].(float64)

	timeout := defaultTimeout
	if timeoutMs > 0 {
		timeout = time.Duration(timeoutMs) * time.Millisecond
	}

	// Get context if not provided
	if context == "" {
		ctx, err := r.getContext(session)
		if err != nil {
			r.sendError(session, cmd.ID, err)
			return
		}
		context = ctx
	}

	// Wait for element
	info, err := r.waitForElement(session, context, selector, timeout)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{
		"tag":  info.Tag,
		"text": info.Text,
		"box": map[string]interface{}{
			"x":      info.Box.X,
			"y":      info.Box.Y,
			"width":  info.Box.Width,
			"height": info.Box.Height,
		},
	})
}

// elementInfo holds parsed element information.
type elementInfo struct {
	Tag  string  `json:"tag"`
	Text string  `json:"text"`
	Box  boxInfo `json:"box"`
}

type boxInfo struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
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

// waitForElement polls until an element is found or timeout.
func (r *Router) waitForElement(session *BrowserSession, context, selector string, timeout time.Duration) (*elementInfo, error) {
	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond

	findScript := `
		(selector) => {
			const el = document.querySelector(selector);
			if (!el) return null;
			const rect = el.getBoundingClientRect();
			return JSON.stringify({
				tag: el.tagName.toLowerCase(),
				text: (el.textContent || '').trim().substring(0, 100),
				box: {
					x: rect.x,
					y: rect.y,
					width: rect.width,
					height: rect.height
				}
			});
		}
	`

	for {
		params := map[string]interface{}{
			"functionDeclaration": findScript,
			"target":              map[string]interface{}{"context": context},
			"arguments": []map[string]interface{}{
				{"type": "string", "value": selector},
			},
			"awaitPromise":    false,
			"resultOwnership": "root",
		}

		resp, err := r.sendInternalCommand(session, "script.callFunction", params)
		if err == nil {
			// The BiDi response structure is nested:
			// { "result": { "realm": "...", "result": { "type": "string", "value": "..." } } }
			var result struct {
				Result struct {
					Result struct {
						Type  string `json:"type"`
						Value string `json:"value,omitempty"`
					} `json:"result"`
				} `json:"result"`
			}
			if err := json.Unmarshal(resp, &result); err == nil {
				if result.Result.Result.Type == "string" && result.Result.Result.Value != "" {
					var info elementInfo
					if err := json.Unmarshal([]byte(result.Result.Result.Value), &info); err == nil {
						return &info, nil
					}
				}
			}
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout after %s waiting for '%s': element not found", timeout, selector)
		}

		time.Sleep(interval)
	}
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
		}

		// Forward message to client
		if err := session.Client.Send(msg); err != nil {
			fmt.Printf("[router] Failed to send to client %d: %v\n", session.Client.ID, err)
			return
		}
	}
}

// sendInternalCommand sends a BiDi command and waits for the response.
func (r *Router) sendInternalCommand(session *BrowserSession, method string, params map[string]interface{}) (json.RawMessage, error) {
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
	case <-time.After(60 * time.Second):
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
