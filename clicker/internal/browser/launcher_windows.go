//go:build windows

package browser

import (
	"bytes"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

// platformChromeArgs returns Windows-specific Chrome launch arguments.
func platformChromeArgs() []string {
	// Chrome for Testing sandbox cannot access its own executable in AppData
	// due to Windows filesystem permission restrictions.
	return []string{"--no-sandbox"}
}

// setProcGroup prevents chromedriver from inheriting the parent's console
// window on Windows. This stops Chrome's stderr messages (GPU errors,
// "DevTools listening on...") from leaking into the terminal.
func setProcGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}

// killProcessGroup is a no-op on Windows — killByPid already uses taskkill /T
// which kills the entire process tree.
func killProcessGroup(pid int) {}

// killByPid kills a process tree by PID on Windows.
func killByPid(pid int) {
	// /T kills the entire process tree, /F forces termination
	exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprintf("%d", pid)).Run()
}

// killBrowserProfileProcesses terminates the Firefox processes started against
// this session's profile dir. taskkill /T on the launched PID misses them
// because the Windows launcher process re-parents the real browser out of our
// tree (#622); matching on the unique profile instead is parentage-independent
// and leaves the user's own Firefox untouched.
func killBrowserProfileProcesses(profileDir string) {
	script := firefoxProfileKillScript(profileDir)
	exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Run()
}

// waitForProcessDead polls until the given PID has exited or timeout is reached.
func waitForProcessDead(pid int, timeout time.Duration) {
	// Brief initial sleep to let the OS reap process table entries
	time.Sleep(50 * time.Millisecond)

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		// tasklist /FI filters by PID and /FO CSV gives parseable output.
		out, err := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH").Output()
		if err != nil || !bytes.Contains(out, []byte(fmt.Sprintf("%d", pid))) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}
