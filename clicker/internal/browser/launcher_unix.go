//go:build !windows

package browser

import (
	"os/exec"
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

// listProcessTable returns every running process's command line as one blob,
// so the temp-dir sweep can tell whether a candidate dir still belongs to a
// live browser (chrome/chromedriver/crashpad carry the dir in argv).
func listProcessTable() (string, error) {
	out, err := exec.Command("ps", "-axo", "command=").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
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
