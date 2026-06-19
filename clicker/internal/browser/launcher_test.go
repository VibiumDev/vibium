package browser

import (
	"slices"
	"testing"
)

// TestChromeArgsCustomEnv verifies that flags supplied via VIBIUM_CHROME_ARGS
// are appended to the Chrome argv so users can pass --no-sandbox etc. in
// root/CI/container environments (issue #141).
func TestChromeArgsCustomEnv(t *testing.T) {
	t.Setenv("VIBIUM_CHROME_ARGS", "--no-sandbox --disable-gpu")

	args := chromeArgs(false)

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

	args := chromeArgs(false)

	if slices.Contains(args, "") {
		t.Errorf("expected no empty-string args, got %v", args)
	}
	if slices.Contains(args, "--no-sandbox") {
		t.Errorf("did not expect --no-sandbox when env unset, got %v", args)
	}
}

// TestChromeArgsCustomEnvWhitespace verifies that extra whitespace between and
// around flags is split cleanly without producing empty-string args.
func TestChromeArgsCustomEnvWhitespace(t *testing.T) {
	t.Setenv("VIBIUM_CHROME_ARGS", "  --no-sandbox   --disable-gpu  ")

	args := chromeArgs(false)

	if slices.Contains(args, "") {
		t.Errorf("expected no empty-string args from whitespace, got %v", args)
	}
	if !slices.Contains(args, "--no-sandbox") || !slices.Contains(args, "--disable-gpu") {
		t.Errorf("expected both flags parsed from padded input, got %v", args)
	}
}
