package api

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// scriptedSession replays a fixed sequence of responses; the last entry
// repeats if called again.
type scriptedSession struct {
	steps []struct {
		resp json.RawMessage
		err  error
	}
	calls int
	nav   *NavigationTracker
}

func (f *scriptedSession) step(resp string, err error) {
	var raw json.RawMessage
	if resp != "" {
		raw = json.RawMessage(resp)
	}
	f.steps = append(f.steps, struct {
		resp json.RawMessage
		err  error
	}{raw, err})
}

func (f *scriptedSession) SendBidiCommand(method string, params map[string]interface{}) (json.RawMessage, error) {
	i := f.calls
	if i >= len(f.steps) {
		i = len(f.steps) - 1
	}
	f.calls++
	return f.steps[i].resp, f.steps[i].err
}

func (f *scriptedSession) SendBidiCommandWithTimeout(method string, params map[string]interface{}, timeout time.Duration) (json.RawMessage, error) {
	return f.SendBidiCommand(method, params)
}

func (f *scriptedSession) GetContextID() (string, error)  { return "ctx", nil }
func (f *scriptedSession) SetLastElementBox(box *BoxInfo) {}
func (f *scriptedSession) NavTracker() *NavigationTracker { return f.nav }

const evalOK = `{"result":{"type":"success","result":{"type":"string","value":"ok"},"realm":"r1"}}`

// A navigation destroys the realm under an in-flight eval; the browsing
// context survives, so the eval must retry against the new document instead
// of surfacing the one-per-navigation error (#335).
func TestEvalRetriesWhenNavigationDestroysRealm(t *testing.T) {
	f := &scriptedSession{}
	// Proxy-shaped error envelope, as chromedriver reports the torn realm.
	f.step(`{"type":"error","error":"unknown error","message":"unknown error: Execution context was destroyed"}`, nil)
	f.step(evalOK, nil)

	out, err := EvalSimpleScript(f, "ctx", "() => 'x'")
	if err != nil {
		t.Fatalf("EvalSimpleScript: %v", err)
	}
	if out != "ok" || f.calls != 2 {
		t.Fatalf("out=%q calls=%d, want ok after one retry", out, f.calls)
	}
}

// The MCP path reports the same condition as a Go error from the client.
func TestEvalRetriesOnClientSideDestroyedError(t *testing.T) {
	f := &scriptedSession{}
	f.step("", errors.New("BiDi error: unknown error - Execution context was destroyed"))
	f.step(evalOK, nil)

	out, err := EvalSimpleScript(f, "ctx", "() => 'x'")
	if err != nil || out != "ok" {
		t.Fatalf("out=%q err=%v, want retry to succeed", out, err)
	}
}

// Firefox aborts the in-flight query instead of naming the realm.
func TestEvalRetriesOnFirefoxAbort(t *testing.T) {
	f := &scriptedSession{}
	f.step("", errors.New("BiDi error: unknown error - Actor 'MessageHandlerFrame' destroyed before query 'EvaluateExpression' could be resolved"))
	f.step(evalOK, nil)

	if out, err := EvalSimpleScript(f, "ctx", "() => 'x'"); err != nil || out != "ok" {
		t.Fatalf("out=%q err=%v, want retry to succeed", out, err)
	}
}

// Errors with no navigation in the picture must surface immediately: a script
// exception is the caller's bug, not a race.
func TestEvalDoesNotRetryUnrelatedErrors(t *testing.T) {
	f := &scriptedSession{}
	f.step(`{"result":{"type":"exception","exceptionDetails":{"text":"boom","lineNumber":1}}}`, nil)
	f.step(evalOK, nil)

	_, err := EvalSimpleScript(f, "ctx", "() => { throw new Error('boom') }")
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("err=%v, want the exception surfaced", err)
	}
	if f.calls != 1 {
		t.Fatalf("calls=%d, an unrelated error must not retry", f.calls)
	}
}

// A realm that never comes back stops at the budget instead of retrying
// forever.
func TestEvalRetryIsBounded(t *testing.T) {
	f := &scriptedSession{}
	f.step(`{"type":"error","error":"unknown error","message":"unknown error: Execution context was destroyed"}`, nil)

	start := time.Now()
	_, err := EvalSimpleScript(f, "ctx", "() => 'x'")
	elapsed := time.Since(start)
	if err == nil || !strings.Contains(err.Error(), "context was destroyed") {
		t.Fatalf("err=%v, want the persistent error surfaced", err)
	}
	if elapsed > evalNavigationRetryBudget+time.Second {
		t.Fatalf("retried for %v, want the budget respected", elapsed)
	}
}
