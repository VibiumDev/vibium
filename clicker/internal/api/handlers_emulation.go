package api

import (
	"encoding/json"
	"fmt"
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

	s := NewAPISession(r, session, context)
	if err := EmulateMedia(s, context, overrides); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// emulateMediaScript is the JS that installs/updates matchMedia overrides.
// The override wraps native matchMedia once (idempotent) and intercepts
// queries for configured CSS media features.
const emulateMediaScript = "(overridesJSON) => {\n" +
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

// EmulateMedia overrides CSS media features in the browser via a JS matchMedia override.
// The overrides map can contain keys: media, colorScheme, reducedMotion, forcedColors, contrast.
// Values can be strings (to override) or nil (to reset).
func EmulateMedia(s Session, context string, overrides map[string]interface{}) error {
	overridesJSON, err := json.Marshal(overrides)
	if err != nil {
		return fmt.Errorf("failed to serialize overrides: %w", err)
	}

	resp, err := s.SendBidiCommand("script.callFunction", map[string]interface{}{
		"functionDeclaration": emulateMediaScript,
		"target":              map[string]interface{}{"context": context},
		"arguments": []map[string]interface{}{
			{"type": "string", "value": string(overridesJSON)},
		},
		"awaitPromise":    false,
		"resultOwnership": "root",
	})
	if err != nil {
		return err
	}
	return checkBidiError(resp)
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
func (r *Router) handlePageSetWindow(session *BrowserSession, cmd bidiCommand) {
	state, _ := cmd.Params["state"].(string)
	width, hasWidth := cmd.Params["width"].(float64)
	height, hasHeight := cmd.Params["height"].(float64)
	x, hasX := cmd.Params["x"].(float64)
	y, hasY := cmd.Params["y"].(float64)

	opts := SetWindowOpts{State: state}
	if hasWidth {
		w := int(width)
		opts.Width = &w
	}
	if hasHeight {
		h := int(height)
		opts.Height = &h
	}
	if hasX {
		xv := int(x)
		opts.X = &xv
	}
	if hasY {
		yv := int(y)
		opts.Y = &yv
	}

	if err := SetWindow(NewAPISession(r, session, ""), opts); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// handlePageWindow handles vibium:page.window — returns the current OS window state and dimensions.
func (r *Router) handlePageWindow(session *BrowserSession, cmd bidiCommand) {
	s := NewAPISession(r, session, "")
	win, err := GetWindow(s)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{
		"state":  win.State,
		"x":      win.X,
		"y":      win.Y,
		"width":  win.Width,
		"height": win.Height,
	})
}

// handlePageSetGeolocation handles vibium:page.setGeolocation — overrides geolocation.
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

	s := NewAPISession(r, session, context)
	if err := SetGeolocation(s, context, lat, lng, accuracy); err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// ---------------------------------------------------------------------------
// Exported standalone functions — usable from both proxy and MCP.
// ---------------------------------------------------------------------------

// WindowInfo holds OS browser window state and dimensions.
type WindowInfo struct {
	State  string `json:"state"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// GetWindow returns the current OS browser window state and dimensions.
// Uses BiDi browser.getClientWindows.
func GetWindow(s Session) (*WindowInfo, error) {
	win, _, err := activeClientWindow(s)
	return win, err
}

// activeClientWindow returns the focused client window (or the first one when
// none reports focus) along with its BiDi client window id.
func activeClientWindow(s Session) (*WindowInfo, string, error) {
	resp, err := s.SendBidiCommand("browser.getClientWindows", map[string]interface{}{})
	if err != nil {
		return nil, "", fmt.Errorf("failed to get window: %w", err)
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		return nil, "", bidiErr
	}

	var getResult struct {
		Result struct {
			ClientWindows []struct {
				WindowInfo
				ClientWindow string `json:"clientWindow"`
				Active       bool   `json:"active"`
			} `json:"clientWindows"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &getResult); err != nil {
		return nil, "", fmt.Errorf("failed to parse getClientWindows: %w", err)
	}
	windows := getResult.Result.ClientWindows
	if len(windows) == 0 {
		return nil, "", fmt.Errorf("no client windows available")
	}
	chosen := windows[0]
	for _, win := range windows {
		if win.Active {
			chosen = win
			break
		}
	}
	return &chosen.WindowInfo, chosen.ClientWindow, nil
}

// SetWindowOpts specifies the desired window state and/or dimensions.
type SetWindowOpts struct {
	State  string // "maximized", "minimized", "fullscreen", "normal", or ""
	X      *int
	Y      *int
	Width  *int
	Height *int
}

// SetWindow sets the OS browser window size, position, or state.
// Uses BiDi browser.setClientWindowState: the classic WebDriver endpoint
// operates on the session's current window handle, which goes stale once
// the page it points at is closed.
func SetWindow(s Session, opts SetWindowOpts) error {
	_, id, err := activeClientWindow(s)
	if err != nil {
		return err
	}

	params := map[string]interface{}{"clientWindow": id}
	switch opts.State {
	case "maximized", "minimized", "fullscreen":
		params["state"] = opts.State
	case "", "normal":
		params["state"] = "normal"
		if opts.Width != nil {
			params["width"] = *opts.Width
		}
		if opts.Height != nil {
			params["height"] = *opts.Height
		}
		if opts.X != nil {
			params["x"] = *opts.X
		}
		if opts.Y != nil {
			params["y"] = *opts.Y
		}
	default:
		return fmt.Errorf("unsupported window state: %s", opts.State)
	}

	resp, err := s.SendBidiCommand("browser.setClientWindowState", params)
	if err != nil {
		return fmt.Errorf("failed to set window: %w", err)
	}
	return checkBidiError(resp)
}

// ViewportCenter returns the viewport's center point. Pointer actions with a
// fixed origin break as soon as the viewport is not the size the constant
// assumed: Firefox rejects out-of-bounds coordinates outright, and on larger
// viewports a fixed point can land inside whatever scrollable element happens
// to cover it (#443, #444).
func ViewportCenter(s Session, context string) (int, int, error) {
	resp, err := CallScript(s, context,
		`() => JSON.stringify({ width: window.innerWidth, height: window.innerHeight })`,
		[]map[string]interface{}{})
	if err != nil {
		return 0, 0, err
	}
	val, err := parseScriptResult(resp)
	if err != nil {
		return 0, 0, err
	}
	var size struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}
	if err := json.Unmarshal([]byte(val), &size); err != nil {
		return 0, 0, fmt.Errorf("failed to parse viewport: %w", err)
	}
	return size.Width / 2, size.Height / 2, nil
}

// geolocationScript returns the JS that overrides navigator.geolocation with
// the given coordinates. The coordinates are baked into the declaration
// because addPreloadScript passes no arguments to its function.
//
// The script is written to run any number of times in a document: the first
// run installs an override that reads the coordinates from a window slot at
// query time, later runs only update the slot. Repeated setGeolocation calls
// stack one preload script each, and on a new document they all run in the
// order they were added, so the last call wins without any script-removal
// bookkeeping.
func geolocationScript(coordsJSON string) string {
	return "() => {\n" +
		"window.__vibiumGeoCoords = " + coordsJSON + ";\n" +
		"if (window.__vibiumGeoInstalled) return 'ok';\n" +
		"window.__vibiumGeoInstalled = true;\n" +
		"const pos = () => ({ coords: { latitude: window.__vibiumGeoCoords.latitude,\n" +
		"  longitude: window.__vibiumGeoCoords.longitude, accuracy: window.__vibiumGeoCoords.accuracy,\n" +
		"  altitude: null, altitudeAccuracy: null, heading: null, speed: null }, timestamp: Date.now() });\n" +
		"const geo = navigator.geolocation;\n" +
		"geo.getCurrentPosition = function(success, error, options) { success(pos()); };\n" +
		"geo.watchPosition = function(success, error, options) { success(pos()); return 0; };\n" +
		"return 'ok';\n" +
		"}"
}

// SetGeolocation overrides the browser geolocation via a JS override, for the
// current document and every later document in the context.
func SetGeolocation(s Session, context string, lat, lon, accuracy float64) error {
	coordsJSON, _ := json.Marshal(map[string]float64{
		"latitude":  lat,
		"longitude": lon,
		"accuracy":  accuracy,
	})
	decl := geolocationScript(string(coordsJSON))

	resp, err := s.SendBidiCommand("script.callFunction", map[string]interface{}{
		"functionDeclaration": decl,
		"target":              map[string]interface{}{"context": context},
		"awaitPromise":        false,
		"resultOwnership":     "root",
	})
	if err != nil {
		return err
	}
	if err := checkBidiError(resp); err != nil {
		return err
	}

	// The call above dies with the document, so a navigation or reload
	// silently dropped the override (#345). Register the same script as a
	// preload so every new document in this context gets it re-applied.
	resp, err = s.SendBidiCommand("script.addPreloadScript", map[string]interface{}{
		"functionDeclaration": decl,
		"contexts":            []interface{}{context},
	})
	if err != nil {
		return err
	}
	return checkBidiError(resp)
}
