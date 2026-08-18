package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// fakeFirefoxInstall lays out a cache entry the way Mozilla's archives
// unpack, so executable resolution finds it.
func fakeFirefoxInstall(t *testing.T, cacheDir, channel, version string) {
	t.Helper()
	versionDir := filepath.Join(cacheDir, "firefox", channel, version)
	exe := FirefoxPathInVersion(versionDir)
	if err := os.MkdirAll(filepath.Dir(exe), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// VIBIUM_FIREFOX_VERSION pins which cached version launches: newest-cached
// would silently run a different Firefox than the pin installed (#326).
func TestFirefoxExecutablePinnedVersionWins(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("cache layout test uses unix permissions")
	}
	cacheDir := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cacheDir)
	t.Setenv("VIBIUM_FIREFOX_PATH", "")
	t.Setenv("VIBIUM_FIREFOX_CHANNEL", "")
	fakeFirefoxInstall(t, cacheDir, "release", "153.0.4")
	fakeFirefoxInstall(t, cacheDir, "release", "154.0")

	t.Setenv("VIBIUM_FIREFOX_VERSION", "153.0.4")
	got, err := GetFirefoxExecutable()
	if err != nil {
		t.Fatalf("GetFirefoxExecutable() error = %v", err)
	}
	want := FirefoxPathInVersion(filepath.Join(cacheDir, "firefox", "release", "153.0.4"))
	if got != want {
		t.Errorf("GetFirefoxExecutable() = %q, want the pinned %q", got, want)
	}
}

// A pin that is not in the cache is an error, not a fallback: install will
// fetch exactly that version, and launching anything else defeats the pin.
func TestFirefoxExecutableMissingPinnedVersionErrors(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cacheDir)
	t.Setenv("VIBIUM_FIREFOX_PATH", "")
	t.Setenv("VIBIUM_FIREFOX_CHANNEL", "")
	fakeFirefoxInstall(t, cacheDir, "release", "154.0")

	t.Setenv("VIBIUM_FIREFOX_VERSION", "153.0.4")
	if _, err := GetFirefoxExecutable(); err == nil {
		t.Fatal("GetFirefoxExecutable() should error when the pinned version is not cached")
	}
}

// Without a pin the newest cached version wins, as before.
func TestFirefoxExecutableUnpinnedPicksNewest(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cacheDir)
	t.Setenv("VIBIUM_FIREFOX_PATH", "")
	t.Setenv("VIBIUM_FIREFOX_CHANNEL", "")
	t.Setenv("VIBIUM_FIREFOX_VERSION", "")
	fakeFirefoxInstall(t, cacheDir, "release", "153.0.4")
	fakeFirefoxInstall(t, cacheDir, "release", "154.0")

	got, err := GetFirefoxExecutable()
	if err != nil {
		t.Fatalf("GetFirefoxExecutable() error = %v", err)
	}
	want := FirefoxPathInVersion(filepath.Join(cacheDir, "firefox", "release", "154.0"))
	if got != want {
		t.Errorf("GetFirefoxExecutable() = %q, want the newest %q", got, want)
	}
}
