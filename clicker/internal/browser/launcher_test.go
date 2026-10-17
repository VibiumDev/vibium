package browser

import (
	"runtime"
	"slices"
	"testing"
)

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

// TestChromeArgsCustomEnv verifies that flags supplied via VIBIUM_CHROME_ARGS
// are appended to the Chrome argv so users can pass --no-sandbox etc. in
// root/CI/container environments (issue #141).
func TestChromeArgsCustomEnv(t *testing.T) {
	t.Setenv("VIBIUM_CHROME_ARGS", "--no-sandbox --disable-gpu")

	args := chromeArgs(false, nil)

	if !slices.Contains(args, "--no-sandbox") {
		t.Errorf("expected --no-sandbox in chrome args, got %v", args)
	}
	if !slices.Contains(args, "--disable-gpu") {
		t.Errorf("expected --disable-gpu in chrome args, got %v", args)
	}
}

// TestChromeArgsCustomEnvUnset verifies the argv is unchanged when the env var
// is absent, and that no empty-string args ever leak into the list.
func TestChromeArgsCustomEnvUnset(t *testing.T) {
	t.Setenv("VIBIUM_CHROME_ARGS", "")

	args := chromeArgs(false, nil)

	if slices.Contains(args, "") {
		t.Errorf("expected no empty-string args, got %v", args)
	}
	// Windows always gets --no-sandbox from platformChromeArgs (the Chrome
	// for Testing sandbox cannot access its own executable under AppData),
	// so absence is only the expectation elsewhere (#660).
	if runtime.GOOS != "windows" && slices.Contains(args, "--no-sandbox") {
		t.Errorf("did not expect --no-sandbox when env unset, got %v", args)
	}
}

// TestChromeArgsCustomEnvWhitespace verifies that extra whitespace between and
// around flags is split cleanly without producing empty-string args.
func TestChromeArgsCustomEnvWhitespace(t *testing.T) {
	t.Setenv("VIBIUM_CHROME_ARGS", "  --no-sandbox   --disable-gpu  ")

	args := chromeArgs(false, nil)

	if slices.Contains(args, "") {
		t.Errorf("expected no empty-string args from whitespace, got %v", args)
	}
	if !slices.Contains(args, "--no-sandbox") || !slices.Contains(args, "--disable-gpu") {
		t.Errorf("expected both flags parsed from padded input, got %v", args)
	}
}

// Caller-supplied flags (StartOptions.args / --browser-arg, #640) are appended
// last, after the defaults and the env var, so an explicit flag wins.
func TestChromeArgsExtraAppendedLast(t *testing.T) {
	t.Setenv("VIBIUM_CHROME_ARGS", "")
	args := chromeArgs(false, []string{"--user-agent=Bot/1", "--lang=fr"})

	if !slices.Contains(args, "--user-agent=Bot/1") || !slices.Contains(args, "--lang=fr") {
		t.Fatalf("expected the extra flags in chrome args, got %v", args)
	}
	if got := args[len(args)-1]; got != "--lang=fr" {
		t.Errorf("extra flags should come last; last arg = %q", got)
	}
}
