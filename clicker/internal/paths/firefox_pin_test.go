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

// VIBIUM_ENGINE_VERSION pins which cached version launches: newest-cached
// would silently run a different Firefox than the pin installed (#326).
func TestFirefoxExecutablePinnedVersionWins(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("cache layout test uses unix permissions")
	}
	cacheDir := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cacheDir)
	t.Setenv("VIBIUM_ENGINE_PATH", "")
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "")
	fakeFirefoxInstall(t, cacheDir, "release", "153.0.4")
	fakeFirefoxInstall(t, cacheDir, "release", "154.0")

	t.Setenv("VIBIUM_ENGINE_VERSION", "153.0.4")
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
	t.Setenv("VIBIUM_ENGINE_PATH", "")
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "")
	fakeFirefoxInstall(t, cacheDir, "release", "154.0")

	t.Setenv("VIBIUM_ENGINE_VERSION", "153.0.4")
	if _, err := GetFirefoxExecutable(); err == nil {
		t.Fatal("GetFirefoxExecutable() should error when the pinned version is not cached")
	}
}

// The release channel resolves the baked pin even when a newer version is
// cached. Newest-cached meant a Firefox pin bump never installed and a
// rollback never launched (#584).
func TestFirefoxExecutableReleaseHonorsBakedPin(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cacheDir)
	t.Setenv("VIBIUM_ENGINE_PATH", "")
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "")
	t.Setenv("VIBIUM_ENGINE_VERSION", "")
	fakeFirefoxInstall(t, cacheDir, "release", PinnedFirefoxVersion)
	fakeFirefoxInstall(t, cacheDir, "release", "999.0")

	got, err := GetFirefoxExecutable()
	if err != nil {
		t.Fatalf("GetFirefoxExecutable() error = %v", err)
	}
	want := FirefoxPathInVersion(filepath.Join(cacheDir, "firefox", "release", PinnedFirefoxVersion))
	if got != want {
		t.Errorf("GetFirefoxExecutable() = %q, want the baked %q", got, want)
	}
}

// A release cache without the pinned version fails instead of launching
// whatever is newest; ensure-install then downloads the pin (#584).
func TestFirefoxExecutableReleaseRequiresBakedPin(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cacheDir)
	t.Setenv("VIBIUM_ENGINE_PATH", "")
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "")
	t.Setenv("VIBIUM_ENGINE_VERSION", "")
	fakeFirefoxInstall(t, cacheDir, "release", "154.0")

	if _, err := GetFirefoxExecutable(); err == nil {
		t.Fatal("GetFirefoxExecutable() should error when the pinned version is not cached")
	}
}

// Beta has no baked pin and keeps resolving newest-cached.
func TestFirefoxExecutableBetaPicksNewest(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cacheDir)
	t.Setenv("VIBIUM_ENGINE_PATH", "")
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "beta")
	t.Setenv("VIBIUM_ENGINE_VERSION", "")
	fakeFirefoxInstall(t, cacheDir, "beta", "153.0b4")
	fakeFirefoxInstall(t, cacheDir, "beta", "154.0b1")

	got, err := GetFirefoxExecutable()
	if err != nil {
		t.Fatalf("GetFirefoxExecutable() error = %v", err)
	}
	want := FirefoxPathInVersion(filepath.Join(cacheDir, "firefox", "beta", "154.0b1"))
	if got != want {
		t.Errorf("GetFirefoxExecutable() = %q, want the newest %q", got, want)
	}
}

// A wrong-architecture cached Firefox must be rejected and deleted so
// ensure-install downloads the right build instead of launching into
// "exec format error" (#704, the Firefox twin of the Chrome fix #699).
func TestFirefoxExecutableRejectsWrongArch(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cacheDir)
	t.Setenv("VIBIUM_ENGINE_PATH", "")
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "")
	t.Setenv("VIBIUM_ENGINE_VERSION", "153.0.4")

	dir := filepath.Join(cacheDir, "firefox", "release", "153.0.4")
	buildTinyBinary(t, foreignArch(), FirefoxPathInVersion(dir))

	if _, err := GetFirefoxExecutable(); err == nil {
		t.Error("GetFirefoxExecutable() with a wrong-arch cached pin: want error, got nil")
	}
	// The rejected directory is deleted so the reinstall extracts cleanly.
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("wrong-arch Firefox version dir should be deleted, Stat() err = %v", err)
	}

	// Rebuilt for the host arch, the same pin resolves.
	buildTinyBinary(t, runtime.GOARCH, FirefoxPathInVersion(dir))
	if _, err := GetFirefoxExecutable(); err != nil {
		t.Errorf("GetFirefoxExecutable() with a host-arch cached pin: %v", err)
	}
}
