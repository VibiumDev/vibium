package bidi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// classicCreateTimeout bounds the classic POST /session round-trip. Cloud
// grids boot a VM per session, which can take tens of seconds. The daemon
// IPC read deadline is 60s, so this must stay under it or the CLI reports
// a timeout while the daemon's create eventually succeeds.
const classicCreateTimeout = 55 * time.Second

// ClassicSession is a WebDriver classic session created over HTTP by
// ResolveEndpoint. Grids and cloud providers release the browser (and stop
// billing) on DELETE /session/<id>; leaving teardown to the provider's idle
// timeout marks the session as timed out instead of completed.
type ClassicSession struct {
	baseURL string // normalized endpoint base, without the /session suffix
	ID      string
	headers http.Header
}

// IsClassicEndpoint reports whether the connect URL is an HTTP(S) WebDriver
// classic endpoint (a Selenium Grid hub, chromedriver, a cloud grid) rather
// than a BiDi WebSocket URL.
func IsClassicEndpoint(endpoint string) bool {
	u, err := url.Parse(endpoint)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

// ResolveEndpoint turns any supported connect URL into a dialable BiDi
// WebSocket URL. ws/wss URLs pass through unchanged. http/https URLs are
// classic WebDriver endpoints: a session is created via POST /session with
// webSocketUrl:true (plus extraCaps), and the BiDi URL the endpoint returns
// is used. The *ClassicSession is non-nil only in that case; the caller must
// Delete it once done with the browser.
func ResolveEndpoint(endpoint string, headers http.Header, extraCaps map[string]interface{}) (string, *ClassicSession, error) {
	if !IsClassicEndpoint(endpoint) {
		return endpoint, nil, nil
	}
	return createClassicSession(endpoint, headers, extraCaps)
}

// createClassicSession creates a WebDriver classic session and returns the
// BiDi webSocketUrl from its capabilities.
func createClassicSession(endpoint string, headers http.Header, extraCaps map[string]interface{}) (string, *ClassicSession, error) {
	baseURL, headers, err := normalizeClassicEndpoint(endpoint, headers)
	if err != nil {
		return "", nil, err
	}

	alwaysMatch := make(map[string]interface{}, len(extraCaps)+1)
	for k, v := range extraCaps {
		alwaysMatch[k] = v
	}
	// BiDi is the whole point of the connection — force it regardless of caps.
	alwaysMatch["webSocketUrl"] = true

	body, err := json.Marshal(map[string]interface{}{
		"capabilities": map[string]interface{}{"alwaysMatch": alwaysMatch},
	})
	if err != nil {
		return "", nil, fmt.Errorf("marshal capabilities: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/session", bytes.NewReader(body))
	if err != nil {
		return "", nil, fmt.Errorf("build session request: %w", err)
	}
	req.Header = cloneHeader(headers)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	client := &http.Client{Timeout: classicCreateTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("create session at %s: %w", baseURL, err)
	}
	defer resp.Body.Close()

	// Screenshot-sized payloads never come back from session create; 1MB is
	// plenty for capabilities and keeps a misbehaving endpoint bounded.
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", nil, fmt.Errorf("read session response: %w", err)
	}

	var parsed struct {
		Value struct {
			SessionID    string                 `json:"sessionId"`
			Capabilities map[string]interface{} `json:"capabilities"`
			Error        string                 `json:"error"`
			Message      string                 `json:"message"`
		} `json:"value"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", nil, fmt.Errorf("session create returned %s with unparseable body %.200q", resp.Status, raw)
	}

	if parsed.Value.Error != "" || resp.StatusCode != http.StatusOK {
		msg := parsed.Value.Message
		if msg == "" {
			msg = fmt.Sprintf("%s: %.200s", resp.Status, raw)
		}
		return "", nil, fmt.Errorf("session not created: %s", msg)
	}

	session := &ClassicSession{
		baseURL: baseURL,
		ID:      parsed.Value.SessionID,
		headers: headers,
	}

	wsURL, _ := parsed.Value.Capabilities["webSocketUrl"].(string)
	if wsURL == "" {
		// Don't leave a billable session running behind an error.
		session.Delete()
		return "", nil, fmt.Errorf(
			"%s created a session but returned no webSocketUrl capability — the remote end does not support WebDriver BiDi", baseURL)
	}

	return wsURL, session, nil
}

// Delete ends the classic session via DELETE /session/<id>, which is what
// releases the slot on grids and cloud providers.
func (s *ClassicSession) Delete() error {
	if s == nil || s.ID == "" {
		return nil
	}
	req, err := http.NewRequest(http.MethodDelete, s.baseURL+"/session/"+s.ID, nil)
	if err != nil {
		return err
	}
	req.Header = cloneHeader(s.headers)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("delete session %s: %w", s.ID, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete session %s: %s", s.ID, resp.Status)
	}
	return nil
}

// normalizeClassicEndpoint strips a trailing slash or /session suffix (both
// https://hub/wd/hub and https://hub/wd/hub/session mean the same endpoint)
// and folds URL userinfo into a Basic Authorization header, the form cloud
// grids like https://USER:KEY@hub-cloud.browserstack.com/wd/hub expect.
func normalizeClassicEndpoint(endpoint string, headers http.Header) (string, http.Header, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", nil, fmt.Errorf("parse endpoint URL: %w", err)
	}

	headers = foldUserinfo(u, headers)

	u.Path = strings.TrimSuffix(strings.TrimSuffix(u.Path, "/"), "/session")
	return strings.TrimSuffix(u.String(), "/"), headers, nil
}

// foldUserinfo moves user:pass credentials out of the URL into a Basic
// Authorization header (unless one is already set). Mutates u, returns the
// headers to use — always non-nil.
func foldUserinfo(u *url.URL, headers http.Header) http.Header {
	headers = cloneHeader(headers)
	if u.User != nil {
		if headers.Get("Authorization") == "" {
			password, _ := u.User.Password()
			headers.Set("Authorization", "Basic "+basicAuth(u.User.Username(), password))
		}
		u.User = nil
	}
	return headers
}

func basicAuth(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}

func cloneHeader(h http.Header) http.Header {
	clone := make(http.Header, len(h))
	for k, vals := range h {
		clone[k] = append([]string(nil), vals...)
	}
	return clone
}
