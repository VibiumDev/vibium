package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/vibium/clicker/internal/daemon"
	"github.com/vibium/clicker/internal/mcp"
	"github.com/vibium/clicker/internal/paths"
)

// daemonCall sends a tool call to the daemon, auto-starting if needed.
// Returns the result or an error.
func daemonCall(toolName string, args map[string]interface{}) (*mcp.ToolsCallResult, error) {
	// First attempt
	result, err := daemon.Call(toolName, args)
	if err == nil {
		return result, nil
	}

	// If connection refused, try auto-starting the daemon
	if !isConnectionError(err) {
		return nil, err
	}

	// Clean stale PID/socket files
	daemon.CleanStale()

	// Auto-start daemon
	if err := autoStartDaemon(); err != nil {
		return nil, fmt.Errorf("auto-start daemon: %w", err)
	}

	// Retry
	return daemon.Call(toolName, args)
}

// autoStartDaemon spawns a daemon process in the background and waits for it.
func autoStartDaemon() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find executable: %w", err)
	}

	args := []string{"daemon", "start", "--_internal", "--idle-timeout=30m"}
	if headless {
		args = append(args, "--headless")
	}

	cmd := exec.Command(exe, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	setSysProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start daemon process: %w", err)
	}

	// Poll for socket availability
	socketPath, err := paths.GetSocketPath()
	if err != nil {
		return fmt.Errorf("get socket path: %w", err)
	}

	return waitForSocket(socketPath, 5*time.Second)
}

// isConnectionError returns true if the error indicates the daemon is not running.
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	// Check for common connection-refused patterns
	if _, ok := err.(*net.OpError); ok {
		return true
	}
	// The daemon client wraps errors, so check the message
	errMsg := err.Error()
	for _, pattern := range []string{
		"connect to daemon",
		"connection refused",
		"no such file or directory",
		"The system cannot find the path",  // Windows named pipe not found
		"The system cannot find the file",  // Windows named pipe not found (alt)
	} {
		if containsString(errMsg, pattern) {
			return true
		}
	}
	return false
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
