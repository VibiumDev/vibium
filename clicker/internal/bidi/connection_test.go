package bidi

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// A browser that has stopped reading its socket must not block Send forever:
// the write deadline turns the stall into a bounded error so the stdin scanner
// goroutine that forwards standard commands is never wedged (#397). The server
// upgrades the connection and then never reads, so once the OS send buffer
// fills, WriteMessage blocks until the deadline.
func TestSendWriteDeadlineUnblocksAStalledBrowser(t *testing.T) {
	upgrader := websocket.Upgrader{}
	stop := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		// Never read: let the peer's send buffer fill. Hold the connection
		// open until the test is done so the block is backpressure, not a
		// closed socket.
		<-stop
	}))
	defer srv.Close()
	defer close(stop)

	wsURL := "ws://" + strings.TrimPrefix(srv.URL, "http://")
	conn, err := Connect(wsURL)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer conn.Close()

	conn.writeDeadline = 500 * time.Millisecond

	// Big payloads fill the socket buffer faster than a trickle of small ones.
	payload := strings.Repeat("x", 1<<20)

	done := make(chan error, 1)
	go func() {
		// The first writes succeed into the kernel buffer; one eventually
		// blocks and must then fail on the deadline rather than hang.
		for i := 0; i < 256; i++ {
			if err := conn.Send(payload); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Send never blocked; test did not exercise the deadline")
		}
		netErr, ok := err.(interface{ Timeout() bool })
		if !ok || !netErr.Timeout() {
			t.Fatalf("Send error = %v, want a write timeout", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Send blocked past the write deadline — the stall was not bounded")
	}
}

// A failed write fails the connection closed so a reader parked in Receive
// (a frozen browser never trips its own read deadline) unblocks and the
// session can tear down (#397). Without this the write deadline alone would
// leave the reader blocked until the 120s read deadline.
func TestFailedWriteUnblocksReceive(t *testing.T) {
	upgrader := websocket.Upgrader{}
	stop := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		<-stop // never read, never write
	}))
	defer srv.Close()
	defer close(stop)

	conn, err := Connect("ws://" + strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer conn.Close()
	conn.writeDeadline = 500 * time.Millisecond

	// A reader is parked in Receive; the server never sends, so only the
	// write failure below can unblock it before the 120s read deadline.
	recvErr := make(chan error, 1)
	go func() {
		_, err := conn.Receive()
		recvErr <- err
	}()

	payload := strings.Repeat("x", 1<<20)
	go func() {
		for i := 0; i < 256; i++ {
			if conn.Send(payload) != nil {
				return
			}
		}
	}()

	select {
	case <-recvErr:
		// Unblocked by the write failure — as intended.
	case <-time.After(10 * time.Second):
		t.Fatal("Receive stayed blocked after a failed write; session cannot tear down")
	}

	if !conn.closed.Load() {
		t.Error("connection not marked closed after a failed write")
	}
}

// With a reader draining the socket, Send stays fast and never trips the
// deadline — the healthy path is unaffected.
func TestSendSucceedsWhenBrowserReads(t *testing.T) {
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	conn, err := Connect("ws://" + u.Host)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer conn.Close()

	conn.writeDeadline = 500 * time.Millisecond
	payload := strings.Repeat("x", 1<<20)
	for i := 0; i < 256; i++ {
		if err := conn.Send(payload); err != nil {
			t.Fatalf("Send to a reading peer failed: %v", err)
		}
	}
}
