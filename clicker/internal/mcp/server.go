// Package mcp implements the Model Context Protocol (MCP) server.
// It provides a JSON-RPC 2.0 interface over stdio for LLM agents.
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/vibium/clicker/internal/log"
)

// JSON-RPC 2.0 request structure
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"` // Can be string, number, or null
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSON-RPC 2.0 response structure
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// JSON-RPC 2.0 error structure
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// MCP-specific types

type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
}

type ClientCapabilities struct {
	Roots    *RootsCapability    `json:"roots,omitempty"`
	Sampling *SamplingCapability `json:"sampling,omitempty"`
}

type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type SamplingCapability struct{}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

type ServerCapabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type ToolsCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type ToolsCallResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

type Content struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`     // For images (base64)
	MimeType string `json:"mimeType,omitempty"` // For images
}

// Server is the MCP server that handles JSON-RPC over stdio.
type Server struct {
	reader   *bufio.Reader
	writer   io.Writer
	handlers *Handlers
	version  string
}

// ServerOptions configures the MCP server.
type ServerOptions struct {
	ScreenshotDir string // Directory for saving screenshots (empty = disabled)
}

// NewServer creates a new MCP server.
func NewServer(version string, opts ServerOptions) *Server {
	return &Server{
		reader:   bufio.NewReader(os.Stdin),
		writer:   os.Stdout,
		handlers: NewHandlers(opts.ScreenshotDir),
		version:  version,
	}
}

// Run starts the server loop, reading requests from stdin and writing responses to stdout.
func (s *Server) Run() error {
	for {
		line, err := s.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil // Clean exit
			}
			return fmt.Errorf("read error: %w", err)
		}

		// Skip empty lines
		if len(line) <= 1 {
			continue
		}

		response := s.handleRequest(line)
		if response != nil {
			if err := s.writeResponse(response); err != nil {
				return fmt.Errorf("write error: %w", err)
			}
		}
	}
}

// handleRequest parses and routes a JSON-RPC request.
func (s *Server) handleRequest(data []byte) *Response {
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return &Response{
			JSONRPC: "2.0",
			Error: &Error{
				Code:    ParseError,
				Message: "Parse error",
				Data:    err.Error(),
			},
		}
	}

	// Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    InvalidRequest,
				Message: "Invalid Request",
				Data:    "jsonrpc must be '2.0'",
			},
		}
	}

	// Route to handler
	result, err := s.route(req)

	// Notifications (no ID) don't get a response (even on error)
	if req.ID == nil {
		return nil
	}

	if err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   err,
		}
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// route dispatches requests to the appropriate handler.
func (s *Server) route(req Request) (interface{}, *Error) {
	log.Debug("mcp request", "method", req.Method, "id", req.ID)

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req.Params)
	case "initialized", "notifications/initialized":
		// Notification, no response needed
		return nil, nil
	case "tools/list":
		return s.handleToolsList()
	case "tools/call":
		return s.handleToolsCall(req.Params)
	default:
		return nil, &Error{
			Code:    MethodNotFound,
			Message: "Method not found",
			Data:    req.Method,
		}
	}
}

// handleInitialize handles the initialize request.
func (s *Server) handleInitialize(params json.RawMessage) (interface{}, *Error) {
	var p InitializeParams
	if params != nil {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &Error{
				Code:    InvalidParams,
				Message: "Invalid params",
				Data:    err.Error(),
			}
		}
	}

	return InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{},
		},
		ServerInfo: ServerInfo{
			Name:    "vibium",
			Version: s.version,
		},
	}, nil
}

// handleToolsList returns the list of available tools.
func (s *Server) handleToolsList() (interface{}, *Error) {
	return ToolsListResult{
		Tools: GetToolSchemas(),
	}, nil
}

// handleToolsCall executes a tool and returns the result.
func (s *Server) handleToolsCall(params json.RawMessage) (interface{}, *Error) {
	var p ToolsCallParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &Error{
			Code:    InvalidParams,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	result, err := s.handlers.Call(p.Name, p.Arguments)
	if err != nil {
		return ToolsCallResult{
			Content: []Content{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	return result, nil
}

// writeResponse writes a JSON-RPC response to stdout.
func (s *Server) writeResponse(resp *Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(s.writer, "%s\n", data)
	return err
}

// Close cleans up the server resources.
func (s *Server) Close() {
	s.handlers.Close()
}
