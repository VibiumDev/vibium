package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// The browser has one id namespace, and a nested vibium client numbers its
// commands from the same base as the router's internal counter. Forwarded
// verbatim, a client id can equal an in-flight internal command's id, and the
// browser's response then resolves the wrong waiter (#723). The router must
// renumber forwarded commands and restore the client's id on the response.
func TestForwardedClientIDCannotCollideWithInternalCommands(t *testing.T) {
	gotEval := make(chan struct{})
	upgrader := websocket.Upgrader{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		var heldEvalID int64 = -1
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

			switch cmd.Method {
			case "session.status":
				// ready:false makes the router attach to this endpoint's
				// session rather than create one.
				conn.WriteMessage(websocket.TextMessage,
					[]byte(`{"id":`+strconv.FormatInt(cmd.ID, 10)+`,"type":"success","result":{"ready":false,"message":"already has session"}}`))
			case "script.evaluate":
				// The client's forwarded command: hold its response until the
				// colliding internal command has arrived.
				heldEvalID = cmd.ID
				close(gotEval)
			case "browsingContext.getTree":
				// Answer the held client command first, then getTree. With a
				// shared id both responses carry the same number, and the
				// first one lands in the internal waiter.
				conn.WriteMessage(websocket.TextMessage,
					[]byte(`{"id":`+strconv.FormatInt(heldEvalID, 10)+`,"type":"success","result":{"value":"client"}}`))
				conn.WriteMessage(websocket.TextMessage,
					[]byte(`{"id":`+strconv.FormatInt(cmd.ID, 10)+`,"type":"success","result":{"contexts":[{"context":"ctx-1"}]}}`))
			default:
				conn.WriteMessage(websocket.TextMessage,
					[]byte(`{"id":`+strconv.FormatInt(cmd.ID, 10)+`,"type":"success","result":{}}`))
			}
		}
	}))
	defer ts.Close()

	router := NewRouter("chrome", true, "ws"+strings.TrimPrefix(ts.URL, "http"), nil, nil)
	client := &recordingClient{}
	t.Cleanup(router.CloseAll)
	router.OnClientConnect(client)

	val, ok := router.sessions.Load(client.ID())
	if !ok {
		t.Fatal("router has no session for the client")
	}
	session := val.(*BrowserSession)

	// The nested-vibium shape: the client picks exactly the id the router's
	// internal counter will hand out next.
	session.internalCmdsMu.Lock()
	n := session.nextInternalID
	session.internalCmdsMu.Unlock()

	router.OnClientMessage(client, fmt.Sprintf(`{"id":%d,"method":"script.evaluate","params":{}}`, n))
	select {
	case <-gotEval:
	case <-time.After(10 * time.Second):
		t.Fatal("forwarded command never reached the browser")
	}

	resp, err := router.sendInternalCommand(session, "browsingContext.getTree", map[string]interface{}{})
	if err != nil {
		t.Fatalf("internal getTree failed: %v", err)
	}
	var tree struct {
		Result struct {
			Contexts []struct {
				Context string `json:"context"`
			} `json:"contexts"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &tree); err != nil || len(tree.Result.Contexts) == 0 {
		t.Errorf("internal command received the client's response instead of its own: %s", resp)
	}

	// The client must get its own response back, under its own id.
	want := fmt.Sprintf(`"id":%d`, n)
	deadline := time.Now().Add(5 * time.Second)
	for {
		var got string
		client.mu.Lock()
		for _, msg := range client.sent {
			if strings.Contains(msg, want) && strings.Contains(msg, `"type":"success"`) {
				got = msg
			}
		}
		client.mu.Unlock()
		if got != "" {
			if !strings.Contains(got, `"value":"client"`) {
				t.Fatalf("client received someone else's response for its id: %s", got)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("client never received the response to its forwarded command")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// A response id may only be claimed once. The reader used to send onto the
// cap-1 internal channel while the map entry stayed live, so a duplicate id
// blocked the session's only reader goroutine forever (#723). Deleting the
// entry before the send makes a second match impossible.
func TestDuplicateResponseIDDoesNotBlockTheReader(t *testing.T) {
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

			switch cmd.Method {
			case "session.status":
				conn.WriteMessage(websocket.TextMessage,
					[]byte(`{"id":`+strconv.FormatInt(cmd.ID, 10)+`,"type":"success","result":{"ready":false,"message":"already has session"}}`))
			case "test.dup":
				// Two responses for one pending internal id, then an event
				// that proves the reader is still alive.
				conn.WriteMessage(websocket.TextMessage, []byte(`{"id":777,"type":"success","result":{}}`))
				conn.WriteMessage(websocket.TextMessage, []byte(`{"id":777,"type":"success","result":{}}`))
				conn.WriteMessage(websocket.TextMessage,
					[]byte(`{"method":"browsingContext.load","params":{"context":"c1","url":"http://example.test/","navigation":"nav-1"}}`))
			default:
				conn.WriteMessage(websocket.TextMessage,
					[]byte(`{"id":`+strconv.FormatInt(cmd.ID, 10)+`,"type":"success","result":{}}`))
			}
		}
	}))
	defer ts.Close()

	router := NewRouter("chrome", true, "ws"+strings.TrimPrefix(ts.URL, "http"), nil, nil)
	client := &recordingClient{}
	t.Cleanup(router.CloseAll)
	router.OnClientConnect(client)

	val, ok := router.sessions.Load(client.ID())
	if !ok {
		t.Fatal("router has no session for the client")
	}
	session := val.(*BrowserSession)

	// A pending internal command whose waiter has not yet consumed its
	// response — the state a colliding or duplicated id hits.
	session.internalCmdsMu.Lock()
	session.internalCmds[777] = make(chan json.RawMessage, 1)
	session.internalCmdsMu.Unlock()

	router.OnClientMessage(client, `{"id":1,"method":"test.dup","params":{}}`)

	deadline := time.Now().Add(5 * time.Second)
	for {
		client.mu.Lock()
		alive := false
		for _, msg := range client.sent {
			if strings.Contains(msg, "browsingContext.load") {
				alive = true
			}
		}
		client.mu.Unlock()
		if alive {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("reader never delivered the event after a duplicate response id; the cap-1 send is blocking the loop")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
