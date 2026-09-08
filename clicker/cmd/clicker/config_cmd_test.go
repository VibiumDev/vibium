package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// The templates carry API keys the moment a user fills them in, so the mode
// is load-bearing: a world-readable copy is how a key reaches anything else
// running on the machine. WriteFile respects the caller's umask, so this also
// covers a permissive umask widening the file behind us.
func TestConfigInitWritesOwnerOnly(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VIBIUM_CONFIG_DIR", filepath.Join(dir, "vibium"))

	old := syscallUmask(0)
	defer syscallUmask(old)

	cmd := &cobra.Command{}
	for _, tmpl := range configTemplates {
		if err := writeConfigTemplate(cmd, tmpl, false); err != nil {
			t.Fatalf("%s: %v", tmpl.name, err)
		}
		path := filepath.Join(dir, "vibium", tmpl.file)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("%s: %v", tmpl.name, err)
		}
		if got := info.Mode().Perm(); got != 0600 {
			t.Errorf("%s mode = %o, want 600", tmpl.file, got)
		}
	}
}

// A bare init must never overwrite real credentials.
func TestConfigInitRefusesToClobber(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VIBIUM_CONFIG_DIR", dir)
	path := filepath.Join(dir, configTemplates[0].file)
	if err := os.WriteFile(path, []byte("export OPENAI_API_KEY='real'\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	err := writeConfigTemplate(cmd, configTemplates[0], false)
	if err == nil {
		t.Fatal("expected a refusal, got nil")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error should name the escape hatch, got: %v", err)
	}
	body, _ := os.ReadFile(path)
	if !strings.Contains(string(body), "real") {
		t.Error("the existing file was modified")
	}
}

// --force replaces the file but must not destroy what was there.
func TestConfigInitForceKeepsBackup(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VIBIUM_CONFIG_DIR", dir)
	path := filepath.Join(dir, configTemplates[0].file)
	if err := os.WriteFile(path, []byte("export OPENAI_API_KEY='real'\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	if err := writeConfigTemplate(cmd, configTemplates[0], true); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatalf("no backup: %v", err)
	}
	if !strings.Contains(string(body), "real") {
		t.Error("backup lost the previous contents")
	}
}

// The shipped templates must stay sourceable: every setting line an `export`,
// everything else a comment. A stray bare line makes `source` fail.
func TestTemplatesAreSourceable(t *testing.T) {
	for _, tmpl := range configTemplates {
		for i, line := range strings.Split(*tmpl.content, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "export ") {
				continue
			}
			t.Errorf("%s line %d is neither comment nor export: %q", tmpl.file, i+1, line)
		}
	}
}
