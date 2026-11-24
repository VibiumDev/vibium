package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// networkQuiet is how long a page must have no request in flight to count as idle.
const networkQuiet = 500 * time.Millisecond

// NetworkTracker records the requests each browsing context has in flight.
type NetworkTracker struct {
	mu       sync.Mutex
	inFlight map[string]map[string]struct{} // context -> request ids
	quiet    map[string]time.Time           // context -> when its last request ended
	start    time.Time
}

// NewNetworkTracker creates an empty tracker.
func NewNetworkTracker() *NetworkTracker {
	return &NetworkTracker{
		inFlight: make(map[string]map[string]struct{}),
		quiet:    make(map[string]time.Time),
		start:    time.Now(),
	}
}

// Observe updates the tracker from a raw BiDi message; callers can hand it every message.
func (t *NetworkTracker) Observe(raw []byte) {
	if t == nil || !(bytes.Contains(raw, []byte(`"network.`)) || bytes.Contains(raw, []byte(`"browsingContext.navigationStarted"`))) {
		return
	}
	var evt struct {
		Method string `json:"method"`
		Params struct {
			Context string `json:"context"`
			Request struct {
				ID  string `json:"request"`
				URL string `json:"url"`
			} `json:"request"`
		} `json:"params"`
	}
	if json.Unmarshal(raw, &evt) != nil || evt.Params.Context == "" {
		return
	}
	context, id := evt.Params.Context, evt.Params.Request.ID

	t.mu.Lock()
	defer t.mu.Unlock()
	switch evt.Method {
	case "network.beforeRequestSent":
		if id == "" || !tracksURL(evt.Params.Request.URL) {
			return
		}
		if t.inFlight[context] == nil {
			t.inFlight[context] = make(map[string]struct{})
		}
		t.inFlight[context][id] = struct{}{}
	case "network.responseCompleted", "network.fetchError":
		if _, ok := t.inFlight[context][id]; !ok {
			return
		}
		delete(t.inFlight[context], id)
		t.quiet[context] = time.Now()
	case "browsingContext.navigationStarted":
		// The old document's requests may never report an end.
		delete(t.inFlight, context)
		t.quiet[context] = time.Now()
	}
}

// tracksURL leaves out requests that never touch the network or never end.
func tracksURL(url string) bool {
	for _, scheme := range []string{"data:", "blob:", "about:", "ws:", "wss:"} {
		if strings.HasPrefix(url, scheme) {
			return false
		}
	}
	return true
}

// Idle reports how many requests a context has in flight and how long it has had none.
func (t *NetworkTracker) Idle(context string) (inFlight int, quiet time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if n := len(t.inFlight[context]); n > 0 {
		return n, 0
	}
	since, ok := t.quiet[context]
	if !ok {
		since = t.start
	}
	return 0, time.Since(since)
}

// waitForNetworkIdle waits until the page has loaded and has had no request in flight for networkQuiet.
func waitForNetworkIdle(s Session, tracker *NetworkTracker, navigations *NavigationTracker, context string, timeout time.Duration) error {
	started := time.Now()
	deadline := started.Add(timeout)
	for {
		inFlight, quiet := tracker.Idle(context)
		// The window also runs from the wait's own start: a request begun just before it may not have reported yet.
		if inFlight == 0 && min(quiet, time.Since(started)) >= networkQuiet && !navigations.IsNavigating(context) {
			if state, err := EvalSimpleScript(s, context, "() => document.readyState"); err == nil && state == "complete" {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout after %s waiting for the network to be idle: %d requests in flight", timeout, inFlight)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
