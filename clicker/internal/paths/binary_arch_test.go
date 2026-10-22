package paths

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// fakeELF returns a minimal ELF64 header carrying the given e_machine.
func fakeELF(machine uint16) []byte {
	h := make([]byte, 64)
	copy(h, "\x7fELF")
	h[4] = 2                                 // ELFCLASS64
	h[5] = 1                                 // ELFDATA2LSB
	h[6] = 1                                 // EV_CURRENT
	binary.LittleEndian.PutUint16(h[16:], 3) // ET_DYN
	binary.LittleEndian.PutUint16(h[18:], machine)
	return h
}

func writeTemp(t *testing.T, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "bin")
	if err := os.WriteFile(path, content, 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestElfMachineMatches(t *testing.T) {
	x86 := writeTemp(t, fakeELF(0x3e))
	arm := writeTemp(t, fakeELF(0xb7))
	text := writeTemp(t, []byte("#!/bin/sh\n"))
	short := writeTemp(t, []byte("\x7fEL"))

	cases := []struct {
		name   string
		path   string
		goarch string
		want   bool
	}{
		{"x86 binary on amd64", x86, "amd64", true},
		{"x86 binary on arm64", arm, "amd64", false},
		{"arm binary on arm64", arm, "arm64", true},
		{"arm binary on amd64", x86, "arm64", false},
		// Only a positively identified mismatch rejects: scripts, truncated
		// files, unmapped architectures, and missing files are all trusted.
		{"non-ELF content", text, "arm64", true},
		{"truncated header", short, "arm64", true},
		{"unmapped goarch", x86, "riscv64", true},
		{"missing file", filepath.Join(t.TempDir(), "absent"), "arm64", true},
	}
	for _, c := range cases {
		if got := elfMachineMatches(c.path, c.goarch); got != c.want {
			t.Errorf("%s: elfMachineMatches(%s, %s) = %v, want %v", c.name, c.path, c.goarch, got, c.want)
		}
	}
}

// installWrongArchChrome fakes a cached version whose binaries are valid ELF
// for the other architecture, the state the released npm package's
// postinstall left on arm64 CI (#699).
func installWrongArchChrome(t *testing.T, cacheDir string, segments ...string) {
	t.Helper()
	other := uint16(0x3e) // EM_X86_64
	if runtime.GOARCH == "amd64" {
		other = 0xb7 // EM_AARCH64
	}
	dir := filepath.Join(append([]string{cacheDir, "chrome-for-testing"}, segments...)...)
	for _, p := range []string{getChromePathInVersion(dir), getChromedriverPathInVersion(dir)} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, fakeELF(other), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

// A pinned version directory holding wrong-architecture binaries is not an
// install: resolving it must fail so ensure-install downloads the right
// build, instead of certifying it and dying at launch with an exec format
// error (#699).
func TestResolveVersionDirRejectsWrongArchPin(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the architecture check only applies to Linux ELF binaries")
	}
	cache := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cache)
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "")
	t.Setenv("VIBIUM_ENGINE_VERSION", "")
	installWrongArchChrome(t, cache, PinnedChromeVersion)

	if _, err := resolveVersionDir(""); err == nil {
		t.Error("resolveVersionDir() with a wrong-arch pin cached: want error, got nil")
	}
}

// Newest-cached selection (non-stable channels) skips wrong-architecture
// directories rather than resolving them.
func TestResolveVersionDirNewestSkipsWrongArch(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the architecture check only applies to Linux ELF binaries")
	}
	cache := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cache)
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "beta")
	t.Setenv("VIBIUM_ENGINE_VERSION", "")
	installFakeChrome(t, cache, "beta", "100.0.0.1")
	installWrongArchChrome(t, cache, "beta", "999.0.0.0")

	dir, err := resolveVersionDir("beta")
	if err != nil {
		t.Fatalf("resolveVersionDir(beta) error = %v", err)
	}
	if got := filepath.Base(dir); got != "100.0.0.1" {
		t.Errorf("resolveVersionDir(beta) = %s, want the right-arch 100.0.0.1", got)
	}
}
