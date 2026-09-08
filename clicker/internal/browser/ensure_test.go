package browser

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vibium/clicker/internal/paths"
)

func TestSkipBrowserDownload(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"", false},
		{"0", false},
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"yes", false},
	}
	for _, tt := range tests {
		t.Setenv("VIBIUM_SKIP_BROWSER_DOWNLOAD", tt.value)
		if got := SkipBrowserDownload(); got != tt.want {
			t.Errorf("SkipBrowserDownload() with %q = %v, want %v", tt.value, got, tt.want)
		}
	}
}

func TestProgressWriterKnownTotal(t *testing.T) {
	var dst, out bytes.Buffer
	pw := &progressWriter{dst: &dst, total: 100 << 20, out: &out}

	chunk := make([]byte, 5<<20)
	for i := 0; i < 20; i++ {
		if _, err := pw.Write(chunk); err != nil {
			t.Fatal(err)
		}
	}

	if dst.Len() != 100<<20 {
		t.Errorf("dst got %d bytes, want %d", dst.Len(), 100<<20)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	// One line per 10% step: 10%..100%.
	if len(lines) != 10 {
		t.Fatalf("got %d progress lines, want 10:\n%s", len(lines), out.String())
	}
	if want := "  10% of 100.0 MB"; lines[0] != want {
		t.Errorf("first line = %q, want %q", lines[0], want)
	}
	if want := "  100% of 100.0 MB"; lines[9] != want {
		t.Errorf("last line = %q, want %q", lines[9], want)
	}
}

func TestProgressWriterSmallChunksPrintEachStepOnce(t *testing.T) {
	var dst, out bytes.Buffer
	pw := &progressWriter{dst: &dst, total: 1000, out: &out}

	chunk := make([]byte, 1)
	for i := 0; i < 1000; i++ {
		if _, err := pw.Write(chunk); err != nil {
			t.Fatal(err)
		}
	}

	if got := strings.Count(out.String(), "\n"); got != 10 {
		t.Errorf("got %d progress lines, want 10:\n%s", got, out.String())
	}
}

func TestProgressWriterUnknownTotal(t *testing.T) {
	var dst, out bytes.Buffer
	// -1 mirrors resp.ContentLength for a chunked response.
	pw := &progressWriter{dst: &dst, total: -1, out: &out}

	chunk := make([]byte, 10<<20)
	for i := 0; i < 6; i++ { // 60 MB total
		if _, err := pw.Write(chunk); err != nil {
			t.Fatal(err)
		}
	}

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d progress lines, want 2 (25 MB and 50 MB):\n%s", len(lines), out.String())
	}
	if want := "  25 MB downloaded"; lines[0] != want {
		t.Errorf("first line = %q, want %q", lines[0], want)
	}
	if want := "  50 MB downloaded"; lines[1] != want {
		t.Errorf("second line = %q, want %q", lines[1], want)
	}
}

// A per-call channel must decide the install on its own: the daemon resolves
// --channel per request, so consulting VIBIUM_ENGINE_CHANNEL here would report
// beta installed because release is, then fail the launch.
func TestEngineInstalledForChannelIgnoresEnvironmentChannel(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("VIBIUM_CACHE_DIR", cache)
	t.Setenv("VIBIUM_ENGINE_CHANNEL", "release")
	t.Setenv("VIBIUM_ENGINE_PATH", "")
	t.Setenv("VIBIUM_ENGINE_VERSION", "")

	exe := paths.FirefoxPathInVersion(filepath.Join(cache, "firefox", "release", "155.0.1"))
	if err := os.MkdirAll(filepath.Dir(exe), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}

	if !EngineInstalledForChannel("firefox", "release") {
		t.Error("release channel should be installed")
	}
	if EngineInstalledForChannel("firefox", "beta") {
		t.Error("beta channel should not be installed")
	}
	if !EngineInstalledForChannel("firefox", "") {
		t.Error("empty channel should fall back to VIBIUM_ENGINE_CHANNEL (release)")
	}
}

// The skip switch must still produce the old hard launch error rather than a
// silent download, whatever channel is asked for.
func TestEnsureInstalledForChannelHonorsSkip(t *testing.T) {
	t.Setenv("VIBIUM_CACHE_DIR", t.TempDir())
	t.Setenv("VIBIUM_SKIP_BROWSER_DOWNLOAD", "1")
	if err := EnsureInstalledForChannel("firefox", "beta"); err != nil {
		t.Errorf("EnsureInstalledForChannel with skip set = %v, want nil (no-op)", err)
	}
}
