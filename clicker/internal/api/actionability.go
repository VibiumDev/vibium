package api

import (
	"encoding/json"
	"fmt"
	"time"
)

// ActionCheck represents a specific actionability check.
type ActionCheck int

const (
	CheckVisible        ActionCheck = iota // non-zero bbox, not display:none/visibility:hidden
	CheckStable                            // bbox unchanged after 50ms delay
	CheckReceivesEvents                    // elementFromPoint at center hits element or descendant
	CheckEnabled                           // not [disabled], aria-disabled, or in disabled fieldset
	CheckEditable                          // enabled + not readonly + valid text input type
)

// Check sets matching Playwright's actionability matrix.
var (
	ClickChecks  = []ActionCheck{CheckVisible, CheckStable, CheckReceivesEvents, CheckEnabled}
	HoverChecks  = []ActionCheck{CheckVisible, CheckStable, CheckReceivesEvents}
	FillChecks   = []ActionCheck{CheckVisible, CheckEnabled, CheckEditable}
	SelectChecks = []ActionCheck{CheckVisible, CheckEnabled}
	ScrollChecks = []ActionCheck{CheckStable}
)

// actionableResult is the JSON structure returned by the combined actionability script.
type actionableResult struct {
	Status string  `json:"status"` // "ok", "not_found", "failed"
	Check  string  `json:"check,omitempty"`
	Reason string  `json:"reason,omitempty"`
	Tag    string  `json:"tag,omitempty"`
	Text   string  `json:"text,omitempty"`
	Box    BoxInfo `json:"box,omitempty"`
}

// checksContain returns true if the check set includes the given check.
func checksContain(checks []ActionCheck, c ActionCheck) bool {
	for _, ch := range checks {
		if ch == c {
			return true
		}
	}
	return false
}

// buildActionableScript builds a synchronous JS function that finds an element,
// scrolls into view, and runs all applicable actionability checks inline
// (except stability, which is handled on the Go side).
// One BiDi round-trip per call regardless of how many checks are needed.
func buildActionableScript(ep ElementParams, checks []ActionCheck) (string, []map[string]interface{}) {
	checkVisible := checksContain(checks, CheckVisible)
	checkReceivesEvents := checksContain(checks, CheckReceivesEvents)
	checkEnabled := checksContain(checks, CheckEnabled)
	checkEditable := checksContain(checks, CheckEditable)

	hasSemantic := ep.Role != "" || ep.Text != "" || ep.Label != "" || ep.Placeholder != "" ||
		ep.Alt != "" || ep.Title != "" || ep.Testid != "" || ep.Xpath != ""

	if !hasSemantic && ep.Selector != "" {
		return buildCSSActionableScript(ep, checkVisible, checkReceivesEvents, checkEnabled, checkEditable)
	}
	return buildSemanticActionableScript(ep, checkVisible, checkReceivesEvents, checkEnabled, checkEditable)
}

func buildCSSActionableScript(ep ElementParams, checkVisible, checkReceivesEvents, checkEnabled, checkEditable bool) (string, []map[string]interface{}) {
	args := []map[string]interface{}{
		{"type": "string", "value": ep.Scope},
		{"type": "string", "value": ep.Selector},
		{"type": "number", "value": ep.Index},
		{"type": "boolean", "value": ep.HasIndex},
		{"type": "boolean", "value": checkVisible},
		{"type": "boolean", "value": checkReceivesEvents},
		{"type": "boolean", "value": checkEnabled},
		{"type": "boolean", "value": checkEditable},
	}

	script := `
		(scope, selector, index, hasIndex, chkVisible, chkEvents, chkEnabled, chkEditable) => {
			const root = scope ? document.querySelector(scope) : document;
			if (!root) return JSON.stringify({status:'not_found'});
			let el;
			if (hasIndex) {
				const all = root.querySelectorAll(selector);
				el = all[index];
			} else {
				el = root.querySelector(selector);
			}
			if (!el) return JSON.stringify({status:'not_found'});

			if (el.scrollIntoViewIfNeeded) {
				el.scrollIntoViewIfNeeded(true);
			} else {
				el.scrollIntoView({ block: 'center', inline: 'nearest' });
			}
			const rect = el.getBoundingClientRect();
` + actionabilityCheckBody() + `
			return JSON.stringify({
				status:'ok',
				tag: el.tagName.toLowerCase(),
				text: (el.innerText || '').trim(),
				box: { x: rect.x, y: rect.y, width: rect.width, height: rect.height }
			});
		}
	`
	return script, args
}

func buildSemanticActionableScript(ep ElementParams, checkVisible, checkReceivesEvents, checkEnabled, checkEditable bool) (string, []map[string]interface{}) {
	args := []map[string]interface{}{
		{"type": "string", "value": ep.Scope},
		{"type": "string", "value": ep.Selector},
		{"type": "string", "value": ep.Role},
		{"type": "string", "value": ep.Text},
		{"type": "string", "value": ep.Label},
		{"type": "string", "value": ep.Placeholder},
		{"type": "string", "value": ep.Alt},
		{"type": "string", "value": ep.Title},
		{"type": "string", "value": ep.Testid},
		{"type": "string", "value": ep.Xpath},
		{"type": "number", "value": ep.Index},
		{"type": "boolean", "value": ep.HasIndex},
		{"type": "boolean", "value": checkVisible},
		{"type": "boolean", "value": checkReceivesEvents},
		{"type": "boolean", "value": checkEnabled},
		{"type": "boolean", "value": checkEditable},
	}

	script := `
		(scope, selector, role, text, label, placeholder, alt, title, testid, xpath, index, hasIndex, chkVisible, chkEvents, chkEnabled, chkEditable) => {
			const root = scope ? document.querySelector(scope) : document;
			if (!root) return JSON.stringify({status:'not_found'});
	` + semanticMatchesHelper() + `
			const found = collectMatches(root, selector, role, text, label, placeholder, alt, title, testid, xpath);
			let el;
			if (hasIndex) {
				el = found[index];
			} else {
				el = pickBest(found, text);
			}
			if (!el) return JSON.stringify({status:'not_found'});

			if (el.scrollIntoViewIfNeeded) {
				el.scrollIntoViewIfNeeded(true);
			} else {
				el.scrollIntoView({ block: 'center', inline: 'nearest' });
			}
			const rect = el.getBoundingClientRect();
` + actionabilityCheckBody() + `
			return JSON.stringify({
				status:'ok',
				tag: el.tagName.toLowerCase(),
				text: (el.innerText || '').trim(),
				box: { x: rect.x, y: rect.y, width: rect.width, height: rect.height }
			});
		}
	`
	return script, args
}

// actionabilityCheckBody returns the shared JS check body appended after
// element finding + scroll. Uses the boolean flags passed as arguments.
// Stability is NOT checked here — it's handled on the Go side via two calls.
func actionabilityCheckBody() string {
	return `
			if (chkVisible) {
				if (rect.width === 0 || rect.height === 0)
					return JSON.stringify({status:'failed', check:'visible', reason:'zero size'});
				const style = window.getComputedStyle(el);
				if (style.visibility === 'hidden')
					return JSON.stringify({status:'failed', check:'visible', reason:'visibility: hidden'});
				if (style.display === 'none')
					return JSON.stringify({status:'failed', check:'visible', reason:'display: none'});
			}
			if (chkEnabled) {
				if (el.disabled === true)
					return JSON.stringify({status:'failed', check:'enabled', reason:'disabled attribute'});
				if (el.getAttribute('aria-disabled') === 'true')
					return JSON.stringify({status:'failed', check:'enabled', reason:'aria-disabled'});
				const fs = el.closest('fieldset[disabled]');
				if (fs) {
					const legend = fs.querySelector('legend');
					if (!legend || !legend.contains(el))
						return JSON.stringify({status:'failed', check:'enabled', reason:'inside disabled fieldset'});
				}
			}
			if (chkEditable) {
				if (el.readOnly === true)
					return JSON.stringify({status:'failed', check:'editable', reason:'readonly attribute'});
				if (el.getAttribute('aria-readonly') === 'true')
					return JSON.stringify({status:'failed', check:'editable', reason:'aria-readonly'});
				const tag = el.tagName.toLowerCase();
				if (tag === 'input') {
					const t = (el.type || 'text').toLowerCase();
					// Types whose value can be set via fill. Includes the value-bearing
					// picker types (range/color/date/time family), not just text inputs.
					// Excludes checkbox/radio (use check), file (use upload), and
					// button/submit/reset/image (not value inputs).
					const fillableTypes = ['text','password','email','number','search','tel','url',
						'range','color','date','time','datetime-local','month','week'];
					if (!fillableTypes.includes(t))
						return JSON.stringify({status:'failed', check:'editable', reason:'input type ' + t + ' not editable'});
				} else if (tag !== 'textarea' && !el.isContentEditable) {
					return JSON.stringify({status:'failed', check:'editable', reason:'not a text input element'});
				}
			}
			if (chkEvents) {
				const cx = rect.x + rect.width/2, cy = rect.y + rect.height/2;
				const hit = document.elementFromPoint(cx, cy);
				if (!hit || (el !== hit && !el.contains(hit)))
					return JSON.stringify({status:'failed', check:'receivesEvents', reason:'element is obscured'});
			}
`
}

// callActionableScript runs the combined actionability script (synchronous,
// awaitPromise: false) and returns the parsed result.
func callActionableScript(s Session, context, script string, args []map[string]interface{}) (*actionableResult, error) {
	resp, err := CallScript(s, context, script, args)
	if err != nil {
		return nil, err
	}

	val, err := parseScriptResult(resp)
	if err != nil {
		return nil, err
	}

	var result actionableResult
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, fmt.Errorf("failed to parse actionability result: %w", err)
	}
	return &result, nil
}

// WaitForActionable polls until the element is found and passes all actionability checks,
// or times out. Returns the element info on success.
//
// Stability is checked on the Go side: after all JS-side checks pass, we sleep 50ms
// and re-run the script to compare bounding boxes. This avoids needing awaitPromise: true.
func WaitForActionable(s Session, context string, ep ElementParams, checks []ActionCheck) (*ElementInfo, error) {
	ep = ep.withDefaultTimeout()
	needStable := checksContain(checks, CheckStable)
	// Build script without stability (handled on Go side)
	checksWithoutStable := make([]ActionCheck, 0, len(checks))
	for _, c := range checks {
		if c != CheckStable {
			checksWithoutStable = append(checksWithoutStable, c)
		}
	}
	script, args := buildActionableScript(ep, checksWithoutStable)

	deadline := time.Now().Add(ep.Timeout)
	interval := 100 * time.Millisecond
	var lastResult *actionableResult

	for {
		result, err := callActionableScript(s, context, script, args)
		if err == nil {
			lastResult = result
			if result.Status == "ok" {
				if needStable {
					// Check stability: sleep 50ms, re-run, compare bbox
					time.Sleep(50 * time.Millisecond)
					result2, err2 := callActionableScript(s, context, script, args)
					if err2 == nil && result2.Status == "ok" {
						if result.Box == result2.Box {
							return &ElementInfo{Tag: result2.Tag, Text: result2.Text, Box: result2.Box}, nil
						}
						// Not stable — set lastResult to indicate instability and retry
						lastResult = &actionableResult{
							Status: "failed",
							Check:  "stable",
							Reason: "element is moving or resizing",
						}
					}
					// If second call failed, retry the whole loop
				} else {
					return &ElementInfo{Tag: result.Tag, Text: result.Text, Box: result.Box}, nil
				}
			}
		}

		if time.Now().After(deadline) {
			if lastResult != nil {
				if lastResult.Status == "not_found" {
					return nil, fmt.Errorf("timeout after %s: element not found", ep.Timeout)
				}
				return nil, fmt.Errorf("timeout after %s: %s check failed — %s", ep.Timeout, lastResult.Check, lastResult.Reason)
			}
			return nil, fmt.Errorf("timeout after %s waiting for element", ep.Timeout)
		}

		time.Sleep(interval)
	}
}

// resolveWithActionability resolves an element with actionability checks.
// If Force is set or no checks are needed, falls back to plain ResolveElement.
func resolveWithActionability(s Session, context string, ep ElementParams, checks []ActionCheck) (*ElementInfo, error) {
	// Timeout normalization (auto-wait default) happens in ResolveElement /
	// WaitForActionable, which both branches below funnel through.
	if ep.Force || len(checks) == 0 {
		return ResolveElement(s, context, ep)
	}
	info, err := WaitForActionable(s, context, ep, checks)
	if err == nil && info != nil {
		s.SetLastElementBox(&info.Box)
	}
	return info, err
}
