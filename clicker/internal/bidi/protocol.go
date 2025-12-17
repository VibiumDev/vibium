package bidi

import (
	"encoding/json"
	"sync/atomic"
)

// commandID is an atomic counter for generating unique command IDs.
var commandID int64

// NextID returns the next unique command ID.
func NextID() int64 {
	return atomic.AddInt64(&commandID, 1)
}

// Command represents a BiDi command to be sent to the browser.
type Command struct {
	ID     int64       `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
}

// Response represents a BiDi response from the browser.
type Response struct {
	ID     int64           `json:"id,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *ErrorData      `json:"error,omitempty"`
}

// ErrorData represents an error in a BiDi response.
type ErrorData struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Event represents a BiDi event from the browser.
type Event struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Message is a generic BiDi message that can be either a response or event.
type Message struct {
	// Response fields
	ID     *int64          `json:"id,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  json.RawMessage `json:"error,omitempty"`

	// Event fields
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
}

// IsResponse returns true if the message is a response (has an ID).
func (m *Message) IsResponse() bool {
	return m.ID != nil
}

// IsEvent returns true if the message is an event (has a method but no ID).
func (m *Message) IsEvent() bool {
	return m.Method != "" && m.ID == nil
}

// IsError returns true if the message is an error response.
func (m *Message) IsError() bool {
	return len(m.Error) > 0
}

// GetError parses the error field and returns an ErrorData.
func (m *Message) GetError() (*ErrorData, error) {
	if len(m.Error) == 0 {
		return nil, nil
	}

	// Try to unmarshal as ErrorData object
	var errData ErrorData
	if err := json.Unmarshal(m.Error, &errData); err != nil {
		// If that fails, it might be a plain string
		var errStr string
		if err := json.Unmarshal(m.Error, &errStr); err != nil {
			return nil, err
		}
		return &ErrorData{Error: errStr, Message: errStr}, nil
	}
	return &errData, nil
}

// NewCommand creates a new BiDi command with a unique ID.
func NewCommand(method string, params interface{}) *Command {
	return &Command{
		ID:     NextID(),
		Method: method,
		Params: params,
	}
}

// Marshal serializes a command to JSON.
func (c *Command) Marshal() ([]byte, error) {
	return json.Marshal(c)
}

// UnmarshalMessage parses a JSON message into a Message struct.
func UnmarshalMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
