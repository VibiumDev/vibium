//go:build !windows

package process

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/vibium/clicker/internal/log"
)

// ReapOrphans kills browser processes launched from pathMarker (vibium's
// browser cache dir) that no longer have a live vibium ancestor. Best-effort:
// a failure to scan or kill is logged and otherwise ignored, since the launch
// this protects can still succeed.
func ReapOrphans(pathMarker string) {
	if pathMarker == "" {
		return
	}
	out, err := exec.Command("ps", "-axo", "pid=,ppid=,command=").Output()
	if err != nil {
		log.Debug("orphan scan failed", "error", err)
		return
	}
	for _, pid := range orphanPIDs(parseProcTable(string(out)), pathMarker, os.Getpid()) {
		log.Info("killing orphaned browser process", "pid", pid)
		if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
			log.Debug("failed to kill orphaned process", "pid", pid, "error", err)
		}
	}
}
