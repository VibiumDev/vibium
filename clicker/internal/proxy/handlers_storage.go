package proxy

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// --- BiDi cookie types ---

// bidiCookieValue represents a BiDi BytesValue: {type: "string", value: "..."}.
type bidiCookieValue struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// bidiCookie represents a cookie from storage.getCookies response.
type bidiCookie struct {
	Name     string           `json:"name"`
	Value    bidiCookieValue  `json:"value"`
	Domain   string           `json:"domain"`
	Path     string           `json:"path"`
	Size     int              `json:"size"`
	HTTPOnly bool             `json:"httpOnly"`
	Secure   bool             `json:"secure"`
	SameSite string           `json:"sameSite"`
	Expiry   *json.RawMessage `json:"expiry,omitempty"`
}

// --- Handlers ---

// handleContextCookies handles vibium:context.cookies — returns cookies for the user context.
// Optional URL filtering via the "urls" param.
func (r *Router) handleContextCookies(session *BrowserSession, cmd bidiCommand) {
	userContext, _ := cmd.Params["userContext"].(string)
	if userContext == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("userContext is required"))
		return
	}

	cookies, err := r.getCookiesForContext(session, userContext)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Filter by URLs if provided
	if urlsRaw, ok := cmd.Params["urls"]; ok {
		if urlSlice, ok := urlsRaw.([]interface{}); ok && len(urlSlice) > 0 {
			urls := make([]string, 0, len(urlSlice))
			for _, u := range urlSlice {
				if s, ok := u.(string); ok {
					urls = append(urls, s)
				}
			}
			cookies = filterCookiesByURLs(cookies, urls)
		}
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"cookies": cookies})
}

// handleContextSetCookies handles vibium:context.setCookies — sets cookies via storage.setCookie (one at a time).
func (r *Router) handleContextSetCookies(session *BrowserSession, cmd bidiCommand) {
	userContext, _ := cmd.Params["userContext"].(string)
	if userContext == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("userContext is required"))
		return
	}

	cookiesRaw, ok := cmd.Params["cookies"].([]interface{})
	if !ok || len(cookiesRaw) == 0 {
		r.sendError(session, cmd.ID, fmt.Errorf("cookies array is required"))
		return
	}

	partition := map[string]interface{}{
		"type":        "storageKey",
		"userContext": userContext,
	}

	for _, cRaw := range cookiesRaw {
		c, ok := cRaw.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := c["name"].(string)
		value, _ := c["value"].(string)
		domain, _ := c["domain"].(string)

		// If domain is empty but url is provided, extract hostname from url
		if domain == "" {
			if urlStr, ok := c["url"].(string); ok && urlStr != "" {
				if parsed, err := url.Parse(urlStr); err == nil {
					domain = parsed.Hostname()
				}
			}
		}

		if name == "" || domain == "" {
			r.sendError(session, cmd.ID, fmt.Errorf("cookie name and domain (or url) are required"))
			return
		}

		bidiCookie := map[string]interface{}{
			"name":  name,
			"value": map[string]interface{}{"type": "string", "value": value},
			"domain": domain,
			"path":  "/",
		}

		if path, ok := c["path"].(string); ok && path != "" {
			bidiCookie["path"] = path
		}
		if httpOnly, ok := c["httpOnly"].(bool); ok {
			bidiCookie["httpOnly"] = httpOnly
		}
		if secure, ok := c["secure"].(bool); ok {
			bidiCookie["secure"] = secure
		}
		if sameSite, ok := c["sameSite"].(string); ok && sameSite != "" {
			bidiCookie["sameSite"] = sameSite
		}
		if expiry, ok := c["expiry"].(float64); ok {
			bidiCookie["expiry"] = int(expiry)
		}

		params := map[string]interface{}{
			"cookie":    bidiCookie,
			"partition": partition,
		}

		resp, err := r.sendInternalCommand(session, "storage.setCookie", params)
		if err != nil {
			r.sendError(session, cmd.ID, err)
			return
		}
		if bidiErr := checkBidiError(resp); bidiErr != nil {
			r.sendError(session, cmd.ID, bidiErr)
			return
		}
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// handleContextClearCookies handles vibium:context.clearCookies — deletes all cookies for the user context.
func (r *Router) handleContextClearCookies(session *BrowserSession, cmd bidiCommand) {
	userContext, _ := cmd.Params["userContext"].(string)
	if userContext == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("userContext is required"))
		return
	}

	params := map[string]interface{}{
		"partition": map[string]interface{}{
			"type":        "storageKey",
			"userContext": userContext,
		},
	}

	// Optional filter for selective deletion
	if filter, ok := cmd.Params["filter"].(map[string]interface{}); ok {
		params["filter"] = filter
	}

	resp, err := r.sendInternalCommand(session, "storage.deleteCookies", params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		r.sendError(session, cmd.ID, bidiErr)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// handleContextStorageState handles vibium:context.storageState — returns cookies + localStorage + sessionStorage.
func (r *Router) handleContextStorageState(session *BrowserSession, cmd bidiCommand) {
	userContext, _ := cmd.Params["userContext"].(string)
	if userContext == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("userContext is required"))
		return
	}

	// 1. Get cookies
	cookies, err := r.getCookiesForContext(session, userContext)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// 2. Find a browsing context in this user context
	context, err := r.findContextForUserContext(session, userContext)
	if err != nil {
		// No page open — return cookies only with empty origins
		r.sendSuccess(session, cmd.ID, map[string]interface{}{
			"cookies": cookies,
			"origins": []interface{}{},
		})
		return
	}

	// 3. Evaluate JS to read localStorage + sessionStorage
	storageScript := `() => {
		const ls = {};
		for (let i = 0; i < localStorage.length; i++) {
			const key = localStorage.key(i);
			ls[key] = localStorage.getItem(key);
		}
		const ss = {};
		for (let i = 0; i < sessionStorage.length; i++) {
			const key = sessionStorage.key(i);
			ss[key] = sessionStorage.getItem(key);
		}
		return JSON.stringify({ origin: location.origin, localStorage: ls, sessionStorage: ss });
	}`

	storageJSON, err := r.evalSimpleScript(session, context, storageScript)
	if err != nil {
		// Script eval failed (e.g. about:blank) — return cookies with empty origins
		r.sendSuccess(session, cmd.ID, map[string]interface{}{
			"cookies": cookies,
			"origins": []interface{}{},
		})
		return
	}

	var storageData struct {
		Origin         string            `json:"origin"`
		LocalStorage   map[string]string `json:"localStorage"`
		SessionStorage map[string]string `json:"sessionStorage"`
	}
	if err := json.Unmarshal([]byte(storageJSON), &storageData); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("failed to parse storage data: %w", err))
		return
	}

	// Convert maps to {name, value} arrays
	lsItems := make([]map[string]string, 0)
	for k, v := range storageData.LocalStorage {
		lsItems = append(lsItems, map[string]string{"name": k, "value": v})
	}
	ssItems := make([]map[string]string, 0)
	for k, v := range storageData.SessionStorage {
		ssItems = append(ssItems, map[string]string{"name": k, "value": v})
	}

	origins := []map[string]interface{}{
		{
			"origin":         storageData.Origin,
			"localStorage":   lsItems,
			"sessionStorage": ssItems,
		},
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{
		"cookies": cookies,
		"origins": origins,
	})
}

// handleContextAddInitScript handles vibium:context.addInitScript — adds a preload script scoped to the user context.
func (r *Router) handleContextAddInitScript(session *BrowserSession, cmd bidiCommand) {
	userContext, _ := cmd.Params["userContext"].(string)
	if userContext == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("userContext is required"))
		return
	}

	script, _ := cmd.Params["script"].(string)
	if script == "" {
		r.sendError(session, cmd.ID, fmt.Errorf("script is required"))
		return
	}

	// Wrap user script as an IIFE
	wrappedScript := fmt.Sprintf("() => { %s }", script)

	params := map[string]interface{}{
		"functionDeclaration": wrappedScript,
		"userContexts":        []string{userContext},
	}

	resp, err := r.sendInternalCommand(session, "script.addPreloadScript", params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		r.sendError(session, cmd.ID, bidiErr)
		return
	}

	var result struct {
		Result struct {
			Script string `json:"script"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("failed to parse addPreloadScript response: %w", err))
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"script": result.Result.Script})
}

// --- Helper functions ---

// getCookiesForContext fetches and normalizes cookies for a user context.
func (r *Router) getCookiesForContext(session *BrowserSession, userContext string) ([]map[string]interface{}, error) {
	params := map[string]interface{}{
		"partition": map[string]interface{}{
			"type":        "storageKey",
			"userContext": userContext,
		},
	}

	resp, err := r.sendInternalCommand(session, "storage.getCookies", params)
	if err != nil {
		return nil, err
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		return nil, bidiErr
	}

	var result struct {
		Result struct {
			Cookies []bidiCookie `json:"cookies"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse getCookies response: %w", err)
	}

	return normalizeCookies(result.Result.Cookies), nil
}

// normalizeCookies converts BiDi cookie objects to plain maps with string values.
func normalizeCookies(bidiCookies []bidiCookie) []map[string]interface{} {
	cookies := make([]map[string]interface{}, 0, len(bidiCookies))
	for _, c := range bidiCookies {
		cookie := map[string]interface{}{
			"name":     c.Name,
			"value":    c.Value.Value,
			"domain":   c.Domain,
			"path":     c.Path,
			"size":     c.Size,
			"httpOnly": c.HTTPOnly,
			"secure":   c.Secure,
			"sameSite": c.SameSite,
		}
		if c.Expiry != nil {
			cookie["expiry"] = c.Expiry
		}
		cookies = append(cookies, cookie)
	}
	return cookies
}

// filterCookiesByURLs filters cookies to only those matching the given URLs.
func filterCookiesByURLs(cookies []map[string]interface{}, urls []string) []map[string]interface{} {
	parsed := make([]*url.URL, 0, len(urls))
	for _, u := range urls {
		if p, err := url.Parse(u); err == nil {
			parsed = append(parsed, p)
		}
	}

	filtered := make([]map[string]interface{}, 0)
	for _, cookie := range cookies {
		domain, _ := cookie["domain"].(string)
		path, _ := cookie["path"].(string)
		for _, u := range parsed {
			if domainMatches(domain, u.Hostname()) && pathMatches(path, u.Path) {
				filtered = append(filtered, cookie)
				break
			}
		}
	}
	return filtered
}

// domainMatches checks if a cookie domain matches a hostname.
// A cookie domain ".example.com" matches "example.com" and "sub.example.com".
func domainMatches(cookieDomain, hostname string) bool {
	if cookieDomain == "" || hostname == "" {
		return false
	}
	// Exact match
	if cookieDomain == hostname {
		return true
	}
	// Leading dot means subdomain match
	if strings.HasPrefix(cookieDomain, ".") {
		bare := cookieDomain[1:]
		return hostname == bare || strings.HasSuffix(hostname, cookieDomain)
	}
	// Cookie domain without dot — also match with dot prefix
	return hostname == cookieDomain || strings.HasSuffix(hostname, "."+cookieDomain)
}

// pathMatches checks if a cookie path matches a URL path.
func pathMatches(cookiePath, urlPath string) bool {
	if cookiePath == "" || cookiePath == "/" {
		return true
	}
	if urlPath == "" {
		urlPath = "/"
	}
	return strings.HasPrefix(urlPath, cookiePath)
}

// findContextForUserContext finds a browsing context (page) in the given user context.
func (r *Router) findContextForUserContext(session *BrowserSession, userContext string) (string, error) {
	resp, err := r.sendInternalCommand(session, "browsingContext.getTree", map[string]interface{}{})
	if err != nil {
		return "", err
	}

	var result struct {
		Result struct {
			Contexts []struct {
				Context     string `json:"context"`
				UserContext string `json:"userContext"`
			} `json:"contexts"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse getTree response: %w", err)
	}

	for _, ctx := range result.Result.Contexts {
		if ctx.UserContext == userContext {
			return ctx.Context, nil
		}
	}

	return "", fmt.Errorf("no browsing context found for user context %s", userContext)
}
