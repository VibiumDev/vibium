package paths

import (
	"os"
	"path/filepath"
	"testing"
)

// installFakeChrome creates the files resolveVersionDir requires (both Chrome
// and chromedriver) inside cacheDir under the given channel-relative segments.
func installFakeChrome(t *testing.T, cacheDir string, segments ...string) {
	t.Helper()
	dir := filepath.Join(append([]string{cacheDir, "chrome-for-testing"}, segments...)...)
	for _, p := range []string{getChromePathInVersion(dir), getChromedriverPathInVersion(dir)} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("fake"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResolveVersionDirHonorsPin(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cache)
	installFakeChrome(t, cache, "140.0.7000.10")
	installFakeChrome(t, cache, "141.0.7100.20")

	// Unpinned picks the newest complete version.
	dir, err := resolveVersionDir()
	if err != nil {
		t.Fatalf("resolveVersionDir() error = %v", err)
	}
	if got := filepath.Base(dir); got != "141.0.7100.20" {
		t.Errorf("unpinned resolveVersionDir() = %s, want 141.0.7100.20", got)
	}

	// A pin selects that version even with a newer one cached.
	t.Setenv("VIBIUM_ENGINE_VERSION", "140.0.7000.10")
	dir, err = resolveVersionDir()
	if err != nil {
		t.Fatalf("pinned resolveVersionDir() error = %v", err)
	}
	if got := filepath.Base(dir); got != "140.0.7000.10" {
		t.Errorf("pinned resolveVersionDir() = %s, want 140.0.7000.10", got)
	}

	// A pin on a version that is not cached fails instead of silently
	// falling back to whatever is newest.
	t.Setenv("VIBIUM_ENGINE_VERSION", "139.0.6900.1")
	if _, err := resolveVersionDir(); err == nil {
		t.Error("resolveVersionDir() with missing pinned version: want error, got nil")
	}
}

func TestChromeChannelDirsAreSeparate(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cache)
	installFakeChrome(t, cache, "141.0.7100.20")
	// A beta with a higher version number than stable.
	installFakeChrome(t, cache, "beta", "142.0.7200.5")

	// Stable resolution must not pick up the beta despite its higher version.
	dir, err := resolveVersionDir()
	if err != nil {
		t.Fatalf("stable resolveVersionDir() error = %v", err)
	}
	if got := filepath.Base(dir); got != "141.0.7100.20" {
		t.Errorf("stable resolveVersionDir() = %s, want 141.0.7100.20", got)
	}

	// Beta resolution sees only the beta subdirectory.
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "beta")
	dir, err = resolveVersionDir()
	if err != nil {
		t.Fatalf("beta resolveVersionDir() error = %v", err)
	}
	if got := filepath.Base(dir); got != "142.0.7200.5" {
		t.Errorf("beta resolveVersionDir() = %s, want 142.0.7200.5", got)
	}
	if got := filepath.Base(filepath.Dir(dir)); got != "beta" {
		t.Errorf("beta version dir parent = %s, want beta", got)
	}
}

func TestChromeChannelDefaultsToStable(t *testing.T) {
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "")
	if got := ChromeChannel(); got != "stable" {
		t.Errorf("ChromeChannel() = %q, want stable", got)
	}
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "beta")
	if got := ChromeChannel(); got != "beta" {
		t.Errorf("ChromeChannel() = %q, want beta", got)
	}
}
