package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/vibium/clicker/internal/agent"
	"github.com/vibium/clicker/internal/browser"
	"github.com/vibium/clicker/internal/paths"
)

// Vars, not consts, so tests can shrink them to hermetic sizes.
var (
	dialTimeout = 2 * time.Second
	readTimeout = 60 * time.Second

	// launchGrace is added to the read deadline once when the daemon reports
	// a browser launch in progress. Derived from the launch path's own bounds
	// so the client outlasts a legitimately slow launch without hiding a
	// wedged daemon: no launch notification means the plain readTimeout
	// still applies (#407).
	launchGrace = browser.LaunchBudget
)

// ToolError is an error the daemon itself reported. It means the daemon was
// reached and answered, so callers must not mistake it for the daemon being
// down — a remote browser refusing a connection produces error text that
// looks exactly like an unreachable daemon socket.
type ToolError struct {
	Msg string
}

func (e *ToolError) Error() string { return e.Msg }

// Call sends a tools/call request to the daemon and returns the result.
func Call(toolName string, args map[string]interface{}) (*agent.ToolsCallResult, error) {
	params := agent.ToolsCallParams{
		Name:      toolName,
		Arguments: args,
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal params: %w", err)
	}

	resp, err := sendRequest("tools/call", json.RawMessage(paramsJSON))
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, &ToolError{Msg: fmt.Sprintf("daemon error: %s", resp.Error.Message)}
	}

	// Parse the result as ToolsCallResult
	resultJSON, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}

	var result agent.ToolsCallResult
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}

	if result.IsError {
		if len(result.Content) > 0 {
			return nil, &ToolError{Msg: result.Content[0].Text}
		}
		return nil, &ToolError{Msg: "tool call failed"}
	}

	return &result, nil
}

// Status sends a daemon/status request and returns the result.
func Status() (*StatusResult, error) {
	resp, err := sendRequest("daemon/status", nil)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("daemon error: %s", resp.Error.Message)
	}

	resultJSON, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}

	var result StatusResult
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}

	return &result, nil
}

// Shutdown sends a daemon/shutdown request.
func Shutdown() error {
	resp, err := sendRequest("daemon/shutdown", nil)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("daemon error: %s", resp.Error.Message)
	}

	return nil
}

// sendRequest sends a JSON-RPC request to the daemon socket and returns the response.
func sendRequest(method string, params json.RawMessage) (*agent.Response, error) {
	socketPath, err := paths.GetSocketPath()
	if err != nil {
		return nil, fmt.Errorf("get socket path: %w", err)
	}

	conn, err := dial(socketPath, dialTimeout)
	if err != nil {
		return nil, fmt.Errorf("connect to daemon: %w", err)
	}
	defer conn.Close()

	req := agent.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := fmt.Fprintf(conn, "%s\n", data); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	conn.SetReadDeadline(time.Now().Add(readTimeout))
	// bufio.Reader grows as needed; bufio.Scanner failed with "token too long"
	// on any response over its fixed buffer — a long page's text, a large
	// storage state (#209).
	reader := bufio.NewReader(conn)
	var line []byte
	extended := false
	for {
		line, err = reader.ReadBytes('\n')
		if err != nil && len(line) == 0 {
			if err != io.EOF {
				return nil, fmt.Errorf("read response: %w", err)
			}
			return nil, fmt.Errorf("daemon closed connection without response")
		}
		if err != nil {
			break // partial final line; let the response parse report it
		}

		// The daemon may send notifications (no id) ahead of the response.
		// A launch notification means the daemon committed to a browser
		// launch, so the response can legitimately take the launch bounds
		// plus a normal command on top; extend the deadline once from here.
		// Unknown notifications are skipped without extending.
		var msg struct {
			Method string          `json:"method"`
			ID     json.RawMessage `json:"id"`
		}
		if json.Unmarshal(line, &msg) == nil && msg.Method != "" && len(msg.ID) == 0 {
			if msg.Method == launchingBrowserMethod && !extended {
				extended = true
				conn.SetReadDeadline(time.Now().Add(launchGrace + readTimeout))
			}
			continue
		}
		break
	}

	var resp agent.Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &resp, nil
}
