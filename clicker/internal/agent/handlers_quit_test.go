package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/vibium/clicker/internal/bidi"
)

// fakeBidiServer accepts a WebSocket and answers every command with a success
// envelope, so Close()'s session.end returns at once instead of waiting out
// the command timeout. Returns the ws:// URL.
func fakeBidiServer(t *testing.T) (string, func()) {
	t.Helper()
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		for {
			_, data, err := ws.ReadMessage()
			if err != nil {
				return
			}
			var cmd struct {
				ID int `json:"id"`
			}
			if json.Unmarshal(data, &cmd) == nil && cmd.ID > 0 {
				reply, _ := json.Marshal(map[string]interface{}{
					"id": cmd.ID, "type": "success", "result": map[string]interface{}{},
				})
				if ws.WriteMessage(websocket.TextMessage, reply) != nil {
					return
				}
			}
		}
	}))
	return "ws" + strings.TrimPrefix(srv.URL, "http"), srv.Close
}

func quitText(t *testing.T, h *Handlers) string {
	t.Helper()
	res, err := h.browserQuit(nil)
	if err != nil {
		t.Fatalf("browserQuit: %v", err)
	}
	if len(res.Content) == 0 {
		t.Fatal("browserQuit returned no content")
	}
	return res.Content[0].Text
}

// A session attached by URL (ownsRemote false, no classic session, no local
// launch) is left running on stop, so the message must say so rather than
// claim it closed the browser (#647).
func TestBrowserQuitAttachedSessionSaysDetached(t *testing.T) {
	url, stop := fakeBidiServer(t)
	defer stop()

	h := NewHandlers("", "chrome", true, url, nil, nil)
	conn, err := bidi.Connect(url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	h.conn = conn
	h.client = bidi.NewClient(conn)
	h.ownsRemote = false // attached, not created here

	text := quitText(t, h)
	if strings.Contains(text, "Browser session closed") {
		t.Errorf("attached session must not report closed, got %q", text)
	}
	if !strings.Contains(text, "left running") {
		t.Errorf("message should say the session was left running, got %q", text)
	}
}

// A remote session this process created (ownsRemote) is ended on stop, so the
// closed message is accurate.
func TestBrowserQuitOwnedRemoteSaysClosed(t *testing.T) {
	url, stop := fakeBidiServer(t)
	defer stop()

	h := NewHandlers("", "chrome", true, url, nil, nil)
	conn, err := bidi.Connect(url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	h.conn = conn
	h.client = bidi.NewClient(conn)
	h.ownsRemote = true // created here, so stop ends it

	if text := quitText(t, h); !strings.Contains(text, "Browser session closed") {
		t.Errorf("owned remote session should report closed, got %q", text)
	}
}
