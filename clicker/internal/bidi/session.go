package bidi

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// eventBufferSize bounds the queue between the reader and the event
// dispatcher. When the handler falls this far behind, events are dropped
// rather than letting them stall the socket reader.
const eventBufferSize = 256

// Client is a BiDi client that wraps a WebSocket connection. A dedicated
// reader goroutine owns all reads on the connection and routes responses to
// waiting callers by command ID, so commands can be sent concurrently and
// the socket stays drained even while no command is in flight.
type Client struct {
	conn    *Connection
	verbose atomic.Bool

	// mu guards pending and eventHandler.
	mu           sync.Mutex
	pending      map[int64]chan *Message
	eventHandler func(msg string)

	events chan string

	// droppedEvents counts events discarded because the events buffer was
	// full. Read it with DroppedEvents.
	droppedEvents atomic.Uint64

	// readErr is written by the reader before it closes readerDone, so it
	// may only be read after readerDone is closed.
	readErr    error
	readerDone chan struct{}
}

// NewClient creates a new BiDi client from a WebSocket connection and starts
// its reader. The client owns all reads on the connection from this point;
// nothing else may call conn.Receive for the connection's lifetime.
func NewClient(conn *Connection) *Client {
	c := &Client{
		conn:       conn,
		pending:    make(map[int64]chan *Message),
		events:     make(chan string, eventBufferSize),
		readerDone: make(chan struct{}),
	}
	go c.readLoop()
	go c.dispatchEvents()
	return c
}

// SetVerbose enables or disables verbose logging of JSON messages.
func (c *Client) SetVerbose(verbose bool) {
	c.verbose.Store(verbose)
}

// SetEventHandler sets a callback for BiDi events. Pass nil to stop
// forwarding events. Safe to call while the client is in use.
func (c *Client) SetEventHandler(handler func(msg string)) {
	c.mu.Lock()
	c.eventHandler = handler
	c.mu.Unlock()
}

// DroppedEvents returns how many events have been discarded over the
// client's lifetime because the event buffer was full. Consumers that care
// about gaps (like the recorder) can diff this counter across an interval.
func (c *Client) DroppedEvents() uint64 {
	return c.droppedEvents.Load()
}

// readLoop is the only reader of the connection. Routing every message
// through one goroutine keeps concurrent SendCommand callers from stealing
// each other's responses and keeps events flowing between commands.
func (c *Client) readLoop() {
	defer func() {
		close(c.readerDone)
		close(c.events)
		// A connection without a reader is unusable; closing it stops the
		// ping loop and makes later sends fail instead of hang.
		c.conn.Close()
	}()

	for {
		raw, err := c.conn.Receive()
		if err != nil {
			c.readErr = fmt.Errorf("connection read failed: %w", err)
			return
		}

		msg, err := UnmarshalMessage([]byte(raw))
		if err != nil {
			if c.verbose.Load() {
				fmt.Printf("       <-- %s\n", raw)
			}
			// A single malformed frame should not take down the connection.
			continue
		}

		// Concurrent commands interleave in verbose output; the [id] prefix
		// matches each <-- line to its --> line.
		if c.verbose.Load() {
			if msg.ID != nil {
				fmt.Printf("       <-- [%d] %s\n", *msg.ID, raw)
			} else {
				fmt.Printf("       <-- %s\n", raw)
			}
		}

		if msg.ID != nil {
			c.mu.Lock()
			ch, ok := c.pending[*msg.ID]
			if ok {
				delete(c.pending, *msg.ID)
			}
			c.mu.Unlock()
			// A missing entry means the caller gave up; drop the response.
			// The channel holds one message, so the send never blocks.
			if ok {
				ch <- msg
			}
			continue
		}

		if msg.IsEvent() {
			select {
			case c.events <- raw:
			default:
				// Dropping beats stalling every command response behind a
				// slow event handler. Warn directly on stderr (once): the
				// default log level discards Warn, and silent data loss is
				// the one thing that must stay visible in quiet mode.
				if c.droppedEvents.Add(1) == 1 {
					fmt.Fprintln(os.Stderr, "vibium: BiDi event consumer too slow; dropping events (warned once, count available via DroppedEvents)")
				}
			}
		}
	}
}

// dispatchEvents delivers events in arrival order on its own goroutine so a
// blocking handler can never stall the reader.
func (c *Client) dispatchEvents() {
	for raw := range c.events {
		c.mu.Lock()
		handler := c.eventHandler
		c.mu.Unlock()
		if handler != nil {
			handler(raw)
		}
	}
}

// defaultCommandTimeout is the maximum time to wait for a BiDi command response.
const defaultCommandTimeout = 60 * time.Second

// SendCommand sends a BiDi command and waits for the response (60s timeout).
func (c *Client) SendCommand(method string, params interface{}) (*Message, error) {
	return c.SendCommandWithTimeout(method, params, defaultCommandTimeout)
}

// SendCommandWithTimeout sends a BiDi command and waits for the response with a custom timeout.
func (c *Client) SendCommandWithTimeout(method string, params interface{}, timeout time.Duration) (*Message, error) {
	cmd := NewCommand(method, params)

	data, err := cmd.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}

	if c.verbose.Load() {
		fmt.Printf("       --> [%d] %s\n", cmd.ID, string(data))
	}

	// Register before sending so the response cannot race past us.
	ch := make(chan *Message, 1)
	c.mu.Lock()
	select {
	case <-c.readerDone:
		c.mu.Unlock()
		return nil, fmt.Errorf("cannot send %s: %w", method, c.readErr)
	default:
	}
	c.pending[cmd.ID] = ch
	c.mu.Unlock()

	// Unregister on every exit path so a late response finds no entry
	// instead of a channel nobody reads.
	defer func() {
		c.mu.Lock()
		delete(c.pending, cmd.ID)
		c.mu.Unlock()
	}()

	if err := c.conn.Send(string(data)); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case msg := <-ch:
		return responseOrError(msg)
	case <-timer.C:
		return nil, fmt.Errorf("timeout waiting for response to %s after %s", method, timeout)
	case <-c.readerDone:
		// The response may have been routed just before the reader died.
		select {
		case msg := <-ch:
			return responseOrError(msg)
		default:
		}
		return nil, fmt.Errorf("connection lost waiting for response to %s: %w", method, c.readErr)
	}
}

// responseOrError converts a routed response message into the caller-facing result.
func responseOrError(msg *Message) (*Message, error) {
	if msg.IsError() {
		errData, _ := msg.GetError()
		if errData != nil {
			return nil, fmt.Errorf("BiDi error: %s - %s", errData.Error, errData.Message)
		}
		return nil, fmt.Errorf("BiDi error: %s", string(msg.Error))
	}
	return msg, nil
}

// SessionStatusResult represents the result of session.status command.
type SessionStatusResult struct {
	Ready   bool   `json:"ready"`
	Message string `json:"message"`
}

// SessionStatus sends a session.status command and returns the result.
func (c *Client) SessionStatus() (*SessionStatusResult, error) {
	msg, err := c.SendCommand("session.status", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	var result SessionStatusResult
	if err := json.Unmarshal(msg.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse session.status result: %w", err)
	}

	return &result, nil
}

// SessionNewResult represents the result of session.new command.
type SessionNewResult struct {
	SessionID    string                 `json:"sessionId"`
	Capabilities map[string]interface{} `json:"capabilities"`
}

// SessionNew sends a session.new command and returns the result.
func (c *Client) SessionNew(capabilities map[string]interface{}) (*SessionNewResult, error) {
	params := map[string]interface{}{
		"capabilities": capabilities,
	}

	msg, err := c.SendCommand("session.new", params)
	if err != nil {
		return nil, err
	}

	var result SessionNewResult
	if err := json.Unmarshal(msg.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse session.new result: %w", err)
	}

	return &result, nil
}

// Close closes the underlying connection and waits for the reader goroutine
// to exit, so callers (tests especially) get a deterministic shutdown.
func (c *Client) Close() error {
	err := c.conn.Close()
	<-c.readerDone
	return err
}
