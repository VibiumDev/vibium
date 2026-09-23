package verifier

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// The site under test (#575). A declared site does three things: the model's
// input carries it as a trusted line, relative browser_navigate targets
// resolve against it before the executors' absolute-URL validation, and the
// operation opens it first unless the current page is already on its origin,
// so an operation on a browser just used keeps its state. Navigation is not
// locked to the origin; login and payment flows cross domains.

// ParseSiteURL validates a declared site under test. The rules match the
// executors' navigation checks so a base that validates can also be opened.
func ParseSiteURL(site string) (*url.URL, error) {
	u, err := url.Parse(site)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
		return nil, fmt.Errorf("the site under test must be an absolute HTTP(S) URL without credentials")
	}
	return u, nil
}

// WithSite resolves relative browser_navigate targets against the site under
// test. Absolute targets and other tools pass through unchanged.
func WithSite(inner ToolExecutor, base *url.URL) ToolExecutor {
	return &siteExecutor{inner: inner, base: base}
}

type siteExecutor struct {
	inner ToolExecutor
	base  *url.URL
}

func (s *siteExecutor) Tools() []Tool { return s.inner.Tools() }

func (s *siteExecutor) Execute(ctx context.Context, name string, args map[string]interface{}) (Observation, error) {
	if name == "browser_navigate" {
		if target, ok := args["url"].(string); ok {
			if ref, err := url.Parse(target); err == nil && !ref.IsAbs() {
				resolved := map[string]interface{}{}
				for k, v := range args {
					resolved[k] = v
				}
				resolved["url"] = s.base.ResolveReference(ref).String()
				args = resolved
			}
		}
	}
	return s.inner.Execute(ctx, name, args)
}

// OpenSite navigates to the site under test unless the current page already
// shares its origin. A page whose URL cannot be read (fresh browser,
// about:blank) counts as elsewhere and is navigated.
func OpenSite(ctx context.Context, executor ToolExecutor, base *url.URL) error {
	if obs, err := executor.Execute(ctx, "browser_get_url", map[string]interface{}{}); err == nil {
		if current, err := url.Parse(observedURL(obs.Text)); err == nil && sameOrigin(current, base) {
			return nil
		}
	}
	if _, err := executor.Execute(ctx, "browser_navigate", map[string]interface{}{"url": base.String()}); err != nil {
		return fmt.Errorf("could not open the site under test: %w", err)
	}
	return nil
}

// observedURL reads a URL from a browser_get_url observation: raw text on
// the CLI/MCP executor, a {"url": ...} object on the SDK executor.
func observedURL(text string) string {
	var wrapped struct {
		URL string `json:"url"`
	}
	if json.Unmarshal([]byte(text), &wrapped) == nil && wrapped.URL != "" {
		return wrapped.URL
	}
	return strings.TrimSpace(text)
}

func sameOrigin(a, b *url.URL) bool {
	return a.Scheme == b.Scheme && canonicalHost(a) == canonicalHost(b)
}

// canonicalHost drops a default port so example.com and example.com:443
// compare equal under https.
func canonicalHost(u *url.URL) string {
	host := u.Host
	if u.Scheme == "http" {
		host = strings.TrimSuffix(host, ":80")
	}
	if u.Scheme == "https" {
		host = strings.TrimSuffix(host, ":443")
	}
	return host
}
