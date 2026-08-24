package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/vibium/clicker/internal/agent"
	"github.com/vibium/clicker/internal/log"
	"github.com/vibium/clicker/internal/paths"
)

// StatusResult is returned by daemon/status.
type StatusResult struct {
	Version   string `json:"version"`
	PID       int    `json:"pid"`
	Uptime    string `json:"uptime"`
	Socket    string `json:"socket"`
	StartTime string `json:"startTime"`
	Session   string `json:"session"`
}

// launchingBrowserMethod is the notification the daemon writes before a tool
// call launches a browser, so the client can extend its read deadline to
// cover the launch bounds instead of timing out mid-launch (#407). Sent as a
// JSON-RPC notification (no id) ahead of the response on the same connection.
const launchingBrowserMethod = "daemon/launchingBrowser"

// handleConnection processes a single client connection.
// Each connection sends one JSON-RPC request and receives one response,
// optionally preceded by notifications.
func (d *Daemon) handleConnection(conn net.Conn) {
	defer conn.Close()

	d.touchActivity()

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	// bufio.Reader grows as needed; bufio.Scanner fails once a message exceeds
	// its fixed buffer, and raising that cap only moves the ceiling (#209).
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil && len(line) == 0 {
		return
	}

	// The handler runs synchronously in this goroutine, so the notification
	// write cannot interleave with the response write below.
	notifyLaunch := func() {
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		fmt.Fprintf(conn, "{\"jsonrpc\":\"2.0\",\"method\":%q}\n", launchingBrowserMethod)
	}

	response := d.handleRequest(line, notifyLaunch)
	if response == nil {
		return
	}

	data, err := json.Marshal(response)
	if err != nil {
		log.Debug("marshal error", "error", err)
		return
	}

	conn.SetWriteDeadline(time.Now().Add(60 * time.Second))
	fmt.Fprintf(conn, "%s\n", data)
}

// handleRequest parses and routes a JSON-RPC request. notifyLaunch is invoked
// if handling the request starts a browser launch.
func (d *Daemon) handleRequest(data []byte, notifyLaunch func()) *agent.Response {
	var req agent.Request
	if err := json.Unmarshal(data, &req); err != nil {
		return &agent.Response{
			JSONRPC: "2.0",
			Error: &agent.Error{
				Code:    agent.ParseError,
				Message: "Parse error",
				Data:    err.Error(),
			},
		}
	}

	if req.JSONRPC != "2.0" {
		return &agent.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &agent.Error{
				Code:    agent.InvalidRequest,
				Message: "Invalid Request",
				Data:    "jsonrpc must be '2.0'",
			},
		}
	}

	result, mcpErr := d.route(req, notifyLaunch)

	if req.ID == nil {
		return nil
	}

	if mcpErr != nil {
		return &agent.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   mcpErr,
		}
	}

	return &agent.Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// route dispatches requests to the appropriate handler.
func (d *Daemon) route(req agent.Request, notifyLaunch func()) (interface{}, *agent.Error) {
	log.Debug("daemon request", "method", req.Method, "id", req.ID)

	switch req.Method {
	case "daemon/status":
		return d.handleStatus()
	case "daemon/shutdown":
		go d.Shutdown() // Shutdown asynchronously so we can send response
		return map[string]string{"status": "shutting down"}, nil
	case "tools/call":
		return d.handleToolsCall(req.Params, notifyLaunch)
	case "tools/list":
		return agent.ToolsListResult{
			Tools: agent.GetToolSchemas(),
		}, nil
	case "initialize":
		return d.handleInitialize()
	case "initialized", "notifications/initialized":
		return nil, nil
	default:
		return nil, &agent.Error{
			Code:    agent.MethodNotFound,
			Message: "Method not found",
			Data:    req.Method,
		}
	}
}

// handleStatus returns daemon status information.
func (d *Daemon) handleStatus() (interface{}, *agent.Error) {
	return StatusResult{
		Version:   d.version,
		PID:       pidSelf(),
		Uptime:    time.Since(d.startTime).Truncate(time.Second).String(),
		Socket:    d.socketPath,
		StartTime: d.startTime.Format(time.RFC3339),
		Session:   paths.SessionName(),
	}, nil
}

// handleInitialize handles the MCP initialize request.
func (d *Daemon) handleInitialize() (interface{}, *agent.Error) {
	return agent.InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: agent.ServerCapabilities{
			Tools: &agent.ToolsCapability{},
		},
		ServerInfo: agent.ServerInfo{
			Name:    "vibium",
			Version: d.version,
		},
	}, nil
}

// handleToolsCall executes a tool and returns the result.
func (d *Daemon) handleToolsCall(params json.RawMessage, notifyLaunch func()) (interface{}, *agent.Error) {
	var p agent.ToolsCallParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &agent.Error{
			Code:    agent.InvalidParams,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	// Serialize handler access — handlers are not thread-safe. The launch
	// callback targets this request's connection, so it is installed and
	// cleared under the same lock.
	d.mu.Lock()
	d.handlers.SetLaunchNotify(notifyLaunch)
	result, err := d.handlers.Call(p.Name, p.Arguments)
	d.handlers.SetLaunchNotify(nil)
	d.mu.Unlock()

	if err != nil {
		return agent.ToolsCallResult{
			Content: []agent.Content{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	return result, nil
}

func pidSelf() int {
	return os.Getpid()
}
