package bidi

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestSubscribeEventsBatchSuccess(t *testing.T) {
	var calls [][]string
	rejected := SubscribeEvents(func(events []string) error {
		calls = append(calls, events)
		return nil
	}, []string{"a", "b", "c"})

	if rejected != nil {
		t.Fatalf("no event should be rejected, got %v", rejected)
	}
	if len(calls) != 1 {
		t.Fatalf("an accepted batch must subscribe exactly once, got %d calls", len(calls))
	}
}

// An engine that does not know one event name rejects the whole batch, the
// way Firefox rejects any batch containing browsingContext.navigationAborted.
// The fallback must keep every name the engine does accept.
func TestSubscribeEventsFallsBackPerEvent(t *testing.T) {
	subscribed := map[string]bool{}
	rejected := SubscribeEvents(func(events []string) error {
		for _, ev := range events {
			if ev == "bad" {
				return fmt.Errorf("invalid argument: bad is not a valid event name")
			}
		}
		for _, ev := range events {
			subscribed[ev] = true
		}
		return nil
	}, []string{"a", "bad", "c"})

	if !reflect.DeepEqual(rejected, []string{"bad"}) {
		t.Fatalf("rejected = %v, want [bad]", rejected)
	}
	for _, ev := range []string{"a", "c"} {
		if !subscribed[ev] {
			t.Fatalf("valid event %q was not subscribed after batch failure", ev)
		}
	}
	if subscribed["bad"] {
		t.Fatal("rejected event must not be recorded as subscribed")
	}
}

// A dead connection fails the batch and every per-event retry. The warning
// must then carry the transport error, not blame the event names.
// Swaps os.Stderr to capture the warning, so it must never run in parallel.
func TestSubscribeEventsTransportFailureReportsBatchError(t *testing.T) {
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	defer func() { os.Stderr = origStderr }()

	rejected := SubscribeEvents(func([]string) error {
		return fmt.Errorf("connection lost waiting for response to session.subscribe")
	}, []string{"a", "b"})

	w.Close()
	os.Stderr = origStderr
	out, _ := io.ReadAll(r)

	if !reflect.DeepEqual(rejected, []string{"a", "b"}) {
		t.Fatalf("rejected = %v, want all events", rejected)
	}
	if !strings.Contains(string(out), "connection lost") {
		t.Fatalf("warning must carry the batch error, got: %q", out)
	}
	if strings.Contains(string(out), "rejected event subscription for") {
		t.Fatalf("transport failure must not be reported as rejected names, got: %q", out)
	}
}

func TestSubscribeEventsEmptyList(t *testing.T) {
	called := false
	if rejected := SubscribeEvents(func([]string) error { called = true; return nil }, nil); rejected != nil {
		t.Fatalf("rejected = %v, want nil", rejected)
	}
	if called {
		t.Fatal("no subscribe call expected for an empty event list")
	}
}
