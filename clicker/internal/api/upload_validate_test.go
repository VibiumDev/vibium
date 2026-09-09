package api

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Regression tests for #480: a bad upload path used to go straight to the
// engine, and what it meant depended on which engine answered.
func TestValidateUploadFiles(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.txt")
	if err := os.WriteFile(good, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("existing file passes", func(t *testing.T) {
		if err := validateUploadFiles([]string{good}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("empty list passes", func(t *testing.T) {
		if err := validateUploadFiles(nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("missing file is rejected", func(t *testing.T) {
		err := validateUploadFiles([]string{filepath.Join(dir, "nope.txt")})
		if err == nil || !strings.Contains(err.Error(), "does not exist") {
			t.Fatalf("want does-not-exist error, got: %v", err)
		}
	})

	t.Run("directory is rejected", func(t *testing.T) {
		err := validateUploadFiles([]string{dir})
		if err == nil || !strings.Contains(err.Error(), "is a directory") {
			t.Fatalf("want directory error, got: %v", err)
		}
	})

	t.Run("unreadable file is rejected", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Getuid() == 0 {
			t.Skip("permission bits are not enforced here")
		}
		locked := filepath.Join(dir, "locked.txt")
		if err := os.WriteFile(locked, []byte("secret"), 0o000); err != nil {
			t.Fatal(err)
		}
		err := validateUploadFiles([]string{locked})
		if err == nil || !strings.Contains(err.Error(), "cannot read") {
			t.Fatalf("want cannot-read error, got: %v", err)
		}
	})

	t.Run("second file missing rejects the whole call", func(t *testing.T) {
		err := validateUploadFiles([]string{good, filepath.Join(dir, "nope.txt")})
		if err == nil {
			t.Fatal("want error, got nil")
		}
	})
}
