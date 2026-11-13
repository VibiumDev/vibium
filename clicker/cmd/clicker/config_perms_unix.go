//go:build !windows

package main

import "os"

// restrictToOwner makes the file readable and writable by its owner only.
// WriteFile's mode respects umask, so an inherited one can widen the file
// behind us; chmod after the write closes that.
func restrictToOwner(path string) error {
	return os.Chmod(path, 0600)
}
