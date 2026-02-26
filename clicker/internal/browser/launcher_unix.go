//go:build !windows

package browser

import (
	"os/exec"
	"syscall"
)

// setProcGroup sets the process group for the command (Unix only).
func setProcGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killByPid sends SIGKILL to a process by PID.
func killByPid(pid int) {
	syscall.Kill(pid, syscall.SIGKILL)
}

// skipGracefulShutdown returns false on Unix. The graceful DELETE request
// works fine because Unix process trees are stable and killByPid uses
// SIGKILL on individual PIDs found via pgrep.
func skipGracefulShutdown() bool { return false }
