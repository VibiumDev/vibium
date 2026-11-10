package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDefaultSessionPaths(t *testing.T) {
	t.Setenv("VIBIUM_SESSION", "")

	sock, err := GetSocketPath()
	if err != nil {
		t.Fatalf("GetSocketPath: %v", err)
	}
	if runtime.GOOS == "windows" {
		if sock != `\\.\pipe\vibium` {
			t.Errorf("socket = %q, want \\\\.\\pipe\\vibium", sock)
		}
	} else if filepath.Base(sock) != "vibium.sock" {
		t.Errorf("socket basename = %q, want vibium.sock", filepath.Base(sock))
	}

	pid, err := GetPIDPath()
	if err != nil {
		t.Fatalf("GetPIDPath: %v", err)
	}
	if filepath.Base(pid) != "vibium.pid" {
		t.Errorf("pid basename = %q, want vibium.pid", filepath.Base(pid))
	}
}

func TestNamedSessionPaths(t *testing.T) {
	t.Setenv("VIBIUM_SESSION", "projA")

	sock, err := GetSocketPath()
	if err != nil {
		t.Fatalf("GetSocketPath: %v", err)
	}
	if runtime.GOOS == "windows" {
		if sock != `\\.\pipe\vibium-projA` {
			t.Errorf("socket = %q, want \\\\.\\pipe\\vibium-projA", sock)
		}
	} else if filepath.Base(sock) != "vibium-projA.sock" {
		t.Errorf("socket basename = %q, want vibium-projA.sock", filepath.Base(sock))
	}

	pid, err := GetPIDPath()
	if err != nil {
		t.Fatalf("GetPIDPath: %v", err)
	}
	if filepath.Base(pid) != "vibium-projA.pid" {
		t.Errorf("pid basename = %q, want vibium-projA.pid", filepath.Base(pid))
	}
}

func TestInvalidSessionNameRejected(t *testing.T) {
	t.Setenv("VIBIUM_SESSION", "bad/name")

	if _, err := GetSocketPath(); err == nil {
		t.Error("GetSocketPath: want error for session name with slash")
	}
	if _, err := GetPIDPath(); err == nil {
		t.Error("GetPIDPath: want error for session name with slash")
	}
}

func TestValidateSessionName(t *testing.T) {
	valid := []string{"", "a", "projA", "proj-a_1", strings.Repeat("x", 64)}
	for _, name := range valid {
		if err := ValidateSessionName(name); err != nil {
			t.Errorf("ValidateSessionName(%q) = %v, want nil", name, err)
		}
	}

	invalid := []string{"bad/name", "a b", "a.b", `a\b`, strings.Repeat("x", 65)}
	for _, name := range invalid {
		if err := ValidateSessionName(name); err == nil {
			t.Errorf("ValidateSessionName(%q) = nil, want error", name)
		}
	}
}

func TestSocketPathLengthLimit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("named pipes have no socket path length limit")
	}

	// A long runtime dir + long session exceeds the sockaddr_un limit.
	t.Setenv("VIBIUM_RUNTIME_DIR", t.TempDir()+"/"+strings.Repeat("d", 80))
	t.Setenv("VIBIUM_SESSION", strings.Repeat("s", 40))
	if _, err := GetSocketPath(); err == nil {
		t.Error("GetSocketPath: want error for socket path exceeding sockaddr_un limit")
	}

	// The same session fits under a short runtime dir.
	short := shortRuntimeDir(t)
	t.Setenv("VIBIUM_RUNTIME_DIR", short)
	if _, err := GetSocketPath(); err != nil {
		t.Errorf("GetSocketPath with short runtime dir: %v", err)
	}
}

// shortRuntimeDir makes a dir whose path stays well under sun_path even
// with a long session suffix; t.TempDir can be too long on macOS.
func shortRuntimeDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "vib")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// The socket must not live under the cache dir: long cache paths (CI
// workspaces, per-test caches) used to stop every command (#727).
func TestSocketPathIgnoresCacheDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("named pipes have no socket path length limit")
	}

	t.Setenv("VIBIUM_SESSION", "")
	t.Setenv("VIBIUM_RUNTIME_DIR", "")
	t.Setenv("XDG_RUNTIME_DIR", "")
	cache := "/tmp/" + strings.Repeat("c", 150)
	t.Setenv("VIBIUM_CACHE_DIR", cache)

	sock, err := GetSocketPath()
	if err != nil {
		t.Fatalf("GetSocketPath with a long cache dir: %v", err)
	}
	if strings.HasPrefix(sock, cache) {
		t.Errorf("socket %q lives under VIBIUM_CACHE_DIR", sock)
	}
}

func TestDaemonDirPrecedence(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the daemon dir on Windows is the cache dir")
	}

	own := shortRuntimeDir(t)
	xdg := shortRuntimeDir(t)

	t.Setenv("VIBIUM_RUNTIME_DIR", own)
	t.Setenv("XDG_RUNTIME_DIR", xdg)
	dir, err := GetDaemonDir()
	if err != nil {
		t.Fatalf("GetDaemonDir: %v", err)
	}
	if dir != own {
		t.Errorf("dir = %q, want VIBIUM_RUNTIME_DIR %q", dir, own)
	}

	t.Setenv("VIBIUM_RUNTIME_DIR", "")
	dir, err = GetDaemonDir()
	if err != nil {
		t.Fatalf("GetDaemonDir: %v", err)
	}
	if want := filepath.Join(xdg, "vibium"); dir != want {
		t.Errorf("dir = %q, want %q", dir, want)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Errorf("runtime dir mode = %#o, want 0700", info.Mode().Perm())
	}
}

func TestRuntimeDirRefusesOpenPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no unix permission bits")
	}

	dir := shortRuntimeDir(t)
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VIBIUM_RUNTIME_DIR", dir)
	if _, err := GetDaemonDir(); err == nil {
		t.Error("GetDaemonDir: want error for a group/world accessible runtime dir")
	}
}

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
	// Windows resolves the cache from LOCALAPPDATA (USERPROFILE as the
	// fallback) and never reads HOME or XDG_CACHE_HOME, so without these the
	// tests seed the machine's real cache and resolve against whatever is
	// actually installed there (#658).
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))
	t.Setenv("USERPROFILE", home)
	// These tests seed the default-channel layout, so the ambient channel
	// must not redirect resolution to a channel subdirectory. The Beta
	// Watch workflow exports VIBIUM_ENGINE_CHANNEL=beta job-wide, which
	// failed every one of them regardless of the browser (#479).
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "")
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
// Runs on the beta channel: stable now resolves its baked pin exactly
// (#579), so newest-complete selection only applies to moving channels.
func TestResolvesChromeAndDriverAsAPair(t *testing.T) {
	cft := withCache(t)
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "beta")
	cft = filepath.Join(cft, "beta")
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
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "beta")
	cft = filepath.Join(cft, "beta")
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
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "beta")
	cft = filepath.Join(cft, "beta")
	seedVersion(t, cft, "146.0.7000.10", true, false)

	if _, err := GetChromeExecutable(); err == nil {
		t.Error("a cache with no complete pair should not resolve")
	}
	if _, err := GetChromedriverPath(); err == nil {
		t.Error("a cache with no complete pair should not resolve")
	}
}
