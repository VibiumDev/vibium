package browser

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
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

// user.js must come out identical run to run: map iteration order previously
// made every launch write the prefs in a different order (#318).
func TestWriteFirefoxPrefsSortedAndDeterministic(t *testing.T) {
	dir := t.TempDir()
	if err := writeFirefoxPrefs(dir); err != nil {
		t.Fatalf("writeFirefoxPrefs() error = %v", err)
	}
	first, err := os.ReadFile(filepath.Join(dir, "user.js"))
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimRight(string(first), "\n"), "\n")
	if len(lines) != len(firefoxPrefs) {
		t.Fatalf("user.js has %d lines, want %d", len(lines), len(firefoxPrefs))
	}
	if !sort.StringsAreSorted(lines) {
		t.Errorf("user.js lines are not sorted:\n%s", string(first))
	}

	dir2 := t.TempDir()
	if err := writeFirefoxPrefs(dir2); err != nil {
		t.Fatalf("writeFirefoxPrefs() error = %v", err)
	}
	second, err := os.ReadFile(filepath.Join(dir2, "user.js"))
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Error("two writes produced different user.js contents")
	}
}
