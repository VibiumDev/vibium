//go:build !windows

package daemon

import (
	"net"
	"time"
)

// dial connects to the daemon socket.
func dial(addr string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("unix", addr, timeout)
}
