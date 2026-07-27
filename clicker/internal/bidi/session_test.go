package bidi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// testCommand is the shape of commands as the fake server sees them.
type testCommand struct {
	ID     int64                  `json:"id"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

// testServerConn wraps a server-side websocket with a write mutex so handler
// goroutines can interleave writes safely.
type testServerConn struct {
	ws *websocket.Conn
	mu sync.Mutex
}

func (s *testServerConn) writeJSON(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Write errors during teardown are expected; callers assert on the
	// client side instead.
	s.ws.WriteMessage(websocket.TextMessage, data)
}

func (s *testServerConn) readCommand() (testCommand, error) {
	var cmd testCommand
	_, data, err := s.ws.ReadMessage()
	if err != nil {
		return cmd, err
	}
	if err := json.Unmarshal(data, &cmd); err != nil {
		return cmd, err
	}
	return cmd, nil
}

func (s *testServerConn) respond(id int64, result map[string]interface{}) {
	s.writeJSON(map[string]interface{}{"type": "success", "id": id, "result": result})
}

func (s *testServerConn) event(method string, params map[string]interface{}) {
	s.writeJSON(map[string]interface{}{"type": "event", "method": method, "params": params})
}

// newTestClient starts an in-process websocket server driven by handler and
// returns a Client connected to it.
func newTestClient(t *testing.T, handler func(*testServerConn)) *Client {
	t.Helper()

	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		handler(&testServerConn{ws: ws})
	}))
	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, err := Connect(url)
	if err != nil {
		t.Fatalf("connect to fake server: %v", err)
	}
	client := NewClient(conn)
	t.Cleanup(func() { client.Close() })
	return client
}

// TestConcurrentCommands sends commands from many goroutines while the server
// answers out of order and interleaves events. Every caller must get the
// response to its own command.
func TestConcurrentCommands(t *testing.T) {
	client := newTestClient(t, func(s *testServerConn) {
		for {
			cmd, err := s.readCommand()
			if err != nil {
				return
			}
			// Answer on a separate goroutine with an id-dependent delay so
			// responses arrive out of order relative to the sends.
			go func(cmd testCommand) {
				time.Sleep(time.Duration(cmd.ID%7) * time.Millisecond)
				s.event("log.entryAdded", map[string]interface{}{"text": "noise"})
				s.respond(cmd.ID, map[string]interface{}{"tag": cmd.Params["tag"]})
			}(cmd)
		}
	})

	const goroutines = 8
	const commands = 20

	var wg sync.WaitGroup
	errs := make(chan error, goroutines*commands)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < commands; i++ {
				tag := fmt.Sprintf("g%d-c%d", g, i)
				msg, err := client.SendCommandWithTimeout("test.echo", map[string]interface{}{"tag": tag}, 5*time.Second)
				if err != nil {
					errs <- fmt.Errorf("%s: %v", tag, err)
					return
				}
				var result struct {
					Tag string `json:"tag"`
				}
				if err := json.Unmarshal(msg.Result, &result); err != nil {
					errs <- fmt.Errorf("%s: parse result: %v", tag, err)
					return
				}
				if result.Tag != tag {
					errs <- fmt.Errorf("caller %s got response for %q", tag, result.Tag)
					return
				}
			}
		}(g)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

// TestBlockedCommandDoesNotBlockOthers withholds the response to one command
// until a second command has been sent and answered, proving a gated response
// (like one waiting on a native dialog) no longer wedges the connection.
func TestBlockedCommandDoesNotBlockOthers(t *testing.T) {
	var slowID int64
	slowSent := make(chan struct{})
	client := newTestClient(t, func(s *testServerConn) {
		for {
			cmd, err := s.readCommand()
			if err != nil {
				return
			}
			switch cmd.Method {
			case "test.slow":
				slowID = cmd.ID
				close(slowSent)
			case "test.fast":
				s.respond(cmd.ID, map[string]interface{}{"which": "fast"})
				s.respond(slowID, map[string]interface{}{"which": "slow"})
			}
		}
	})

	slowErr := make(chan error, 1)
	go func() {
		_, err := client.SendCommandWithTimeout("test.slow", nil, 5*time.Second)
		slowErr <- err
	}()

	select {
	case <-slowSent:
	case <-time.After(2 * time.Second):
		t.Fatal("server never received test.slow")
	}

	if _, err := client.SendCommandWithTimeout("test.fast", nil, 2*time.Second); err != nil {
		t.Fatalf("fast command blocked behind unanswered slow command: %v", err)
	}

	select {
	case err := <-slowErr:
		if err != nil {
			t.Fatalf("slow command failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("slow command never completed after its response was sent")
	}
}

// TestReaderDeathFailsPendingCommands closes the server mid-flight and
// expects every pending caller to fail promptly, later sends to fail
// immediately, and all client goroutines to exit.
func TestReaderDeathFailsPendingCommands(t *testing.T) {
	baseline := runtime.NumGoroutine()

	const pending = 4
	client := newTestClient(t, func(s *testServerConn) {
		for i := 0; i < pending; i++ {
			if _, err := s.readCommand(); err != nil {
				return
			}
		}
		s.ws.Close()
	})

	var wg sync.WaitGroup
	errCh := make(chan error, pending)
	for i := 0; i < pending; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.SendCommandWithTimeout("test.hang", nil, 30*time.Second)
			errCh <- err
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("pending commands did not fail after server closed the connection")
	}
	close(errCh)
	for err := range errCh {
		if err == nil {
			t.Error("pending command succeeded after connection death, want error")
		} else if !strings.Contains(err.Error(), "connection lost waiting for response") {
			t.Errorf("pending command error = %q, want it to say the connection was lost", err)
		}
	}

	start := time.Now()
	if _, err := client.SendCommand("test.after", nil); err == nil {
		t.Error("send after reader death succeeded, want error")
	} else if !strings.Contains(err.Error(), "cannot send") {
		t.Errorf("post-death send error = %q, want it to say the send was refused", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("send after reader death took %s, want immediate failure", elapsed)
	}

	// Teardown is asynchronous, so poll. The slack covers goroutines owned
	// by the still-running httptest server, not the client.
	deadline := time.Now().Add(5 * time.Second)
	for runtime.NumGoroutine() > baseline+3 {
		if time.Now().After(deadline) {
			buf := make([]byte, 1<<20)
			n := runtime.Stack(buf, true)
			t.Fatalf("goroutines leaked: baseline %d, now %d\n%s", baseline, runtime.NumGoroutine(), buf[:n])
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestLateResponseAfterTimeoutIsDiscarded times out a command, then has the
// server deliver the stale response while another command is in flight. The
// stale response must be dropped, not delivered to the wrong caller.
func TestLateResponseAfterTimeoutIsDiscarded(t *testing.T) {
	client := newTestClient(t, func(s *testServerConn) {
		var lateID int64
		for {
			cmd, err := s.readCommand()
			if err != nil {
				return
			}
			switch cmd.Method {
			case "test.late":
				lateID = cmd.ID
			case "test.flush":
				s.respond(lateID, map[string]interface{}{"which": "late"})
				s.respond(cmd.ID, map[string]interface{}{"which": "flush"})
			}
		}
	})

	if _, err := client.SendCommandWithTimeout("test.late", nil, 50*time.Millisecond); err == nil {
		t.Fatal("expected timeout error for withheld response")
	}

	msg, err := client.SendCommandWithTimeout("test.flush", nil, 2*time.Second)
	if err != nil {
		t.Fatalf("command after timeout failed: %v", err)
	}
	var result struct {
		Which string `json:"which"`
	}
	if err := json.Unmarshal(msg.Result, &result); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if result.Which != "flush" {
		t.Fatalf("got response %q, want %q", result.Which, "flush")
	}
}

// TestDroppedEventsCounted blocks the event handler and floods the client
// with more events than the buffer holds. The overflow must be counted, not
// silently discarded, and command responses must keep flowing.
func TestDroppedEventsCounted(t *testing.T) {
	const eventCount = eventBufferSize + 144

	client := newTestClient(t, func(s *testServerConn) {
		for {
			cmd, err := s.readCommand()
			if err != nil {
				return
			}
			if cmd.Method == "test.flood" {
				for i := 0; i < eventCount; i++ {
					s.event("test.event", map[string]interface{}{"seq": i})
				}
			}
			s.respond(cmd.ID, map[string]interface{}{})
		}
	})

	release := make(chan struct{})
	defer close(release)
	client.SetEventHandler(func(string) { <-release })

	// The response is written after the events, so once it arrives every
	// event has passed through the reader's buffer-or-drop select.
	if _, err := client.SendCommandWithTimeout("test.flood", nil, 5*time.Second); err != nil {
		t.Fatalf("command failed during event flood: %v", err)
	}

	// The blocked dispatcher holds at most one event beyond the full buffer.
	dropped := client.DroppedEvents()
	min := uint64(eventCount - eventBufferSize - 1)
	max := uint64(eventCount - eventBufferSize)
	if dropped < min || dropped > max {
		t.Fatalf("DroppedEvents() = %d, want between %d and %d", dropped, min, max)
	}
}

// TestFirstDropWarnsOnStderrOnce swaps stderr for a pipe and floods the
// client twice. The drop warning must appear exactly once — visible without
// verbose logging, but not repeated per drop or per storm.
func TestFirstDropWarnsOnStderrOnce(t *testing.T) {
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = origStderr }()

	const eventCount = eventBufferSize + 50
	client := newTestClient(t, func(s *testServerConn) {
		for {
			cmd, err := s.readCommand()
			if err != nil {
				return
			}
			if cmd.Method == "test.flood" {
				for i := 0; i < eventCount; i++ {
					s.event("test.event", map[string]interface{}{"seq": i})
				}
			}
			s.respond(cmd.ID, map[string]interface{}{})
		}
	})

	release := make(chan struct{})
	defer close(release)
	client.SetEventHandler(func(string) { <-release })

	// Two storms; each response arrives after its events, so all drops have
	// happened (and warned, if they were going to) before the read below.
	for i := 0; i < 2; i++ {
		if _, err := client.SendCommandWithTimeout("test.flood", nil, 5*time.Second); err != nil {
			t.Fatalf("flood %d failed: %v", i, err)
		}
	}
	if client.DroppedEvents() == 0 {
		t.Fatal("test did not force any drops")
	}

	os.Stderr = origStderr
	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read captured stderr: %v", err)
	}

	warns := strings.Count(string(out), "dropping events")
	if warns != 1 {
		t.Fatalf("drop warning appeared %d times, want exactly 1; stderr:\n%s", warns, out)
	}
}

// TestCloseShutsDownCleanly checks that Close returns only after the reader
// has exited, later sends fail immediately, and no goroutines are left.
func TestCloseShutsDownCleanly(t *testing.T) {
	baseline := runtime.NumGoroutine()

	client := newTestClient(t, func(s *testServerConn) {
		for {
			if _, err := s.readCommand(); err != nil {
				return
			}
		}
	})

	closed := make(chan error, 1)
	go func() { closed <- client.Close() }()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not return; reader never exited")
	}

	// Close waited on the reader, so readerDone is closed and the failure
	// path is immediate, not a timeout.
	start := time.Now()
	if _, err := client.SendCommand("test.after", nil); err == nil {
		t.Error("send after Close succeeded, want error")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("send after Close took %s, want immediate failure", elapsed)
	}

	// Same slack for httptest-server goroutines as the reader-death test.
	deadline := time.Now().Add(5 * time.Second)
	for runtime.NumGoroutine() > baseline+3 {
		if time.Now().After(deadline) {
			buf := make([]byte, 1<<20)
			n := runtime.Stack(buf, true)
			t.Fatalf("goroutines leaked after Close: baseline %d, now %d\n%s", baseline, runtime.NumGoroutine(), buf[:n])
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestEventsForwardedInOrder checks that events reach the handler in arrival
// order and that event delivery does not depend on a command being in flight.
func TestEventsForwardedInOrder(t *testing.T) {
	const eventCount = 50
	client := newTestClient(t, func(s *testServerConn) {
		for {
			cmd, err := s.readCommand()
			if err != nil {
				return
			}
			for i := 0; i < eventCount; i++ {
				s.event("test.event", map[string]interface{}{"seq": i})
			}
			s.respond(cmd.ID, map[string]interface{}{})
		}
	})

	var mu sync.Mutex
	var got []int
	allReceived := make(chan struct{})
	client.SetEventHandler(func(raw string) {
		var evt struct {
			Params struct {
				Seq int `json:"seq"`
			} `json:"params"`
		}
		if err := json.Unmarshal([]byte(raw), &evt); err != nil {
			return
		}
		mu.Lock()
		got = append(got, evt.Params.Seq)
		if len(got) == eventCount {
			close(allReceived)
		}
		mu.Unlock()
	})

	if _, err := client.SendCommandWithTimeout("test.go", nil, 2*time.Second); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	select {
	case <-allReceived:
	case <-time.After(2 * time.Second):
		mu.Lock()
		n := len(got)
		mu.Unlock()
		t.Fatalf("received %d of %d events", n, eventCount)
	}

	mu.Lock()
	defer mu.Unlock()
	for i, seq := range got {
		if seq != i {
			t.Fatalf("event %d arrived at position %d", seq, i)
		}
	}
}
