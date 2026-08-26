//go:build windows

package process

// ReapOrphans is a no-op on Windows: cleanup there kills by executable name
// (see the daemon shutdown path), and the ps-based ancestry scan has no
// direct equivalent.
func ReapOrphans(pathMarker string) {}
