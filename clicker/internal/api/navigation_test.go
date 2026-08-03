package api

import (
	"testing"
	"time"
)

const navStarted = `{"method":"browsingContext.navigationStarted","params":{"context":"ctx-1","url":"http://x/"}}`
const navLoad = `{"method":"browsingContext.load","params":{"context":"ctx-1","url":"http://x/"}}`

func TestTracksNavigationInFlight(t *testing.T) {
	tr := NewNavigationTracker()

	if tr.IsNavigating("ctx-1") {
		t.Fatal("nothing observed yet, should not be navigating")
	}

	tr.Observe([]byte(navStarted))
	if !tr.IsNavigating("ctx-1") {
		t.Fatal("navigationStarted should mark the context in flight")
	}
	if tr.IsNavigating("ctx-2") {
		t.Fatal("other contexts must be unaffected")
	}

	tr.Observe([]byte(navLoad))
	if tr.IsNavigating("ctx-1") {
		t.Fatal("load should settle the context")
	}
}

func TestFailedNavigationDoesNotWedgeContext(t *testing.T) {
	// Without these, an abandoned navigation would leave the context marked in
	// flight forever and every later capture would wait out its full timeout.
	for _, ev := range []string{
		`{"method":"browsingContext.navigationFailed","params":{"context":"ctx-1"}}`,
		`{"method":"browsingContext.navigationAborted","params":{"context":"ctx-1"}}`,
	} {
		tr := NewNavigationTracker()
		tr.Observe([]byte(navStarted))
		tr.Observe([]byte(ev))
		if tr.IsNavigating("ctx-1") {
			t.Fatalf("%s should settle the context", ev)
		}
	}
}

func TestWaitForSettledReturnsWhenLoadArrives(t *testing.T) {
	tr := NewNavigationTracker()
	tr.Observe([]byte(navStarted))

	go func() {
		time.Sleep(20 * time.Millisecond)
		tr.Observe([]byte(navLoad))
	}()

	start := time.Now()
	if !tr.WaitForSettled("ctx-1", 2*time.Second) {
		t.Fatal("should report settled, not timed out")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("waited %s, should have woken on load", elapsed)
	}
}

func TestWaitForSettledTimesOut(t *testing.T) {
	tr := NewNavigationTracker()
	tr.Observe([]byte(navStarted))

	if tr.WaitForSettled("ctx-1", 30*time.Millisecond) {
		t.Fatal("should report not-settled when the navigation never completes")
	}
}

func TestWaitForSettledIsNoOpWhenIdle(t *testing.T) {
	tr := NewNavigationTracker()
	start := time.Now()
	if !tr.WaitForSettled("ctx-1", time.Minute) {
		t.Fatal("an idle context is already settled")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("blocked %s on an idle context", elapsed)
	}
}

func TestNotifyStartFiresOnNavigation(t *testing.T) {
	// This is the signal a capture races against: the navigation an action
	// triggers is only reported after that action's command has returned.
	tr := NewNavigationTracker()
	started, cancel := tr.NotifyStart("ctx-1")
	defer cancel()

	go func() {
		time.Sleep(10 * time.Millisecond)
		tr.Observe([]byte(navStarted))
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("NotifyStart should fire when a navigation begins")
	}
}

func TestNotifyStartCancelDeregisters(t *testing.T) {
	tr := NewNavigationTracker()
	_, cancel := tr.NotifyStart("ctx-1")
	cancel()

	tr.Observe([]byte(navStarted))

	tr.mu.Lock()
	n := len(tr.starting["ctx-1"])
	tr.mu.Unlock()
	if n != 0 {
		t.Fatalf("cancelled waiter should be gone, found %d", n)
	}
}

func TestNilTrackerIsSafe(t *testing.T) {
	// Sessions without a recorder carry no tracker.
	var tr *NavigationTracker
	tr.Observe([]byte(navStarted))
	if tr.IsNavigating("ctx-1") {
		t.Fatal("nil tracker never navigates")
	}
	if !tr.WaitForSettled("ctx-1", time.Second) {
		t.Fatal("nil tracker is always settled")
	}
	ch, cancel := tr.NotifyStart("ctx-1")
	if ch != nil {
		t.Fatal("nil tracker returns no channel")
	}
	cancel()
	tr.Clear("ctx-1")
}
