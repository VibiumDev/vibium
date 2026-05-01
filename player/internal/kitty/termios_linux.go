//go:build linux

package kitty

import "golang.org/x/sys/unix"

const (
	termiosGetReq = unix.TCGETS
	termiosSetReq = unix.TCSETS
)
