package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
)

// Firefox does not implement browsingContext.navigationAborted and rejects
// any session.subscribe batch containing it with "invalid argument". The
// router batches all its events into one subscribe, so that one name used to
// take down every subscription: no console, download, popup, dialog, network,
// or navigation events ever arrived on the Firefox path (#348, #323, #324).
// This pins the fallback: after a rejected batch, every event the browser
// does accept must still get subscribed individually.
func TestSubscribeFallsBackWhenBatchIsRejected(t *testing.T) {
	var mu sync.Mutex
	subscribed := map[string]bool{}

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
				Params struct {
					Events []string `json:"events"`
				} `json:"params"`
			}
			if json.Unmarshal(raw, &cmd) != nil {
				continue
			}

			// ready:false makes the router attach to this endpoint's session
			// rather than create one, keeping the handshake to one command.
			result := `{}`
			reject := false
			if cmd.Method == "session.status" {
				result = `{"ready":false,"message":"already has session"}`
			}
			if cmd.Method == "session.subscribe" {
				for _, ev := range cmd.Params.Events {
					if ev == "browsingContext.navigationAborted" {
						reject = true
					}
				}
				if !reject {
					mu.Lock()
					for _, ev := range cmd.Params.Events {
						subscribed[ev] = true
					}
					mu.Unlock()
				}
			}

			id := strconv.FormatInt(cmd.ID, 10)
			if reject {
				conn.WriteMessage(websocket.TextMessage,
					[]byte(`{"id":`+id+`,"type":"error","error":"invalid argument","message":"browsingContext.navigationAborted is not a valid event name"}`))
				continue
			}
			conn.WriteMessage(websocket.TextMessage,
				[]byte(`{"id":`+id+`,"type":"success","result":`+result+`}`))
		}
	}))
	defer ts.Close()

	router := NewRouter("firefox", true, "ws"+strings.TrimPrefix(ts.URL, "http"), nil, nil)
	client := &recordingClient{}
	t.Cleanup(router.CloseAll)

	router.OnClientConnect(client)

	if _, ok := router.sessions.Load(client.ID()); !ok {
		t.Fatal("router has no session for the client")
	}

	wanted := []string{
		"browsingContext.contextCreated",
		"network.beforeRequestSent",
		"network.responseCompleted",
		"browsingContext.userPromptOpened",
		"browsingContext.userPromptClosed",
		"log.entryAdded",
		"browsingContext.downloadWillBegin",
		"browsingContext.downloadEnd",
		"browsingContext.load",
		"browsingContext.navigationStarted",
		"browsingContext.navigationFailed",
		"browsingContext.fragmentNavigated",
		"browsingContext.historyUpdated",
	}
	mu.Lock()
	defer mu.Unlock()
	for _, ev := range wanted {
		if !subscribed[ev] {
			t.Errorf("event %s was never subscribed after the batch was rejected", ev)
		}
	}
	if subscribed["browsingContext.navigationAborted"] {
		t.Error("browser recorded a subscription for the event it rejects")
	}
}
