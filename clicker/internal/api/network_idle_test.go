package api

import (
	"fmt"
	"testing"
	"time"
)

func networkEvent(method, context, id, url string) []byte {
	return []byte(fmt.Sprintf(`{"type":"event","method":%q,"params":{"context":%q,"request":{"request":%q,"url":%q}}}`, method, context, id, url))
}

func TestNetworkTrackerCountsRequestsInFlight(t *testing.T) {
	tracker := NewNetworkTracker()
	inFlight := func(context string) int {
		n, _ := tracker.Idle(context)
		return n
	}
	if n, quiet := tracker.Idle("page"); n != 0 || quiet < 0 {
		t.Fatalf("a page with no requests is idle, got %d in flight", n)
	}

	tracker.Observe(networkEvent("network.beforeRequestSent", "page", "1", "http://example.test/a"))
	tracker.Observe(networkEvent("network.beforeRequestSent", "page", "2", "http://example.test/b"))
	tracker.Observe(networkEvent("network.beforeRequestSent", "page", "2", "http://example.test/redirected"))
	tracker.Observe(networkEvent("network.beforeRequestSent", "other", "3", "http://example.test/c"))
	if inFlight("page") != 2 || inFlight("other") != 1 {
		t.Fatalf("page = %d, other = %d; want 2 and 1", inFlight("page"), inFlight("other"))
	}

	tracker.Observe(networkEvent("network.responseCompleted", "page", "1", "http://example.test/a"))
	tracker.Observe(networkEvent("network.fetchError", "page", "2", "http://example.test/redirected"))
	if n, quiet := tracker.Idle("page"); n != 0 || quiet > time.Minute {
		t.Fatalf("page should have just become idle, got %d in flight, quiet %s", n, quiet)
	}
	if inFlight("other") != 1 {
		t.Fatal("ending the page's requests must not end another context's")
	}
}

func TestNetworkTrackerIgnoresRequestsThatNeverEnd(t *testing.T) {
	tracker := NewNetworkTracker()
	for i, url := range []string{"data:text/plain,x", "blob:http://example.test/1", "about:blank", "ws://example.test/socket", "wss://example.test/socket"} {
		tracker.Observe(networkEvent("network.beforeRequestSent", "page", fmt.Sprint(i), url))
	}
	tracker.Observe(networkEvent("network.beforeRequestSent", "", "9", "http://example.test/worker"))
	tracker.Observe([]byte(`{"type":"success","id":7,"result":{"note":"network.beforeRequestSent"}}`))
	if n, _ := tracker.Idle("page"); n != 0 {
		t.Fatalf("%d requests tracked, want 0", n)
	}
}

func TestNetworkTrackerForgetsRequestsWhenThePageNavigates(t *testing.T) {
	tracker := NewNetworkTracker()
	tracker.Observe(networkEvent("network.beforeRequestSent", "page", "1", "http://example.test/old"))
	tracker.Observe([]byte(`{"type":"event","method":"browsingContext.navigationStarted","params":{"context":"page","url":"http://example.test/new"}}`))
	if n, _ := tracker.Idle("page"); n != 0 {
		t.Fatalf("%d requests left from the old document, want 0", n)
	}
	tracker.Observe(networkEvent("network.responseCompleted", "page", "1", "http://example.test/old"))
	tracker.Observe(networkEvent("network.beforeRequestSent", "page", "2", "http://example.test/new"))
	if n, _ := tracker.Idle("page"); n != 1 {
		t.Fatalf("%d requests in flight after the navigation request, want 1", n)
	}
}
