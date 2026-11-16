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

// Chrome rejects a script queued behind a navigation once, without running it (#811).
func TestPageEvalIsResentWhenNavigationReplacedItsRealm(t *testing.T) {
	upgrader := websocket.Upgrader{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		evals := 0
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
			reply := `{"id":` + id + `,"type":"success","result":{}}`
			switch cmd.Method {
			case "session.status":
				reply = `{"id":` + id + `,"type":"success","result":{"ready":false,"message":"already has session"}}`
			case "script.evaluate":
				evals++
				if evals == 1 {
					reply = `{"id":` + id + `,"type":"error","error":"unknown error","message":"unknown error: Cannot find context with specified id"}`
				} else {
					reply = `{"id":` + id + `,"type":"success","result":{"type":"success","result":{"type":"string","value":"Next"},"realm":"r2"}}`
				}
			}
			conn.WriteMessage(websocket.TextMessage, []byte(reply))
		}
	}))
	defer ts.Close()

	router := NewRouter("chrome", true, "ws"+strings.TrimPrefix(ts.URL, "http"), nil, nil)
	client := &recordingClient{}
	t.Cleanup(router.CloseAll)

	router.OnClientConnect(client)
	router.OnClientMessage(client, `{"id":7,"method":"vibium:page.eval","params":{"context":"ctx","expression":"document.title"}}`)

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
			if !strings.Contains(got, `"success"`) || !strings.Contains(got, "Next") {
				t.Fatalf("eval should answer from the new document, got %s", got)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("no response with id 7 reached the client")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
