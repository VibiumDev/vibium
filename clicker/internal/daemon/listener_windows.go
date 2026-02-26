//go:build windows

package daemon

import (
	"net"

	"github.com/Microsoft/go-winio"
)

// listen creates a named pipe listener on Windows.
func listen(socketPath string) (net.Listener, error) {
	return winio.ListenPipe(socketPath, nil)
}
