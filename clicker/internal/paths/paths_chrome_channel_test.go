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

// Stable resolves the baked pin even when a newer version is cached. The
// newest-cached rule meant a pin bump never installed (any cached Chrome
// satisfied IsInstalled) and a pin rollback never launched (#579).
func TestResolveVersionDirStableHonorsBakedPin(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cache)
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "")
	t.Setenv("VIBIUM_ENGINE_VERSION", "")
	installFakeChrome(t, cache, PinnedChromeVersion)
	installFakeChrome(t, cache, "999.0.0.0")

	dir, err := resolveVersionDir("")
	if err != nil {
		t.Fatalf("resolveVersionDir() error = %v", err)
	}
	if got := filepath.Base(dir); got != PinnedChromeVersion {
		t.Errorf("stable resolveVersionDir() = %s, want the baked %s", got, PinnedChromeVersion)
	}
}

// A stable cache without the pinned version fails instead of launching
// whatever is newest; the ensure-install path then downloads the pin (#579).
func TestResolveVersionDirStableRequiresBakedPin(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cache)
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "")
	t.Setenv("VIBIUM_ENGINE_VERSION", "")
	installFakeChrome(t, cache, "150.0.7000.10")

	if _, err := resolveVersionDir(""); err == nil {
		t.Error("resolveVersionDir() without the pinned version cached: want error, got nil")
	}
}

func TestResolveVersionDirHonorsEnvPin(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cache)
	// Pin the channel: the ambient environment may carry
	// VIBIUM_ENGINE_CHANNEL=beta (the Beta Watch workflow does), and this
	// test seeds the default-channel layout (#479).
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "")
	installFakeChrome(t, cache, "140.0.7000.10")
	installFakeChrome(t, cache, PinnedChromeVersion)

	// The env pin beats both the baked pin and anything newer cached.
	t.Setenv("VIBIUM_ENGINE_VERSION", "140.0.7000.10")
	dir, err := resolveVersionDir("")
	if err != nil {
		t.Fatalf("pinned resolveVersionDir() error = %v", err)
	}
	if got := filepath.Base(dir); got != "140.0.7000.10" {
		t.Errorf("pinned resolveVersionDir() = %s, want 140.0.7000.10", got)
	}

	// A pin on a version that is not cached fails instead of silently
	// falling back to whatever is newest.
	t.Setenv("VIBIUM_ENGINE_VERSION", "139.0.6900.1")
	if _, err := resolveVersionDir(""); err == nil {
		t.Error("resolveVersionDir() with missing pinned version: want error, got nil")
	}
}

// Moving channels have no baked pin and keep resolving newest-cached.
func TestResolveVersionDirBetaPicksNewest(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cache)
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "beta")
	t.Setenv("VIBIUM_ENGINE_VERSION", "")
	installFakeChrome(t, cache, "beta", "141.0.7100.20")
	installFakeChrome(t, cache, "beta", "142.0.7200.5")

	dir, err := resolveVersionDir("")
	if err != nil {
		t.Fatalf("beta resolveVersionDir() error = %v", err)
	}
	if got := filepath.Base(dir); got != "142.0.7200.5" {
		t.Errorf("beta resolveVersionDir() = %s, want 142.0.7200.5", got)
	}
}

func TestChromeChannelDirsAreSeparate(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cache)
	// Pin the channel: the ambient environment may carry
	// VIBIUM_ENGINE_CHANNEL=beta (the Beta Watch workflow does), and this
	// test seeds the default-channel layout (#479).
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "")
	t.Setenv("VIBIUM_ENGINE_VERSION", "")
	installFakeChrome(t, cache, PinnedChromeVersion)
	// A beta with a higher version number than stable's pin.
	installFakeChrome(t, cache, "beta", "999.0.7200.5")

	// Stable resolution must not pick up the beta despite its higher version.
	dir, err := resolveVersionDir("")
	if err != nil {
		t.Fatalf("stable resolveVersionDir() error = %v", err)
	}
	if got := filepath.Base(dir); got != PinnedChromeVersion {
		t.Errorf("stable resolveVersionDir() = %s, want %s", got, PinnedChromeVersion)
	}

	// Beta resolution sees only the beta subdirectory.
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "beta")
	dir, err = resolveVersionDir("")
	if err != nil {
		t.Fatalf("beta resolveVersionDir() error = %v", err)
	}
	if got := filepath.Base(dir); got != "999.0.7200.5" {
		t.Errorf("beta resolveVersionDir() = %s, want 999.0.7200.5", got)
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

// A per-call channel must override the environment's channel: the daemon
// resolves browser_start's channel argument explicitly, and an env lookup
// underneath it would launch a different Chrome than it validated (#525).
func TestResolveVersionDirExplicitChannelBeatsEnv(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cache)
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "")
	t.Setenv("VIBIUM_ENGINE_VERSION", "")
	installFakeChrome(t, cache, PinnedChromeVersion)
	installFakeChrome(t, cache, "beta", "142.0.7200.5")

	dir, err := resolveVersionDir("beta")
	if err != nil {
		t.Fatalf(`resolveVersionDir("beta") error = %v`, err)
	}
	if got := filepath.Base(dir); got != "142.0.7200.5" {
		t.Errorf(`resolveVersionDir("beta") = %s, want 142.0.7200.5`, got)
	}

	// And the reverse: an explicit stable ignores an env beta.
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "beta")
	dir, err = resolveVersionDir("stable")
	if err != nil {
		t.Fatalf(`resolveVersionDir("stable") error = %v`, err)
	}
	if got := filepath.Base(dir); got != PinnedChromeVersion {
		t.Errorf(`resolveVersionDir("stable") = %s, want %s`, got, PinnedChromeVersion)
	}
}

func TestValidChannel(t *testing.T) {
	cases := []struct {
		engine, channel string
		want            bool
	}{
		{"chrome", "", true},
		{"chrome", "stable", true},
		{"chrome", "canary", true},
		{"chrome", "release", false},
		{"firefox", "release", true},
		{"firefox", "canary", false},
		{"firefox", "", true},
	}
	for _, c := range cases {
		if got := ValidChannel(c.engine, c.channel); got != c.want {
			t.Errorf("ValidChannel(%q, %q) = %v, want %v", c.engine, c.channel, got, c.want)
		}
	}
}
