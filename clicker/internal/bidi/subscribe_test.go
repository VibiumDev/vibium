package bidi

import (
	"fmt"
	"reflect"
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

func TestSubscribeEventsEmptyList(t *testing.T) {
	called := false
	if rejected := SubscribeEvents(func([]string) error { called = true; return nil }, nil); rejected != nil {
		t.Fatalf("rejected = %v, want nil", rejected)
	}
	if called {
		t.Fatal("no subscribe call expected for an empty event list")
	}
}
