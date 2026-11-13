//go:build windows

package main

import (
	"os/exec"
	"os/user"
	"strings"
	"testing"
)

// assertOwnerOnly fails the test unless the file's DACL grants access to the
// current user alone. Group names localize across Windows languages but the
// user name does not, and after restrictToOwner the user must be the only
// entry, so matching on it stays locale-safe.
func assertOwnerOnly(t *testing.T, path string) {
	t.Helper()
	u, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("icacls", path).CombinedOutput()
	if err != nil {
		t.Fatalf("icacls %s: %v: %s", path, err, out)
	}
	var grants []string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, ":(") {
			grants = append(grants, strings.TrimSpace(line))
		}
	}
	if len(grants) != 1 {
		t.Errorf("%s: want exactly one ACL entry, got %d:\n%s", path, len(grants), out)
		return
	}
	if !strings.Contains(strings.ToLower(grants[0]), strings.ToLower(u.Username)) {
		t.Errorf("%s: the single ACL entry is not the current user: %s", path, grants[0])
	}
}
