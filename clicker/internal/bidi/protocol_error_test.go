package bidi

import "testing"

// A WebDriver BiDi error response carries the error code in `error` and the
// human-readable detail in a sibling `message` field:
//
//	{"type":"error","id":1,"error":"invalid argument","message":"invalid argument: ..."}
//
// GetError must report the detail, not a second copy of the code (#343).
func TestGetErrorKeepsSiblingMessage(t *testing.T) {
	raw := `{"type":"error","id":1,"error":"invalid argument",` +
		`"message":"Invalid URL: not-a-real-url"}`

	msg, err := UnmarshalMessage([]byte(raw))
	if err != nil {
		t.Fatalf("UnmarshalMessage: %v", err)
	}
	if !msg.IsError() {
		t.Fatal("expected an error message")
	}

	errData, err := msg.GetError()
	if err != nil {
		t.Fatalf("GetError: %v", err)
	}
	if errData.Error != "invalid argument" {
		t.Errorf("code = %q, want %q", errData.Error, "invalid argument")
	}
	want := "Invalid URL: not-a-real-url"
	if errData.Message != want {
		t.Errorf("detail = %q, want %q", errData.Message, want)
	}
}

// The nested shape must keep working.
func TestGetErrorNestedObjectStillParses(t *testing.T) {
	raw := `{"type":"error","id":1,"error":{"error":"invalid argument","message":"detail here"}}`
	msg, err := UnmarshalMessage([]byte(raw))
	if err != nil {
		t.Fatalf("UnmarshalMessage: %v", err)
	}
	errData, err := msg.GetError()
	if err != nil {
		t.Fatalf("GetError: %v", err)
	}
	if errData.Message != "detail here" {
		t.Errorf("detail = %q, want %q", errData.Message, "detail here")
	}
}

// A bare string error with no sibling message still reports the code rather
// than an empty detail.
func TestGetErrorBareStringWithoutMessage(t *testing.T) {
	msg, err := UnmarshalMessage([]byte(`{"type":"error","id":1,"error":"unknown error"}`))
	if err != nil {
		t.Fatalf("UnmarshalMessage: %v", err)
	}
	errData, err := msg.GetError()
	if err != nil {
		t.Fatalf("GetError: %v", err)
	}
	if errData.Message != "unknown error" {
		t.Errorf("detail = %q, want the code as fallback", errData.Message)
	}
}
