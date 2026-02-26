//go:build windows

package browser

import (
	"fmt"
	"os/exec"
)

// setProcGroup is a no-op on Windows.
func setProcGroup(cmd *exec.Cmd) {
	// Windows doesn't use process groups the same way
}

// skipGracefulShutdown returns true on Windows because the graceful DELETE
// request can cause chromedriver to exit before taskkill /T runs, orphaning
// Chrome child processes. taskkill /T handles the full tree kill instead.
func skipGracefulShutdown() bool { return true }

// killByPid kills a process tree by PID on Windows.
func killByPid(pid int) {
	// /T kills the entire process tree, /F forces termination
	exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprintf("%d", pid)).Run()
}
