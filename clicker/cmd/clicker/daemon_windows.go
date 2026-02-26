//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

// setSysProcAttr configures the child process for Windows.
// CREATE_NEW_PROCESS_GROUP detaches the daemon from the parent console.
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
