package api

import (
	"os"
	"path/filepath"
	"testing"
)

// A borrowed Grok session token authenticates provider calls the same way
// XAI_API_KEY does, so recordings must treat both token values as secrets.
func TestRecordingRedactsGrokAuthTokens(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GROK_HOME", "")
	if err := os.MkdirAll(filepath.Join(home, ".grok"), 0o700); err != nil {
		t.Fatal(err)
	}
	body := `{"https://auth.x.ai/user":{"key":"GROK-ACCESS-SENTINEL","refresh_token":"GROK-REFRESH-SENTINEL","oidc_issuer":"https://auth.x.ai"}}`
	if err := os.WriteFile(filepath.Join(home, ".grok", "auth.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	r := NewRecorder()
	r.Start(RecordingStartOptions{}, nil)
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, sentinel := range []string{"GROK-ACCESS-SENTINEL", "GROK-REFRESH-SENTINEL"} {
		if !r.secrets[sentinel] {
			t.Fatalf("%s not registered for redaction", sentinel)
		}
	}
}
