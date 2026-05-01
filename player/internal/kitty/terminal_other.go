//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd

package kitty

import (
	"os"
	"time"
)

func isTerminal(_ *os.File) bool {
	return false
}

func query(_ *os.File, _ *os.File, _ string, _ time.Duration) (string, bool) {
	return "", false
}
