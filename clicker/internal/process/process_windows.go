//go:build windows

package process

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"

	"github.com/vibium/clicker/internal/log"
)

// killProcess kills a process tree on Windows.
func killProcess(cmd *exec.Cmd) {
	if cmd.Process != nil {
		exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprintf("%d", cmd.Process.Pid)).Run()
	}
}

// setupSignalNotify sets up signal notification for Windows.
func setupSignalNotify(c chan os.Signal) {
	signal.Notify(c, os.Interrupt)
}

// WithCleanup wraps a function with panic recovery that ensures browser cleanup.
func WithCleanup(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("panic recovered, cleaning up browsers", "panic", r)
			KillAll()
			panic(r)
		}
	}()
	fn()
}
