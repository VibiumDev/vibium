package api

import (
	"encoding/json"
	"sync"
	"time"
)

// NavigationTracker records which browsing contexts have a navigation in
// flight.
//
// browsingContext.captureScreenshot does not answer while the target context is
// navigating: Chrome holds the command until the new document settles, so a
// capture issued right after a click that submits a form burns its entire
// timeout and returns nothing (#289). The action still succeeds, so the only
// visible symptom is that every navigating action costs the timeout.
//
// Waiting for the navigation first turns "5s, then no frame" into "as long as
// the navigation actually takes, then the frame that shows its result" — which
// is the frame a filmstrip most wants, since it is the outcome of the action.
type NavigationTracker struct {
	mu       sync.Mutex
	inFlight map[string]chan struct{}   // context -> closed when the navigation settles
	starting map[string][]chan struct{} // context -> waiters woken when one begins
}

// NewNavigationTracker creates an empty tracker.
func NewNavigationTracker() *NavigationTracker {
	return &NavigationTracker{
		inFlight: make(map[string]chan struct{}),
		starting: make(map[string][]chan struct{}),
	}
}

// Observe updates the tracker from a raw BiDi event. Anything that is not a
// navigation lifecycle event is ignored, so callers can hand it every event.
func (t *NavigationTracker) Observe(raw []byte) {
	if t == nil {
		return
	}
	var evt struct {
		Method string `json:"method"`
		Params struct {
			Context string `json:"context"`
		} `json:"params"`
	}
	if err := json.Unmarshal(raw, &evt); err != nil {
		return
	}
	if evt.Params.Context == "" {
		return
	}

	switch evt.Method {
	case "browsingContext.navigationStarted":
		t.mu.Lock()
		if _, ok := t.inFlight[evt.Params.Context]; !ok {
			t.inFlight[evt.Params.Context] = make(chan struct{})
		}
		for _, w := range t.starting[evt.Params.Context] {
			close(w)
		}
		delete(t.starting, evt.Params.Context)
		t.mu.Unlock()

	// load is the settle signal; the failure events exist so an abandoned
	// navigation cannot leave a context marked in-flight forever.
	case "browsingContext.load",
		"browsingContext.navigationFailed",
		"browsingContext.navigationAborted":
		t.Clear(evt.Params.Context)
	}
}

// IsNavigating reports whether a navigation is in flight for a context.
func (t *NavigationTracker) IsNavigating(context string) bool {
	if t == nil || context == "" {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	_, ok := t.inFlight[context]
	return ok
}

// WaitForSettled blocks until the context's navigation completes, the timeout
// elapses, or there was no navigation to begin with. It reports whether the
// context is settled — false means the timeout won and a command issued now may
// still block.
func (t *NavigationTracker) WaitForSettled(context string, timeout time.Duration) bool {
	if t == nil || context == "" {
		return true
	}

	t.mu.Lock()
	done, ok := t.inFlight[context]
	t.mu.Unlock()
	if !ok {
		return true
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
		return true
	case <-timer.C:
		return false
	}
}

// Clear marks a context settled and wakes anything waiting on it. Also used
// when a context goes away, so a destroyed context leaves nothing behind.
func (t *NavigationTracker) Clear(context string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	if done, ok := t.inFlight[context]; ok {
		delete(t.inFlight, context)
		close(done)
	}
	t.mu.Unlock()
}

// NotifyStart returns a channel closed when a navigation begins for a context,
// plus a cancel to deregister. A navigation started by an action is only
// reported ~10ms after the action's command returns, far too late to check for
// up front, so a capture has to react to one starting underneath it rather than
// predict it (#289).
func (t *NavigationTracker) NotifyStart(context string) (<-chan struct{}, func()) {
	if t == nil || context == "" {
		return nil, func() {}
	}
	ch := make(chan struct{})
	t.mu.Lock()
	t.starting[context] = append(t.starting[context], ch)
	t.mu.Unlock()

	return ch, func() {
		t.mu.Lock()
		defer t.mu.Unlock()
		w := t.starting[context]
		for i, c := range w {
			if c == ch {
				t.starting[context] = append(w[:i], w[i+1:]...)
				break
			}
		}
		if len(t.starting[context]) == 0 {
			delete(t.starting, context)
		}
	}
}
