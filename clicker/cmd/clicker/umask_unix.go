//go:build !windows

package main

import "syscall"

func syscallUmask(mask int) int { return syscall.Umask(mask) }
