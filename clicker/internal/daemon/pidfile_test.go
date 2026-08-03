package daemon

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/vibium/clicker/internal/paths"
)

func TestWritePIDUsesOwnerOnlyPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not meaningful on windows")
	}

	withTempCacheDir(t)

	if err := WritePID(); err != nil {
		t.Fatalf("WritePID() error = %v", err)
	}
	t.Cleanup(func() {
		if err := RemovePID(); err != nil {
			t.Fatalf("RemovePID() error = %v", err)
		}
	})

	dir, err := paths.GetDaemonDir()
	if err != nil {
		t.Fatalf("GetDaemonDir() error = %v", err)
	}
	assertMode(t, dir, 0o700)

	pidPath, err := paths.GetPIDPath()
	if err != nil {
		t.Fatalf("GetPIDPath() error = %v", err)
	}
	assertMode(t, pidPath, 0o600)
}

func TestSecureSocketPathUsesOwnerOnlyPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix socket permissions do not apply on windows")
	}

	socketPath := filepath.Join(t.TempDir(), "vibium.sock")
	if err := os.WriteFile(socketPath, []byte("socket placeholder"), 0o666); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := secureSocketPath(socketPath); err != nil {
		t.Fatalf("secureSocketPath() error = %v", err)
	}
	assertMode(t, socketPath, 0o600)
}

func withTempCacheDir(t *testing.T) {
	t.Helper()

	baseDir, err := os.MkdirTemp("/tmp", "vibium-daemon-")
	if err != nil {
		t.Fatalf("MkdirTemp() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(baseDir); err != nil {
			t.Fatalf("RemoveAll() error = %v", err)
		}
	})

	cacheDir := filepath.Join(baseDir, "cache")
	original := os.Getenv("VIBIUM_CACHE_DIR")
	if err := os.Setenv("VIBIUM_CACHE_DIR", cacheDir); err != nil {
		t.Fatalf("Setenv() error = %v", err)
	}
	t.Cleanup(func() {
		if original == "" {
			if err := os.Unsetenv("VIBIUM_CACHE_DIR"); err != nil {
				t.Fatalf("Unsetenv() error = %v", err)
			}
			return
		}
		if err := os.Setenv("VIBIUM_CACHE_DIR", original); err != nil {
			t.Fatalf("restore env error = %v", err)
		}
	})
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}

	if got := info.Mode().Perm(); got != want {
		t.Fatalf("mode for %s = %04o, want %04o", path, got, want)
	}
}
