package verifier

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vibium/clicker/internal/paths"
)

const (
	CredentialAPIKey = "api_key"
	CredentialOAuth  = "oauth"

	xaiOIDCIssuer    = "https://auth.x.ai"
	xaiOAuthClientID = "b1a00492-073a-47ea-816f-4c329264a828"
	xaiOAuthScope    = "openid profile email offline_access grok-cli:access api:access"
	xaiDeviceURL     = "https://auth.x.ai/oauth2/device/code"
	xaiTokenURL      = "https://auth.x.ai/oauth2/token"
	xaiRefreshSkew   = 120 * time.Second
	xaiDeviceGrant   = "urn:ietf:params:oauth:grant-type:device_code"
	xaiAuthFileName  = "xai-auth.json"
)

// Test hooks. Production leaves these nil.
var (
	xaiAuthHTTP *http.Client
	xaiAuthNow  func() time.Time
	xaiSleep    func(time.Duration)
)

type xaiVibiumAuth struct {
	Issuer       string `json:"issuer"`
	ClientID     string `json:"client_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

type xaiTokenSet struct {
	access   string
	refresh  string
	tokenTyp string
}

type xaiDeviceCode struct {
	deviceCode      string
	userCode        string
	verificationURI string
	interval        time.Duration
	expiresIn       time.Duration
}

type xaiStore struct {
	path    string
	kind    string // "vibium" or "grok"
	grokKey string
	raw     []byte
	access  string
	refresh string
	client  string
}

func applyXAICredentials(c *Config) error {
	if strings.TrimSpace(c.APIKey) != "" {
		c.CredentialSource = CredentialAPIKey
		return nil
	}
	c.APIKey = ""
	token, err := resolveXAIAccessToken(context.Background())
	if err != nil {
		return fmt.Errorf("xAI login could not be refreshed; run vibium login xai, or export XAI_API_KEY")
	}
	if token != "" {
		c.APIKey = token
		c.CredentialSource = CredentialOAuth
	}
	return nil
}

func resolveXAIAccessToken(ctx context.Context) (string, error) {
	store, ok, err := loadXAIStore()
	if err != nil {
		return "", err
	}
	if !ok {
		return "", nil
	}
	if store.access != "" && !xaiAccessNeedsRefresh(store.access) {
		return store.access, nil
	}
	if store.refresh == "" {
		return "", fmt.Errorf("xAI login expired")
	}
	access, refresh, err := refreshXAIAccess(ctx, xaiTokenURL, store.client, store.refresh)
	if err != nil {
		return "", err
	}
	if refresh == "" {
		refresh = store.refresh
	}
	if err := persistXAITokens(store, access, refresh); err != nil {
		return "", err
	}
	return access, nil
}

func loadXAIStore() (xaiStore, bool, error) {
	if store, ok, err := loadVibiumXAIStore(); err != nil || ok {
		return store, ok, err
	}
	return loadGrokXAIStore()
}

func loadVibiumXAIStore() (xaiStore, bool, error) {
	path, err := xaiVibiumAuthPath()
	if err != nil {
		return xaiStore{}, false, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return xaiStore{}, false, nil
		}
		return xaiStore{}, false, err
	}
	var file xaiVibiumAuth
	if json.Unmarshal(raw, &file) != nil || (strings.TrimSpace(file.AccessToken) == "" && strings.TrimSpace(file.RefreshToken) == "") {
		return xaiStore{}, false, nil
	}
	client := file.ClientID
	if client == "" {
		client = xaiOAuthClientID
	}
	return xaiStore{
		path:    path,
		kind:    "vibium",
		raw:     raw,
		access:  file.AccessToken,
		refresh: file.RefreshToken,
		client:  client,
	}, true, nil
}

func loadGrokXAIStore() (xaiStore, bool, error) {
	path, err := xaiGrokAuthPath()
	if err != nil {
		return xaiStore{}, false, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return xaiStore{}, false, nil
		}
		return xaiStore{}, false, err
	}
	var file map[string]json.RawMessage
	if json.Unmarshal(raw, &file) != nil {
		return xaiStore{}, false, nil
	}
	type grokEntry struct {
		Key          string `json:"key"`
		RefreshToken string `json:"refresh_token"`
		OIDCIssuer   string `json:"oidc_issuer"`
		ClientID     string `json:"oidc_client_id"`
	}
	fromEntry := func(key string, entry grokEntry) xaiStore {
		client := entry.ClientID
		if client == "" {
			client = xaiOAuthClientID
		}
		return xaiStore{
			path:    path,
			kind:    "grok",
			grokKey: key,
			raw:     raw,
			access:  entry.Key,
			refresh: entry.RefreshToken,
			client:  client,
		}
	}
	var prefixKey string
	var prefixEntry grokEntry
	for key, value := range file {
		var entry grokEntry
		if json.Unmarshal(value, &entry) != nil {
			continue
		}
		if entry.Key == "" && entry.RefreshToken == "" {
			continue
		}
		if entry.OIDCIssuer == xaiOIDCIssuer {
			return fromEntry(key, entry), true, nil
		}
		if prefixKey == "" && strings.HasPrefix(key, xaiOIDCIssuer) {
			prefixKey, prefixEntry = key, entry
		}
	}
	if prefixKey != "" {
		return fromEntry(prefixKey, prefixEntry), true, nil
	}
	return xaiStore{}, false, nil
}

func persistXAITokens(store xaiStore, access, refresh string) error {
	switch store.kind {
	case "vibium":
		obj, err := jsonObject(store.raw)
		if err != nil {
			obj = map[string]json.RawMessage{}
		}
		setJSONString(obj, "access_token", access)
		if refresh != "" {
			setJSONString(obj, "refresh_token", refresh)
		}
		data, err := json.Marshal(obj)
		if err != nil {
			return err
		}
		return writePrivateFile(store.path, data)
	case "grok":
		path, err := xaiVibiumAuthPath()
		if err != nil {
			return err
		}
		data, err := json.Marshal(xaiVibiumAuth{
			Issuer:       xaiOIDCIssuer,
			ClientID:     store.client,
			AccessToken:  access,
			RefreshToken: refresh,
			TokenType:    "Bearer",
		})
		if err != nil {
			return err
		}
		return writePrivateFile(path, data)
	default:
		return fmt.Errorf("unknown xAI auth store")
	}
}

func xaiAccessNeedsRefresh(token string) bool {
	exp, ok := jwtExp(token)
	if !ok {
		return true
	}
	return !xaiNow().Before(exp.Add(-xaiRefreshSkew))
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

func refreshXAIAccess(ctx context.Context, tokenURL, clientID, refreshToken string) (string, string, error) {
	if err := validateXAIAuthURL(tokenURL); err != nil {
		return "", "", err
	}
	if clientID == "" {
		clientID = xaiOAuthClientID
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", clientID)
	body, status, err := postXAIForm(ctx, tokenURL, form)
	if err != nil {
		return "", "", err
	}
	if status < 200 || status >= 300 {
		return "", "", fmt.Errorf("xAI token refresh failed")
	}
	tokens, err := parseXAITokenResponse(body)
	if err != nil {
		return "", "", err
	}
	return tokens.access, tokens.refresh, nil
}

func validateXAIAuthURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" {
		return fmt.Errorf("xAI auth URL is invalid")
	}
	if u.Scheme != "https" {
		return fmt.Errorf("xAI auth URL must be https")
	}
	host := strings.ToLower(u.Hostname())
	if host != "auth.x.ai" && !strings.HasSuffix(host, ".x.ai") {
		return fmt.Errorf("xAI auth URL host is not x.ai")
	}
	return nil
}

func validateXAIVerificationURI(raw string) error {
	if strings.IndexFunc(raw, func(r rune) bool { return r < 32 || r == 127 }) >= 0 {
		return fmt.Errorf("xAI returned an invalid verification URL")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil {
		return fmt.Errorf("xAI returned an invalid verification URL")
	}
	if u.Scheme != "https" {
		return fmt.Errorf("xAI returned an unsupported verification URL")
	}
	host := strings.ToLower(u.Hostname())
	if host != "auth.x.ai" && host != "accounts.x.ai" && !strings.HasSuffix(host, ".x.ai") {
		return fmt.Errorf("xAI returned an unsupported verification URL")
	}
	return nil
}

// LoginXAI runs RFC 8628 device-code login and writes the vibium xAI store.
func LoginXAI(ctx context.Context, prompt io.Writer) (string, error) {
	if prompt == nil {
		prompt = io.Discard
	}
	device, err := requestXAIDeviceCode(ctx)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(prompt, "To sign in to xAI, open this URL in a browser:\n\n  %s\n\nThen enter this code:\n\n  %s\n\nWaiting for authorization...\n", device.verificationURI, device.userCode)
	tokens, err := pollXAIDeviceToken(ctx, device)
	if err != nil {
		return "", err
	}
	path, err := xaiVibiumAuthPath()
	if err != nil {
		return "", err
	}
	tokenType := tokens.tokenTyp
	if tokenType == "" {
		tokenType = "Bearer"
	}
	data, err := json.Marshal(xaiVibiumAuth{
		Issuer:       xaiOIDCIssuer,
		ClientID:     xaiOAuthClientID,
		AccessToken:  tokens.access,
		RefreshToken: tokens.refresh,
		TokenType:    tokenType,
	})
	if err != nil {
		return "", err
	}
	if err := writePrivateFile(path, data); err != nil {
		return "", err
	}
	return path, nil
}

// LogoutXAI deletes only the vibium xAI store. ~/.grok/auth.json is left alone.
func LogoutXAI() error {
	path, err := xaiVibiumAuthPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// XAIAuthSecrets returns access and refresh token values from known stores
// so recordings can redact them. Values only; never log the map.
func XAIAuthSecrets() []string {
	var secrets []string
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			secrets = append(secrets, value)
		}
	}
	if path, err := xaiVibiumAuthPath(); err == nil {
		var file xaiVibiumAuth
		if raw, err := os.ReadFile(path); err == nil && json.Unmarshal(raw, &file) == nil {
			add(file.AccessToken)
			add(file.RefreshToken)
		}
	}
	if path, err := xaiGrokAuthPath(); err == nil {
		var file map[string]json.RawMessage
		if raw, err := os.ReadFile(path); err == nil && json.Unmarshal(raw, &file) == nil {
			type grokEntry struct {
				Key          string `json:"key"`
				RefreshToken string `json:"refresh_token"`
			}
			for _, value := range file {
				var entry grokEntry
				if json.Unmarshal(value, &entry) != nil {
					continue
				}
				add(entry.Key)
				add(entry.RefreshToken)
			}
		}
	}
	return secrets
}

func requestXAIDeviceCode(ctx context.Context) (xaiDeviceCode, error) {
	if err := validateXAIAuthURL(xaiDeviceURL); err != nil {
		return xaiDeviceCode{}, err
	}
	form := url.Values{}
	form.Set("client_id", xaiOAuthClientID)
	form.Set("scope", xaiOAuthScope)
	body, status, err := postXAIForm(ctx, xaiDeviceURL, form)
	if err != nil {
		return xaiDeviceCode{}, err
	}
	if status < 200 || status >= 300 {
		return xaiDeviceCode{}, fmt.Errorf("xAI device-code request failed")
	}
	var response struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
	}
	if json.Unmarshal(body, &response) != nil || response.DeviceCode == "" || response.UserCode == "" || response.VerificationURI == "" {
		return xaiDeviceCode{}, fmt.Errorf("xAI device-code response was invalid")
	}
	for _, c := range response.UserCode {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
			return xaiDeviceCode{}, fmt.Errorf("xAI returned an invalid user code")
		}
	}
	if err := validateXAIVerificationURI(response.VerificationURI); err != nil {
		return xaiDeviceCode{}, err
	}
	interval := response.Interval
	if interval <= 0 {
		interval = 5
	}
	expires := response.ExpiresIn
	if expires <= 0 {
		expires = 900
	}
	return xaiDeviceCode{
		deviceCode:      response.DeviceCode,
		userCode:        response.UserCode,
		verificationURI: response.VerificationURI,
		interval:        time.Duration(interval) * time.Second,
		expiresIn:       time.Duration(expires) * time.Second,
	}, nil
}

func pollXAIDeviceToken(ctx context.Context, device xaiDeviceCode) (xaiTokenSet, error) {
	if err := validateXAIAuthURL(xaiTokenURL); err != nil {
		return xaiTokenSet{}, err
	}
	deadline := xaiNow().Add(device.expiresIn)
	interval := device.interval
	for {
		xaiPause(interval)
		if err := ctx.Err(); err != nil {
			return xaiTokenSet{}, err
		}
		if !xaiNow().Before(deadline) {
			return xaiTokenSet{}, fmt.Errorf("xAI device code expired; run vibium login xai again")
		}
		form := url.Values{}
		form.Set("grant_type", xaiDeviceGrant)
		form.Set("device_code", device.deviceCode)
		form.Set("client_id", xaiOAuthClientID)
		body, status, err := postXAIForm(ctx, xaiTokenURL, form)
		if err != nil {
			return xaiTokenSet{}, err
		}
		if status >= 200 && status < 300 {
			return parseXAITokenResponse(body)
		}
		var tokenErr struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(body, &tokenErr)
		switch tokenErr.Error {
		case "authorization_pending":
			continue
		case "slow_down":
			interval += 5 * time.Second
			continue
		case "access_denied":
			return xaiTokenSet{}, fmt.Errorf("xAI authorization denied")
		case "expired_token":
			return xaiTokenSet{}, fmt.Errorf("xAI device code expired; run vibium login xai again")
		default:
			return xaiTokenSet{}, fmt.Errorf("xAI device-code login failed")
		}
	}
}

func parseXAITokenResponse(body []byte) (xaiTokenSet, error) {
	var response struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
	}
	if json.Unmarshal(body, &response) != nil || strings.TrimSpace(response.AccessToken) == "" {
		return xaiTokenSet{}, fmt.Errorf("xAI token response was invalid")
	}
	return xaiTokenSet{access: response.AccessToken, refresh: response.RefreshToken, tokenTyp: response.TokenType}, nil
}

func postXAIForm(ctx context.Context, endpoint string, form url.Values) ([]byte, int, error) {
	if err := validateXAIAuthURL(endpoint); err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, 0, fmt.Errorf("invalid xAI auth endpoint")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := xaiHTTPClient().Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, 0, ctx.Err()
		}
		return nil, 0, fmt.Errorf("xAI auth request failed")
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, 0, fmt.Errorf("xAI auth response was invalid")
	}
	return data, resp.StatusCode, nil
}

func xaiHTTPClient() *http.Client {
	if xaiAuthHTTP != nil {
		return xaiAuthHTTP
	}
	return &http.Client{
		Timeout:       30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func xaiNow() time.Time {
	if xaiAuthNow != nil {
		return xaiAuthNow()
	}
	return time.Now()
}

func xaiPause(d time.Duration) {
	if d <= 0 {
		return
	}
	if xaiSleep != nil {
		xaiSleep(d)
		return
	}
	time.Sleep(d)
}

func xaiVibiumAuthPath() (string, error) {
	dir, err := paths.GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, xaiAuthFileName), nil
}

func xaiGrokAuthPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".grok", "auth.json"), nil
}

func writePrivateFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0600); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Chmod(path, 0600)
}

func jsonObject(raw []byte) (map[string]json.RawMessage, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	if obj == nil {
		obj = map[string]json.RawMessage{}
	}
	return obj, nil
}

func setJSONString(obj map[string]json.RawMessage, key, value string) {
	encoded, _ := json.Marshal(value)
	obj[key] = encoded
}
