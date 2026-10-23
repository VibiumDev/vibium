package browser

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// waitForProcessArgs polls the process table until the presence of needle
// matches want, so the test never races ps against process startup/teardown.
// It shells ps directly rather than using the production helper, so the test
// also compiles (and demonstrably fails) on a tree without the fix.
func waitForProcessArgs(t *testing.T, needle string, want bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		out, err := exec.Command("ps", "axww", "-o", "args=").Output()
		if err == nil && strings.Contains(string(out), needle) == want {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("process table never showed needle=%v for %s", want, needle)
}

// A temp dir referenced by a live process's command line must survive the
// sweep no matter how old its mtime is; once nothing references it, the same
// dir is reclaimed (#706). Before the fix, the sweep deleted a healthy
// sibling browser's dirs the moment they aged past minAge, wedging it.
func TestCleanupSparesLiveReferencedDirs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses tail and TMPDIR")
	}
	t.Setenv("TMPDIR", t.TempDir())

	dir := filepath.Join(os.TempDir(), "com.google.chrome.for.testing.sweeptest")
	keep := filepath.Join(dir, "keep")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keep, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-10 * time.Minute)
	if err := os.Chtimes(dir, old, old); err != nil {
		t.Fatal(err)
	}

	// A stand-in for a live browser: its argv references a file in the dir,
	// the way Chrome's --user-data-dir does.
	cmd := exec.Command("tail", "-f", keep)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()
	waitForProcessArgs(t, keep, true)

	CleanupOrphanedBrowserTempDirs(time.Minute)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("dir referenced by a live process was deleted: %v", err)
	}

	cmd.Process.Kill()
	cmd.Wait()
	waitForProcessArgs(t, keep, false)

	CleanupOrphanedBrowserTempDirs(time.Minute)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("unreferenced dir should be deleted, Stat err = %v", err)
	}
}

// Fresh directories stay protected by minAge alone, covering the window
// between a dir's creation and its browser appearing in the process table.
func TestCleanupSparesFreshDirs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses TMPDIR")
	}
	t.Setenv("TMPDIR", t.TempDir())

	dir := filepath.Join(os.TempDir(), "org.chromium.Chromium.scoped_dir.fresh")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	CleanupOrphanedBrowserTempDirs(time.Minute)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("fresh dir was deleted: %v", err)
	}
}
