package bidi

import (
	"strings"
	"sync/atomic"
	"testing"
)

const staleRealmMessage = "unknown error: Cannot find context with specified id"

// rejectFirst answers the first n commands with a stale-realm error and the rest with success.
func rejectFirst(n int32, seen *atomic.Int32) func(*testServerConn) {
	return func(s *testServerConn) {
		for {
			cmd, err := s.readCommand()
			if err != nil {
				return
			}
			if seen.Add(1) <= n {
				s.writeJSON(map[string]interface{}{"type": "error", "id": cmd.ID, "error": "unknown error", "message": staleRealmMessage})
				continue
			}
			s.respond(cmd.ID, map[string]interface{}{"type": "success"})
		}
	}
}

func TestScriptCommandIsResentWhenItNeverRan(t *testing.T) {
	for _, method := range []string{"script.evaluate", "script.callFunction"} {
		var seen atomic.Int32
		client := newTestClient(t, rejectFirst(1, &seen))

		if _, err := client.SendCommand(method, map[string]interface{}{}); err != nil {
			t.Fatalf("%s: %v, want the resend to succeed", method, err)
		}
		if got := seen.Load(); got != 2 {
			t.Fatalf("%s: server saw %d commands, want 2", method, got)
		}
	}
}

func TestOnlyScriptCommandsAreResent(t *testing.T) {
	var seen atomic.Int32
	client := newTestClient(t, rejectFirst(1, &seen))

	_, err := client.SendCommand("browsingContext.navigate", map[string]interface{}{})
	if err == nil || !strings.Contains(err.Error(), "Cannot find context") {
		t.Fatalf("err=%v, want the rejection surfaced", err)
	}
	if got := seen.Load(); got != 1 {
		t.Fatalf("server saw %d commands, want 1", got)
	}
}

func TestOtherScriptErrorsAreNotResent(t *testing.T) {
	calls := 0
	_, err := RetryStaleRealm("script.evaluate", func() (int, error) {
		calls++
		return 0, &protocolTestError{"no such frame: context gone"}
	})
	if err == nil || calls != 1 {
		t.Fatalf("err=%v calls=%d, want one call and the error", err, calls)
	}
}

func TestResendStopsAtTheBudget(t *testing.T) {
	calls := 0
	_, err := RetryStaleRealm("script.evaluate", func() (int, error) {
		calls++
		return 0, &protocolTestError{staleRealmMessage}
	})
	if err == nil || calls < 2 {
		t.Fatalf("err=%v calls=%d, want resends and then the error", err, calls)
	}
}

type protocolTestError struct{ msg string }

func (e *protocolTestError) Error() string { return e.msg }
