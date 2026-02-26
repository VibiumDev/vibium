package daemon

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/vibium/clicker/internal/paths"
)

// WritePID writes the current process PID to the PID file.
func WritePID() error {
	pidPath, err := paths.GetPIDPath()
	if err != nil {
		return fmt.Errorf("get PID path: %w", err)
	}

	dir, err := paths.GetDaemonDir()
	if err != nil {
		return fmt.Errorf("get daemon dir: %w", err)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create daemon dir: %w", err)
	}

	return os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0644)
}

// ReadPID reads the PID from the PID file. Returns 0 if file doesn't exist.
func ReadPID() (int, error) {
	pidPath, err := paths.GetPIDPath()
	if err != nil {
		return 0, err
	}

	data, err := os.ReadFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID file content: %w", err)
	}

	return pid, nil
}

// RemovePID removes the PID file.
func RemovePID() error {
	pidPath, err := paths.GetPIDPath()
	if err != nil {
		return err
	}
	err = os.Remove(pidPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// CleanStale removes PID and socket files if the recorded process is no longer running.
func CleanStale() {
	pid, err := ReadPID()
	if err != nil || pid == 0 {
		return
	}

	if processExists(pid) {
		return
	}

	// Process is dead — clean up stale files
	RemovePID()

	// On Unix, remove the stale socket file. On Windows, named pipes are
	// managed by the kernel and don't leave files to clean up.
	if runtime.GOOS != "windows" {
		socketPath, err := paths.GetSocketPath()
		if err == nil {
			os.Remove(socketPath)
		}
	}
}
