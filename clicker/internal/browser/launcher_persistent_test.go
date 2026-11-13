package browser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A user-supplied profile dir and its contents must survive Close (#715);
// only the throwaway temp profile may be removed.
func TestClosePersistentUserDataDirSurvives(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "profile")
	marker := filepath.Join(dir, "Cookies")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte("state"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := &LaunchResult{UserDataDir: dir, Persistent: true}
	if err := r.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("persistent profile contents gone after Close: %v", err)
	}
	if string(data) != "state" {
		t.Fatalf("marker = %q, want %q", data, "state")
	}
}

func TestCloseTempUserDataDirRemoved(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "profile")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	r := &LaunchResult{UserDataDir: dir}
	if err := r.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("temp profile still exists after Close (stat err = %v)", err)
	}
}

// prepareUserDataDir must create a missing dir and return an absolute path,
// so a relative CLI value works and the launch flags are unambiguous.
func TestPrepareUserDataDirCreatesAndResolves(t *testing.T) {
	got, err := prepareUserDataDir(filepath.Join(t.TempDir(), "nested", "profile"))
	if err != nil {
		t.Fatalf("prepareUserDataDir() error = %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("prepareUserDataDir() = %q, want absolute path", got)
	}
	if info, err := os.Stat(got); err != nil || !info.IsDir() {
		t.Fatalf("dir not created: %v", err)
	}
}

// The orphan temp-dir sweep only globs vibium's own temp-dir name patterns,
// so a persistent profile outside os.TempDir() can never match. This pins
// that the sweep leaves such a dir alone even when it is stale by mtime.
func TestCleanupOrphanedBrowserTempDirsIgnoresUserDirs(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "vibium-profile")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-10 * time.Minute)
	if err := os.Chtimes(dir, old, old); err != nil {
		t.Fatal(err)
	}

	CleanupOrphanedBrowserTempDirs(0)

	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("user dir swept: %v", err)
	}
}

// Existing user prefs in a persistent profile must be kept; vibium's own
// prefs are replaced in place so the write is idempotent (#715).
func TestWriteFirefoxPrefsKeepsUnrelatedPrefs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "user.js")
	existing := `user_pref("browser.startup.page", 3);
user_pref("custom.user.pref", "keep-me");
`
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := writeFirefoxPrefs(dir); err != nil {
		t.Fatalf("writeFirefoxPrefs() error = %v", err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	got := string(first)
	if !strings.Contains(got, `user_pref("custom.user.pref", "keep-me");`) {
		t.Errorf("unrelated user pref clobbered:\n%s", got)
	}
	if strings.Count(got, `user_pref("browser.startup.page"`) != 1 {
		t.Errorf("overridden pref not replaced exactly once:\n%s", got)
	}
	if !strings.Contains(got, `user_pref("browser.startup.page", 0);`) {
		t.Errorf("vibium pref value not applied:\n%s", got)
	}

	if err := writeFirefoxPrefs(dir); err != nil {
		t.Fatalf("second writeFirefoxPrefs() error = %v", err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(second) != got {
		t.Errorf("writeFirefoxPrefs not idempotent:\nfirst:\n%s\nsecond:\n%s", got, second)
	}
}
