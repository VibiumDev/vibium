package api

import (
	"encoding/json"
	"fmt"
)

// handleClockInstall handles vibium:clock.install — injects the fake clock script.
// Registers it as a preload script so it persists across navigations.
// Options: time (epoch ms to set as initial time), timezone (IANA timezone ID).
func (r *Router) handleClockInstall(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	// Inject into the current page immediately
	_, err = r.evalSimpleScript(session, context, ClockScript)
	if err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("failed to install clock: %w", err))
		return
	}

	// Register as preload script (once per session) so it auto-runs on future navigations
	session.mu.Lock()
	needPreload := session.clockPreloadScriptID == ""
	session.mu.Unlock()

	if needPreload {
		resp, err := r.sendInternalCommand(session, "script.addPreloadScript", map[string]interface{}{
			"functionDeclaration": ClockScript,
			"contexts":            []interface{}{context},
		})
		if err != nil {
			r.sendError(session, cmd.ID, fmt.Errorf("failed to register clock preload: %w", err))
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

		session.mu.Lock()
		session.clockPreloadScriptID = result.Result.Script
		session.mu.Unlock()
	}

	// If initial time is provided, set it
	if timeVal, ok := cmd.Params["time"].(float64); ok {
		_, err = r.evalSimpleScript(session, context,
			fmt.Sprintf("() => { window.__vibiumClock.setSystemTime(%v); return 'ok'; }", timeVal))
		if err != nil {
			r.sendError(session, cmd.ID, fmt.Errorf("failed to set initial time: %w", err))
			return
		}
	}

	// If timezone is provided, override it via BiDi emulation.setTimezoneOverride
	if tz, ok := cmd.Params["timezone"].(string); ok && tz != "" {
		if err := r.setTimezoneOverride(session, context, tz); err != nil {
			r.sendError(session, cmd.ID, fmt.Errorf("failed to set timezone: %w", err))
			return
		}
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// runClockOp evaluates a method call against the page's installed fake clock.
// If the clock was never installed (window.__vibiumClock is undefined) it sends
// a clear, actionable error instead of silently doing nothing — previously the
// thrown ReferenceError was swallowed and the call appeared to succeed while
// having no effect (issues #125, #137).
func (r *Router) runClockOp(session *BrowserSession, cmd bidiCommand, context, opName, call string) {
	res, err := r.evalSimpleScript(session, context,
		fmt.Sprintf("() => { if (!window.__vibiumClock) return 'NOT_INSTALLED'; window.__vibiumClock.%s; return 'ok'; }", call))
	if err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("clock.%s failed: %w", opName, err))
		return
	}
	if res == "NOT_INSTALLED" {
		r.sendError(session, cmd.ID, fmt.Errorf("clock not installed: call clock.install() before clock.%s()", opName))
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// handleClockFastForward handles vibium:clock.fastForward — jump forward N ms, fire due timers once.
func (r *Router) handleClockFastForward(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	ticks, ok := cmd.Params["ticks"].(float64)
	if !ok {
		r.sendError(session, cmd.ID, fmt.Errorf("ticks is required"))
		return
	}

	r.runClockOp(session, cmd, context, "fastForward", fmt.Sprintf("fastForward(%v)", ticks))
}

// handleClockRunFor handles vibium:clock.runFor — advance N ms, fire all callbacks systematically.
func (r *Router) handleClockRunFor(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	ticks, ok := cmd.Params["ticks"].(float64)
	if !ok {
		r.sendError(session, cmd.ID, fmt.Errorf("ticks is required"))
		return
	}

	r.runClockOp(session, cmd, context, "runFor", fmt.Sprintf("runFor(%v)", ticks))
}

// handleClockPauseAt handles vibium:clock.pauseAt — jump to a time and pause.
func (r *Router) handleClockPauseAt(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	time, ok := cmd.Params["time"].(float64)
	if !ok {
		r.sendError(session, cmd.ID, fmt.Errorf("time is required"))
		return
	}

	r.runClockOp(session, cmd, context, "pauseAt", fmt.Sprintf("pauseAt(%v)", time))
}

// handleClockResume handles vibium:clock.resume — resume real-time progression.
func (r *Router) handleClockResume(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.runClockOp(session, cmd, context, "resume", "resume()")
}

// handleClockSetFixedTime handles vibium:clock.setFixedTime — freeze Date.now() at a value.
func (r *Router) handleClockSetFixedTime(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	time, ok := cmd.Params["time"].(float64)
	if !ok {
		r.sendError(session, cmd.ID, fmt.Errorf("time is required"))
		return
	}

	r.runClockOp(session, cmd, context, "setFixedTime", fmt.Sprintf("setFixedTime(%v)", time))
}

// handleClockSetSystemTime handles vibium:clock.setSystemTime — set Date.now() without firing timers.
func (r *Router) handleClockSetSystemTime(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	time, ok := cmd.Params["time"].(float64)
	if !ok {
		r.sendError(session, cmd.ID, fmt.Errorf("time is required"))
		return
	}

	r.runClockOp(session, cmd, context, "setSystemTime", fmt.Sprintf("setSystemTime(%v)", time))
}

// handleClockSetTimezone handles vibium:clock.setTimezone — override or reset the browser timezone.
// Pass timezone as an IANA timezone ID (e.g. "America/New_York"), or empty string to reset.
func (r *Router) handleClockSetTimezone(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	tz, _ := cmd.Params["timezone"].(string)

	if tz == "" {
		// Reset to default — pass null for timezone
		if err := r.clearTimezoneOverride(session, context); err != nil {
			r.sendError(session, cmd.ID, fmt.Errorf("failed to clear timezone: %w", err))
			return
		}
	} else {
		if err := r.setTimezoneOverride(session, context, tz); err != nil {
			r.sendError(session, cmd.ID, fmt.Errorf("failed to set timezone: %w", err))
			return
		}
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{})
}

// ---------------------------------------------------------------------------
// Exported standalone clock/timezone functions — usable from both proxy and MCP.
// ---------------------------------------------------------------------------

// SetTimezone overrides the browser timezone via BiDi emulation.setTimezoneOverride.
func SetTimezone(s Session, context, timezone string) error {
	resp, err := s.SendBidiCommand("emulation.setTimezoneOverride", map[string]interface{}{
		"timezone": timezone,
		"contexts": []interface{}{context},
	})
	if err != nil {
		return err
	}
	return checkBidiError(resp)
}

// ClearTimezone resets the browser timezone to the system default.
func ClearTimezone(s Session, context string) error {
	resp, err := s.SendBidiCommand("emulation.setTimezoneOverride", map[string]interface{}{
		"timezone": nil,
		"contexts": []interface{}{context},
	})
	if err != nil {
		return err
	}
	return checkBidiError(resp)
}

// setTimezoneOverride uses BiDi emulation.setTimezoneOverride to set the browser timezone.
func (r *Router) setTimezoneOverride(session *BrowserSession, context string, timezone string) error {
	resp, err := r.sendInternalCommand(session, "emulation.setTimezoneOverride", map[string]interface{}{
		"timezone": timezone,
		"contexts": []interface{}{context},
	})
	if err != nil {
		return err
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		return bidiErr
	}
	return nil
}

// clearTimezoneOverride resets the browser timezone to the system default.
func (r *Router) clearTimezoneOverride(session *BrowserSession, context string) error {
	resp, err := r.sendInternalCommand(session, "emulation.setTimezoneOverride", map[string]interface{}{
		"timezone": nil,
		"contexts": []interface{}{context},
	})
	if err != nil {
		return err
	}
	if bidiErr := checkBidiError(resp); bidiErr != nil {
		return bidiErr
	}
	return nil
}
