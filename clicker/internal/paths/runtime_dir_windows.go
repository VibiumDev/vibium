//go:build windows

package paths

// checkRuntimeDir is a no-op: the daemon dir on Windows is the cache dir.
func checkRuntimeDir(string) error { return nil }
