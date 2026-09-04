package bidi

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	errs "github.com/vibium/clicker/internal/errors"
)

// maxMessageSize is the maximum size of a WebSocket message (10MB).
// This accommodates large screenshots from high-resolution displays (e.g., retina, 4K).
const maxMessageSize = 10 * 1024 * 1024

// Connection represents a WebSocket connection.
type Connection struct {
	conn   *websocket.Conn
	mu     sync.Mutex
	closed atomic.Bool
	done   chan struct{} // closed on Close() to stop the ping loop
}

// readDeadline is the timeout for each WebSocket read operation.
// Must be longer than pingInterval so pongs have time to arrive.
const readDeadline = 120 * time.Second

// pingInterval is how often we send WebSocket pings to keep the connection alive.
const pingInterval = 30 * time.Second

// Connect establishes a WebSocket connection to the given URL.
func Connect(url string) (*Connection, error) {
	return ConnectWithHeaders(url, nil)
}

// ConnectWithHeaders establishes a WebSocket connection with optional HTTP headers.
// Headers are sent during the WebSocket handshake (useful for authentication tokens).
//
// The URL goes through NormalizeEndpoint first, so an http:// or https:// hub
// URL and credentials in its userinfo field both work (#101).
func ConnectWithHeaders(rawURL string, headers http.Header) (*Connection, error) {
	endpoint, urlHeaders, err := NormalizeEndpoint(rawURL)
	if err != nil {
		return nil, &errs.ConnectionError{URL: RedactEndpoint(rawURL), Cause: err}
	}

	// Credentials in the URL only fill a gap: an explicit header from
	// VIBIUM_CONNECT_API_KEY or --connect-header is a deliberate choice and
	// wins. Clone rather than mutate — the router reuses one header map for
	// every client it connects.
	if len(urlHeaders) > 0 {
		merged := headers.Clone()
		if merged == nil {
			merged = make(http.Header)
		}
		for key, vals := range urlHeaders {
			if merged.Get(key) == "" {
				merged[key] = vals
			}
		}
		headers = merged
	}

	dialer := websocket.Dialer{
		ReadBufferSize:   maxMessageSize,
		WriteBufferSize:  maxMessageSize,
		HandshakeTimeout: 30 * time.Second,
	}
	conn, _, err := dialer.Dial(endpoint, headers)
	if err != nil {
		return nil, &errs.ConnectionError{URL: RedactEndpoint(endpoint), Cause: err}
	}

	// Set read limit to handle large messages (e.g., screenshots from high-res displays)
	conn.SetReadLimit(maxMessageSize)

	// Set up pong handler to extend read deadline on activity
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(readDeadline))
		return nil
	})

	c := &Connection{
		conn: conn,
		done: make(chan struct{}),
	}
	go c.pingLoop()
	return c, nil
}

// pingLoop sends WebSocket pings at regular intervals to keep the connection
// alive and allow the pong handler to extend the read deadline.
func (c *Connection) pingLoop() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			if c.closed.Load() {
				return
			}
			c.mu.Lock()
			err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second))
			c.mu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

// Send sends a text message over the WebSocket.
func (c *Connection) Send(msg string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed.Load() {
		return fmt.Errorf("connection closed")
	}

	return c.conn.WriteMessage(websocket.TextMessage, []byte(msg))
}

// Receive receives a text message from the WebSocket.
// Blocks until a message is received or the read deadline (120s) expires.
func (c *Connection) Receive() (string, error) {
	if c.closed.Load() {
		return "", fmt.Errorf("connection closed")
	}

	// Set a read deadline to detect dead connections (e.g., Chrome crash without TCP close)
	c.conn.SetReadDeadline(time.Now().Add(readDeadline))

	msgType, msg, err := c.conn.ReadMessage()
	if err != nil {
		return "", err
	}

	if msgType != websocket.TextMessage {
		return "", fmt.Errorf("expected text message, got type %d", msgType)
	}

	return string(msg), nil
}

// Close closes the WebSocket connection.
func (c *Connection) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}

	close(c.done)

	// Send close message
	c.mu.Lock()
	c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	c.mu.Unlock()

	return c.conn.Close()
}
