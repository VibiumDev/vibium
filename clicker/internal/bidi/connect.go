package bidi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RemoteSession describes the session a remote connect ended up using.
type RemoteSession struct {
	ID string // empty when attached to a session the endpoint does not name

	// Created reports whether this process created the session. Only its
	// creator may end it — attaching and then sending session.end would
	// destroy a browser the caller merely borrowed.
	Created bool
}

// ConnectRemote connects to a remote BiDi endpoint, creates a client, and
// establishes a session. The returned client owns all reads on the connection;
// callers that need to read the connection themselves must use
// AttachOrNewSessionOnConn instead.
func ConnectRemote(url string, headers http.Header) (*Connection, *Client, *RemoteSession, error) {
	conn, err := ConnectWithHeaders(url, headers)
	if err != nil {
		return nil, nil, nil, err
	}

	client := NewClient(conn)

	// An endpoint that already carries a session rejects session.new with
	// "session not created" (spec: remote end steps for session.new, step 1).
	// Attach to that session instead of trying to create a second one.
	if status, err := client.SessionStatus(); err == nil && !status.Ready {
		return conn, client, &RemoteSession{ID: SessionIDFromURL(url)}, nil
	}

	result, err := client.SessionNew(map[string]interface{}{})
	if err != nil {
		conn.Close()
		return nil, nil, nil, err
	}

	return conn, client, &RemoteSession{ID: result.SessionID, Created: true}, nil
}

// AttachOrNewSessionOnConn performs the session handshake with direct reads on
// a connection that has no Client attached. Callers that will own reads on the
// connection afterward (like the api router) use this instead of NewClient,
// whose reader goroutine keeps the socket for the connection's lifetime.
func AttachOrNewSessionOnConn(conn *Connection, endpoint string, capabilities map[string]interface{}) (*RemoteSession, error) {
	// See ConnectRemote: an endpoint that already carries a session (a
	// chromedriver webSocketUrl, a Selenium Grid /se/bidi URL) reports
	// ready:false and cannot create a second one — attach to it instead.
	var status SessionStatusResult
	if raw, err := commandOnConn(conn, "session.status", map[string]interface{}{}); err == nil {
		if err := json.Unmarshal(raw, &status); err == nil && !status.Ready {
			return &RemoteSession{ID: SessionIDFromURL(endpoint)}, nil
		}
	}

	result, err := SessionNewOnConn(conn, capabilities)
	if err != nil {
		return nil, err
	}
	return &RemoteSession{ID: result.SessionID, Created: true}, nil
}

// SessionNewOnConn creates a session with direct reads on a connection that has
// no Client attached. Only for endpoints known to have no session yet (a
// freshly launched chromedriver); use AttachOrNewSessionOnConn for user-supplied
// endpoints, which may already carry one.
func SessionNewOnConn(conn *Connection, capabilities map[string]interface{}) (*SessionNewResult, error) {
	raw, err := commandOnConn(conn, "session.new", map[string]interface{}{
		"capabilities": capabilities,
	})
	if err != nil {
		return nil, err
	}

	var result SessionNewResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("failed to parse session.new result: %w", err)
	}
	return &result, nil
}

// SessionIDFromURL extracts the WebDriver session ID from a BiDi endpoint URL
// such as ws://host:9515/session/<id> or a Selenium Grid
// ws://host:4444/session/<id>/se/bidi. Returns "" when the URL names no session.
func SessionIDFromURL(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i, p := range parts {
		if p == "session" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// commandOnConn sends one command and waits for its response with direct reads
// on a connection that has no Client attached.
func commandOnConn(conn *Connection, method string, params map[string]interface{}) (json.RawMessage, error) {
	cmd := NewCommand(method, params)

	data, err := cmd.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}

	if err := conn.Send(string(data)); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	// The deadline is only checked between messages, so a connection that
	// goes fully silent blocks in Receive until its 120s read deadline, not
	// this 60s one. Acceptable for a handshake-only path and matches the
	// behavior before the single-reader refactor.
	deadline := time.Now().Add(defaultCommandTimeout)
	for time.Now().Before(deadline) {
		raw, err := conn.Receive()
		if err != nil {
			return nil, fmt.Errorf("failed to receive response: %w", err)
		}

		msg, err := UnmarshalMessage([]byte(raw))
		if err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		// Nothing is subscribed yet, so anything that is not the handshake
		// response (early events, mostly) can be skipped.
		if msg.ID == nil || *msg.ID != cmd.ID {
			continue
		}

		if _, err := responseOrError(msg); err != nil {
			return nil, err
		}

		return msg.Result, nil
	}

	return nil, fmt.Errorf("timeout waiting for response to %s after %s", method, defaultCommandTimeout)
}
