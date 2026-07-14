package bidi

import (
	"encoding/json"
	"reflect"
	"testing"
)

// parseRemoteValue decodes a raw BiDi RemoteValue JSON payload the same way the
// Evaluate/CallFunction code paths do, so tests exercise the real decode shape
// (numbers become float64, nested values become map[string]interface{}).
func parseRemoteValue(t *testing.T, raw string) RemoteValue {
	t.Helper()
	var rv RemoteValue
	if err := json.Unmarshal([]byte(raw), &rv); err != nil {
		t.Fatalf("unmarshal %q: %v", raw, err)
	}
	return rv
}

func TestConvertRemoteValue(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want interface{}
	}{
		{
			name: "string",
			raw:  `{"type":"string","value":"hello"}`,
			want: "hello",
		},
		{
			name: "number",
			raw:  `{"type":"number","value":3}`,
			want: float64(3),
		},
		{
			name: "boolean",
			raw:  `{"type":"boolean","value":true}`,
			want: true,
		},
		{
			name: "null",
			raw:  `{"type":"null"}`,
			want: nil,
		},
		{
			name: "undefined",
			raw:  `{"type":"undefined"}`,
			want: nil,
		},
		{
			name: "array of primitives",
			raw: `{"type":"array","value":[` +
				`{"type":"number","value":1},` +
				`{"type":"number","value":2},` +
				`{"type":"number","value":3}]}`,
			want: []interface{}{float64(1), float64(2), float64(3)},
		},
		{
			name: "flat object",
			raw: `{"type":"object","value":[` +
				`["title",{"type":"string","value":"Example"}],` +
				`["count",{"type":"number","value":5}]]}`,
			want: map[string]interface{}{"title": "Example", "count": float64(5)},
		},
		{
			name: "nested object",
			raw: `{"type":"object","value":[` +
				`["a",{"type":"object","value":[` +
				`["b",{"type":"number","value":1}]]}]]}`,
			want: map[string]interface{}{
				"a": map[string]interface{}{"b": float64(1)},
			},
		},
		{
			name: "array of objects",
			raw: `{"type":"array","value":[` +
				`{"type":"object","value":[["x",{"type":"number","value":1}]]}]}`,
			want: []interface{}{
				map[string]interface{}{"x": float64(1)},
			},
		},
		{
			name: "object with non-string key is skipped",
			raw: `{"type":"object","value":[` +
				`[{"type":"object","value":[]},{"type":"number","value":1}],` +
				`["ok",{"type":"number","value":2}]]}`,
			want: map[string]interface{}{"ok": float64(2)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertRemoteValue(parseRemoteValue(t, tt.raw))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertRemoteValue() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
