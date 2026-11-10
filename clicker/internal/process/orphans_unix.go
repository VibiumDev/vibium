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
// this protects can still succeed. Returns the pids it judged orphaned and
// sent SIGKILL (best-effort; a pid may already be gone).
func ReapOrphans(pathMarker string) []int {
	if pathMarker == "" {
		return nil
	}
	// -ww: never truncate the command column. Without it ps caps each line at
	// the terminal width (COLUMNS when there is no tty), and a vibium pipe
	// whose binary sits under a long path — a self-hosted runner workdir, a
	// Python venv's site-packages — gets its name cut mid-word. The ancestry
	// walk then finds no vibium owner and SIGKILLs a live session's browser
	// the moment a sibling vibium launches (#701, third cause).
	out, err := exec.Command("ps", "-axwwo", "pid=,ppid=,command=").Output()
	if err != nil {
		log.Debug("orphan scan failed", "error", err)
		return nil
	}
	entries := parseProcTable(string(out))
	byPid := make(map[int]procEntry, len(entries))
	for _, e := range entries {
		byPid[e.pid] = e
	}
	var killed []int
	for _, pid := range orphanPIDs(entries, pathMarker, os.Getpid()) {
		// Warn, not Info: this kill is invisible to the vibium that owns the
		// process if the judgment is wrong, and a kill that cannot be seen at
		// the default log level is undiagnosable from CI logs (#701).
		cmdline := byPid[pid].cmd
		if len(cmdline) > 160 {
			cmdline = cmdline[:160] + "..."
		}
		log.Warn("killing orphaned browser process",
			"pid", pid, "ppid", byPid[pid].ppid, "cmdline", cmdline)
		if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
			log.Debug("failed to kill orphaned process", "pid", pid, "error", err)
		}
		killed = append(killed, pid)
	}
	return killed
}
