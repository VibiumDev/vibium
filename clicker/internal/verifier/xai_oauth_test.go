package verifier

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func isolateXAIAuth(t *testing.T) (home, configDir string) {
	t.Helper()
	home = t.TempDir()
	configDir = filepath.Join(home, "vibium")
	t.Setenv("HOME", home)
	t.Setenv("VIBIUM_CONFIG_DIR", configDir)
	t.Setenv("XAI_API_KEY", "")
	t.Setenv("VIBIUM_AI_PROVIDER", "xai")
	t.Setenv("VIBIUM_AI_MODEL", "grok-4")
	t.Setenv("VIBIUM_AI_BASE_URL", "")
	t.Setenv("VIBIUM_AI_REASONING_EFFORT", "")
	t.Setenv("OPENAI_API_KEY", "")
	xaiSleep = func(time.Duration) {}
	t.Cleanup(func() {
		xaiAuthHTTP = nil
		xaiAuthNow = nil
		xaiSleep = nil
	})
	return home, configDir
}

func jwtWithExp(exp int64) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"exp":%d}`, exp)))
	return header + "." + payload + ".sig"
}

func xaiRewriteClient(server *httptest.Server) *http.Client {
	target, _ := url.Parse(server.URL)
	return &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			clone := req.Clone(req.Context())
			clone.URL.Scheme = target.Scheme
			clone.URL.Host = target.Host
			clone.Host = target.Host
			clone.RequestURI = ""
			return http.DefaultTransport.RoundTrip(clone)
		}),
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func writeVibiumAuth(t *testing.T, configDir, access, refresh string) string {
	t.Helper()
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(configDir, xaiAuthFileName)
	data, _ := json.Marshal(xaiVibiumAuth{
		Issuer:       xaiOIDCIssuer,
		ClientID:     xaiOAuthClientID,
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
	})
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeGrokAuth(t *testing.T, home, access, refresh string) string {
	t.Helper()
	dir := filepath.Join(home, ".grok")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "auth.json")
	entry := map[string]interface{}{
		"key":                           access,
		"refresh_token":                 refresh,
		"oidc_issuer":                   xaiOIDCIssuer,
		"oidc_client_id":                xaiOAuthClientID,
		"email":                         "dev@example.test",
		"coding_data_retention_opt_out": true,
	}
	body, _ := json.Marshal(map[string]interface{}{
		"https://auth.x.ai::" + xaiOAuthClientID: entry,
		"https://other.example::other": map[string]string{
			"key":           "OTHER-PROVIDER-TOKEN",
			"refresh_token": "OTHER-REFRESH",
		},
	})
	if err := os.WriteFile(path, body, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestXAIAPIKeyWinsOverOAuthFile(t *testing.T) {
	_, configDir := isolateXAIAuth(t)
	writeVibiumAuth(t, configDir, "oauth-token", "oauth-refresh")
	t.Setenv("XAI_API_KEY", "env-api-key")
	for _, role := range []string{"run", "check"} {
		c, err := ConfigForRole(role)
		if err != nil || c.APIKey != "env-api-key" || c.CredentialSource != CredentialAPIKey {
			t.Fatalf("%s: %+v %v", role, c, err)
		}
	}
}

func TestXAIOauthFromVibiumStore(t *testing.T) {
	_, configDir := isolateXAIAuth(t)
	writeVibiumAuth(t, configDir, jwtWithExp(time.Now().Add(time.Hour).Unix()), "refresh")
	c, err := ConfigForRole("check")
	if err != nil || c.CredentialSource != CredentialOAuth || !strings.Contains(c.APIKey, ".") {
		t.Fatalf("vibium oauth: %+v %v", c, err)
	}
	if c.APIKey == "refresh" {
		t.Fatal("used refresh token as bearer")
	}
}

func TestXAIOauthFromGrokAuthJSON(t *testing.T) {
	home, _ := isolateXAIAuth(t)
	token := jwtWithExp(time.Now().Add(time.Hour).Unix())
	writeGrokAuth(t, home, token, "grok-refresh")
	c, err := ConfigForRole("run")
	if err != nil || c.APIKey != token || c.CredentialSource != CredentialOAuth {
		t.Fatalf("grok oauth: %+v %v", c, err)
	}
}

func TestXAIVibiumStorePreferredOverGrok(t *testing.T) {
	home, configDir := isolateXAIAuth(t)
	vibium := jwtWithExp(time.Now().Add(time.Hour).Unix())
	grok := jwtWithExp(time.Now().Add(2 * time.Hour).Unix())
	writeVibiumAuth(t, configDir, vibium, "vibium-refresh")
	writeGrokAuth(t, home, grok, "grok-refresh")
	c, err := ConfigForRole("check")
	if err != nil || c.APIKey != vibium {
		t.Fatalf("did not prefer vibium store: %+v %v", c, err)
	}
}

func TestXAIExpiredJWTRefreshesAndWritesRotatedRefresh(t *testing.T) {
	home, configDir := isolateXAIAuth(t)
	now := time.Unix(1_700_000_000, 0)
	xaiAuthNow = func() time.Time { return now }
	expired := jwtWithExp(now.Add(30 * time.Second).Unix())
	fresh := jwtWithExp(now.Add(time.Hour).Unix())
	writeVibiumAuth(t, configDir, expired, "old-refresh")
	writeGrokAuth(t, home, "unused-grok", "unused-grok-refresh")

	var gotGrant, gotRefresh, gotClient string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth2/token" || r.Method != http.MethodPost {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_ = r.ParseForm()
		gotGrant, gotRefresh, gotClient = r.FormValue("grant_type"), r.FormValue("refresh_token"), r.FormValue("client_id")
		json.NewEncoder(w).Encode(map[string]string{
			"access_token":  fresh,
			"refresh_token": "rotated-refresh",
			"token_type":    "Bearer",
		})
	}))
	defer server.Close()
	xaiAuthHTTP = xaiRewriteClient(server)

	c, err := ConfigForRole("check")
	if err != nil || c.APIKey != fresh || c.CredentialSource != CredentialOAuth {
		t.Fatalf("refresh: %+v %v", c, err)
	}
	if gotGrant != "refresh_token" || gotRefresh != "old-refresh" || gotClient != xaiOAuthClientID {
		t.Fatalf("refresh form: grant=%s refresh=%s client=%s", gotGrant, gotRefresh, gotClient)
	}
	raw, err := os.ReadFile(filepath.Join(configDir, xaiAuthFileName))
	if err != nil {
		t.Fatal(err)
	}
	var stored xaiVibiumAuth
	if json.Unmarshal(raw, &stored) != nil || stored.AccessToken != fresh || stored.RefreshToken != "rotated-refresh" {
		t.Fatalf("vibium store not rotated: %s", raw)
	}
	if strings.Contains(string(raw), "unused-grok") {
		t.Fatal("wrote grok tokens into vibium store")
	}
	grokRaw, _ := os.ReadFile(filepath.Join(home, ".grok", "auth.json"))
	if !strings.Contains(string(grokRaw), "unused-grok-refresh") {
		t.Fatal("refresh of vibium store mutated grok auth.json")
	}
}

func TestXAIGrokRefreshCopiesToVibiumStoreWithoutWritingGrok(t *testing.T) {
	home, configDir := isolateXAIAuth(t)
	now := time.Unix(1_700_000_000, 0)
	xaiAuthNow = func() time.Time { return now }
	expired := jwtWithExp(now.Unix() - 10)
	fresh := jwtWithExp(now.Add(time.Hour).Unix())
	path := writeGrokAuth(t, home, expired, "grok-old-refresh")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.FormValue("refresh_token") != "grok-old-refresh" {
			t.Error("wrong grok refresh token")
		}
		json.NewEncoder(w).Encode(map[string]string{"access_token": fresh, "refresh_token": "grok-new-refresh"})
	}))
	defer server.Close()
	xaiAuthHTTP = xaiRewriteClient(server)

	c, err := ConfigForRole("run")
	if err != nil || c.APIKey != fresh {
		t.Fatalf("grok refresh: %+v %v", c, err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("refresh mutated ~/.grok/auth.json")
	}
	if !strings.Contains(string(after), "grok-old-refresh") || !strings.Contains(string(after), "OTHER-PROVIDER-TOKEN") {
		t.Fatal("lost grok auth.json contents")
	}
	raw, err := os.ReadFile(filepath.Join(configDir, xaiAuthFileName))
	if err != nil {
		t.Fatal(err)
	}
	var stored xaiVibiumAuth
	if json.Unmarshal(raw, &stored) != nil || stored.AccessToken != fresh || stored.RefreshToken != "grok-new-refresh" {
		t.Fatalf("did not copy rotated tokens to vibium store: %s", raw)
	}
}

func TestXAIRefreshRefusesNonHTTPSAndNonXAITokenURL(t *testing.T) {
	contacted := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		contacted = true
	}))
	defer server.Close()
	for _, raw := range []string{
		"http://auth.x.ai/oauth2/token",
		"https://evil.example/oauth2/token",
		"https://auth.x.ai.evil.com/oauth2/token",
		"https://user:pass@auth.x.ai/oauth2/token",
		server.URL + "/oauth2/token",
	} {
		contacted = false
		_, _, err := refreshXAIAccess(context.Background(), raw, xaiOAuthClientID, "refresh")
		if err == nil || contacted {
			t.Fatalf("accepted or contacted %s: err=%v contacted=%v", raw, err, contacted)
		}
	}
	if err := validateXAIAuthURL(xaiTokenURL); err != nil {
		t.Fatal(err)
	}
	if err := validateXAIAuthURL(xaiDeviceURL); err != nil {
		t.Fatal(err)
	}
}

func TestXAIChecksNameBothCredentialPaths(t *testing.T) {
	isolateXAIAuth(t)
	c, err := ConfigForRole("check")
	if err == nil {
		t.Fatal("expected missing credential error")
	}
	if !strings.Contains(err.Error(), "XAI_API_KEY") || !strings.Contains(err.Error(), "vibium login xai") {
		t.Fatalf("error should name both paths: %v", err)
	}
	if strings.TrimSpace(err.Error()) == "XAI_API_KEY is required" {
		t.Fatal("only named the API key")
	}
	found := false
	for _, check := range c.Checks() {
		if check.Variable == "Grok login" && strings.Contains(check.Error, "vibium login xai") {
			found = true
		}
		if check.Variable == "XAI_API_KEY" {
			t.Fatal("missing oauth still named XAI_API_KEY")
		}
	}
	if !found {
		t.Fatalf("checks: %+v", c.Checks())
	}
}

func TestXAIOauthCheckNameIsGrokLogin(t *testing.T) {
	_, configDir := isolateXAIAuth(t)
	writeVibiumAuth(t, configDir, jwtWithExp(time.Now().Add(time.Hour).Unix()), "refresh")
	c, err := ConfigForRole("check")
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range c.Checks() {
		if check.Variable == "XAI_API_KEY" {
			t.Fatal("oauth config still checked XAI_API_KEY")
		}
		if check.Variable == "Grok login" && check.Error != "" {
			t.Fatalf("oauth check failed: %+v", check)
		}
	}
}

func TestXAILoginWritesOwnerOnlyFile(t *testing.T) {
	_, configDir := isolateXAIAuth(t)
	var polls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/device/code":
			_ = r.ParseForm()
			if r.FormValue("client_id") != xaiOAuthClientID || !strings.Contains(r.FormValue("scope"), "api:access") {
				t.Errorf("device form: %s", r.Form.Encode())
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"device_code":      "SECRET-DEVICE-CODE",
				"user_code":        "ABCD-EFGH",
				"verification_uri": "https://accounts.x.ai/oauth2/device",
				"expires_in":       600,
				"interval":         5,
			})
		case "/oauth2/token":
			polls++
			_ = r.ParseForm()
			if r.FormValue("device_code") != "SECRET-DEVICE-CODE" {
				t.Error("token poll missing device code")
			}
			if polls == 1 {
				w.WriteHeader(400)
				json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
				return
			}
			json.NewEncoder(w).Encode(map[string]string{
				"access_token":  "login-access",
				"refresh_token": "login-refresh",
				"token_type":    "Bearer",
			})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	xaiAuthHTTP = xaiRewriteClient(server)

	var out strings.Builder
	path, err := LoginXAI(context.Background(), &out)
	if err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join(configDir, xaiAuthFileName)
	if path != wantPath {
		t.Fatalf("path=%s want %s", path, wantPath)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode=%o", info.Mode().Perm())
	}
	raw, _ := os.ReadFile(path)
	var stored xaiVibiumAuth
	if json.Unmarshal(raw, &stored) != nil || stored.AccessToken != "login-access" || stored.RefreshToken != "login-refresh" || stored.Issuer != xaiOIDCIssuer {
		t.Fatalf("store: %s", raw)
	}
	printed := out.String()
	if !strings.Contains(printed, "https://accounts.x.ai/oauth2/device") || !strings.Contains(printed, "ABCD-EFGH") {
		t.Fatalf("missing URI or user code: %s", printed)
	}
	if strings.Contains(printed, "SECRET-DEVICE-CODE") || strings.Contains(printed, "login-access") || strings.Contains(printed, "login-refresh") {
		t.Fatal("printed a secret")
	}
	if polls < 2 {
		t.Fatal("did not poll past authorization_pending")
	}
}

func TestXAILogoutRemovesVibiumFileOnly(t *testing.T) {
	home, configDir := isolateXAIAuth(t)
	vibium := writeVibiumAuth(t, configDir, "vibium-access", "vibium-refresh")
	grok := writeGrokAuth(t, home, "grok-access", "grok-refresh")
	if err := LogoutXAI(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(vibium); !os.IsNotExist(err) {
		t.Fatalf("vibium store still present: %v", err)
	}
	raw, err := os.ReadFile(grok)
	if err != nil || !strings.Contains(string(raw), "grok-access") {
		t.Fatal("logout touched ~/.grok/auth.json")
	}
	if err := LogoutXAI(); err != nil {
		t.Fatal(err)
	}
}

func TestXAIOauthStillSharesRunAndCheckDefaults(t *testing.T) {
	_, configDir := isolateXAIAuth(t)
	token := jwtWithExp(time.Now().Add(time.Hour).Unix())
	writeVibiumAuth(t, configDir, token, "refresh")
	run, err := ConfigForRole("run")
	if err != nil {
		t.Fatal(err)
	}
	check, err := ConfigForRole("check")
	if err != nil {
		t.Fatal(err)
	}
	if run.APIKey != check.APIKey || run.Provider != "xai" || check.Provider != "xai" || run.Model != "grok-4" || check.Model != "grok-4" {
		t.Fatalf("roles diverged: run=%+v check=%+v", run, check)
	}
	if run.CredentialSource != CredentialOAuth || check.CredentialSource != CredentialOAuth {
		t.Fatal("expected oauth source")
	}
}

func TestXAICredentialSourceOmittedFromJSON(t *testing.T) {
	c := Config{Provider: "xai", Model: "grok-4", APIKey: "secret", CredentialSource: CredentialOAuth}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "oauth") || strings.Contains(string(data), "CredentialSource") || strings.Contains(string(data), "api_key") {
		t.Fatalf("credential source leaked: %s", data)
	}
}

func TestXAIAuthSecretsDoNotPanicOnMissingFiles(t *testing.T) {
	isolateXAIAuth(t)
	if secrets := XAIAuthSecrets(); len(secrets) != 0 {
		t.Fatalf("unexpected secrets: %d", len(secrets))
	}
}

func TestXAILoginOutputWriterOptional(t *testing.T) {
	isolateXAIAuth(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "device") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"device_code": "d", "user_code": "CODE-1",
				"verification_uri": "https://accounts.x.ai/oauth2/device",
				"expires_in":       60, "interval": 1,
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"access_token": "a", "refresh_token": "r"})
	}))
	defer server.Close()
	xaiAuthHTTP = xaiRewriteClient(server)
	if _, err := LoginXAI(context.Background(), io.Discard); err != nil {
		t.Fatal(err)
	}
}
