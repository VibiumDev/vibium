package bidi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ConnectRemote connects to a remote BiDi endpoint, creates a client,
// and establishes a session. Returns the connection, client, and session ID.
// The returned client owns all reads on the connection; callers that need to
// read the connection themselves must use SessionNewOnConn instead.
func ConnectRemote(url string, headers http.Header) (*Connection, *Client, string, error) {
	conn, err := ConnectWithHeaders(url, headers)
	if err != nil {
		return nil, nil, "", err
	}

	client := NewClient(conn)

	result, err := client.SessionNew(map[string]interface{}{})
	if err != nil {
		conn.Close()
		return nil, nil, "", err
	}

	return conn, client, result.SessionID, nil
}

// SessionNewOnConn performs the session.new handshake with direct reads on a
// connection that has no Client attached. Callers that will own reads on the
// connection afterward (like the api router) use this instead of NewClient,
// whose reader goroutine keeps the socket for the connection's lifetime.
func SessionNewOnConn(conn *Connection, capabilities map[string]interface{}) (*SessionNewResult, error) {
	cmd := NewCommand("session.new", map[string]interface{}{
		"capabilities": capabilities,
	})

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

		var result SessionNewResult
		if err := json.Unmarshal(msg.Result, &result); err != nil {
			return nil, fmt.Errorf("failed to parse session.new result: %w", err)
		}
		return &result, nil
	}

	return nil, fmt.Errorf("timeout waiting for response to session.new after %s", defaultCommandTimeout)
}
