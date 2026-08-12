package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Download behavior used to be established in a bare goroutine, so a client whose
// first command started a download could be served first: the file landed in the
// browser's own download directory, outside the session's temp dir, where
// download.saveAs could not find it (#351).
//
// OnClientConnect runs before either transport starts reading client messages
// (server.handleWebSocket, cmd/clicker/pipe.go), which is why session.subscribe
// is synchronous there. This pins that setupDownloads now shares that guarantee.
//
// The proof is the held response, not timing: the browser reports when
// browser.setDownloadBehavior arrives and then answers nothing until the test
// says so. Only a synchronous setup can still be inside OnClientConnect at that
// point, so a backgrounded one fails here however the scheduler behaves.
func TestDownloadBehaviorIsEstablishedDuringConnect(t *testing.T) {
	arrived := make(chan struct{}) // closed when setDownloadBehavior reaches the browser
	release := make(chan struct{}) // closed by the test to let it be answered

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

			// ready:false makes the router attach to this endpoint's session
			// rather than create one, keeping the handshake to one command.
			result := `{}`
			if cmd.Method == "session.status" {
				result = `{"ready":false,"message":"already has session"}`
			}
			if cmd.Method == "browser.setDownloadBehavior" {
				close(arrived)
				<-release
			}

			conn.WriteMessage(websocket.TextMessage,
				[]byte(`{"id":`+strconv.FormatInt(cmd.ID, 10)+`,"type":"success","result":`+result+`}`))
		}
	}))
	defer ts.Close()

	router := NewRouter("chrome", true, "ws"+strings.TrimPrefix(ts.URL, "http"), nil)
	client := &recordingClient{}
	t.Cleanup(router.CloseAll)

	connected := make(chan struct{})
	go func() {
		router.OnClientConnect(client)
		close(connected)
	}()

	select {
	case <-arrived:
	case <-time.After(10 * time.Second):
		close(release)
		t.Fatal("browser.setDownloadBehavior never reached the browser during connect")
	}

	// Its response is being held, so connect must still be waiting on it.
	select {
	case <-connected:
		close(release)
		t.Fatal("OnClientConnect returned while download behavior was still being established; " +
			"a client command could be served before the browser knows where downloads go")
	case <-time.After(200 * time.Millisecond):
	}

	close(release)

	select {
	case <-connected:
	case <-time.After(10 * time.Second):
		t.Fatal("OnClientConnect never returned after download behavior was established")
	}

	val, ok := router.sessions.Load(client.ID())
	if !ok {
		t.Fatal("router has no session for the client")
	}
	session := val.(*BrowserSession)
	session.mu.Lock()
	dir := session.downloadDir
	session.mu.Unlock()
	if dir == "" {
		t.Fatal("download dir must be set once connect returns")
	}
}

// recordingClient is a ClientTransport that keeps whatever the router sends it.
type recordingClient struct {
	mu   sync.Mutex
	sent []string
}

func (c *recordingClient) ID() uint64 { return 1 }

func (c *recordingClient) Send(msg string) error {
	c.mu.Lock()
	c.sent = append(c.sent, msg)
	c.mu.Unlock()
	return nil
}

func (c *recordingClient) Close() error { return nil }
