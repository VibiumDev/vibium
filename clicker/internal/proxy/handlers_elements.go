package proxy

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// elementInfo holds parsed element information.
type elementInfo struct {
	Tag  string  `json:"tag"`
	Text string  `json:"text"`
	Box  boxInfo `json:"box"`
}

type boxInfo struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// handleVibiumFind handles the vibium:find command with wait-for-selector.
// Accepts CSS selector (string) or semantic selector params (role, text, label, etc.).
func (r *Router) handleVibiumFind(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	timeoutMs, _ := cmd.Params["timeout"].(float64)
	timeout := defaultTimeout
	if timeoutMs > 0 {
		timeout = time.Duration(timeoutMs) * time.Millisecond
	}

	script, args := buildFindScript(cmd.Params, false)

	info, err := r.waitForElementWithScript(session, context, script, args, timeout)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{
		"tag":  info.Tag,
		"text": info.Text,
		"box": map[string]interface{}{
			"x":      info.Box.X,
			"y":      info.Box.Y,
			"width":  info.Box.Width,
			"height": info.Box.Height,
		},
	})
}

// handleVibiumFindAll handles the vibium:findAll command.
// Returns all matching elements with their info.
func (r *Router) handleVibiumFindAll(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	timeoutMs, _ := cmd.Params["timeout"].(float64)
	timeout := defaultTimeout
	if timeoutMs > 0 {
		timeout = time.Duration(timeoutMs) * time.Millisecond
	}

	hasText, _ := cmd.Params["hasText"].(string)
	has, _ := cmd.Params["has"].(string)

	script, args := buildFindScript(cmd.Params, true)

	// Add filter args
	args = append(args,
		map[string]interface{}{"type": "string", "value": hasText},
		map[string]interface{}{"type": "string", "value": has},
	)

	elements, err := r.waitForElements(session, context, script, args, hasText, has, timeout)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{
		"elements": elements,
		"count":    len(elements),
	})
}

// buildFindScript builds the JS function and arguments for element finding.
// It dispatches based on which selector params are present.
// If findAll is true, returns all matching elements; otherwise returns the first match.
func buildFindScript(params map[string]interface{}, findAll bool) (string, []map[string]interface{}) {
	selector, _ := params["selector"].(string)
	scope, _ := params["scope"].(string)
	role, _ := params["role"].(string)
	text, _ := params["text"].(string)
	label, _ := params["label"].(string)
	placeholder, _ := params["placeholder"].(string)
	alt, _ := params["alt"].(string)
	title, _ := params["title"].(string)
	testid, _ := params["testid"].(string)
	xpath, _ := params["xpath"].(string)

	args := []map[string]interface{}{
		{"type": "string", "value": scope},
	}

	// Determine which strategy to use based on params
	hasSemantic := role != "" || text != "" || label != "" || placeholder != "" || alt != "" || title != "" || testid != "" || xpath != ""

	if !hasSemantic && selector != "" {
		// Pure CSS selector (original behavior)
		args = append(args, map[string]interface{}{"type": "string", "value": selector})
		if findAll {
			return buildCSSFindAllScript(), args
		}
		return buildCSSFindScript(), args
	}

	// Build semantic/combo selector
	args = append(args,
		map[string]interface{}{"type": "string", "value": selector},
		map[string]interface{}{"type": "string", "value": role},
		map[string]interface{}{"type": "string", "value": text},
		map[string]interface{}{"type": "string", "value": label},
		map[string]interface{}{"type": "string", "value": placeholder},
		map[string]interface{}{"type": "string", "value": alt},
		map[string]interface{}{"type": "string", "value": title},
		map[string]interface{}{"type": "string", "value": testid},
		map[string]interface{}{"type": "string", "value": xpath},
	)

	if findAll {
		return buildSemanticFindAllScript(), args
	}
	return buildSemanticFindScript(), args
}

func buildCSSFindScript() string {
	return `
		(scope, selector) => {
			const root = scope ? document.querySelector(scope) : document;
			if (!root) return null;
			const el = root.querySelector(selector);
			if (!el) return null;
			const rect = el.getBoundingClientRect();
			return JSON.stringify({
				tag: el.tagName.toLowerCase(),
				text: (el.textContent || '').trim().substring(0, 100),
				box: { x: rect.x, y: rect.y, width: rect.width, height: rect.height }
			});
		}
	`
}

func buildCSSFindAllScript() string {
	return `
		(scope, selector, hasText, has) => {
			const root = scope ? document.querySelector(scope) : document;
			if (!root) return '[]';
			let els = Array.from(root.querySelectorAll(selector));
			if (hasText) {
				els = els.filter(el => (el.textContent || '').includes(hasText));
			}
			if (has) {
				els = els.filter(el => el.querySelector(has) !== null);
			}
			return JSON.stringify(els.map((el, i) => {
				const rect = el.getBoundingClientRect();
				return {
					tag: el.tagName.toLowerCase(),
					text: (el.textContent || '').trim().substring(0, 100),
					box: { x: rect.x, y: rect.y, width: rect.width, height: rect.height },
					index: i
				};
			}));
		}
	`
}

func semanticMatchesHelper() string {
	return `
			const IMPLICIT_ROLES = {
				A: (el) => el.hasAttribute('href') ? 'link' : '',
				AREA: (el) => el.hasAttribute('href') ? 'link' : '',
				ARTICLE: () => 'article',
				ASIDE: () => 'complementary',
				BUTTON: () => 'button',
				DETAILS: () => 'group',
				DIALOG: () => 'dialog',
				FOOTER: () => 'contentinfo',
				FORM: () => 'form',
				H1: () => 'heading', H2: () => 'heading', H3: () => 'heading',
				H4: () => 'heading', H5: () => 'heading', H6: () => 'heading',
				HEADER: () => 'banner',
				HR: () => 'separator',
				IMG: (el) => el.getAttribute('alt') ? 'img' : 'presentation',
				INPUT: (el) => {
					const t = (el.getAttribute('type') || 'text').toLowerCase();
					const map = {button:'button',checkbox:'checkbox',image:'button',
						number:'spinbutton',radio:'radio',range:'slider',
						reset:'button',search:'searchbox',submit:'button',text:'textbox',
						email:'textbox',tel:'textbox',url:'textbox',password:'textbox'};
					return map[t] || 'textbox';
				},
				LI: () => 'listitem',
				MAIN: () => 'main',
				MENU: () => 'list',
				NAV: () => 'navigation',
				OL: () => 'list',
				OPTION: () => 'option',
				OUTPUT: () => 'status',
				PROGRESS: () => 'progressbar',
				SECTION: () => 'region',
				SELECT: (el) => el.hasAttribute('multiple') ? 'listbox' : 'combobox',
				SUMMARY: () => 'button',
				TABLE: () => 'table',
				TBODY: () => 'rowgroup', THEAD: () => 'rowgroup', TFOOT: () => 'rowgroup',
				TD: () => 'cell',
				TEXTAREA: () => 'textbox',
				TH: () => 'columnheader',
				TR: () => 'row',
				UL: () => 'list',
			};

			function getImplicitRole(el) {
				const explicit = el.getAttribute('role');
				if (explicit) return explicit.toLowerCase();
				const fn = IMPLICIT_ROLES[el.tagName];
				return fn ? fn(el).toLowerCase() : '';
			}

			function matches(el, selector, role, text, label, placeholder, alt, title, testid) {
				if (selector && !el.matches(selector)) return false;
				if (role) {
					if (getImplicitRole(el) !== role.toLowerCase()) return false;
				}
				if (text) {
					const elText = (el.textContent || '').trim();
					if (!elText.includes(text)) return false;
				}
				if (label) {
					const ariaLabel = el.getAttribute('aria-label') || '';
					const labelledBy = el.getAttribute('aria-labelledby');
					let labelText = ariaLabel;
					if (labelledBy) {
						const labelEl = document.getElementById(labelledBy);
						if (labelEl) labelText = labelText || (labelEl.textContent || '').trim();
					}
					if (el.id) {
						const assocLabel = document.querySelector('label[for="' + el.id + '"]');
						if (assocLabel) labelText = labelText || (assocLabel.textContent || '').trim();
					}
					if (!labelText.includes(label)) return false;
				}
				if (placeholder && el.getAttribute('placeholder') !== placeholder) return false;
				if (alt && el.getAttribute('alt') !== alt) return false;
				if (title && el.getAttribute('title') !== title) return false;
				if (testid && el.getAttribute('data-testid') !== testid) return false;
				return true;
			}

			function toInfo(el) {
				const rect = el.getBoundingClientRect();
				return {
					tag: el.tagName.toLowerCase(),
					text: (el.textContent || '').trim().substring(0, 100),
					box: { x: rect.x, y: rect.y, width: rect.width, height: rect.height }
				};
			}

			function collectMatches(root, selector, role, text, label, placeholder, alt, title, testid, xpath) {
				const found = [];
				if (xpath) {
					const xr = document.evaluate(xpath, root, null, XPathResult.ORDERED_NODE_SNAPSHOT_TYPE, null);
					for (let i = 0; i < xr.snapshotLength; i++) {
						const el = xr.snapshotItem(i);
						if (el && el.nodeType === 1 && matches(el, selector, role, text, label, placeholder, alt, title, testid)) {
							found.push(el);
						}
					}
				} else {
					const walker = document.createTreeWalker(root, NodeFilter.SHOW_ELEMENT);
					let node;
					while (node = walker.nextNode()) {
						if (matches(node, selector, role, text, label, placeholder, alt, title, testid)) {
							found.push(node);
						}
					}
				}
				return found;
			}

			function pickBest(found, text) {
				if (found.length === 0) return null;
				if (!text || found.length === 1) return found[0];
				let best = found[0];
				let bestLen = (best.textContent || '').length;
				for (let i = 1; i < found.length; i++) {
					const len = (found[i].textContent || '').length;
					if (len < bestLen) {
						best = found[i];
						bestLen = len;
					}
				}
				return best;
			}
	`
}

func buildSemanticFindScript() string {
	return `
		(scope, selector, role, text, label, placeholder, alt, title, testid, xpath) => {
			const root = scope ? document.querySelector(scope) : document;
			if (!root) return null;
` + semanticMatchesHelper() + `
			const found = collectMatches(root, selector, role, text, label, placeholder, alt, title, testid, xpath);
			const best = pickBest(found, text);
			if (!best) return null;
			return JSON.stringify(toInfo(best));
		}
	`
}

func buildSemanticFindAllScript() string {
	return `
		(scope, selector, role, text, label, placeholder, alt, title, testid, xpath, hasText, has) => {
			const root = scope ? document.querySelector(scope) : document;
			if (!root) return '[]';
` + semanticMatchesHelper() + `
			let found = collectMatches(root, selector, role, text, label, placeholder, alt, title, testid, xpath);
			if (hasText) {
				found = found.filter(el => (el.textContent || '').includes(hasText));
			}
			if (has) {
				found = found.filter(el => el.querySelector(has) !== null);
			}
			return JSON.stringify(found.map((el, i) => {
				const info = toInfo(el);
				info.index = i;
				return info;
			}));
		}
	`
}

// waitForElementWithScript polls until an element is found using a custom script.
func (r *Router) waitForElementWithScript(session *BrowserSession, context, script string, args []map[string]interface{}, timeout time.Duration) (*elementInfo, error) {
	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond

	// Build a description for the timeout error
	desc := describeSelector(args)

	for {
		params := map[string]interface{}{
			"functionDeclaration": script,
			"target":              map[string]interface{}{"context": context},
			"arguments":           args,
			"awaitPromise":        false,
			"resultOwnership":     "root",
		}

		resp, err := r.sendInternalCommand(session, "script.callFunction", params)
		if err == nil {
			var result struct {
				Result struct {
					Result struct {
						Type  string `json:"type"`
						Value string `json:"value,omitempty"`
					} `json:"result"`
				} `json:"result"`
			}
			if err := json.Unmarshal(resp, &result); err == nil {
				if result.Result.Result.Type == "string" && result.Result.Result.Value != "" {
					var info elementInfo
					if err := json.Unmarshal([]byte(result.Result.Result.Value), &info); err == nil {
						return &info, nil
					}
				}
			}
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout after %s waiting for '%s': element not found", timeout, desc)
		}

		time.Sleep(interval)
	}
}

// waitForElements polls until at least one matching element is found, then returns all.
func (r *Router) waitForElements(session *BrowserSession, context, script string, args []map[string]interface{}, hasText, has string, timeout time.Duration) ([]map[string]interface{}, error) {
	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond

	desc := describeSelector(args)

	for {
		params := map[string]interface{}{
			"functionDeclaration": script,
			"target":              map[string]interface{}{"context": context},
			"arguments":           args,
			"awaitPromise":        false,
			"resultOwnership":     "root",
		}

		resp, err := r.sendInternalCommand(session, "script.callFunction", params)
		if err == nil {
			var result struct {
				Result struct {
					Result struct {
						Type  string `json:"type"`
						Value string `json:"value,omitempty"`
					} `json:"result"`
				} `json:"result"`
			}
			if err := json.Unmarshal(resp, &result); err == nil {
				if result.Result.Result.Type == "string" && result.Result.Result.Value != "" {
					var elements []map[string]interface{}
					if err := json.Unmarshal([]byte(result.Result.Result.Value), &elements); err == nil && len(elements) > 0 {
						return elements, nil
					}
				}
			}
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout after %s waiting for '%s': no elements found", timeout, desc)
		}

		time.Sleep(interval)
	}
}

// describeSelector builds a human-readable description of the selector for error messages.
func describeSelector(args []map[string]interface{}) string {
	var parts []string
	// args[0] is scope, args[1] is selector (for CSS) or selector (for semantic)
	if len(args) > 1 {
		if v, ok := args[1]["value"].(string); ok && v != "" {
			parts = append(parts, v)
		}
	}
	// For semantic selectors, args[2..] are role, text, label, etc.
	labels := []string{"role", "text", "label", "placeholder", "alt", "title", "testid", "xpath"}
	for i, label := range labels {
		idx := i + 2
		if idx < len(args) {
			if v, ok := args[idx]["value"].(string); ok && v != "" {
				parts = append(parts, fmt.Sprintf("%s=%s", label, v))
			}
		}
	}
	if len(parts) == 0 {
		return "element"
	}
	return strings.Join(parts, ", ")
}

// waitForElement polls until an element matching the CSS selector is found or timeout.
// This is the legacy method used by click/type handlers.
func (r *Router) waitForElement(session *BrowserSession, context, selector string, timeout time.Duration) (*elementInfo, error) {
	script, args := buildFindScript(map[string]interface{}{"selector": selector}, false)
	return r.waitForElementWithScript(session, context, script, args, timeout)
}
