package verifier

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testJWT(t *testing.T, exp time.Time) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	payload, err := json.Marshal(map[string]int64{"exp": exp.Unix()})
	if err != nil {
		t.Fatal(err)
	}
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}

func writeGrokAuth(t *testing.T, home, token, refresh string) {
	t.Helper()
	dir := filepath.Join(home, ".grok")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"https://auth.x.ai/user":{"key":%q,"refresh_token":%q,"oidc_issuer":"https://auth.x.ai"}}`, token, refresh)
	if err := os.WriteFile(filepath.Join(dir, "auth.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func xaiTestEnv(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GROK_HOME", "")
	t.Setenv("VIBIUM_AI_PROVIDER", "xai")
	t.Setenv("VIBIUM_AI_MODEL", "grok-4")
	t.Setenv("VIBIUM_AI_BASE_URL", "")
	t.Setenv("VIBIUM_AI_REASONING_EFFORT", "")
	t.Setenv("XAI_API_KEY", "")
	return home
}

func TestXAIAPIKeyStillWins(t *testing.T) {
	home := xaiTestEnv(t)
	writeGrokAuth(t, home, testJWT(t, time.Now().Add(time.Hour)), "refresh-1")
	t.Setenv("XAI_API_KEY", "xai-console-key")
	c, err := ConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if c.APIKey != "xai-console-key" || c.CredentialSource != CredentialAPIKey {
		t.Fatalf("source %q key %q", c.CredentialSource, c.APIKey)
	}
}

func TestXAIUsesFreshGrokSession(t *testing.T) {
	home := xaiTestEnv(t)
	token := testJWT(t, time.Now().Add(time.Hour))
	writeGrokAuth(t, home, token, "refresh-1")
	c, err := ConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if c.APIKey != token || c.CredentialSource != CredentialGrokSession {
		t.Fatalf("source %q", c.CredentialSource)
	}
	name, problem := c.credentialCheck()
	if name != "Grok login" || problem != "" {
		t.Fatalf("check %q %q", name, problem)
	}
}

func TestXAIExpiredGrokSessionNamesBothRemedies(t *testing.T) {
	home := xaiTestEnv(t)
	writeGrokAuth(t, home, testJWT(t, time.Now().Add(-time.Hour)), "refresh-1")
	_, err := ConfigFromEnv()
	if err == nil || !strings.Contains(err.Error(), "grok login") || !strings.Contains(err.Error(), "XAI_API_KEY") {
		t.Fatalf("err %v", err)
	}
}

func TestXAISkewTreatsNearExpiryAsExpired(t *testing.T) {
	home := xaiTestEnv(t)
	writeGrokAuth(t, home, testJWT(t, time.Now().Add(time.Minute)), "refresh-1")
	c, err := ConfigFromEnv()
	if err == nil || c.CredentialSource != credentialGrokExpired {
		t.Fatalf("source %q err %v", c.CredentialSource, err)
	}
}

func TestXAINoSessionFallsBackToKeyMessage(t *testing.T) {
	xaiTestEnv(t)
	_, err := ConfigFromEnv()
	if err == nil || !strings.Contains(err.Error(), "XAI_API_KEY") || !strings.Contains(err.Error(), "grok login") {
		t.Fatalf("err %v", err)
	}
}

func TestXAIMalformedTokenCountsAsExpired(t *testing.T) {
	home := xaiTestEnv(t)
	writeGrokAuth(t, home, "not-a-jwt", "refresh-1")
	c, err := ConfigFromEnv()
	if err == nil || c.CredentialSource != credentialGrokExpired {
		t.Fatalf("source %q err %v", c.CredentialSource, err)
	}
}

func TestGrokAuthPathHonorsGROKHOME(t *testing.T) {
	home := xaiTestEnv(t)
	root := filepath.Join(home, "grok-root")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	token := testJWT(t, time.Now().Add(time.Hour))
	body := fmt.Sprintf(`{"https://auth.x.ai/user":{"key":%q,"oidc_issuer":"https://auth.x.ai"}}`, token)
	if err := os.WriteFile(filepath.Join(root, "auth.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GROK_HOME", root)
	c, err := ConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if c.APIKey != token {
		t.Fatal("GROK_HOME auth.json was not used")
	}
}

func TestXAIAuthSecretsListsTokenValues(t *testing.T) {
	home := xaiTestEnv(t)
	token := testJWT(t, time.Now().Add(time.Hour))
	writeGrokAuth(t, home, token, "refresh-secret")
	secrets := XAIAuthSecrets()
	joined := strings.Join(secrets, "\n")
	if !strings.Contains(joined, token) || !strings.Contains(joined, "refresh-secret") {
		t.Fatalf("secrets missing values: %d entries", len(secrets))
	}
}

func TestXAISessionNeverTouchesTheGrokFile(t *testing.T) {
	home := xaiTestEnv(t)
	writeGrokAuth(t, home, testJWT(t, time.Now().Add(time.Hour)), "refresh-1")
	path := filepath.Join(home, ".grok", "auth.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ConfigFromEnv(); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("auth.json changed")
	}
}
