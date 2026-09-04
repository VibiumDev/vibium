package bidi

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

// The URL shapes cloud browser providers document. Each one used to reach the
// dialer verbatim and come back as "malformed ws or wss URL" (#101).
func TestNormalizeEndpointRewritesProviderURLs(t *testing.T) {
	basic := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:key"))

	tests := []struct {
		name     string
		in       string
		wantURL  string
		wantAuth string // expected Authorization header, "" for none
	}{
		{"ws is left alone", "ws://127.0.0.1:9515/session", "ws://127.0.0.1:9515/session", ""},
		{"wss is left alone", "wss://cloud.example.com/bidi", "wss://cloud.example.com/bidi", ""},
		{"https becomes wss", "https://hub-cloud.browserstack.com", "wss://hub-cloud.browserstack.com", ""},
		{"http becomes ws", "http://127.0.0.1:9515/session", "ws://127.0.0.1:9515/session", ""},
		{
			"browserstack hub URL from the issue",
			"https://user:key@hub-cloud.browserstack.com",
			"wss://hub-cloud.browserstack.com",
			basic,
		},
		{
			"userinfo moves off a wss URL too",
			"wss://user:key@cloud.example.com/bidi",
			"wss://cloud.example.com/bidi",
			basic,
		},
		{
			"a username with no password still authenticates",
			"https://user@hub.example.com",
			"wss://hub.example.com",
			"Basic " + base64.StdEncoding.EncodeToString([]byte("user:")),
		},
		{
			"percent-encoded credentials are decoded before encoding",
			"https://user%40corp.com:p%40ss@hub.example.com",
			"wss://hub.example.com",
			"Basic " + base64.StdEncoding.EncodeToString([]byte("user@corp.com:p@ss")),
		},
		{
			"the query string survives the rewrite",
			"https://user:key@hub.example.com/wd/hub?os=Windows",
			"wss://hub.example.com/wd/hub?os=Windows",
			basic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, headers, err := NormalizeEndpoint(tt.in)
			if err != nil {
				t.Fatalf("NormalizeEndpoint(%q) returned error: %v", tt.in, err)
			}
			if got != tt.wantURL {
				t.Errorf("url = %q, want %q", got, tt.wantURL)
			}
			if auth := headers.Get("Authorization"); auth != tt.wantAuth {
				t.Errorf("Authorization = %q, want %q", auth, tt.wantAuth)
			}
		})
	}
}

// A real mistake must be named, not passed to the dialer to come back as
// "malformed" — that opacity is what made #101 hard to diagnose.
func TestNormalizeEndpointRejectsBadURLsClearly(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", "empty endpoint URL"},
		{"whitespace only", "   ", "empty endpoint URL"},
		{"no scheme", "hub-cloud.browserstack.com", "no scheme"},
		{"unsupported scheme", "ftp://hub.example.com", "unsupported endpoint scheme"},
		{"no host", "wss:///session", "no host"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := NormalizeEndpoint(tt.in)
			if err == nil {
				t.Fatalf("NormalizeEndpoint(%q) should have failed", tt.in)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want it to mention %q", err, tt.want)
			}
			if strings.Contains(err.Error(), "malformed ws or wss URL") {
				t.Errorf("error must not be the dialer's opaque message: %q", err)
			}
		})
	}
}

func TestRedactEndpointHidesCredentials(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			"password is replaced",
			"https://user:key@hub-cloud.browserstack.com",
			"https://user:redacted@hub-cloud.browserstack.com",
		},
		{
			"a bare username is not a secret",
			"https://user@hub.example.com",
			"https://user@hub.example.com",
		},
		{"no userinfo, no change", "wss://cloud.example.com/bidi", "wss://cloud.example.com/bidi"},
		{"plain local URL", "ws://127.0.0.1:9515/session", "ws://127.0.0.1:9515/session"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RedactEndpoint(tt.in); got != tt.want {
				t.Errorf("RedactEndpoint(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// url.Parse rejects some strings outright. Redaction still may not print the
// credential, so the whole userinfo field goes.
func TestRedactEndpointHidesCredentialsInUnparseableURLs(t *testing.T) {
	got := RedactEndpoint("https://user:k[e]y@hub.example.com/wd/hub")
	if strings.Contains(got, "k[e]y") {
		t.Fatalf("credential leaked from an unparseable URL: %q", got)
	}
	for _, want := range []string{"hub.example.com", "/wd/hub", redactedSecret} {
		if !strings.Contains(got, want) {
			t.Errorf("redacted URL %q lost %q", got, want)
		}
	}
}

// authCapturingServer accepts one WebSocket handshake and reports the
// Authorization header it arrived with.
func authCapturingServer(t *testing.T) (endpoint string, auth <-chan string) {
	t.Helper()

	got := make(chan string, 1)
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- r.Header.Get("Authorization")
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		ws.ReadMessage() // hold the connection open until the client closes
	}))
	t.Cleanup(srv.Close)

	// srv.URL is http://127.0.0.1:port — exactly the scheme the dialer used
	// to reject. Credentials go in the userinfo field, as BrowserStack
	// documents its hub URL.
	return strings.Replace(srv.URL, "http://", "http://user:key@", 1), got
}

// End to end over a real handshake: the http:// URL connects and the
// credential arrives as a Basic Authorization header (#101).
func TestConnectWithHeadersAcceptsHTTPURLWithCredentials(t *testing.T) {
	endpoint, gotAuth := authCapturingServer(t)

	conn, err := ConnectWithHeaders(endpoint, nil)
	if err != nil {
		t.Fatalf("connect to %q failed: %v", RedactEndpoint(endpoint), err)
	}
	defer conn.Close()

	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:key"))
	if auth := <-gotAuth; auth != want {
		t.Errorf("Authorization = %q, want %q", auth, want)
	}
}

// An explicit header is a deliberate choice (VIBIUM_CONNECT_API_KEY,
// --connect-header) and must outrank a credential left in the URL.
func TestConnectWithHeadersPrefersExplicitAuthorization(t *testing.T) {
	endpoint, gotAuth := authCapturingServer(t)

	headers := http.Header{}
	headers.Set("Authorization", "Bearer explicit-token")

	conn, err := ConnectWithHeaders(endpoint, headers)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer conn.Close()

	if auth := <-gotAuth; auth != "Bearer explicit-token" {
		t.Errorf("Authorization = %q, want the explicit header to win", auth)
	}
	// The router reuses one header map across every client it connects, so
	// the caller's map must come back untouched.
	if got := headers.Get("Authorization"); got != "Bearer explicit-token" {
		t.Errorf("caller's header map was mutated: Authorization = %q", got)
	}
	if len(headers) != 1 {
		t.Errorf("caller's header map grew to %d entries", len(headers))
	}
}

// Headers unrelated to auth still reach the endpoint alongside URL credentials.
func TestConnectWithHeadersKeepsUnrelatedHeaders(t *testing.T) {
	gotTrace := make(chan string, 1)
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTrace <- r.Header.Get("X-Trace-Id")
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		ws.ReadMessage()
	}))
	t.Cleanup(srv.Close)

	endpoint := strings.Replace(srv.URL, "http://", "http://user:key@", 1)
	headers := http.Header{}
	headers.Set("X-Trace-Id", "abc123")

	conn, err := ConnectWithHeaders(endpoint, headers)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer conn.Close()

	if trace := <-gotTrace; trace != "abc123" {
		t.Errorf("X-Trace-Id = %q, want it preserved", trace)
	}
}

// The credential must not reach stderr, which clients drain and show the user.
func TestConnectWithHeadersRedactsCredentialsInErrors(t *testing.T) {
	// Port 1 is not listening, so the dial fails with the URL in its error.
	_, err := ConnectWithHeaders("https://user:supersecret@127.0.0.1:1/session", nil)
	if err == nil {
		t.Fatal("connecting to a closed port should fail")
	}
	if strings.Contains(err.Error(), "supersecret") {
		t.Fatalf("credential leaked into the error: %v", err)
	}
}

// A bad URL fails before the dial, and its error names the problem instead of
// repeating the dialer's "malformed ws or wss URL".
func TestConnectWithHeadersNamesBadURLs(t *testing.T) {
	_, err := ConnectWithHeaders("hub-cloud.browserstack.com", nil)
	if err == nil {
		t.Fatal("a schemeless URL should fail")
	}
	if !strings.Contains(err.Error(), "no scheme") {
		t.Errorf("error should name the missing scheme, got: %v", err)
	}
}
