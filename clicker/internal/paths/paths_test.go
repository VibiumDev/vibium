package paths

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedVersion creates a version directory holding the requested binaries.
func seedVersion(t *testing.T, cftDir, version string, chrome, driver bool) {
	t.Helper()
	dir := filepath.Join(cftDir, version)
	if chrome {
		p := getChromePathInVersion(dir)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("chrome"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if driver {
		p := getChromedriverPathInVersion(dir)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("driver"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

// withCache points the cache at a temp dir for the duration of a test.
func withCache(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	dir, err := GetChromeForTestingDir()
	if err != nil {
		t.Fatalf("GetChromeForTestingDir: %v", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// A cache holding Chrome under one version and chromedriver under another used
// to resolve to a mismatched pair, which IsInstalled() then certified (#265).
func TestResolvesChromeAndDriverAsAPair(t *testing.T) {
	cft := withCache(t)
	seedVersion(t, cft, "146.0.7000.10", true, false) // chrome only
	seedVersion(t, cft, "147.0.8000.20", false, true) // driver only
	seedVersion(t, cft, "145.0.6000.30", true, true)  // the only complete pair

	chrome, err := GetChromeExecutable()
	if err != nil {
		t.Fatalf("GetChromeExecutable: %v", err)
	}
	driver, err := GetChromedriverPath()
	if err != nil {
		t.Fatalf("GetChromedriverPath: %v", err)
	}

	if !strings.Contains(chrome, "145.0.6000.30") {
		t.Errorf("chrome resolved to %q, want the complete 145 pair", chrome)
	}
	if !strings.Contains(driver, "145.0.6000.30") {
		t.Errorf("driver resolved to %q, want the complete 145 pair", driver)
	}
}

func TestPrefersNewestCompleteVersion(t *testing.T) {
	cft := withCache(t)
	seedVersion(t, cft, "99.0.1000.1", true, true)
	seedVersion(t, cft, "100.0.1000.1", true, true)

	chrome, err := GetChromeExecutable()
	if err != nil {
		t.Fatalf("GetChromeExecutable: %v", err)
	}
	// Lexically "99.0..." sorts after "100.0...", so this also pins that the
	// comparison is numeric rather than os.ReadDir's string order.
	if !strings.Contains(chrome, "100.0.1000.1") {
		t.Errorf("chrome resolved to %q, want the newest complete version", chrome)
	}
}

func TestNoCompleteVersionIsNotFound(t *testing.T) {
	cft := withCache(t)
	seedVersion(t, cft, "146.0.7000.10", true, false)

	if _, err := GetChromeExecutable(); err == nil {
		t.Error("a cache with no complete pair should not resolve")
	}
	if _, err := GetChromedriverPath(); err == nil {
		t.Error("a cache with no complete pair should not resolve")
	}
}
