package bidi

import (
	"fmt"
	"sync"
)

// Client is a BiDi client that wraps a WebSocket connection.
type Client struct {
	conn    *Connection
	verbose bool

	// Single channel for all events
	events chan *Event

	// For dispatching command responses
	pendingCommands   map[int64]chan *Message
	pendingCommandsMu sync.Mutex

	// Event loop state
	eventLoopRunning bool
	eventLoopMu      sync.Mutex
}

// NewClient creates a new BiDi client from a WebSocket connection.
func NewClient(conn *Connection) *Client {
	return &Client{
		conn:            conn,
		events:          make(chan *Event, 100),
		pendingCommands: make(map[int64]chan *Message),
	}
}

// SetVerbose enables or disables verbose logging of JSON messages.
func (c *Client) SetVerbose(verbose bool) {
	c.verbose = verbose
}

// Events returns the channel that receives all BiDi events.
// The event loop must be started with StartEventLoop() for events to be delivered.
func (c *Client) Events() <-chan *Event {
	return c.events
}

// StartEventLoop starts a background goroutine that reads messages from the WebSocket
// and dispatches events to the events channel. This must be called before
// events will be delivered.
func (c *Client) StartEventLoop() {
	c.eventLoopMu.Lock()
	if c.eventLoopRunning {
		c.eventLoopMu.Unlock()
		return
	}
	c.eventLoopRunning = true
	c.eventLoopMu.Unlock()

	go c.eventLoop()
}

// eventLoop reads messages from the WebSocket and dispatches them appropriately.
func (c *Client) eventLoop() {
	for {
		resp, err := c.conn.Receive()
		if err != nil {
			// Connection closed or error
			c.eventLoopMu.Lock()
			c.eventLoopRunning = false
			c.eventLoopMu.Unlock()
			close(c.events)
			return
		}

		if c.verbose {
			fmt.Printf("       <-- %s\n", resp)
		}

		msg, err := UnmarshalMessage([]byte(resp))
		if err != nil {
			if c.verbose {
				fmt.Printf("       (failed to parse message: %v)\n", err)
			}
			continue
		}

		// Dispatch command responses
		if msg.ID != nil {
			c.pendingCommandsMu.Lock()
			if ch, ok := c.pendingCommands[*msg.ID]; ok {
				ch <- msg
				delete(c.pendingCommands, *msg.ID)
			}
			c.pendingCommandsMu.Unlock()
			continue
		}

		// Dispatch events
		if msg.IsEvent() {
			event := &Event{
				Method: msg.Method,
				Params: msg.Params,
			}
			select {
			case c.events <- event:
			default:
				if c.verbose {
					fmt.Printf("       (events channel full, dropping event: %s)\n", msg.Method)
				}
			}
		}
	}
}

// SendCommand sends a BiDi command and waits for the response.
func (c *Client) SendCommand(method string, params interface{}) (*Message, error) {
	cmd := NewCommand(method, params)

	data, err := cmd.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}

	if c.verbose {
		fmt.Printf("       --> %s\n", string(data))
	}

	// Check if event loop is running
	c.eventLoopMu.Lock()
	eventLoopRunning := c.eventLoopRunning
	c.eventLoopMu.Unlock()

	if eventLoopRunning {
		// Register a channel for the response
		responseCh := make(chan *Message, 1)
		c.pendingCommandsMu.Lock()
		c.pendingCommands[cmd.ID] = responseCh
		c.pendingCommandsMu.Unlock()

		if err := c.conn.Send(string(data)); err != nil {
			c.pendingCommandsMu.Lock()
			delete(c.pendingCommands, cmd.ID)
			c.pendingCommandsMu.Unlock()
			return nil, fmt.Errorf("failed to send command: %w", err)
		}

		// Wait for response from event loop
		msg := <-responseCh
		if msg.IsError() {
			errData, _ := msg.GetError()
			if errData != nil {
				return nil, fmt.Errorf("BiDi error: %s - %s", errData.Error, errData.Message)
			}
			return nil, fmt.Errorf("BiDi error: %s", string(msg.Error))
		}
		return msg, nil
	}

	// Fallback to synchronous mode when event loop is not running
	if err := c.conn.Send(string(data)); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	// Wait for response with matching ID
	for {
		resp, err := c.conn.Receive()
		if err != nil {
			return nil, fmt.Errorf("failed to receive response: %w", err)
		}

		if c.verbose {
			fmt.Printf("       <-- %s\n", resp)
		}

		msg, err := UnmarshalMessage([]byte(resp))
		if err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		// Check if this is the response we're waiting for
		if msg.ID != nil && *msg.ID == cmd.ID {
			if msg.IsError() {
				errData, _ := msg.GetError()
				if errData != nil {
					return nil, fmt.Errorf("BiDi error: %s - %s", errData.Error, errData.Message)
				}
				return nil, fmt.Errorf("BiDi error: %s", string(msg.Error))
			}
			return msg, nil
		}

		// If it's an event, skip it for now (could be handled by event listener)
		if msg.IsEvent() {
			if c.verbose {
				fmt.Printf("       (event, skipping)\n")
			}
			continue
		}
	}
}

// Close closes the underlying connection.
func (c *Client) Close() error {
	return c.conn.Close()
}
