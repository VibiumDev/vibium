package bidi

import (
	"fmt"
	"sync"

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
	closed bool
}

// Connect establishes a WebSocket connection to the given URL.
func Connect(url string) (*Connection, error) {
	dialer := websocket.Dialer{
		ReadBufferSize:  maxMessageSize,
		WriteBufferSize: maxMessageSize,
	}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		return nil, &errs.ConnectionError{URL: url, Cause: err}
	}

	// Set read limit to handle large messages (e.g., screenshots from high-res displays)
	conn.SetReadLimit(maxMessageSize)

	return &Connection{
		conn: conn,
	}, nil
}

// Send sends a text message over the WebSocket.
func (c *Connection) Send(msg string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("connection closed")
	}

	return c.conn.WriteMessage(websocket.TextMessage, []byte(msg))
}

// Receive receives a text message from the WebSocket.
// Blocks until a message is received.
func (c *Connection) Receive() (string, error) {
	if c.closed {
		return "", fmt.Errorf("connection closed")
	}

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
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	// Send close message
	c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	return c.conn.Close()
}
