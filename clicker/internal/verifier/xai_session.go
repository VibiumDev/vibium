package verifier

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Reuse a signed-in Grok CLI session for the xai provider, read-only.
// Vibium never talks to auth.x.ai: it does not mint, refresh, or store
// tokens, so it cannot invalidate the Grok CLI's own session. When the
// session expires the fix is grok login or XAI_API_KEY.

const (
	CredentialAPIKey      = "api_key"
	CredentialGrokSession = "grok_session"
	credentialGrokExpired = "grok_expired"

	xaiOIDCIssuer = "https://auth.x.ai"
	// Treat a token this close to exp as expired; a Run should not start
	// on a session that dies mid-loop.
	xaiTokenSkew = 2 * time.Minute
)

// xaiSessionNow is a test hook. Production leaves it nil.
var xaiSessionNow func() time.Time

func applyXAICredentials(c *Config) {
	if strings.TrimSpace(c.APIKey) != "" {
		c.CredentialSource = CredentialAPIKey
		return
	}
	token, ok := grokSessionToken()
	if !ok {
		return
	}
	if xaiTokenExpired(token) {
		c.CredentialSource = credentialGrokExpired
		return
	}
	c.APIKey = token
	c.CredentialSource = CredentialGrokSession
}

type grokAuthEntry struct {
	Key          string `json:"key"`
	RefreshToken string `json:"refresh_token"`
	OIDCIssuer   string `json:"oidc_issuer"`
}

// grokSessionToken returns the access token of the auth.x.ai entry in the
// Grok CLI's auth file. Unreadable or unparseable files count as absent;
// the credential check names both remedies either way.
func grokSessionToken() (string, bool) {
	entries, ok := readGrokAuthEntries()
	if !ok {
		return "", false
	}
	var prefixMatch string
	for key, entry := range entries {
		if strings.TrimSpace(entry.Key) == "" {
			continue
		}
		if entry.OIDCIssuer == xaiOIDCIssuer {
			return entry.Key, true
		}
		if prefixMatch == "" && strings.HasPrefix(key, xaiOIDCIssuer) {
			prefixMatch = entry.Key
		}
	}
	return prefixMatch, prefixMatch != ""
}

func readGrokAuthEntries() (map[string]grokAuthEntry, bool) {
	path, err := grokAuthPath()
	if err != nil {
		return nil, false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var entries map[string]grokAuthEntry
	if json.Unmarshal(raw, &entries) != nil || len(entries) == 0 {
		return nil, false
	}
	return entries, true
}

// grokAuthPath honors GROK_HOME the way add-skill does; explicit
// configuration is intent.
func grokAuthPath() (string, error) {
	if root := strings.TrimSpace(os.Getenv("GROK_HOME")); root != "" {
		return filepath.Join(root, "auth.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".grok", "auth.json"), nil
}

// xaiTokenExpired treats tokens without a readable exp claim as expired;
// only a session Vibium can trust to outlive the Run is worth borrowing.
func xaiTokenExpired(token string) bool {
	exp, ok := jwtExp(token)
	if !ok {
		return true
	}
	now := time.Now()
	if xaiSessionNow != nil {
		now = xaiSessionNow()
	}
	return !now.Before(exp.Add(-xaiTokenSkew))
}

func jwtExp(token string) (time.Time, bool) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		payload, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return time.Time{}, false
		}
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if json.Unmarshal(payload, &claims) != nil || claims.Exp == 0 {
		return time.Time{}, false
	}
	return time.Unix(claims.Exp, 0), true
}

// XAIAuthSecrets returns token values from the Grok CLI auth file so
// recordings can redact them. Values only; never log them.
func XAIAuthSecrets() []string {
	entries, ok := readGrokAuthEntries()
	if !ok {
		return nil
	}
	var secrets []string
	for _, entry := range entries {
		for _, value := range []string{entry.Key, entry.RefreshToken} {
			if value = strings.TrimSpace(value); value != "" {
				secrets = append(secrets, value)
			}
		}
	}
	return secrets
}
