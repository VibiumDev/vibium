//go:build !windows

package browser

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// platformChromeArgs returns Unix-specific Chrome launch arguments.
func platformChromeArgs() []string {
	return nil
}

// setProcGroup sets the process group for the command (Unix only).
func setProcGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup kills all processes in the process group of the given PID.
func killProcessGroup(pid int) {
	pgid, err := syscall.Getpgid(pid)
	if err == nil {
		syscall.Kill(-pgid, syscall.SIGKILL)
	}
}

// killByPid sends SIGKILL to a process by PID.
func killByPid(pid int) {
	syscall.Kill(pid, syscall.SIGKILL)
}

// killBrowserProfileProcesses is a no-op on Unix: the process-group kill in
// Close already reaps the whole Firefox tree here. The profile sweep exists
// only to work around the Windows launcher process re-parenting (#622).
func killBrowserProfileProcesses(profileDir string) {}

// waitProcessGroupIdle polls pid's process group until no member other than
// pid itself remains, or timeout passes. Close calls this on a persistent
// profile after the session teardown sent BiDi browser.close: the browser (a
// grandchild via chromedriver on the Chrome path) is exiting and flushing its
// profile, and killing the group under it would lose those writes. SIGTERM is
// no gentler here — chromedriver reacts to it by hard-killing the browser.
// The root (chromedriver, which holds no state) is force-killed right after.
func waitProcessGroupIdle(pid int, timeout time.Duration) {
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		return
	}
	root := strconv.Itoa(pid)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := exec.Command("pgrep", "-g", strconv.Itoa(pgid)).Output()
		if err != nil {
			return // no members left at all
		}
		idle := true
		for _, member := range strings.Fields(string(out)) {
			if member != root {
				idle = false
				break
			}
		}
		if idle {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// waitForProcessDead polls until the given PID has exited or timeout is reached.
func waitForProcessDead(pid int, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if syscall.Kill(pid, 0) != nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}
