//go:build !windows

package main

import (
	"os"
	"testing"
)

// assertOwnerOnly fails the test unless the file is accessible by its owner
// alone: mode 0600 on Unix.
func assertOwnerOnly(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Errorf("%s mode = %o, want 600", path, got)
	}
}
