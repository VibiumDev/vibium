package bidi

import (
	"encoding/json"
	"fmt"
)

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

// SessionSubscribeParams represents the parameters for session.subscribe command.
type SessionSubscribeParams struct {
	Events       []string `json:"events"`
	Contexts     []string `json:"contexts,omitempty"`
	UserContexts []string `json:"userContexts,omitempty"`
}

// SessionSubscribeResult represents the result of session.subscribe command.
type SessionSubscribeResult struct {
	Subscription string `json:"subscription"`
}

// SessionSubscribe sends a session.subscribe command to enable event subscriptions.
// The events parameter is a list of event names to subscribe to (e.g., "browsingContext.load").
// Optionally, contexts or userContexts can be provided to limit the subscription scope.
func (c *Client) SessionSubscribe(events []string, contexts []string, userContexts []string) (*SessionSubscribeResult, error) {
	params := SessionSubscribeParams{
		Events: events,
	}
	if len(contexts) > 0 {
		params.Contexts = contexts
	}
	if len(userContexts) > 0 {
		params.UserContexts = userContexts
	}

	msg, err := c.SendCommand("session.subscribe", params)
	if err != nil {
		return nil, err
	}

	var result SessionSubscribeResult
	if err := json.Unmarshal(msg.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse session.subscribe result: %w", err)
	}

	return &result, nil
}

// SessionUnsubscribeByIDParams represents the parameters for session.unsubscribe by subscription ID.
type SessionUnsubscribeByIDParams struct {
	Subscriptions []string `json:"subscriptions"`
}

// SessionUnsubscribeByEventsParams represents the parameters for session.unsubscribe by event names.
type SessionUnsubscribeByEventsParams struct {
	Events []string `json:"events"`
}

// SessionUnsubscribeByID sends a session.unsubscribe command to remove subscriptions by their IDs.
func (c *Client) SessionUnsubscribeByID(subscriptions []string) error {
	params := SessionUnsubscribeByIDParams{
		Subscriptions: subscriptions,
	}

	_, err := c.SendCommand("session.unsubscribe", params)
	return err
}

// SessionUnsubscribeByEvents sends a session.unsubscribe command to remove subscriptions by event names.
func (c *Client) SessionUnsubscribeByEvents(events []string) error {
	params := SessionUnsubscribeByEventsParams{
		Events: events,
	}

	_, err := c.SendCommand("session.unsubscribe", params)
	return err
}
