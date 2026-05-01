//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package kitty

import (
	"errors"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	_, err = unix.IoctlGetTermios(int(f.Fd()), termiosGetReq)
	return err == nil
}

func query(in, out *os.File, queryText string, timeout time.Duration) (string, bool) {
	fd := int(in.Fd())
	oldTerm, err := unix.IoctlGetTermios(fd, termiosGetReq)
	if err != nil {
		return "", false
	}
	raw := *oldTerm
	raw.Iflag &^= unix.BRKINT | unix.ICRNL | unix.INPCK | unix.ISTRIP | unix.IXON
	raw.Oflag &^= unix.OPOST
	raw.Cflag |= unix.CS8
	raw.Lflag &^= unix.ECHO | unix.ICANON | unix.IEXTEN | unix.ISIG
	raw.Cc[unix.VMIN] = 0
	raw.Cc[unix.VTIME] = 1

	oldFlags, err := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	if err != nil {
		return "", false
	}

	if err := unix.IoctlSetTermios(fd, termiosSetReq, &raw); err != nil {
		return "", false
	}
	defer unix.IoctlSetTermios(fd, termiosSetReq, oldTerm)

	if _, err := unix.FcntlInt(uintptr(fd), unix.F_SETFL, oldFlags|unix.O_NONBLOCK); err != nil {
		return "", false
	}
	defer unix.FcntlInt(uintptr(fd), unix.F_SETFL, oldFlags)

	if _, err := out.WriteString(queryText); err != nil {
		return "", false
	}

	deadline := time.Now().Add(timeout)
	buf := make([]byte, 256)
	var resp []byte
	for time.Now().Before(deadline) {
		n, err := in.Read(buf)
		if n > 0 {
			resp = append(resp, buf[:n]...)
			if containsTerminator(resp) {
				return string(resp), true
			}
		}
		if err != nil && !errors.Is(err, unix.EAGAIN) && !errors.Is(err, unix.EWOULDBLOCK) {
			return string(resp), len(resp) > 0
		}
		time.Sleep(10 * time.Millisecond)
	}
	return string(resp), len(resp) > 0
}

func containsTerminator(b []byte) bool {
	for i := 0; i+1 < len(b); i++ {
		if b[i] == '\x1b' && b[i+1] == '\\' {
			return true
		}
		if b[i] == 't' {
			return true
		}
	}
	return false
}
