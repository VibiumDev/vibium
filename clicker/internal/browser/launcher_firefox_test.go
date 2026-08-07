package browser

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPrepareFirefoxXDGUsesProfile(t *testing.T) {
	profile := t.TempDir()
	config, err := prepareFirefoxXDG(profile)
	if err != nil {
		t.Fatalf("prepareFirefoxXDG() error = %v", err)
	}
	if config != filepath.Join(profile, "xdg-config") {
		t.Fatalf("config = %q", config)
	}
	if info, err := os.Stat(filepath.Join(profile, "Downloads")); err != nil || !info.IsDir() {
		t.Fatalf("private Downloads directory not created: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(config, "user-dirs.dirs"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), filepath.Join(profile, "Downloads")) {
		t.Fatalf("user-dirs.dirs = %q", data)
	}
}

func TestReplaceEnv(t *testing.T) {
	got := replaceEnv([]string{"A=1", "XDG_CONFIG_HOME=old", "B=2"}, "XDG_CONFIG_HOME", "new")
	want := []string{"A=1", "B=2", "XDG_CONFIG_HOME=new"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("replaceEnv() = %v, want %v", got, want)
	}
}
