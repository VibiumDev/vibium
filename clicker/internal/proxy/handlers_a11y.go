package proxy

import (
	"encoding/json"
	"fmt"
)

// handleVibiumElRole handles vibium:el.role — returns the element's computed ARIA role.
func (r *Router) handleVibiumElRole(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	script, args := buildElStateScript(ep, `(() => {
		if (typeof el.computedRole === 'string' && el.computedRole !== '') return el.computedRole;
		const explicit = el.getAttribute('role');
		if (explicit) return explicit.toLowerCase();
		const IMPLICIT_ROLES = {
			A: (e) => e.hasAttribute('href') ? 'link' : '',
			AREA: (e) => e.hasAttribute('href') ? 'link' : '',
			ARTICLE: () => 'article', ASIDE: () => 'complementary',
			BUTTON: () => 'button', DETAILS: () => 'group', DIALOG: () => 'dialog',
			FOOTER: () => 'contentinfo', FORM: () => 'form',
			H1: () => 'heading', H2: () => 'heading', H3: () => 'heading',
			H4: () => 'heading', H5: () => 'heading', H6: () => 'heading',
			HEADER: () => 'banner', HR: () => 'separator',
			IMG: (e) => e.getAttribute('alt') ? 'img' : 'presentation',
			INPUT: (e) => {
				const t = (e.getAttribute('type') || 'text').toLowerCase();
				const m = {button:'button',checkbox:'checkbox',image:'button',
					number:'spinbutton',radio:'radio',range:'slider',
					reset:'button',search:'searchbox',submit:'button',text:'textbox',
					email:'textbox',tel:'textbox',url:'textbox',password:'textbox'};
				return m[t] || 'textbox';
			},
			LI: () => 'listitem', MAIN: () => 'main', MENU: () => 'list',
			NAV: () => 'navigation', OL: () => 'list', OPTION: () => 'option',
			OUTPUT: () => 'status', PROGRESS: () => 'progressbar',
			SECTION: () => 'region',
			SELECT: (e) => e.hasAttribute('multiple') ? 'listbox' : 'combobox',
			SUMMARY: () => 'button', TABLE: () => 'table',
			TBODY: () => 'rowgroup', THEAD: () => 'rowgroup', TFOOT: () => 'rowgroup',
			TD: () => 'cell', TEXTAREA: () => 'textbox', TH: () => 'columnheader',
			TR: () => 'row', UL: () => 'list',
		};
		const fn = IMPLICIT_ROLES[el.tagName];
		return fn ? fn(el) : '';
	})()`)
	val, err := r.evalElementScript(session, context, script, args)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{"role": val})
}

// handleVibiumElLabel handles vibium:el.label — returns the element's accessible name.
func (r *Router) handleVibiumElLabel(session *BrowserSession, cmd bidiCommand) {
	ep := extractElementParams(cmd.Params)
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	script, args := buildElStateScript(ep, `(() => {
		if (typeof el.computedName === 'string' && el.computedName !== '') return el.computedName;
		const ariaLabel = el.getAttribute('aria-label');
		if (ariaLabel) return ariaLabel;
		const labelledBy = el.getAttribute('aria-labelledby');
		if (labelledBy) {
			const parts = labelledBy.split(/\s+/).map(id => {
				const ref = document.getElementById(id);
				return ref ? (ref.textContent || '').trim() : '';
			}).filter(Boolean);
			if (parts.length) return parts.join(' ');
		}
		if (el.id) {
			const assocLabel = document.querySelector('label[for="' + el.id + '"]');
			if (assocLabel) return (assocLabel.textContent || '').trim();
		}
		const parentLabel = el.closest('label');
		if (parentLabel) return (parentLabel.textContent || '').trim();
		const placeholder = el.getAttribute('placeholder');
		if (placeholder) return placeholder;
		const alt = el.getAttribute('alt');
		if (alt) return alt;
		const title = el.getAttribute('title');
		if (title) return title;
		const text = (el.textContent || '').trim();
		if (text) return text;
		return '';
	})()`)
	val, err := r.evalElementScript(session, context, script, args)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}
	r.sendSuccess(session, cmd.ID, map[string]interface{}{"label": val})
}

// handleVibiumPageA11yTree handles vibium:page.a11yTree — returns the accessibility tree.
func (r *Router) handleVibiumPageA11yTree(session *BrowserSession, cmd bidiCommand) {
	context, err := r.resolveContext(session, cmd.Params)
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	interestingOnly := true
	if val, ok := cmd.Params["everything"].(bool); ok {
		interestingOnly = !val
	}

	rootSelector := ""
	if val, ok := cmd.Params["root"].(string); ok {
		rootSelector = val
	}

	args := []map[string]interface{}{
		{"type": "boolean", "value": interestingOnly},
		{"type": "string", "value": rootSelector},
	}

	resp, err := r.sendInternalCommand(session, "script.callFunction", map[string]interface{}{
		"functionDeclaration": a11yTreeScript(),
		"target":              map[string]interface{}{"context": context},
		"arguments":           args,
		"awaitPromise":        false,
		"resultOwnership":     "root",
	})
	if err != nil {
		r.sendError(session, cmd.ID, err)
		return
	}

	val, err := parseScriptResult(resp)
	if err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("a11yTree failed: %w", err))
		return
	}

	var tree interface{}
	if err := json.Unmarshal([]byte(val), &tree); err != nil {
		r.sendError(session, cmd.ID, fmt.Errorf("a11yTree parse failed: %w", err))
		return
	}

	r.sendSuccess(session, cmd.ID, map[string]interface{}{"tree": tree})
}

// a11yTreeScript returns the JS function that builds the accessibility tree.
func a11yTreeScript() string {
	return `(interestingOnly, rootSelector) => {
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

		function getRole(el) {
			if (typeof el.computedRole === 'string' && el.computedRole !== '') return el.computedRole;
			const explicit = el.getAttribute('role');
			if (explicit) return explicit.toLowerCase();
			const fn = IMPLICIT_ROLES[el.tagName];
			return fn ? fn(el) : 'generic';
		}

		function getName(el) {
			if (typeof el.computedName === 'string') return el.computedName;
			const ariaLabel = el.getAttribute('aria-label');
			if (ariaLabel) return ariaLabel;
			const labelledBy = el.getAttribute('aria-labelledby');
			if (labelledBy) {
				const parts = labelledBy.split(/\s+/).map(id => {
					const ref = document.getElementById(id);
					return ref ? (ref.textContent || '').trim() : '';
				}).filter(Boolean);
				if (parts.length) return parts.join(' ');
			}
			if (el.id) {
				const assocLabel = document.querySelector('label[for="' + el.id + '"]');
				if (assocLabel) return (assocLabel.textContent || '').trim();
			}
			const placeholder = el.getAttribute('placeholder');
			if (placeholder) return placeholder;
			const alt = el.getAttribute('alt');
			if (alt) return alt;
			const title = el.getAttribute('title');
			if (title) return title;
			return '';
		}

		function getChildren(el) {
			if (el.shadowRoot) return Array.from(el.shadowRoot.children);
			return Array.from(el.children);
		}

		function getHeadingLevel(el) {
			const tag = el.tagName;
			if (tag === 'H1') return 1;
			if (tag === 'H2') return 2;
			if (tag === 'H3') return 3;
			if (tag === 'H4') return 4;
			if (tag === 'H5') return 5;
			if (tag === 'H6') return 6;
			const level = el.getAttribute('aria-level');
			if (level) return parseInt(level, 10);
			return undefined;
		}

		function buildNode(el) {
			const role = getRole(el);
			const name = getName(el);

			// Collect children first
			const childNodes = [];
			for (const child of getChildren(el)) {
				if (child.nodeType !== 1) continue;
				const nodes = buildNode(child);
				if (nodes) {
					if (Array.isArray(nodes)) {
						childNodes.push(...nodes);
					} else {
						childNodes.push(nodes);
					}
				}
			}

			// If interestingOnly, skip uninteresting nodes (promote their children)
			if (interestingOnly) {
				if (role === 'none' || role === 'presentation') {
					return childNodes.length ? childNodes : null;
				}
				if (role === 'generic' && !name) {
					return childNodes.length ? childNodes : null;
				}
			}

			const node = { role: role };
			if (name) node.name = name;

			// Collect states
			if (el.hasAttribute('disabled') || el.disabled) node.disabled = true;
			if (el.hasAttribute('aria-expanded')) node.expanded = el.getAttribute('aria-expanded') === 'true';
			if (document.activeElement === el) node.focused = true;

			// checked (checkbox, radio, aria-checked)
			if (typeof el.checked === 'boolean' && (el.type === 'checkbox' || el.type === 'radio')) {
				node.checked = el.checked;
			} else if (el.hasAttribute('aria-checked')) {
				const v = el.getAttribute('aria-checked');
				node.checked = v === 'true' ? true : v === 'mixed' ? 'mixed' : false;
			}

			// pressed (toggle buttons)
			if (el.hasAttribute('aria-pressed')) {
				const v = el.getAttribute('aria-pressed');
				node.pressed = v === 'true' ? true : v === 'mixed' ? 'mixed' : false;
			}

			if (el.hasAttribute('aria-selected') && el.getAttribute('aria-selected') === 'true') node.selected = true;
			if (el.hasAttribute('required') || el.required) node.required = true;
			if (el.hasAttribute('readonly') || el.readOnly) node.readonly = true;

			// heading level
			const level = getHeadingLevel(el);
			if (level !== undefined) node.level = level;

			// value for range controls
			if (el.hasAttribute('aria-valuetext')) {
				node.value = el.getAttribute('aria-valuetext');
			} else if (el.hasAttribute('aria-valuenow')) {
				node.value = parseFloat(el.getAttribute('aria-valuenow'));
			} else if ((el.tagName === 'INPUT' && (el.type === 'range' || el.type === 'number')) || el.tagName === 'PROGRESS') {
				node.value = el.value !== undefined && el.value !== '' ? parseFloat(el.value) : undefined;
			}

			if (el.hasAttribute('aria-valuemin')) node.valuemin = parseFloat(el.getAttribute('aria-valuemin'));
			if (el.hasAttribute('aria-valuemax')) node.valuemax = parseFloat(el.getAttribute('aria-valuemax'));

			// description
			const describedBy = el.getAttribute('aria-describedby');
			if (describedBy) {
				const parts = describedBy.split(/\s+/).map(id => {
					const ref = document.getElementById(id);
					return ref ? (ref.textContent || '').trim() : '';
				}).filter(Boolean);
				if (parts.length) node.description = parts.join(' ');
			}

			if (childNodes.length) node.children = childNodes;

			return node;
		}

		const rootEl = rootSelector ? document.querySelector(rootSelector) : document.body;
		if (!rootEl) return JSON.stringify({role: 'WebArea', name: document.title, children: []});

		const children = [];
		for (const child of getChildren(rootEl)) {
			if (child.nodeType !== 1) continue;
			const nodes = buildNode(child);
			if (nodes) {
				if (Array.isArray(nodes)) {
					children.push(...nodes);
				} else {
					children.push(nodes);
				}
			}
		}

		return JSON.stringify({
			role: 'WebArea',
			name: document.title,
			children: children
		});
	}`
}
