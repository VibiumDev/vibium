package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestLogoutXAICommandRemovesVibiumFileOnly(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, "vibium")
	t.Setenv("HOME", home)
	t.Setenv("VIBIUM_CONFIG_DIR", configDir)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}
	vibium := filepath.Join(configDir, "xai-auth.json")
	if err := os.WriteFile(vibium, []byte(`{"access_token":"v","refresh_token":"r"}`), 0600); err != nil {
		t.Fatal(err)
	}
	grokDir := filepath.Join(home, ".grok")
	if err := os.MkdirAll(grokDir, 0700); err != nil {
		t.Fatal(err)
	}
	grok := filepath.Join(grokDir, "auth.json")
	if err := os.WriteFile(grok, []byte(`{"https://auth.x.ai::x":{"key":"grok-access"}}`), 0600); err != nil {
		t.Fatal(err)
	}

	oldJSON := jsonOutput
	jsonOutput = false
	t.Cleanup(func() { jsonOutput = oldJSON })

	cmd := newLogoutXAICmd()
	cmd.SetOut(&strings.Builder{})
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(vibium); !os.IsNotExist(err) {
		t.Fatal("vibium xai-auth.json still present")
	}
	raw, err := os.ReadFile(grok)
	if err != nil || !strings.Contains(string(raw), "grok-access") {
		t.Fatal("logout touched ~/.grok/auth.json")
	}
}

func TestLogoutXAICommandJSONEnvelope(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("VIBIUM_CONFIG_DIR", t.TempDir())
	oldJSON := jsonOutput
	jsonOutput = true
	t.Cleanup(func() { jsonOutput = oldJSON })
	if err := newLogoutXAICmd().RunE(newLogoutXAICmd(), nil); err != nil {
		t.Fatal(err)
	}
}

func TestLoginLogoutCommandsAreRegistered(t *testing.T) {
	root, _ := newRootCmd("vibium")
	found := map[string]bool{}
	walkCommands(root, func(path string, _ *cobra.Command) {
		found[path] = true
	})
	for _, path := range []string{"login", "login xai", "logout", "logout xai"} {
		if !found[path] {
			t.Fatalf("missing command %s", path)
		}
	}
}
