package bidi

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

// A BiDi exception response carries no "result" member, so the failure text is
// only reachable through exceptionDetails (issue #221).
func TestParseScriptResultException(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "exception",
		"exceptionDetails": {
			"text": "ReferenceError: undefinedVariable is not defined",
			"lineNumber": 4,
			"columnNumber": 12
		},
		"realm": "realm-1"
	}`)

	_, err := ParseScriptResult(raw)
	if err == nil {
		t.Fatal("ParseScriptResult() returned nil error for an exception result")
	}

	var se *ScriptException
	if !errors.As(err, &se) {
		t.Fatalf("error is %T, want *ScriptException", err)
	}
	if se.Text != "ReferenceError: undefinedVariable is not defined" {
		t.Errorf("Text = %q, want the exceptionDetails text", se.Text)
	}
	if se.LineNumber != 4 {
		t.Errorf("LineNumber = %d, want 4", se.LineNumber)
	}
	if want := "script exception: ReferenceError: undefinedVariable is not defined"; err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestParseScriptResultExceptionWithoutText(t *testing.T) {
	raw := json.RawMessage(`{"type":"exception","exceptionDetails":{}}`)

	_, err := ParseScriptResult(raw)
	if err == nil {
		t.Fatal("ParseScriptResult() returned nil error for an exception result")
	}
	if err.Error() != "script threw an exception" {
		t.Errorf("Error() = %q, want the fallback message", err.Error())
	}
}

func TestParseScriptResultSuccess(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "success",
		"result": {"type": "string", "value": "hello"},
		"realm": "realm-1"
	}`)

	sr, err := ParseScriptResult(raw)
	if err != nil {
		t.Fatalf("ParseScriptResult() error = %v", err)
	}
	if sr.Type != "success" {
		t.Errorf("Type = %q, want success", sr.Type)
	}
	if got := ConvertRemoteValue(sr.Result); got != "hello" {
		t.Errorf("ConvertRemoteValue() = %#v, want %q", got, "hello")
	}
}

func TestParseScriptResponseUnwrapsEnvelope(t *testing.T) {
	raw := json.RawMessage(`{"result":{"type":"success","result":{"type":"number","value":7}}}`)

	sr, err := ParseScriptResponse(raw)
	if err != nil {
		t.Fatalf("ParseScriptResponse() error = %v", err)
	}
	if got := ConvertRemoteValue(sr.Result); !reflect.DeepEqual(got, float64(7)) {
		t.Errorf("ConvertRemoteValue() = %#v, want float64(7)", got)
	}
}

// The envelope path must surface exceptions too — this is the JS/Python
// page.evaluate route, which previously returned null for a thrown error.
func TestParseScriptResponseSurfacesException(t *testing.T) {
	raw := json.RawMessage(`{"result":{"type":"exception","exceptionDetails":{"text":"boom"}}}`)

	_, err := ParseScriptResponse(raw)
	if err == nil {
		t.Fatal("ParseScriptResponse() returned nil error for an exception result")
	}
	var se *ScriptException
	if !errors.As(err, &se) {
		t.Fatalf("error is %T, want *ScriptException", err)
	}
	if se.Text != "boom" {
		t.Errorf("Text = %q, want %q", se.Text, "boom")
	}
}

func TestParseScriptResponseMissingResult(t *testing.T) {
	if _, err := ParseScriptResponse(json.RawMessage(`{}`)); err == nil {
		t.Fatal("ParseScriptResponse() returned nil error for a response with no result member")
	}
}

// Nested arrays must survive the round trip — the JS-client page.evaluate path
// previously unwrapped only one level, leaving inner elements as typed objects
// (issue #124).
func TestParseScriptResponseNestedArray(t *testing.T) {
	raw := json.RawMessage(`{"result":{"type":"success","result":` +
		`{"type":"array","value":[` +
		`{"type":"array","value":[{"type":"string","value":"a"},{"type":"string","value":"b"}]},` +
		`{"type":"array","value":[{"type":"string","value":"c"}]}]}}}`)

	sr, err := ParseScriptResponse(raw)
	if err != nil {
		t.Fatalf("ParseScriptResponse() error = %v", err)
	}

	want := []interface{}{
		[]interface{}{"a", "b"},
		[]interface{}{"c"},
	}
	if got := ConvertRemoteValue(sr.Result); !reflect.DeepEqual(got, want) {
		t.Errorf("ConvertRemoteValue() = %#v, want %#v", got, want)
	}
}
