package daemon

import (
	"time"

	"github.com/vibium/clicker/internal/paths"
)

// IsRunning checks if a daemon is currently running.
// It verifies both that the PID file exists with a live process
// and that the socket is connectable.
func IsRunning() bool {
	pid, err := ReadPID()
	if err != nil || pid == 0 {
		return false
	}

	if !processExists(pid) {
		return false
	}

	socketPath, err := paths.GetSocketPath()
	if err != nil {
		return false
	}

	return socketConnectable(socketPath)
}

// socketConnectable tests if the daemon socket accepts connections.
func socketConnectable(socketPath string) bool {
	conn, err := dial(socketPath, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
