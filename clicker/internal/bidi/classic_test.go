package bidi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestIsClassicEndpoint(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://grid.example.com/wd/hub", true},
		{"http://localhost:4444", true},
		{"ws://localhost:9515/session", false},
		{"wss://cloud.example.com/bidi", false},
		{"://not a url", false},
	}
	for _, tt := range tests {
		if got := IsClassicEndpoint(tt.url); got != tt.want {
			t.Errorf("IsClassicEndpoint(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

// gridStub is an httptest server that speaks just enough WebDriver classic to
// create and delete one session, plus a BiDi WebSocket the session points at.
type gridStub struct {
	*httptest.Server
	t *testing.T

	sawAuth    string                 // Authorization header on POST /session
	sawCaps    map[string]interface{} // alwaysMatch sent by the client
	deleted    bool
	sessionErr string // if set, POST /session returns this W3C error
	omitWSURL  bool   // if set, session response has no webSocketUrl
}

func newGridStub(t *testing.T) *gridStub {
	g := &gridStub{t: t}
	upgrader := websocket.Upgrader{}

	mux := http.NewServeMux()
	mux.HandleFunc("/wd/hub/session", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "bad method", http.StatusMethodNotAllowed)
			return
		}
		g.sawAuth = r.Header.Get("Authorization")

		var body struct {
			Capabilities struct {
				AlwaysMatch map[string]interface{} `json:"alwaysMatch"`
			} `json:"capabilities"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			g.t.Errorf("session body did not parse: %v", err)
		}
		g.sawCaps = body.Capabilities.AlwaysMatch

		if g.sessionErr != "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"value":{"error":"session not created","message":%q}}`, g.sessionErr)
			return
		}

		caps := map[string]interface{}{}
		if !g.omitWSURL {
			wsURL := "ws" + strings.TrimPrefix(g.Server.URL, "http") + "/session/stub-session-id"
			caps["webSocketUrl"] = wsURL
		}
		resp := map[string]interface{}{
			"value": map[string]interface{}{
				"sessionId":    "stub-session-id",
				"capabilities": caps,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/wd/hub/session/stub-session-id", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			g.deleted = true
			fmt.Fprint(w, `{"value":null}`)
			return
		}
		http.Error(w, "bad method", http.StatusMethodNotAllowed)
	})
	// The BiDi endpoint the created session points at. It carries a session
	// already, so session.status reports ready:false (spec behavior).
	mux.HandleFunc("/session/stub-session-id", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var cmd struct {
				ID     int    `json:"id"`
				Method string `json:"method"`
			}
			json.Unmarshal(msg, &cmd)
			if cmd.Method == "session.status" {
				conn.WriteMessage(websocket.TextMessage,
					[]byte(fmt.Sprintf(`{"type":"success","id":%d,"result":{"ready":false,"message":"session active"}}`, cmd.ID)))
			}
		}
	})

	g.Server = httptest.NewServer(mux)
	return g
}

func TestResolveEndpointPassesThroughWebSocketURLs(t *testing.T) {
	for _, u := range []string{"ws://localhost:9515/session", "wss://cloud.example.com/bidi"} {
		got, classic, err := ResolveEndpoint(u, nil, nil)
		if err != nil || classic != nil || got != u {
			t.Errorf("ResolveEndpoint(%q) = (%q, %v, %v), want passthrough", u, got, classic, err)
		}
	}
}

func TestResolveEndpointCreatesClassicSession(t *testing.T) {
	g := newGridStub(t)
	defer g.Close()

	endpoint := g.URL + "/wd/hub"
	wsURL, classic, err := ResolveEndpoint(endpoint, nil, map[string]interface{}{
		"vendor:options": map[string]interface{}{"os": "OS X"},
	})
	if err != nil {
		t.Fatalf("ResolveEndpoint: %v", err)
	}
	if classic == nil || classic.ID != "stub-session-id" {
		t.Fatalf("classic session = %+v, want ID stub-session-id", classic)
	}
	if !strings.HasSuffix(wsURL, "/session/stub-session-id") {
		t.Errorf("wsURL = %q, want the stub session URL", wsURL)
	}

	// webSocketUrl must always be forced on, and vendor caps preserved.
	if g.sawCaps["webSocketUrl"] != true {
		t.Errorf("alwaysMatch webSocketUrl = %v, want true", g.sawCaps["webSocketUrl"])
	}
	if _, ok := g.sawCaps["vendor:options"]; !ok {
		t.Errorf("vendor capability was dropped: %v", g.sawCaps)
	}

	// The returned URL attaches as an existing session (ready:false → attach).
	conn, err := ConnectWithHeaders(wsURL, nil)
	if err != nil {
		t.Fatalf("connect to session ws: %v", err)
	}
	session, err := AttachOrNewSessionOnConn(conn, wsURL, map[string]interface{}{})
	conn.Close()
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	if session.Created {
		t.Errorf("session.Created = true, want attach to existing session")
	}

	if err := classic.Delete(); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !g.deleted {
		t.Errorf("DELETE /session/<id> never reached the grid")
	}
}

func TestResolveEndpointFoldsUserinfoIntoBasicAuth(t *testing.T) {
	g := newGridStub(t)
	defer g.Close()

	u, _ := url.Parse(g.URL)
	endpoint := fmt.Sprintf("http://user:secret@%s/wd/hub", u.Host)
	_, classic, err := ResolveEndpoint(endpoint, nil, nil)
	if err != nil {
		t.Fatalf("ResolveEndpoint: %v", err)
	}
	defer classic.Delete()

	// base64("user:secret")
	if want := "Basic dXNlcjpzZWNyZXQ="; g.sawAuth != want {
		t.Errorf("Authorization = %q, want %q", g.sawAuth, want)
	}
}

func TestResolveEndpointAcceptsSessionSuffix(t *testing.T) {
	g := newGridStub(t)
	defer g.Close()

	// Both .../wd/hub and .../wd/hub/session name the same endpoint.
	for _, suffix := range []string{"/wd/hub", "/wd/hub/", "/wd/hub/session"} {
		_, classic, err := ResolveEndpoint(g.URL+suffix, nil, nil)
		if err != nil {
			t.Errorf("ResolveEndpoint(...%s): %v", suffix, err)
			continue
		}
		classic.Delete()
	}
}

func TestResolveEndpointSurfacesW3CError(t *testing.T) {
	g := newGridStub(t)
	defer g.Close()
	g.sessionErr = "All parallel tests are currently in use"

	_, _, err := ResolveEndpoint(g.URL+"/wd/hub", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "All parallel tests") {
		t.Errorf("err = %v, want the grid's message surfaced", err)
	}
}

func TestResolveEndpointRejectsAndCleansUpNonBiDiGrid(t *testing.T) {
	g := newGridStub(t)
	defer g.Close()
	g.omitWSURL = true

	_, _, err := ResolveEndpoint(g.URL+"/wd/hub", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "webSocketUrl") {
		t.Errorf("err = %v, want a no-BiDi explanation", err)
	}
	// The half-created session must not be left running (it bills).
	if !g.deleted {
		t.Errorf("session without webSocketUrl was not deleted")
	}
}

func TestConnectWithHeadersFoldsUserinfo(t *testing.T) {
	sawAuth := make(chan string, 1)
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth <- r.Header.Get("Authorization")
		conn, err := upgrader.Upgrade(w, r, nil)
		if err == nil {
			conn.Close()
		}
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	conn, err := ConnectWithHeaders(fmt.Sprintf("ws://user:secret@%s", u.Host), nil)
	if err != nil {
		t.Fatalf("ConnectWithHeaders: %v", err)
	}
	conn.Close()

	if got, want := <-sawAuth, "Basic dXNlcjpzZWNyZXQ="; got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
}
