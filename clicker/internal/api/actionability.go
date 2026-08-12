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
	CheckReceivesEvents                    // elementFromPoint at the in-view center hits element or descendant
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
	// Point is the in-view center the receivesEvents hit test probed, null when
	// that check did not run or nothing of the element is on screen.
	Point *PointInfo `json:"point,omitempty"`
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
			` + PierceQueryJS() + `
			const root = scope ? pierceQuery(document, scope) : document;
			if (!root) return JSON.stringify({status:'not_found'});
			let el;
			if (hasIndex) {
				const all = pierceQueryAll(root, selector);
				el = all[index];
			} else {
				el = pierceQuery(root, selector);
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
				box: { x: rect.x, y: rect.y, width: rect.width, height: rect.height },
				point: inView
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
			` + PierceQueryJS() + `
			const root = scope ? pierceQuery(document, scope) : document;
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
				box: { x: rect.x, y: rect.y, width: rect.width, height: rect.height },
				point: inView
			});
		}
	`
	return script, args
}

// actionabilityCheckBody returns the shared JS check body appended after
// element finding + scroll. Uses the boolean flags passed as arguments.
// Stability is NOT checked here — it's handled on the Go side via two calls.
// FillableInputTypesJS is a JS array literal of the input types whose value
// fill can set: text-like inputs plus the value-bearing pickers. Excludes
// checkbox/radio (use check), file (use upload) and button/submit/reset/image.
//
// One definition, because the action path, `vibium is actionable` and
// element.isEditable each had their own copy and they had already drifted —
// fill worked on input[type=range] while `is actionable` called it not
// editable (#264).
const FillableInputTypesJS = `['text','password','email','number','search','tel','url',` +
	`'range','color','date','time','datetime-local','month','week']`

// EditablePredicateJS is a JS expression, evaluated with `el` in scope, that is
// true when fill can set this element's value.
const EditablePredicateJS = `(!el.disabled && !el.readOnly && el.getAttribute('aria-readonly') !== 'true' && (
	el.tagName.toLowerCase() === 'input'
		? ` + FillableInputTypesJS + `.includes((el.type || 'text').toLowerCase())
		: (el.tagName.toLowerCase() === 'textarea' || el.isContentEditable)
))`

// inViewCenterJS declares inViewCenter(rect): the WebDriver in-view center
// point — the center of the element rect clipped to the viewport, floored to
// whole pixels — or null when the clipped rect is empty.
//
// Probing the *full* rect center made every element more than twice the viewport
// tall or wide permanently unclickable: the center landed off-screen, where
// document.elementFromPoint is defined to return null, and the check blamed an
// obstruction that did not exist (#340). Clipping is what chromedriver's Element
// Click does, so a full-page scrim behaves the same through either driver.
//
// Flooring matters: BiDi input coordinates are integers, so the point returned
// here is also the point the action dispatches at (see InputPoint).
const inViewCenterJS = `
			function inViewCenter(rect) {
				const left   = Math.max(0, rect.x);
				const right  = Math.min(window.innerWidth, rect.x + rect.width);
				const top    = Math.max(0, rect.y);
				const bottom = Math.min(window.innerHeight, rect.y + rect.height);
				if (right <= left || bottom <= top) return null;
				return { x: Math.floor((left + right) / 2), y: Math.floor((top + bottom) / 2) };
			}
`

func actionabilityCheckBody() string {
	return inViewCenterJS + `
			// Only the receivesEvents check needs a hit-test point, and only it
			// reports one — paths without it (fill, select, scroll) keep
			// dispatching at the bounding-box center as before.
			const inView = chkEvents ? inViewCenter(rect) : null;
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
					const fillableTypes = ` + FillableInputTypesJS + `;
					if (!fillableTypes.includes(t))
						return JSON.stringify({status:'failed', check:'editable', reason:'input type ' + t + ' not editable'});
				} else if (tag !== 'textarea' && !el.isContentEditable) {
					return JSON.stringify({status:'failed', check:'editable', reason:'not a text input element'});
				}
			}
			if (chkEvents) {
				// Nothing on screen to click. Distinct from "obscured": no
				// element is on top, and scrolling did not bring this one in.
				if (!inView)
					return JSON.stringify({status:'failed', check:'receivesEvents', reason:'element has no visible area in the viewport'});
				const cx = inView.x, cy = inView.y;
				// document.elementFromPoint stops at a shadow host, so an element
				// inside a shadow root always looked obscured. Descend through
				// each root at the same point to find what is really on top.
				let hit = document.elementFromPoint(cx, cy);
				while (hit && hit.shadowRoot) {
					const inner = hit.shadowRoot.elementFromPoint(cx, cy);
					if (!inner || inner === hit) break;
					hit = inner;
				}
				const at = ' at in-view center (' + cx + ', ' + cy + ')';
				// Separate from the obscured branch: a null hit is a hit-test
				// fault, not an obstruction, and reporting it as one sent a real
				// investigation looking for an overlay that was not there (#340).
				if (!hit)
					return JSON.stringify({status:'failed', check:'receivesEvents', reason:'no element' + at});
				if (el !== hit && !el.contains(hit)) {
					const blocker = hit.tagName.toLowerCase() + (hit.id ? '#' + hit.id : '');
					return JSON.stringify({status:'failed', check:'receivesEvents', reason:'element is obscured by <' + blocker + '>' + at});
				}
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
							return &ElementInfo{Tag: result2.Tag, Text: result2.Text, Box: result2.Box, Point: result2.Point}, nil
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
					return &ElementInfo{Tag: result.Tag, Text: result.Text, Box: result.Box, Point: result.Point}, nil
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
