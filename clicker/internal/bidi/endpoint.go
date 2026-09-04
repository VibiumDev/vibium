package bidi

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// redactedSecret replaces a credential in logs and error messages.
const redactedSecret = "redacted"

// NormalizeEndpoint rewrites a user-supplied BiDi endpoint into the form the
// WebSocket dialer accepts and returns any headers the URL itself implied.
//
// Two shapes that cloud browser providers document are rejected by the dialer
// with the same opaque "malformed ws or wss URL" (#101):
//
//   - An http:// or https:// scheme. BrowserStack's hub is documented as
//     https://hub-cloud.browserstack.com, and the CLI argument, the
//     VIBIUM_CONNECT_URL env var and the clients' connect option all pass that
//     string through verbatim.
//   - Credentials in the userinfo field (https://user:key@host). A WebSocket
//     URI may not carry them at all, so they have to move to an Authorization
//     header.
//
// Both are mechanical rewrites, so do them rather than fail. Anything else (an
// unknown scheme, a missing host) is a real mistake and is named as such
// instead of surfacing as "malformed".
func NormalizeEndpoint(endpoint string) (string, http.Header, error) {
	if strings.TrimSpace(endpoint) == "" {
		return "", nil, fmt.Errorf("empty endpoint URL (expected ws:// or wss://)")
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return "", nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}

	switch u.Scheme {
	case "ws", "wss":
		// Already what the dialer wants.
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "":
		return "", nil, fmt.Errorf("endpoint URL %q has no scheme (expected ws:// or wss://)",
			RedactEndpoint(endpoint))
	default:
		return "", nil, fmt.Errorf("unsupported endpoint scheme %q (expected ws, wss, http or https)", u.Scheme)
	}

	if u.Host == "" {
		return "", nil, fmt.Errorf("endpoint URL %q has no host", RedactEndpoint(endpoint))
	}

	var headers http.Header
	if u.User != nil {
		// url.Parse percent-decodes userinfo, so this is the credential the
		// user actually meant — encode it straight into Basic auth, the
		// scheme every provider that documents this URL shape expects.
		password, _ := u.User.Password()
		headers = make(http.Header)
		headers.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString(
			[]byte(u.User.Username()+":"+password)))
		u.User = nil
	}

	return u.String(), headers, nil
}

// RedactEndpoint returns endpoint with any credential in its userinfo field
// replaced, for logs and error messages. A provider access key lives in that
// field, and connect URLs are printed on stderr, which clients drain and
// surface to the user.
func RedactEndpoint(endpoint string) string {
	if u, err := url.Parse(endpoint); err == nil {
		if u.User == nil {
			return endpoint
		}
		if _, hasPassword := u.User.Password(); hasPassword {
			u.User = url.UserPassword(u.User.Username(), redactedSecret)
		}
		return u.String()
	}

	// Parsing failed, so the credential's boundaries are not reliably known.
	// Drop the whole userinfo field rather than risk printing a key.
	scheme, rest, found := strings.Cut(endpoint, "//")
	if !found {
		return endpoint
	}
	authority, path, hasPath := strings.Cut(rest, "/")
	if at := strings.LastIndex(authority, "@"); at >= 0 {
		authority = redactedSecret + "@" + authority[at+1:]
	}
	out := scheme + "//" + authority
	if hasPath {
		out += "/" + path
	}
	return out
}
