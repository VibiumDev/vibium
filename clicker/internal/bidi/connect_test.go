package bidi

import "testing"

func TestSessionIDFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"ws://127.0.0.1:9515/session/6ee2c5b3676daef013c740110218097c", "6ee2c5b3676daef013c740110218097c"},
		{"ws://localhost:4444/session/04f89ffbe4a92b5178448328f3fa5618/se/bidi", "04f89ffbe4a92b5178448328f3fa5618"},
		{"wss://cloud.example.com/session/abc123/", "abc123"},
		{"ws://localhost:9515/session", ""},
		{"wss://cloud.example.com/bidi", ""},
		{"://not a url", ""},
	}

	for _, tt := range tests {
		if got := SessionIDFromURL(tt.url); got != tt.want {
			t.Errorf("SessionIDFromURL(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}
