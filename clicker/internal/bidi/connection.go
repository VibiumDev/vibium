package bidi

import (
	"fmt"
	"net/http"
	neturl "net/url"
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
	done   chan struct{} // closed on Close() or a failed write, stops the ping loop

	// writeDeadline bounds a single Send. A field, not a const, so tests can
	// shorten it; ConnectWithHeaders sets the default.
	writeDeadline time.Duration
}

// readDeadline is the timeout for each WebSocket read operation.
// Must be longer than pingInterval so pongs have time to arrive.
const readDeadline = 120 * time.Second

// pingInterval is how often we send WebSocket pings to keep the connection alive.
const pingInterval = 30 * time.Second

// defaultWriteDeadline bounds how long a Send may block. A healthy local
// socket write completes in microseconds; this only fires when the browser
// has stopped draining its socket — CPU starvation on a loaded CI runner,
// which stalls the write forever (#397). Without a deadline the single stdin
// scanner goroutine blocks inside the forward at router.go, wedging all
// command intake for the session, and sendInternalCommand blocks before its
// own timeout is ever reached. Generous enough that no healthy write trips
// it, bounded so a wedge fails fast and names the stuck command instead of
// hanging past every client timeout.
const defaultWriteDeadline = 60 * time.Second

// Connect establishes a WebSocket connection to the given URL.
func Connect(url string) (*Connection, error) {
	return ConnectWithHeaders(url, nil)
}

// ConnectWithHeaders establishes a WebSocket connection with optional HTTP headers.
// Headers are sent during the WebSocket handshake (useful for authentication tokens).
func ConnectWithHeaders(url string, headers http.Header) (*Connection, error) {
	// gorilla rejects userinfo in ws URLs outright, but cloud providers hand
	// out ws://user:key@host URLs — fold the credentials into Basic auth.
	if u, err := neturl.Parse(url); err == nil && u.User != nil {
		headers = foldUserinfo(u, headers)
		url = u.String()
	}

	dialer := websocket.Dialer{
		ReadBufferSize:   maxMessageSize,
		WriteBufferSize:  maxMessageSize,
		HandshakeTimeout: 30 * time.Second,
	}
	conn, _, err := dialer.Dial(url, headers)
	if err != nil {
		return nil, &errs.ConnectionError{URL: url, Cause: err}
	}

	// Set read limit to handle large messages (e.g., screenshots from high-res displays)
	conn.SetReadLimit(maxMessageSize)

	// Set up pong handler to extend read deadline on activity
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(readDeadline))
		return nil
	})

	c := &Connection{
		conn:          conn,
		done:          make(chan struct{}),
		writeDeadline: defaultWriteDeadline,
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
//
// The write is bounded by writeDeadline: a browser that has stopped draining
// its socket would otherwise block the caller forever. Many callers run on
// goroutines that can afford to wait, but the standard-command forward runs on
// the single stdin scanner goroutine, so one stalled write freezes all command
// intake for the session (#397). A write that fails (deadline or otherwise)
// leaves the WebSocket framing unusable, so the connection is failed closed:
// the blocked reader in Receive unblocks, the routing goroutine runs session
// teardown, and the client's pending commands get a prompt error instead of
// each waiting out its own timeout against a dead connection.
func (c *Connection) Send(msg string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed.Load() {
		return fmt.Errorf("connection closed")
	}

	if c.writeDeadline > 0 {
		c.conn.SetWriteDeadline(time.Now().Add(c.writeDeadline))
	}
	err := c.conn.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		c.failClosed()
	}
	return err
}

// failClosed tears the connection down after a fatal write. The caller holds
// c.mu. Closing the underlying socket unblocks a reader parked in ReadMessage
// (a frozen browser never trips the read deadline on its own), and closing
// done stops the ping loop. Guarded by the same CompareAndSwap as Close so a
// later Close is a no-op and done is closed once.
func (c *Connection) failClosed() {
	if c.closed.CompareAndSwap(false, true) {
		close(c.done)
		c.conn.Close()
	}
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
