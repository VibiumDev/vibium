package paths

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func foreignArch() string {
	if runtime.GOARCH == "arm64" {
		return "amd64"
	}
	return "arm64"
}

// buildTinyBinary cross-compiles an empty program for goarch (host GOOS) to
// outPath, giving a real executable of a known architecture to match against.
func buildTinyBinary(t *testing.T, goarch, outPath string) {
	t.Helper()
	src := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(src, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", "-o", outPath, src)
	cmd.Env = append(os.Environ(), "GOOS="+runtime.GOOS, "GOARCH="+goarch, "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build GOARCH=%s: %v\n%s", goarch, err, out)
	}
}

func TestBinaryArchMatches(t *testing.T) {
	dir := t.TempDir()

	host := filepath.Join(dir, "host")
	buildTinyBinary(t, runtime.GOARCH, host)
	if !binaryArchMatches(host) {
		t.Errorf("host-arch (%s) binary should match", runtime.GOARCH)
	}

	foreign := filepath.Join(dir, "foreign")
	buildTinyBinary(t, foreignArch(), foreign)
	if binaryArchMatches(foreign) {
		t.Errorf("foreign-arch (%s) binary should not match on %s", foreignArch(), runtime.GOARCH)
	}

	// An unparseable file is treated as matching: the stat checks gate
	// existence, and a parser quirk must not discard a valid install.
	junk := filepath.Join(dir, "junk")
	if err := os.WriteFile(junk, []byte("not an executable"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !binaryArchMatches(junk) {
		t.Error("unparseable file should be treated as matching")
	}
}

// A cached pin whose binaries are the wrong architecture must be rejected
// and deleted so ensure-install downloads the right build instead of
// launching into "exec format error" (#699).
func TestResolveVersionDirRejectsWrongArch(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cache)
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "")
	t.Setenv("VIBIUM_ENGINE_VERSION", "150.0.7000.10")

	dir := filepath.Join(cache, "chrome-for-testing", "150.0.7000.10")
	buildTinyBinary(t, foreignArch(), getChromePathInVersion(dir))
	buildTinyBinary(t, foreignArch(), getChromedriverPathInVersion(dir))

	if _, err := resolveVersionDir(""); err == nil {
		t.Error("resolveVersionDir() with a wrong-arch cached pin: want error, got nil")
	}

	// The rejected directory is deleted so the reinstall extracts into a
	// clean directory instead of over the old files.
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("wrong-arch version dir should be deleted, Stat() err = %v", err)
	}

	// Rebuilt for the host arch, the same pin resolves.
	buildTinyBinary(t, runtime.GOARCH, getChromePathInVersion(dir))
	buildTinyBinary(t, runtime.GOARCH, getChromedriverPathInVersion(dir))
	if _, err := resolveVersionDir(""); err != nil {
		t.Errorf("resolveVersionDir() with a host-arch cached pin: %v", err)
	}
}

// Newest-cached selection (moving channels) must skip a wrong-architecture
// directory rather than resolve it, even when it carries the highest version.
func TestResolveVersionDirNewestSkipsWrongArch(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cache)
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "beta")
	t.Setenv("VIBIUM_ENGINE_VERSION", "")

	good := filepath.Join(cache, "chrome-for-testing", "beta", "100.0.0.1")
	buildTinyBinary(t, runtime.GOARCH, getChromePathInVersion(good))
	buildTinyBinary(t, runtime.GOARCH, getChromedriverPathInVersion(good))
	bad := filepath.Join(cache, "chrome-for-testing", "beta", "999.0.0.0")
	buildTinyBinary(t, foreignArch(), getChromePathInVersion(bad))
	buildTinyBinary(t, foreignArch(), getChromedriverPathInVersion(bad))

	dir, err := resolveVersionDir("beta")
	if err != nil {
		t.Fatalf("resolveVersionDir(beta) error = %v", err)
	}
	if got := filepath.Base(dir); got != "100.0.0.1" {
		t.Errorf("resolveVersionDir(beta) = %s, want the host-arch 100.0.0.1", got)
	}
	if _, err := os.Stat(bad); !os.IsNotExist(err) {
		t.Errorf("wrong-arch version dir should be deleted, Stat() err = %v", err)
	}
}

// A cached Firefox pin whose binary is the wrong architecture must be
// rejected and deleted so ensure-install downloads the right build instead of
// launching into "exec format error" (#704, the Firefox side of #699).
func TestResolveFirefoxVersionDirRejectsWrongArch(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cache)
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "")
	t.Setenv("VIBIUM_ENGINE_VERSION", "140.0.1")

	dir := filepath.Join(cache, "firefox", "release", "140.0.1")
	buildTinyBinary(t, foreignArch(), FirefoxPathInVersion(dir))

	if _, err := resolveFirefoxVersionDir(""); err == nil {
		t.Error("resolveFirefoxVersionDir() with a wrong-arch cached pin: want error, got nil")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("wrong-arch version dir should be deleted, Stat() err = %v", err)
	}

	// Rebuilt for the host arch, the same pin resolves.
	buildTinyBinary(t, runtime.GOARCH, FirefoxPathInVersion(dir))
	if _, err := resolveFirefoxVersionDir(""); err != nil {
		t.Errorf("resolveFirefoxVersionDir() with a host-arch cached pin: %v", err)
	}
}

// Newest-cached selection (the beta channel) must skip a wrong-architecture
// Firefox rather than resolve it, even when it carries the highest version.
func TestResolveFirefoxVersionDirNewestSkipsWrongArch(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cache)
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "beta")
	t.Setenv("VIBIUM_ENGINE_VERSION", "")

	good := filepath.Join(cache, "firefox", "beta", "100.0b1")
	buildTinyBinary(t, runtime.GOARCH, FirefoxPathInVersion(good))
	bad := filepath.Join(cache, "firefox", "beta", "999.0b1")
	buildTinyBinary(t, foreignArch(), FirefoxPathInVersion(bad))

	dir, err := resolveFirefoxVersionDir("beta")
	if err != nil {
		t.Fatalf("resolveFirefoxVersionDir(beta) error = %v", err)
	}
	if got := filepath.Base(dir); got != "100.0b1" {
		t.Errorf("resolveFirefoxVersionDir(beta) = %s, want the host-arch 100.0b1", got)
	}
	if _, err := os.Stat(bad); !os.IsNotExist(err) {
		t.Errorf("wrong-arch version dir should be deleted, Stat() err = %v", err)
	}
}
