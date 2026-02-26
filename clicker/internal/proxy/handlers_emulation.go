package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// handlePageSetViewport handles vibium:page.setViewport — sets the viewport size.
// Uses BiDi browsingContext.setViewport.
func (r *Router) handlePageSetViewport(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	width, _ := cmd.Params["width"].(float64)
	height, _ := cmd.Params["height"].(float64)
	if width == 0 || height == 0 {
		r.sendError(session, cmd.ID, fmt.Errorf("width and height are required"))
		return
	}

	params := map[string]interface{}{
		"context": context,
		"viewport": map[string]interface{}{
			"width":  int(width),
			"height": int(height),
		},
	}

	if dpr, ok := cmd.Params["devicePixelRatio"].(float64); ok && dpr > 0 {
		params["devicePixelRatio"] = dpr
	}

	resp, err := r.sendInternalCommand(session, "browsingContext.setViewport", params)
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

// handlePageViewport handles vibium:page.viewport — returns the current viewport size.
// Uses JS eval since BiDi has no viewport getter.
func (r *Router) handlePageViewport(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	val, err := r.evalSimpleScript(session, context,
		`() => JSON.stringify({ width: window.innerWidth, height: window.innerHeight })`)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	var size struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}
	if err := json.Unmarshal([]byte(val), &size); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("failed to parse viewport: %w", err))
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{
		"width":  size.Width,
		"height": size.Height,
	})
}

// handlePageEmulateMedia handles vibium:page.emulateMedia — overrides CSS media features.
// Uses JS matchMedia override since BiDi has no CSS media feature commands.
// Supports: media, colorScheme, reducedMotion, forcedColors, contrast.
func (r *Router) handlePageEmulateMedia(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Build the overrides object from params.
	// Each option can be a string value or null (to reset).
	overrides := map[string]interface{}{}

	for _, key := range []string{"media", "colorScheme", "reducedMotion", "forcedColors", "contrast"} {
		if val, exists := cmd.Params[key]; exists {
			if val == nil {
				overrides[key] = nil
			} else if s, ok := val.(string); ok {
				overrides[key] = s
			}
		}
	}

	overridesJSON, err := json.Marshal(overrides)
	if err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("failed to serialize overrides: %w", err))
		return
	}

	// JS function that installs/updates matchMedia overrides.
	// The override wraps native matchMedia once (idempotent) and intercepts
	// queries for configured CSS media features.
	// NOTE: In Go backtick strings, characters are literal — so \( in the JS
	// source must be written as \( here (no extra escaping).
	script := "(overridesJSON) => {\n" +
		"const overrides = JSON.parse(overridesJSON);\n" +
		"if (!window.__vibiumMediaOverrides) { window.__vibiumMediaOverrides = {}; }\n" +
		"const featureMap = {\n" +
		"  colorScheme: 'prefers-color-scheme',\n" +
		"  reducedMotion: 'prefers-reduced-motion',\n" +
		"  forcedColors: 'forced-colors',\n" +
		"  contrast: 'prefers-contrast'\n" +
		"};\n" +
		"for (const [key, value] of Object.entries(overrides)) {\n" +
		"  if (value === null) { delete window.__vibiumMediaOverrides[key]; }\n" +
		"  else { window.__vibiumMediaOverrides[key] = value; }\n" +
		"}\n" +
		"if (!window.__vibiumOriginalMatchMedia) {\n" +
		"  window.__vibiumOriginalMatchMedia = window.matchMedia.bind(window);\n" +
		"  window.matchMedia = function(query) {\n" +
		"    const original = window.__vibiumOriginalMatchMedia(query);\n" +
		"    const ov = window.__vibiumMediaOverrides || {};\n" +
		"    if (ov.media !== undefined) {\n" +
		"      const q = query.trim().toLowerCase();\n" +
		"      if (q === 'print' || q === '(print)') return makeResult(original, ov.media === 'print', query);\n" +
		"      if (q === 'screen' || q === '(screen)') return makeResult(original, ov.media === 'screen', query);\n" +
		"    }\n" +
		"    for (const [key, feature] of Object.entries(featureMap)) {\n" +
		"      if (ov[key] !== undefined) {\n" +
		"        const re = new RegExp('\\\\(' + feature + '\\\\s*:\\\\s*([^)]+)\\\\)');\n" +
		"        const m = query.match(re);\n" +
		"        if (m) { return makeResult(original, m[1].trim() === ov[key], query); }\n" +
		"      }\n" +
		"    }\n" +
		"    return original;\n" +
		"  };\n" +
		"}\n" +
		"function makeResult(original, matches, media) {\n" +
		"  return {\n" +
		"    matches: matches, media: media, onchange: original.onchange,\n" +
		"    addListener: original.addListener.bind(original),\n" +
		"    removeListener: original.removeListener.bind(original),\n" +
		"    addEventListener: original.addEventListener.bind(original),\n" +
		"    removeEventListener: original.removeEventListener.bind(original),\n" +
		"    dispatchEvent: original.dispatchEvent.bind(original)\n" +
		"  };\n" +
		"}\n" +
		"return 'ok';\n" +
		"}"

	params := map[string]interface{}{
		"functionDeclaration": script,
		"target":              map[string]interface{}{"context": context},
		"arguments": []map[string]interface{}{
			{"type": "string", "value": string(overridesJSON)},
		},
		"awaitPromise":    false,
		"resultOwnership": "root",
	}

	resp, err := r.sendInternalCommand(session, "script.callFunction", params)
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

// handlePageSetContent handles vibium:page.setContent — replaces the page HTML.
// Uses document.open/write/close to fully replace the document.
func (r *Router) handlePageSetContent(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	html, _ := cmd.Params["html"].(string)

	script := `(html) => { document.open(); document.write(html); document.close(); }`

	params := map[string]interface{}{
		"functionDeclaration": script,
		"target":              map[string]interface{}{"context": context},
		"arguments": []map[string]interface{}{
			{"type": "string", "value": html},
		},
		"awaitPromise":    true,
		"resultOwnership": "root",
	}

	resp, err := r.sendInternalCommand(session, "script.callFunction", params)
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

// handlePageSetWindow handles vibium:page.setWindow — sets the OS browser window size, position, or state.
// Uses chromedriver's classic WebDriver HTTP API since BiDi browser.setClientWindowState
// is not yet available in chromedriver 145.
func (r *Router) handlePageSetWindow(session *BrowserSession, cmd bidiCommand) {
	state, hasState := cmd.Params["state"].(string)
	width, hasWidth := cmd.Params["width"].(float64)
	height, hasHeight := cmd.Params["height"].(float64)
	x, hasX := cmd.Params["x"].(float64)
	y, hasY := cmd.Params["y"].(float64)

	baseURL := fmt.Sprintf("http://localhost:%d/session/%s/window", session.LaunchResult.Port, session.LaunchResult.SessionID)

	// Handle named states (maximize, minimize, fullscreen) via dedicated endpoints
	if hasState && state != "normal" {
		endpoint := ""
		switch state {
		case "maximized":
			endpoint = baseURL + "/maximize"
		case "minimized":
			endpoint = baseURL + "/minimize"
		case "fullscreen":
			endpoint = baseURL + "/fullscreen"
		default:
			r.sendError(session, cmd.ID, fmt.Errorf("unsupported window state: %s", state))
			return
		}

		if err := r.chromedriverPost(endpoint, map[string]interface{}{}); err != nil {
			r.sendError(session, cmd.ID, err)
			return
		}
		r.sendSuccess(session, cmd.ID, map[string]interface{}{})
		return
	}

	// For "normal" state or dimension changes, use /window/rect
	rect := map[string]interface{}{}
	if hasWidth {
		rect["width"] = int(width)
	}
	if hasHeight {
		rect["height"] = int(height)
	}
	if hasX {
		rect["x"] = int(x)
	}
	if hasY {
		rect["y"] = int(y)
	}

	if err := r.chromedriverPost(baseURL+"/rect", rect); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// chromedriverPost sends a POST request to a chromedriver classic WebDriver endpoint.
func (r *Router) chromedriverPost(url string, body map[string]interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("chromedriver request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("chromedriver error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// handlePageWindow handles vibium:page.window — returns the current OS window state and dimensions.
// Uses BiDi browser.getClientWindows.
func (r *Router) handlePageWindow(session *BrowserSession, cmd bidiCommand) {
	resp, err := r.sendInternalCommand(session, "browser.getClientWindows", map[string]interface{}{})
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		r.sendError(session, cmd.ID, bidiErr)
		return
	}

	var getResult struct {
		Result struct {
			ClientWindows []struct {
				State  string `json:"state"`
				X      int    `json:"x"`
				Y      int    `json:"y"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"clientWindows"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &getResult); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("failed to parse getClientWindows: %w", err))
		return
	}
	if len(getResult.Result.ClientWindows) == 0 {
		r.sendError(session, cmd.ID, fmt.Errorf("no client windows available"))
		return
	}

	win := getResult.Result.ClientWindows[0]
	r.sendSuccess(session, cmd.ID, map[string]interface{}{
		"state":  win.State,
		"x":      win.X,
		"y":      win.Y,
		"width":  win.Width,
		"height": win.Height,
	})
}

// handlePageSetGeolocation handles vibium:page.setGeolocation — overrides geolocation.
// Uses JS override of navigator.geolocation since BiDi emulation.setGeolocationOverride
// is not widely supported and requires granted permissions.
func (r *Router) handlePageSetGeolocation(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	lat, hasLat := cmd.Params["latitude"].(float64)
	lng, hasLng := cmd.Params["longitude"].(float64)

	if !hasLat || !hasLng {
		r.sendError(session, cmd.ID, fmt.Errorf("latitude and longitude are required"))
		return
	}

	accuracy := float64(1)
	if acc, ok := cmd.Params["accuracy"].(float64); ok {
		accuracy = acc
	}

	coordsJSON, _ := json.Marshal(map[string]float64{
		"latitude":  lat,
		"longitude": lng,
		"accuracy":  accuracy,
	})

	script := "(coordsJSON) => {\n" +
		"const coords = JSON.parse(coordsJSON);\n" +
		"const geo = navigator.geolocation;\n" +
		"geo.getCurrentPosition = function(success, error, options) {\n" +
		"  success({ coords: { latitude: coords.latitude, longitude: coords.longitude, accuracy: coords.accuracy,\n" +
		"    altitude: null, altitudeAccuracy: null, heading: null, speed: null }, timestamp: Date.now() });\n" +
		"};\n" +
		"geo.watchPosition = function(success, error, options) {\n" +
		"  success({ coords: { latitude: coords.latitude, longitude: coords.longitude, accuracy: coords.accuracy,\n" +
		"    altitude: null, altitudeAccuracy: null, heading: null, speed: null }, timestamp: Date.now() });\n" +
		"  return 0;\n" +
		"};\n" +
		"return 'ok';\n" +
		"}"

	params := map[string]interface{}{
		"functionDeclaration": script,
		"target":              map[string]interface{}{"context": context},
		"arguments": []map[string]interface{}{
			{"type": "string", "value": string(coordsJSON)},
		},
		"awaitPromise":    false,
		"resultOwnership": "root",
	}

	resp, err := r.sendInternalCommand(session, "script.callFunction", params)
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
