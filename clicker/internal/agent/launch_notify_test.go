package agent

import "testing"

// The notify callback must fire exactly once when a call commits to starting
// a browser, before the connect/launch work. A remote URL with nothing
// listening makes the attempt fail fast while still crossing the commit point.
func TestLaunchNotifyFiresOnLaunchAttempt(t *testing.T) {
	h := NewHandlers("", "", false, "ws://127.0.0.1:1/session", nil, nil)
	calls := 0
	h.SetLaunchNotify(func() { calls++ })

	if _, err := h.Call("browser_start", map[string]interface{}{}); err == nil {
		t.Fatal("expected the remote connect to fail")
	}
	if calls != 1 {
		t.Fatalf("launchNotify fired %d times, want 1", calls)
	}
}
