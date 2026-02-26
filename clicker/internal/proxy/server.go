package proxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

// maxMessageSize is the maximum size of a WebSocket message (10MB).
// This accommodates large screenshots from high-resolution displays (e.g., retina, 4K).
const maxMessageSize = 10 * 1024 * 1024

// Server is a WebSocket server that accepts client connections.
type Server struct {
	port       int
	httpServer *http.Server
	upgrader   websocket.Upgrader
	clients    sync.Map // map[uint64]*ClientConn
	nextID     atomic.Uint64
	onConnect  func(*ClientConn)
	onMessage  func(*ClientConn, string)
	onClose    func(*ClientConn)
}

// ClientConn represents a connected WebSocket client.
type ClientConn struct {
	ID     uint64
	conn   *websocket.Conn
	mu     sync.Mutex
	closed bool
	server *Server
}

// ServerOption configures a Server.
type ServerOption func(*Server)

// WithPort sets the port for the server.
func WithPort(port int) ServerOption {
	return func(s *Server) {
		s.port = port
	}
}

// WithOnConnect sets a callback for when a client connects.
func WithOnConnect(fn func(*ClientConn)) ServerOption {
	return func(s *Server) {
		s.onConnect = fn
	}
}

// WithOnMessage sets a callback for when a message is received.
func WithOnMessage(fn func(*ClientConn, string)) ServerOption {
	return func(s *Server) {
		s.onMessage = fn
	}
}

// WithOnClose sets a callback for when a client disconnects.
func WithOnClose(fn func(*ClientConn)) ServerOption {
	return func(s *Server) {
		s.onClose = fn
	}
}

// NewServer creates a new WebSocket server.
func NewServer(opts ...ServerOption) *Server {
	s := &Server{
		port: 9515, // default port
		upgrader: websocket.Upgrader{
			ReadBufferSize:  maxMessageSize,
			WriteBufferSize: maxMessageSize,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins
			},
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Port returns the port the server is listening on.
func (s *Server) Port() int {
	return s.port
}

// Start starts the WebSocket server.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleWebSocket)

	addr := fmt.Sprintf(":%d", s.port)

	// Bind to the port (port 0 = OS-assigned random port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.port, err)
	}

	// Store actual port (important when port=0 for OS-assigned)
	s.port = listener.Addr().(*net.TCPAddr).Port

	s.httpServer = &http.Server{
		Handler: mux,
	}

	// Serve using the listener
	go s.httpServer.Serve(listener)

	return nil
}

// Stop stops the WebSocket server gracefully.
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}

	// Close all client connections
	s.clients.Range(func(key, value interface{}) bool {
		if client, ok := value.(*ClientConn); ok {
			client.Close()
		}
		return true
	})

	return s.httpServer.Shutdown(ctx)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("WebSocket upgrade error: %v\n", err)
		return
	}

	// Set read limit to handle large messages (e.g., screenshots from high-res displays)
	conn.SetReadLimit(maxMessageSize)

	client := &ClientConn{
		ID:     s.nextID.Add(1),
		conn:   conn,
		server: s,
	}

	s.clients.Store(client.ID, client)
	fmt.Printf("[proxy] Client %d connected from %s\n", client.ID, r.RemoteAddr)

	if s.onConnect != nil {
		s.onConnect(client)
	}

	// Handle messages in this goroutine
	s.handleClient(client)
}

func (s *Server) handleClient(client *ClientConn) {
	defer func() {
		s.clients.Delete(client.ID)
		client.Close()
		fmt.Printf("[proxy] Client %d disconnected\n", client.ID)
		if s.onClose != nil {
			s.onClose(client)
		}
	}()

	for {
		msgType, msg, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				fmt.Printf("[proxy] Client %d read error: %v\n", client.ID, err)
			}
			return
		}

		if msgType != websocket.TextMessage {
			continue
		}

		if s.onMessage != nil {
			s.onMessage(client, string(msg))
		}
	}
}

// Send sends a text message to the client.
func (c *ClientConn) Send(msg string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("connection closed")
	}

	return c.conn.WriteMessage(websocket.TextMessage, []byte(msg))
}

// Close closes the client connection.
func (c *ClientConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	// Send close message
	c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	return c.conn.Close()
}
