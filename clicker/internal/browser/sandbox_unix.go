//go:build !windows

package browser

import (
	"errors"
	"os"
	"strings"
)

// errRootSandbox explains the one launch failure that looks like a crash but is
// a deliberate refusal. Chrome will not start its sandbox as root, so a
// container running as root — which most CI base images do — got
// "browser crashed with exit code 1" and nothing to act on (#141).
//
// We do not disable the sandbox automatically. Running a browser unsandboxed is
// a real security tradeoff, and it should be the caller's decision rather than
// something vibium does quietly on their behalf.
var errRootSandbox = errors.New(
	"Chrome cannot run as root with its sandbox enabled.\n" +
		"  Set VIBIUM_CHROME_ARGS=--no-sandbox to disable it, or run as a non-root user.\n" +
		"  Disabling the sandbox removes a security boundary; prefer a non-root user where you can.")

// checkSandboxable returns an actionable error when the browser is certain to
// refuse to start, rather than letting it fail as an opaque crash.
func checkSandboxable() error {
	if os.Geteuid() != 0 {
		return nil
	}
	// Already opted out, so root is fine.
	for _, arg := range strings.Fields(os.Getenv("VIBIUM_CHROME_ARGS")) {
		if arg == "--no-sandbox" {
			return nil
		}
	}
	return errRootSandbox
}
