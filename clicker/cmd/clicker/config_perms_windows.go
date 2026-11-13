//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"os/user"
)

// restrictToOwner cuts the file's ACL down to the owning user. Unix modes do
// not exist here — Chmod(0600) only clears the read-only attribute — so a
// default ACL inherited from the profile directory leaves the file readable
// by other accounts on the machine. icacls ships with every Windows, the
// same way the launcher already relies on taskkill and tasklist.
func restrictToOwner(path string) error {
	u, err := user.Current()
	if err != nil {
		return err
	}
	// /inheritance:r drops the inherited entries; /grant:r replaces whatever
	// is left with the owning user alone. SYSTEM and elevated Administrators
	// can still take ownership either way — the point is keeping ordinary
	// accounts out, matching what 0600 means on Unix.
	out, err := exec.Command("icacls", path, "/inheritance:r", "/grant:r", u.Username+":F").CombinedOutput()
	if err != nil {
		return fmt.Errorf("icacls: %v: %s", err, out)
	}
	return nil
}
