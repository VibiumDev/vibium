//go:build windows

package main

// Windows has no umask; file modes are not enforced the same way.
func syscallUmask(mask int) int { return 0 }
