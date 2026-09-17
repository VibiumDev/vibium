package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	for _, path := range []string{"login", "login status", "login xai", "logout", "logout xai"} {
		if !found[path] {
			t.Fatalf("missing command %s", path)
		}
	}
}

func isolateLoginStatus(t *testing.T) (home, configDir string) {
	t.Helper()
	home = t.TempDir()
	configDir = filepath.Join(home, "vibium")
	t.Setenv("HOME", home)
	t.Setenv("VIBIUM_CONFIG_DIR", configDir)
	t.Setenv("XAI_API_KEY", "")
	oldJSON := jsonOutput
	jsonOutput = false
	t.Cleanup(func() { jsonOutput = oldJSON })
	return home, configDir
}

func writeLoginFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}

func loginTestJWT(t *testing.T) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"exp":%d}`, time.Now().Add(time.Hour).Unix())))
	return header + "." + payload + ".sig"
}

func runLoginStatus(t *testing.T) (string, int) {
	t.Helper()
	var out strings.Builder
	code, err := emitLoginStatus(&out)
	if err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, secret := range []string{"env-api-key-secret", "oauth-token-secret", "oauth-refresh-secret", "grok-access-secret", "grok-refresh-secret"} {
		if strings.Contains(text, secret) {
			t.Fatalf("secret leaked: %s", text)
		}
	}
	return text, code
}

func TestLoginStatusAPIKeyHumanAndJSON(t *testing.T) {
	_, configDir := isolateLoginStatus(t)
	writeLoginFile(t, filepath.Join(configDir, "xai-auth.json"), `{"access_token":"oauth-token-secret","refresh_token":"oauth-refresh-secret"}`)
	t.Setenv("XAI_API_KEY", "env-api-key-secret")

	text, code := runLoginStatus(t)
	if code != 0 || text != "xai: XAI_API_KEY (active; wins over Grok login)\n" {
		t.Fatalf("human: code=%d text=%q", code, text)
	}

	jsonOutput = true
	text, code = runLoginStatus(t)
	if code != 0 {
		t.Fatalf("json exit %d", code)
	}
	var env jsonEnvelope
	if err := json.Unmarshal([]byte(text), &env); err != nil || !env.OK {
		t.Fatalf("envelope: %s %v", text, err)
	}
	if strings.Contains(text, "env-api-key-secret") || strings.Contains(text, "oauth-token-secret") {
		t.Fatalf("secret leaked: %s", text)
	}
	if !strings.Contains(text, `"active":"api_key"`) || !strings.Contains(text, `"api_key":true`) {
		t.Fatalf("json: %s", text)
	}
}

func TestLoginStatusVibiumOAuthHuman(t *testing.T) {
	_, configDir := isolateLoginStatus(t)
	token := loginTestJWT(t)
	writeLoginFile(t, filepath.Join(configDir, "xai-auth.json"), fmt.Sprintf(`{"access_token":%q,"refresh_token":"oauth-refresh-secret"}`, token))
	text, code := runLoginStatus(t)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.HasPrefix(text, "xai: Grok login (Vibium ") || !strings.Contains(text, "xai-auth.json)\n") {
		t.Fatalf("text=%q", text)
	}
	if strings.Contains(text, "token expiring") || strings.Contains(text, token) || strings.Contains(text, "oauth-refresh-secret") {
		t.Fatalf("text=%q", text)
	}
}

func TestLoginStatusGrokCLIOAuthHuman(t *testing.T) {
	home, _ := isolateLoginStatus(t)
	token := loginTestJWT(t)
	writeLoginFile(t, filepath.Join(home, ".grok", "auth.json"), fmt.Sprintf(`{"https://auth.x.ai::x":{"key":%q,"refresh_token":"grok-refresh-secret","oidc_issuer":"https://auth.x.ai"}}`, token))
	text, code := runLoginStatus(t)
	if code != 0 || text != "xai: Grok login (Grok CLI ~/.grok/auth.json)\n" {
		t.Fatalf("code=%d text=%q", code, text)
	}
	if strings.Contains(text, token) {
		t.Fatalf("jwt leaked: %s", text)
	}
}

func TestLoginStatusNotSignedIn(t *testing.T) {
	isolateLoginStatus(t)
	text, code := runLoginStatus(t)
	want := "xai: not signed in\nExport XAI_API_KEY or run: vibium login xai\n"
	if code != 1 || text != want {
		t.Fatalf("code=%d text=%q", code, text)
	}

	jsonOutput = true
	text, code = runLoginStatus(t)
	if code != 1 {
		t.Fatalf("json exit %d", code)
	}
	var env jsonEnvelope
	if err := json.Unmarshal([]byte(text), &env); err != nil || !env.OK {
		t.Fatalf("envelope: %s %v", text, err)
	}
	if !strings.Contains(text, `"active":"none"`) || !strings.Contains(text, `"api_key":false`) {
		t.Fatalf("json: %s", text)
	}
}

func TestLoginStatusExpiredWithoutRefresh(t *testing.T) {
	_, configDir := isolateLoginStatus(t)
	writeLoginFile(t, filepath.Join(configDir, "xai-auth.json"), `{"access_token":"oauth-token-secret"}`)
	text, code := runLoginStatus(t)
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(text, "expired") || !strings.Contains(text, "vibium login xai") {
		t.Fatalf("text=%q", text)
	}
	if strings.Contains(text, "oauth-token-secret") {
		t.Fatalf("secret leaked: %s", text)
	}
}

func TestLoginStatusExpiringWithRefreshStaysSignedIn(t *testing.T) {
	_, configDir := isolateLoginStatus(t)
	writeLoginFile(t, filepath.Join(configDir, "xai-auth.json"), `{"access_token":"oauth-token-secret","refresh_token":"oauth-refresh-secret"}`)
	text, code := runLoginStatus(t)
	if code != 0 || !strings.Contains(text, "token expiring") {
		t.Fatalf("code=%d text=%q", code, text)
	}
}

func TestLoginStatusJSONOmitsTokenFields(t *testing.T) {
	home, configDir := isolateLoginStatus(t)
	token := loginTestJWT(t)
	writeLoginFile(t, filepath.Join(configDir, "xai-auth.json"), fmt.Sprintf(`{"access_token":%q,"refresh_token":"oauth-refresh-secret"}`, token))
	writeLoginFile(t, filepath.Join(home, ".grok", "auth.json"), `{"https://auth.x.ai::x":{"key":"grok-access-secret","refresh_token":"grok-refresh-secret","oidc_issuer":"https://auth.x.ai"}}`)
	jsonOutput = true
	text, code := runLoginStatus(t)
	if code != 0 {
		t.Fatalf("exit %d %s", code, text)
	}
	for _, leaked := range []string{token, "oauth-refresh-secret", "grok-access-secret", "access_token", "refresh_token", `"path"`} {
		if strings.Contains(text, leaked) {
			t.Fatalf("leaked %q in %s", leaked, text)
		}
	}
	if !strings.Contains(text, `"ok":true`) || !strings.Contains(text, `"source":"vibium"`) {
		t.Fatalf("json: %s", text)
	}
}
