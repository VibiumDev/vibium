package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vibium/clicker/internal/mcp"
	"github.com/vibium/clicker/internal/paths"
)

const (
	dialTimeout = 2 * time.Second
	readTimeout = 60 * time.Second
)

// Call sends a tools/call request to the daemon and returns the result.
func Call(toolName string, args map[string]interface{}) (*mcp.ToolsCallResult, error) {
	params := mcp.ToolsCallParams{
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
		return nil, fmt.Errorf("daemon error: %s", resp.Error.Message)
	}

	// Parse the result as ToolsCallResult
	resultJSON, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}

	var result mcp.ToolsCallResult
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}

	if result.IsError {
		if len(result.Content) > 0 {
			return nil, fmt.Errorf("%s", result.Content[0].Text)
		}
		return nil, fmt.Errorf("tool call failed")
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
func sendRequest(method string, params json.RawMessage) (*mcp.Response, error) {
	socketPath, err := paths.GetSocketPath()
	if err != nil {
		return nil, fmt.Errorf("get socket path: %w", err)
	}

	conn, err := dial(socketPath, dialTimeout)
	if err != nil {
		return nil, fmt.Errorf("connect to daemon: %w", err)
	}
	defer conn.Close()

	req := mcp.Request{
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
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}
		return nil, fmt.Errorf("daemon closed connection without response")
	}

	var resp mcp.Response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &resp, nil
}
