package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// newTestClientConn spins up a real WebSocket pair and returns the server-side
// ClientConn plus the raw client socket, so tests can stop reading on purpose.
func newTestClientConn(t *testing.T) (*ClientConn, *websocket.Conn) {
	t.Helper()

	var (
		mu     sync.Mutex
		server *ClientConn
		ready  = make(chan struct{})
	)

	upgrader := websocket.Upgrader{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		c := &ClientConn{id: 1, conn: conn}
		c.startWriter()
		mu.Lock()
		server = c
		mu.Unlock()
		close(ready)
		// Hold the handler open for the life of the test.
		<-r.Context().Done()
	}))
	t.Cleanup(ts.Close)

	dialer := websocket.Dialer{}
	client, _, err := dialer.Dial("ws"+strings.TrimPrefix(ts.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("server side never accepted the connection")
	}

	mu.Lock()
	defer mu.Unlock()
	return server, client
}

// A client that stops reading must not block the caller. The browser→client
// pump calls Send for every forwarded message, so a synchronous write here
// froze the whole session once the socket buffers filled (issue #232).
func TestSendDoesNotBlockOnStalledClient(t *testing.T) {
	server, _ := newTestClientConn(t)

	// Never read from the client socket. 12MB comfortably exceeds any
	// kernel/gorilla buffering, so the writer goroutine will be stuck.
	payload := strings.Repeat("x", 64*1024)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			if err := server.Send(payload); err != nil {
				return
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Send blocked on a client that stopped reading")
	}
}

func TestSendDeliversInOrder(t *testing.T) {
	server, client := newTestClientConn(t)

	const n = 50
	for i := 0; i < n; i++ {
		if err := server.Send(string(rune('a'+i%26)) + "-msg"); err != nil {
			t.Fatalf("Send %d: %v", i, err)
		}
	}

	client.SetReadDeadline(time.Now().Add(10 * time.Second))
	for i := 0; i < n; i++ {
		_, msg, err := client.ReadMessage()
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		want := string(rune('a'+i%26)) + "-msg"
		if string(msg) != want {
			t.Fatalf("message %d = %q, want %q (ordering must be preserved)", i, msg, want)
		}
	}
}

func TestSendAfterCloseFails(t *testing.T) {
	server, _ := newTestClientConn(t)

	if err := server.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := server.Send("late"); err == nil {
		t.Error("Send after Close should fail")
	}
	// Close must be idempotent — the router calls it on teardown paths that
	// can overlap.
	if err := server.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

func TestCloseDrainsQueuedMessages(t *testing.T) {
	server, client := newTestClientConn(t)

	for i := 0; i < 20; i++ {
		if err := server.Send("queued"); err != nil {
			t.Fatalf("Send %d: %v", i, err)
		}
	}

	closed := make(chan error, 1)
	go func() { closed <- server.Close() }()

	client.SetReadDeadline(time.Now().Add(10 * time.Second))
	for i := 0; i < 20; i++ {
		if _, _, err := client.ReadMessage(); err != nil {
			t.Fatalf("queued message %d was lost: %v", i, err)
		}
	}

	select {
	case err := <-closed:
		if err != nil {
			t.Fatalf("Close: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Close did not return after the queue drained")
	}
}
