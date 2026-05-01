//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package kitty

import "golang.org/x/sys/unix"

const (
	termiosGetReq = unix.TIOCGETA
	termiosSetReq = unix.TIOCSETA
)
