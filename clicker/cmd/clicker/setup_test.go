package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func setupTestEnv(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("VIBIUM_CACHE_DIR", filepath.Join(home, "cache"))
	t.Setenv("VIBIUM_CONFIG_DIR", "")
	for _, key := range []string{"VIBIUM_AI_PROVIDER", "VIBIUM_AI_MODEL", "VIBIUM_AI_BASE_URL", "VIBIUM_AI_REASONING_EFFORT", "OPENAI_API_KEY", "ANTHROPIC_API_KEY", "GOOGLE_API_KEY", "GROK_HOME", "VIBIUM_ENGINE", "VIBIUM_ENGINE_CHANNEL", "VIBIUM_ENGINE_PATH", "VIBIUM_SKIP_BROWSER_DOWNLOAD"} {
		t.Setenv(key, "")
	}
	oldJSON, oldEngine, oldChannel := jsonOutput, engineName, engineChannel
	t.Cleanup(func() { jsonOutput, engineName, engineChannel = oldJSON, oldEngine, oldChannel })
	jsonOutput, engineName, engineChannel = true, "chrome", ""
	return home
}

func seedChrome(t *testing.T, home string) {
	t.Helper()
	cache := filepath.Join(home, "cache")
	version := filepath.Join(cache, "chrome-for-testing", "999.0.0.0")
	chrome := filepath.Join(version, "chrome")
	if runtime.GOOS == "darwin" {
		chrome = filepath.Join(version, "Google Chrome for Testing.app", "Contents", "MacOS", "Google Chrome for Testing")
	} else if runtime.GOOS == "windows" {
		chrome = filepath.Join(version, "chrome.exe")
	}
	if err := os.MkdirAll(filepath.Dir(chrome), 0o755); err != nil {
		t.Fatal(err)
	}
	driver := filepath.Join(version, "chromedriver")
	if runtime.GOOS == "windows" {
		driver = filepath.Join(version, "chromedriver.exe")
	}
	for _, p := range []string{chrome, driver} {
		if err := os.WriteFile(p, []byte("fixture"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
}

func quietUI() *setupUI {
	return &setupUI{in: &bytes.Buffer{}, out: ioDiscard(), err: ioDiscard(), interactive: false}
}

func ioDiscard() *bytes.Buffer { return &bytes.Buffer{} }

func TestSetupAINonInteractiveSkipsWhenMissingAndDoesNotOverwrite(t *testing.T) {
	home := setupTestEnv(t)
	cmd := &cobra.Command{}
	sec := setupAI(cmd, quietUI(), false)
	if sec.Status != "skipped" {
		t.Fatalf("missing file: %+v", sec)
	}
	path := filepath.Join(home, ".config", "vibium", "ai.env")
	if _, err := os.Stat(path); err == nil {
		t.Fatal("non-interactive setup wrote ai.env")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	original := "export VIBIUM_AI_PROVIDER=openai\nexport VIBIUM_AI_MODEL=keep-me\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	sec = setupAI(cmd, quietUI(), false)
	if sec.Status != "skipped" {
		t.Fatalf("existing file: %+v", sec)
	}
	got, _ := os.ReadFile(path)
	if string(got) != original {
		t.Fatalf("ai.env was rewritten: %q", got)
	}
	if _, err := os.Stat(path + ".bak"); err == nil {
		t.Fatal("unexpected backup")
	}
}

func TestSetupSkillsOnlyWhenAgentDirExists(t *testing.T) {
	home := setupTestEnv(t)
	cmd := &cobra.Command{}
	sec := setupSkills(cmd, quietUI(), false)
	if sec.Status != "skipped" {
		t.Fatalf("%+v", sec)
	}
	if _, err := os.Stat(filepath.Join(home, ".grok", "skills", "browser", "SKILL.md")); err == nil {
		t.Fatal("installed grok skills without agent dir")
	}

	if err := os.MkdirAll(filepath.Join(home, ".grok"), 0o755); err != nil {
		t.Fatal(err)
	}
	sec = setupSkills(cmd, quietUI(), false)
	if sec.Status != "done" || sec.Agent != "grok" {
		t.Fatalf("%+v", sec)
	}
	if _, err := os.Stat(filepath.Join(home, ".grok", "skills", "browser", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".grok", "skills", "check", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}

// The wizard shares detection with add-skill: a machine with both agent
// directories gets skills for both, not just the first match.
func TestSetupSkillsInstallsForAllDetectedAgents(t *testing.T) {
	home := setupTestEnv(t)
	for _, dir := range []string{".grok", ".claude"} {
		if err := os.MkdirAll(filepath.Join(home, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	sec := setupSkills(&cobra.Command{}, quietUI(), false)
	if sec.Status != "done" || len(sec.Agents) != 2 {
		t.Fatalf("%+v", sec)
	}
	for _, p := range []string{
		filepath.Join(home, ".claude", "skills", "browser", "SKILL.md"),
		filepath.Join(home, ".grok", "skills", "check", "SKILL.md"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSetupQuickSkipsInstalledBrowser(t *testing.T) {
	home := setupTestEnv(t)
	seedChrome(t, home)
	sec := setupBrowser(&cobra.Command{}, quietUI(), true)
	if sec.Status != "skipped" {
		t.Fatalf("%+v", sec)
	}
}

func TestSetupBrowserReportsPresent(t *testing.T) {
	home := setupTestEnv(t)
	seedChrome(t, home)
	sec := setupBrowser(&cobra.Command{}, quietUI(), false)
	if sec.Status != "done" || !strings.Contains(sec.Message, "already installed") {
		t.Fatalf("%+v", sec)
	}
}

func TestWriteAIEnvModeAndBackup(t *testing.T) {
	home := setupTestEnv(t)
	path := filepath.Join(home, ".config", "vibium", "ai.env")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("export OPENAI_API_KEY='old'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeAIEnv(path, map[string]string{"VIBIUM_AI_PROVIDER": "openai", "VIBIUM_AI_MODEL": "m", "OPENAI_API_KEY": "new"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
	bak, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bak), "old") {
		t.Fatal("backup lost previous contents")
	}
	body, _ := os.ReadFile(path)
	if strings.Contains(string(body), "old") || !strings.Contains(string(body), "export VIBIUM_AI_PROVIDER='openai'") {
		t.Fatalf("written file: %s", body)
	}
}

func TestSetupTryHintOnlyWhenFullyReady(t *testing.T) {
	setupTestEnv(t)
	if setupTryHint(nil) != "" {
		t.Fatal("hint without readiness")
	}
	if setupTryHint(&setupResult{Ready: true}) != "" {
		t.Fatal("hint without AI configuration")
	}
	t.Setenv("VIBIUM_AI_PROVIDER", "openai")
	t.Setenv("VIBIUM_AI_MODEL", "gpt-test")
	t.Setenv("OPENAI_API_KEY", "k")
	if setupTryHint(&setupResult{Ready: false}) != "" {
		t.Fatal("hint on a not-ready run")
	}
	if hint := setupTryHint(&setupResult{Ready: true}); !strings.Contains(hint, "vibium run") {
		t.Fatalf("hint: %q", hint)
	}
}

func TestSetupAIModelDefaultsPerProvider(t *testing.T) {
	home := setupTestEnv(t)
	oldJSON := jsonOutput
	jsonOutput = false
	t.Cleanup(func() { jsonOutput = oldJSON })
	path := filepath.Join(home, ".config", "vibium", "ai.env")
	for _, tc := range []struct{ choice, model string }{
		{"3", "claude-sonnet-4-6"}, // anthropic
		{"4", "gemini-2.5-flash"},  // google
	} {
		t.Setenv("VIBIUM_AI_PROVIDER", "")
		t.Setenv("VIBIUM_AI_MODEL", "")
		os.Remove(path)
		os.Remove(path + ".bak")
		// Provider choice, Enter through the model default, Enter to skip the key.
		in := bytes.NewBufferString(tc.choice + "\n\n\n")
		ui := &setupUI{in: in, out: ioDiscard(), err: ioDiscard(), interactive: true}
		sec := setupAI(&cobra.Command{}, ui, false)
		if sec.Status != "done" {
			t.Fatalf("choice %s: %+v", tc.choice, sec)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), "export VIBIUM_AI_MODEL='"+tc.model+"'") {
			t.Fatalf("choice %s: model default not written: %s", tc.choice, body)
		}
	}
}

func TestParseSelectChoice(t *testing.T) {
	// Digits map onto setupProviders; the accepted range must follow the
	// list so adding a provider never silently truncates the menu.
	opts := providerOptions()
	for i, want := range setupProviders {
		got, err := parseSelectChoice(string(rune('1'+i)), opts)
		if err != nil || setupProviders[got] != want {
			t.Fatalf("choice %d: got %d %v", i+1, got, err)
		}
	}
	got, err := parseSelectChoice("2", opts)
	if err != nil || setupProviders[got] != "xai" {
		t.Fatalf("got %d %v", got, err)
	}
	got, err = parseSelectChoice("local", opts)
	if err != nil || setupProviders[got] != "local" {
		t.Fatalf("got %d %v", got, err)
	}
	if _, err := parseSelectChoice(string(rune('1'+len(setupProviders))), opts); err == nil {
		t.Fatal("accepted digit past the menu")
	}
	if _, err := parseSelectChoice("nope", opts); err == nil {
		t.Fatal("accepted unknown provider")
	}
}
