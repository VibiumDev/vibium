package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSkillInstallation(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"browser", nil},
		{"browser", []string{"browser"}},
		{"check", []string{"check"}},
	} {
		name := tc.name
		cmd := newSkillCmd()
		// Empty arguments must install the same browser skill as the named form.
		cmd.SetArgs(append([]string{}, tc.args...))
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		home, _ := os.UserHomeDir()
		installed, err := os.ReadFile(filepath.Join(home, ".claude", "skills", name, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		var printed bytes.Buffer
		cmd = newSkillCmd()
		cmd.SetOut(&printed)
		cmd.SetArgs([]string{name, "--stdout"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(installed, printed.Bytes()) {
			t.Fatal("installed skill differs from exported skill")
		}
		source, err := os.ReadFile(filepath.Join("..", "..", "..", "skills", name, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(installed, source) {
			t.Fatal("embedded skill is stale; run make build-go")
		}
	}
	cmd := newSkillCmd()
	cmd.SetArgs([]string{"../escape"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("accepted invalid skill name")
	}
}

func TestDefaultSkillIsBrowser(t *testing.T) {
	var out bytes.Buffer
	cmd := newSkillCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--stdout"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.String() != skillMD {
		t.Fatal("default skill changed")
	}
}
