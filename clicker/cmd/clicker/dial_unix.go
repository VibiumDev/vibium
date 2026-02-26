//go:build !windows

package main

import (
	"net"
	"time"
)

// dialSocket connects to the daemon socket.
func dialSocket(addr string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("unix", addr, timeout)
}
