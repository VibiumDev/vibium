package browser

import "testing"

// A session-creation failure used to surface as bare "HTTP 500", discarding the
// body that says why (#107).
func TestDriverErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "webdriver error shape",
			body: `{"value":{"error":"session not created","message":"session not created: This version of ChromeDriver only supports Chrome version 146","stacktrace":"..."}}`,
			want: "session not created: This version of ChromeDriver only supports Chrome version 146",
		},
		{
			name: "error without message",
			body: `{"value":{"error":"unknown error"}}`,
			want: "unknown error",
		},
		{
			name: "not the expected shape falls back to the raw body",
			body: "upstream proxy exploded",
			want: "upstream proxy exploded",
		},
		{
			name: "empty body is stated rather than blank",
			body: "",
			want: "(empty response body)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := driverErrorMessage([]byte(tt.body)); got != tt.want {
				t.Errorf("driverErrorMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDriverErrorMessageTruncatesRunaway(t *testing.T) {
	body := make([]byte, 4000)
	for i := range body {
		body[i] = 'x'
	}
	got := driverErrorMessage(body)
	if len(got) > 520 {
		t.Errorf("message length %d, want it truncated", len(got))
	}
}
