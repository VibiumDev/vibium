package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// An engine-level rejection arrives as a normal response whose payload is an
// error envelope, so it never touched sendInternalCommand's err. Handlers
// that only checked err answered the client with success while the engine
// had refused the command (#509). page.activate was one of them; the check
// now lives in the transport, so this pins the whole class.
func TestEngineRejectionReachesClient(t *testing.T) {
	upgrader := websocket.Upgrader{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var cmd struct {
				ID     int64  `json:"id"`
				Method string `json:"method"`
			}
			if json.Unmarshal(raw, &cmd) != nil {
				continue
			}

			id := strconv.FormatInt(cmd.ID, 10)
			// ready:false makes the router attach to this endpoint's session
			// rather than create one, keeping the handshake to one command.
			if cmd.Method == "session.status" {
				conn.WriteMessage(websocket.TextMessage,
					[]byte(`{"id":`+id+`,"type":"success","result":{"ready":false,"message":"already has session"}}`))
				continue
			}
			if cmd.Method == "browsingContext.activate" {
				conn.WriteMessage(websocket.TextMessage,
					[]byte(`{"id":`+id+`,"type":"error","error":"no such frame","message":"context gone was not found"}`))
				continue
			}
			conn.WriteMessage(websocket.TextMessage,
				[]byte(`{"id":`+id+`,"type":"success","result":{}}`))
		}
	}))
	defer ts.Close()

	router := NewRouter("chrome", true, "ws"+strings.TrimPrefix(ts.URL, "http"), nil, nil)
	client := &recordingClient{}
	t.Cleanup(router.CloseAll)

	router.OnClientConnect(client)
	router.OnClientMessage(client, `{"id":7,"method":"vibium:page.activate","params":{"context":"gone"}}`)

	deadline := time.Now().Add(5 * time.Second)
	for {
		var got string
		client.mu.Lock()
		for _, msg := range client.sent {
			var resp struct {
				ID int64 `json:"id"`
			}
			if json.Unmarshal([]byte(msg), &resp) == nil && resp.ID == 7 {
				got = msg
			}
		}
		client.mu.Unlock()

		if got != "" {
			var resp struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal([]byte(got), &resp); err != nil {
				t.Fatalf("unparseable response: %s", got)
			}
			if resp.Type != "error" {
				t.Fatalf("engine rejected the activate but the client got %s", got)
			}
			if !strings.Contains(resp.Message, "context gone was not found") {
				t.Fatalf("error response should carry the engine's message, got %s", got)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("no response with id 7 reached the client")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
