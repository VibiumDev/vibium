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
	t.Setenv("GROK_HOME", "")
	// Both agent directories exist, so the auto default installs for both.
	for _, dir := range []string{".claude", ".grok"} {
		if err := os.MkdirAll(filepath.Join(homeDir, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
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
		grokInstalled, err := os.ReadFile(filepath.Join(home, ".grok", "skills", name, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(installed, grokInstalled) {
			t.Fatal("claude and grok installs differ")
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

func TestSkillInstallAgentGrokOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GROK_HOME", "")
	cmd := newSkillCmd()
	cmd.SetArgs([]string{"--agent", "grok"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "browser", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatal("claude skill written for grok-only install")
	}
	if _, err := os.ReadFile(filepath.Join(home, ".grok", "skills", "browser", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}

// A fresh machine gets only Claude, the pre-flag behavior; add-skill must
// not create ~/.grok for users who never ran Grok.
func TestSkillDefaultDoesNotCreateGrokOnFreshHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GROK_HOME", "")
	cmd := newSkillCmd()
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.ReadFile(filepath.Join(home, ".claude", "skills", "browser", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".grok")); !os.IsNotExist(err) {
		t.Fatal("default install created ~/.grok on a machine without Grok")
	}
}

func TestSkillDetectionCountsGROKHOMEAsGrok(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GROK_HOME", filepath.Join(home, "grok-root"))
	cmd := newSkillCmd()
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.ReadFile(filepath.Join(home, "grok-root", "skills", "browser", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}

func TestSkillAgentAllForcesBothOnFreshHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GROK_HOME", "")
	cmd := newSkillCmd()
	cmd.SetArgs([]string{"--agent", "all"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		filepath.Join(home, ".claude", "skills", "browser", "SKILL.md"),
		filepath.Join(home, ".grok", "skills", "browser", "SKILL.md"),
	} {
		if _, err := os.ReadFile(p); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGrokSkillDirHonorsGROKHOME(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GROK_HOME", root)
	dir, err := grokSkillDir("browser")
	if err != nil || dir != filepath.Join(root, "skills", "browser") {
		t.Fatalf("got %q %v", dir, err)
	}
}

func TestInstallSkillUnknownAgent(t *testing.T) {
	if err := installSkill("browser", "# test\n", "cursor"); err == nil {
		t.Fatal("expected unknown agent error")
	}
}
