package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// maxMessageSize is the maximum size of a WebSocket message (10MB).
// This accommodates large screenshots from high-resolution displays (e.g., retina, 4K).
const maxMessageSize = 10 * 1024 * 1024

// clientReadDeadline is the timeout for reading from a client WebSocket.
// Generous since clients may be idle between commands.
const clientReadDeadline = 300 * time.Second

// ClientTransport is the interface that both WebSocket and pipe transports implement.
type ClientTransport interface {
	ID() uint64
	Send(msg string) error
	Close() error
}

// Server is a WebSocket server that accepts client connections.
type Server struct {
	host       string
	port       int
	httpServer *http.Server
	upgrader   websocket.Upgrader
	clients    sync.Map // map[uint64]*ClientConn
	nextID     atomic.Uint64
	onConnect  func(ClientTransport)
	onMessage  func(ClientTransport, string)
	onClose    func(ClientTransport)
}

// ClientConn represents a connected WebSocket client.
type ClientConn struct {
	id     uint64
	conn   *websocket.Conn
	mu     sync.Mutex
	closed bool
	server *Server
}

// ID returns the client connection ID.
func (c *ClientConn) ID() uint64 {
	return c.id
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
func WithOnConnect(fn func(ClientTransport)) ServerOption {
	return func(s *Server) {
		s.onConnect = fn
	}
}

// WithOnMessage sets a callback for when a message is received.
func WithOnMessage(fn func(ClientTransport, string)) ServerOption {
	return func(s *Server) {
		s.onMessage = fn
	}
}

// WithOnClose sets a callback for when a client disconnects.
func WithOnClose(fn func(ClientTransport)) ServerOption {
	return func(s *Server) {
		s.onClose = fn
	}
}

// NewServer creates a new WebSocket server.
func NewServer(opts ...ServerOption) *Server {
	s := &Server{
		host: "127.0.0.1",
		port: 9515, // default port
		upgrader: websocket.Upgrader{
			ReadBufferSize:  maxMessageSize,
			WriteBufferSize: maxMessageSize,
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	s.upgrader.CheckOrigin = s.checkOrigin

	return s
}

func (s *Server) listenAddr() string {
	return fmt.Sprintf("%s:%d", s.host, s.port)
}

func (s *Server) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}

	originHost := parsed.Hostname()
	if originHost == "" {
		return false
	}

	if isLoopbackHost(s.host) {
		return isLoopbackHost(originHost)
	}

	return strings.EqualFold(originHost, s.host)
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// Port returns the port the server is listening on.
func (s *Server) Port() int {
	return s.port
}

// Start starts the WebSocket server.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleWebSocket)

	// Bind to the port (port 0 = OS-assigned random port)
	listener, err := net.Listen("tcp", s.listenAddr())
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
		fmt.Fprintf(os.Stderr, "WebSocket upgrade error: %v\n", err)
		return
	}

	// Set read limit to handle large messages (e.g., screenshots from high-res displays)
	conn.SetReadLimit(maxMessageSize)

	client := &ClientConn{
		id:     s.nextID.Add(1),
		conn:   conn,
		server: s,
	}

	s.clients.Store(client.id, client)
	fmt.Fprintf(os.Stderr, "[proxy] Client %d connected from %s\n", client.id, r.RemoteAddr)

	if s.onConnect != nil {
		s.onConnect(client)
	}

	// Handle messages in this goroutine
	s.handleClient(client)
}

func (s *Server) handleClient(client *ClientConn) {
	defer func() {
		s.clients.Delete(client.id)
		client.Close()
		fmt.Fprintf(os.Stderr, "[proxy] Client %d disconnected\n", client.id)
		if s.onClose != nil {
			s.onClose(client)
		}
	}()

	// Set up pong handler to extend read deadline on active connections
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(clientReadDeadline))
		return nil
	})

	for {
		// Set read deadline to detect dead client connections
		client.conn.SetReadDeadline(time.Now().Add(clientReadDeadline))
		msgType, msg, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				fmt.Fprintf(os.Stderr, "[proxy] Client %d read error: %v\n", client.id, err)
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
