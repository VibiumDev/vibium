//go:build !windows

package paths

import (
	"fmt"
	"os"
	"syscall"
)

// checkRuntimeDir refuses a runtime dir another user could control.
func checkRuntimeDir(dir string) error {
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("runtime dir %s is not a directory; set VIBIUM_RUNTIME_DIR to another path", dir)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("runtime dir %s is group/world accessible (%#o); fix its permissions or set VIBIUM_RUNTIME_DIR", dir, info.Mode().Perm())
	}
	if st, ok := info.Sys().(*syscall.Stat_t); ok && int(st.Uid) != os.Getuid() {
		return fmt.Errorf("runtime dir %s is owned by uid %d, not you; set VIBIUM_RUNTIME_DIR", dir, st.Uid)
	}
	return nil
}
