//go:build !windows

package browser

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A healthy sibling browser's profile dir is older than minAge too, so the
// sweep must not delete a dir a live process still references in its argv; a
// genuinely orphaned dir must still be reclaimed (#706).
func TestCleanupOrphanedBrowserTempDirsSparesLiveDirs(t *testing.T) {
	tmp := t.TempDir()
	// os.TempDir() reads TMPDIR, so the sweep only touches this isolated dir.
	t.Setenv("TMPDIR", tmp)

	old := time.Now().Add(-10 * time.Minute)

	liveDir := filepath.Join(tmp, "com.google.chrome.for.testing.live")
	if err := os.MkdirAll(liveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(liveDir, old, old); err != nil {
		t.Fatal(err)
	}

	orphanDir := filepath.Join(tmp, "com.google.chrome.for.testing.orphan")
	if err := os.MkdirAll(orphanDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(orphanDir, old, old); err != nil {
		t.Fatal(err)
	}

	// A live process whose argv references liveDir, the way a running Chrome
	// carries its --user-data-dir. The command is compound ("; :") so sh does
	// not exec-optimize into bare "sleep" and keeps liveDir in its argv, where
	// ps reports it.
	cmd := exec.Command("sh", "-c", "sleep 30; :", "vibium-sweep-test", liveDir)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	// Wait until the process is visible in the table before sweeping.
	deadline := time.Now().Add(5 * time.Second)
	for {
		tbl, err := listProcessTable()
		if err == nil && strings.Contains(tbl, liveDir) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("live process never appeared in the process table")
		}
		time.Sleep(50 * time.Millisecond)
	}

	CleanupOrphanedBrowserTempDirs(time.Minute)

	if _, err := os.Stat(liveDir); os.IsNotExist(err) {
		t.Error("a dir referenced by a live process must not be deleted (#706)")
	}
	if _, err := os.Stat(orphanDir); !os.IsNotExist(err) {
		t.Errorf("an orphaned dir should be reclaimed, Stat() err = %v", err)
	}
}
