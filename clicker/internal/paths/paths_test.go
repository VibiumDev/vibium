package paths

import (
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
