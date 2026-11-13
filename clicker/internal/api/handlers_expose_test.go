package api

import (
	"reflect"
	"testing"
)

// The page posts {name, seq, args} through the expose channel; the router
// turns that into a vibium:expose.call event with the calling realm attached
// (#298).
func TestParseExposeCall(t *testing.T) {
	msg := `{"method":"script.message","params":{"channel":"vibium-expose",` +
		`"data":{"type":"string","value":"{\"name\":\"save\",\"seq\":3,\"args\":[\"x\",2]}"},` +
		`"source":{"context":"ctx-1","realm":"realm-1"}}}`

	params := parseExposeCall(msg)
	if params == nil {
		t.Fatal("a well-formed expose channel message must parse")
	}
	if params["name"] != "save" || params["seq"] != 3.0 {
		t.Fatalf("params = %v", params)
	}
	if !reflect.DeepEqual(params["args"], []interface{}{"x", 2.0}) {
		t.Fatalf("args = %v", params["args"])
	}
	if params["context"] != "ctx-1" || params["realm"] != "realm-1" {
		t.Fatalf("source not carried: %v", params)
	}
}

func TestParseExposeCallIgnoresOtherMessages(t *testing.T) {
	for _, msg := range []string{
		// The WebSocket monitor shares the script.message subscription.
		`{"method":"script.message","params":{"channel":"vibium-ws","data":{"type":"string","value":"{}"},"source":{"context":"c"}}}`,
		`{"method":"log.entryAdded","params":{"type":"console"}}`,
		`{"method":"script.message","params":{"channel":"vibium-expose","data":{"type":"number","value":5},"source":{"context":"c"}}}`,
		`{"method":"script.message","params":{"channel":"vibium-expose","data":{"type":"string","value":"not json"},"source":{"context":"c"}}}`,
		`not json`,
	} {
		if params := parseExposeCall(msg); params != nil {
			t.Fatalf("message %q must not parse as an expose call, got %v", msg, params)
		}
	}
}

// A call with no arguments still carries an empty args array, so clients can
// spread it without a nil check.
func TestParseExposeCallDefaultsArgs(t *testing.T) {
	msg := `{"method":"script.message","params":{"channel":"vibium-expose",` +
		`"data":{"type":"string","value":"{\"name\":\"ping\",\"seq\":1}"},` +
		`"source":{"context":"ctx-1","realm":"realm-1"}}}`
	params := parseExposeCall(msg)
	if params == nil {
		t.Fatal("must parse")
	}
	if !reflect.DeepEqual(params["args"], []interface{}{}) {
		t.Fatalf("args = %v", params["args"])
	}
}

// expose.result params become the three primitive deliver arguments: seq,
// result JSON or null, error message or null.
func TestExposeDeliverArgs(t *testing.T) {
	t.Run("result value", func(t *testing.T) {
		args, err := exposeDeliverArgs(map[string]interface{}{
			"seq": 3.0, "result": map[string]interface{}{"ok": true},
		})
		if err != nil {
			t.Fatal(err)
		}
		want := []map[string]interface{}{
			{"type": "number", "value": 3.0},
			{"type": "string", "value": `{"ok":true}`},
			{"type": "null"},
		}
		if !reflect.DeepEqual(args, want) {
			t.Fatalf("args = %v", args)
		}
	})

	t.Run("error takes precedence", func(t *testing.T) {
		args, err := exposeDeliverArgs(map[string]interface{}{
			"seq": 1.0, "error": "boom", "result": "ignored",
		})
		if err != nil {
			t.Fatal(err)
		}
		if args[1]["type"] != "null" || args[2]["value"] != "boom" {
			t.Fatalf("args = %v", args)
		}
	})

	t.Run("no result means undefined in the page", func(t *testing.T) {
		args, err := exposeDeliverArgs(map[string]interface{}{"seq": 2.0})
		if err != nil {
			t.Fatal(err)
		}
		if args[1]["type"] != "null" || args[2]["type"] != "null" {
			t.Fatalf("args = %v", args)
		}
	})

	t.Run("seq is required", func(t *testing.T) {
		if _, err := exposeDeliverArgs(map[string]interface{}{"result": 1}); err == nil {
			t.Fatal("a missing seq must error")
		}
	})
}
