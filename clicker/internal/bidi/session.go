package bidi

import (
	"encoding/json"
	"fmt"
)

// Client is a BiDi client that wraps a WebSocket connection.
type Client struct {
	conn *Connection
}

// NewClient creates a new BiDi client from a WebSocket connection.
func NewClient(conn *Connection) *Client {
	return &Client{conn: conn}
}

// SendCommand sends a BiDi command and waits for the response.
func (c *Client) SendCommand(method string, params interface{}) (*Message, error) {
	cmd := NewCommand(method, params)

	data, err := cmd.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}

	if err := c.conn.Send(string(data)); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	// Wait for response with matching ID
	for {
		resp, err := c.conn.Receive()
		if err != nil {
			return nil, fmt.Errorf("failed to receive response: %w", err)
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
			continue
		}
	}
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

// Close closes the underlying connection.
func (c *Client) Close() error {
	return c.conn.Close()
}
