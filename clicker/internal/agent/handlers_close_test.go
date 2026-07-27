package agent

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/vibium/clicker/internal/bidi"
)

// TestConcurrentClose reproduces the MCP shutdown race: the cleanup goroutine
// calls Close while browser_quit's Close is in flight on the serve loop.
// Meaningful under -race, where unsynchronized session-field access fails it.
func TestConcurrentClose(t *testing.T) {
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	h := NewHandlers("", true, "", nil)
	conn, err := bidi.Connect("ws" + strings.TrimPrefix(srv.URL, "http"))
	if err != nil {
		t.Fatalf("connect to fake server: %v", err)
	}
	h.conn = conn
	h.client = bidi.NewClient(conn)

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.Close()
		}()
	}
	wg.Wait()

	if h.conn != nil || h.client != nil || h.launchResult != nil {
		t.Fatal("session fields not cleared after concurrent Close")
	}
}
