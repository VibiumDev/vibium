package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
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
	port       int
	httpServer *http.Server
	upgrader   websocket.Upgrader
	clients    sync.Map // map[uint64]*ClientConn
	nextID     atomic.Uint64
	onConnect  func(ClientTransport)
	onMessage  func(ClientTransport, string)
	onClose    func(ClientTransport)
}

// clientWriteDeadline bounds a single socket write. Without it a client that
// stops reading blocks the writer forever and the queue grows without limit.
const clientWriteDeadline = 60 * time.Second

// ClientConn represents a connected WebSocket client.
//
// Sends are queued and drained by a single writer goroutine, with the socket
// write performed outside the lock. The browser→client pump calls Send for
// every forwarded message, so a synchronous write here would let one client
// that stops reading stall the whole session.
type ClientConn struct {
	id     uint64
	conn   *websocket.Conn
	mu     sync.Mutex // guards queue, closed, writeStart
	cond   *sync.Cond
	queue  []string
	closed bool
	done   chan struct{}
	server *Server

	writeStart int64 // unix nanos of the in-progress write, 0 when idle
}

// startWriter begins draining queued messages. Called once per connection.
func (c *ClientConn) startWriter() {
	c.cond = sync.NewCond(&c.mu)
	c.done = make(chan struct{})
	go c.writeLoop()
	go c.watchStalledWrite()
}

// writeLoop drains the queue, writing with the lock released so a stalled
// client never blocks Send callers.
func (c *ClientConn) writeLoop() {
	defer close(c.done)
	for {
		c.mu.Lock()
		for len(c.queue) == 0 && !c.closed {
			c.cond.Wait()
		}
		if len(c.queue) == 0 && c.closed {
			c.mu.Unlock()
			return
		}
		batch := c.queue
		c.queue = nil
		c.writeStart = time.Now().UnixNano()
		c.mu.Unlock()

		var err error
		for _, msg := range batch {
			c.conn.SetWriteDeadline(time.Now().Add(clientWriteDeadline))
			if err = c.conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				break
			}
		}

		c.mu.Lock()
		c.writeStart = 0
		if err != nil {
			// The client is gone or wedged — stop accepting messages so the
			// queue can't grow unboundedly against a reader that never drains.
			c.closed = true
			c.mu.Unlock()
			fmt.Fprintf(os.Stderr, "[proxy] Client %d write failed, dropping connection: %v\n", c.id, err)
			c.conn.Close()
			return
		}
		c.mu.Unlock()
	}
}

// watchStalledWrite reports a write that is currently blocked — the signature
// of a client that stopped reading its socket.
func (c *ClientConn) watchStalledWrite() {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-c.done:
			return
		case <-t.C:
			c.mu.Lock()
			start := c.writeStart
			depth := len(c.queue)
			c.mu.Unlock()
			if start != 0 {
				if blocked := time.Since(time.Unix(0, start)); blocked > 5*time.Second {
					fmt.Fprintf(os.Stderr, "[proxy] client %d write blocked for %.0fs (%d messages queued) — client not reading\n",
						c.id, blocked.Seconds(), depth)
				}
			}
		}
	}
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
	client.startWriter()

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

// Send queues a message for the client. It never blocks and never drops.
func (c *ClientConn) Send(msg string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("connection closed")
	}

	c.queue = append(c.queue, msg)
	c.cond.Signal()
	return nil
}

// Close stops accepting messages, waits for the queue to drain, then closes
// the socket.
func (c *ClientConn) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.cond.Signal()
	c.mu.Unlock()

	<-c.done

	c.conn.SetWriteDeadline(time.Now().Add(clientWriteDeadline))
	c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	return c.conn.Close()
}
