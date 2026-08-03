package api

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// page.evaluate returning a nested array previously kept the inner elements as
// BiDi typed objects, because the array branch unwrapped only one level
// (issue #124).
func TestDeserializeScriptResultNestedArray(t *testing.T) {
	resp := json.RawMessage(`{"result":{"type":"success","result":` +
		`{"type":"array","value":[` +
		`{"type":"array","value":[{"type":"string","value":"a"},{"type":"string","value":"b"}]},` +
		`{"type":"number","value":2}]}}}`)

	got, err := deserializeScriptResult(resp)
	if err != nil {
		t.Fatalf("deserializeScriptResult() error = %v", err)
	}

	want := []interface{}{
		[]interface{}{"a", "b"},
		float64(2),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("deserializeScriptResult() = %#v, want %#v", got, want)
	}
}

func TestDeserializeScriptResultNestedObject(t *testing.T) {
	resp := json.RawMessage(`{"result":{"type":"success","result":` +
		`{"type":"object","value":[` +
		`["outer",{"type":"object","value":[["inner",{"type":"string","value":"v"}]]}]]}}}`)

	got, err := deserializeScriptResult(resp)
	if err != nil {
		t.Fatalf("deserializeScriptResult() error = %v", err)
	}

	want := map[string]interface{}{
		"outer": map[string]interface{}{"inner": "v"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("deserializeScriptResult() = %#v, want %#v", got, want)
	}
}

// A thrown exception previously fell through to the default branch and returned
// a nil value with no error, so page.evaluate reported null instead of failing.
func TestDeserializeScriptResultSurfacesException(t *testing.T) {
	resp := json.RawMessage(`{"result":{"type":"exception","exceptionDetails":` +
		`{"text":"ReferenceError: nope is not defined"}}}`)

	got, err := deserializeScriptResult(resp)
	if err == nil {
		t.Fatalf("deserializeScriptResult() = %#v, want an error", got)
	}
	if !strings.Contains(err.Error(), "ReferenceError: nope is not defined") {
		t.Errorf("error = %q, want it to carry the exception text", err.Error())
	}
}

func TestDeserializeScriptResultPrimitives(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want interface{}
	}{
		{"string", `{"type":"string","value":"hi"}`, "hi"},
		{"number", `{"type":"number","value":3}`, float64(3)},
		{"boolean", `{"type":"boolean","value":true}`, true},
		{"null", `{"type":"null"}`, nil},
		{"undefined", `{"type":"undefined"}`, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := json.RawMessage(`{"result":{"type":"success","result":` + tt.raw + `}}`)
			got, err := deserializeScriptResult(resp)
			if err != nil {
				t.Fatalf("deserializeScriptResult() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("deserializeScriptResult() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// parseScriptResult keeps its string contract and its exception behavior from
// the #111/#117 fix.
func TestParseScriptResultStringContract(t *testing.T) {
	ok := json.RawMessage(`{"result":{"type":"success","result":{"type":"string","value":"payload"}}}`)
	got, err := parseScriptResult(ok)
	if err != nil {
		t.Fatalf("parseScriptResult() error = %v", err)
	}
	if got != "payload" {
		t.Errorf("parseScriptResult() = %q, want %q", got, "payload")
	}

	exc := json.RawMessage(`{"result":{"type":"exception","exceptionDetails":{"text":"Illegal invocation"}}}`)
	if _, err = parseScriptResult(exc); err == nil {
		t.Fatal("parseScriptResult() returned nil error for an exception result")
	} else if err.Error() != "Illegal invocation" {
		t.Errorf("error = %q, want %q", err.Error(), "Illegal invocation")
	}

	null := json.RawMessage(`{"result":{"type":"success","result":{"type":"null"}}}`)
	if _, err = parseScriptResult(null); err == nil || !strings.Contains(err.Error(), "script returned null") {
		t.Errorf("parseScriptResult() error = %v, want it to report a null result", err)
	}
}
