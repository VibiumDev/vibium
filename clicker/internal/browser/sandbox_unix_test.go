//go:build !windows

package browser

import (
	"strings"
	"testing"
)

// Chrome refuses to start its sandbox as root, which surfaced as
// "browser crashed with exit code 1" with nothing to act on (#141).
func TestCheckSandboxableExplainsTheRootRefusal(t *testing.T) {
	if errRootSandbox == nil {
		t.Fatal("expected a root sandbox error to exist")
	}
	msg := errRootSandbox.Error()

	// The message has to name the cause and the fix, since the whole point is
	// that the crash it replaces named neither.
	for _, want := range []string{"root", "VIBIUM_CHROME_ARGS=--no-sandbox", "non-root user"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message should mention %q, got:\n%s", want, msg)
		}
	}
}

func TestCheckSandboxablePassesForNonRoot(t *testing.T) {
	// The test process is not root, so this exercises the ordinary path.
	if err := checkSandboxable(); err != nil {
		t.Fatalf("non-root should launch normally, got: %v", err)
	}
}

func TestCheckSandboxableAcceptsAnExplicitOptOut(t *testing.T) {
	// Someone who has already set --no-sandbox has made the decision; do not
	// block them for being root.
	t.Setenv("VIBIUM_CHROME_ARGS", "--disable-gpu --no-sandbox")
	if err := checkSandboxable(); err != nil {
		t.Fatalf("explicit opt-out should be honoured, got: %v", err)
	}
}
