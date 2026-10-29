package api

// GetSelectorJS returns the JS getSelector(el) function body that generates unique CSS selectors.
func GetSelectorJS() string {
	return `function getSelector(el) {
			// An element inside a shadow root is addressed as "<host> >> <path>",
			// the pierce form the ref resolvers understand. Without the host hop the
			// path stopped at the root, so map minted refs that matched nothing or
			// a light-DOM element with the same path.
			const root = el.getRootNode();
			const shadow = root instanceof ShadowRoot ? root : null;
			const prefix = shadow ? getSelector(shadow.host) + ' >> ' : '';
			if (el.id) return prefix + '#' + CSS.escape(el.id);
			const parts = [];
			let cur = el;
			while (cur && cur !== document.body && cur !== document.documentElement) {
				let seg = cur.tagName.toLowerCase();
				if (cur.id) {
					parts.unshift('#' + CSS.escape(cur.id));
					break;
				}
				const parent = cur.parentElement || shadow;
				if (parent) {
					const siblings = Array.from(parent.children).filter(c => c.tagName === cur.tagName);
					if (siblings.length > 1) {
						const idx = siblings.indexOf(cur) + 1;
						seg += ':nth-of-type(' + idx + ')';
					}
				}
				parts.unshift(seg);
				cur = cur.parentElement;
			}
			if (parts.length === 0) return prefix + el.tagName.toLowerCase();
			// :host anchors the path at the shadow root, as body does in the
			// document; unanchored, "div > button" would match a deeper button first.
			if (!parts[0].startsWith('#')) parts.unshift(shadow ? ':host' : 'body');
			return prefix + parts.join(' > ');
		}`
}

// GetLabelJS returns the JS getLabel(el) function body that generates descriptive labels.
func GetLabelJS() string {
	return `function getLabel(el) {
			const tag = el.tagName.toLowerCase();
			const type = el.getAttribute('type');
			let desc = '[' + tag;
			if (type) desc += ' type="' + type + '"';
			desc += ']';

			const ariaLabel = el.getAttribute('aria-label');
			if (ariaLabel) return desc + ' "' + ariaLabel.substring(0, 60) + '"';

			const placeholder = el.getAttribute('placeholder');
			if (placeholder) return desc + ' placeholder="' + placeholder.substring(0, 60) + '"';

			const title = el.getAttribute('title');
			if (title) return desc + ' title="' + title.substring(0, 60) + '"';

			const text = (el.textContent || '').trim().substring(0, 60);
			if (text) return desc + ' "' + text + '"';

			const name = el.getAttribute('name');
			if (name) return desc + ' name="' + name + '"';

			const src = el.getAttribute('src');
			if (src) return desc + ' src="' + src.substring(0, 60) + '"';

			return desc;
		}`
}

// mapScript returns the JS function that maps interactive elements with refs.
// When a selector is provided, only elements within the matching subtree are returned.
func MapScript() string {
	return `(scopeSelector) => {
		` + GetSelectorJS() + `
		` + GetLabelJS() + `
		` + PierceQueryJS() + `

		const interactive = 'a[href], button, input, textarea, select, [role="button"], [role="link"], [role="checkbox"], [role="radio"], [role="tab"], [role="menuitem"], [role="switch"], [onclick], [tabindex]:not([tabindex="-1"]), summary, details';

		const root = scopeSelector ? pierceQuery(document, scopeSelector) : document;
		if (!root) return JSON.stringify([]);
		// Walk shadow roots too: querySelectorAll stops at the boundary, so
		// web-component UIs listed as empty (#203).
		const els = [];
		{
			const roots = __shadowRootsUnder(root);
			for (let r = 0; r < roots.length; r++) {
				const found = roots[r].querySelectorAll(interactive);
				for (let f = 0; f < found.length; f++) els.push(found[f]);
			}
		}
		const results = [];
		const seen = new Set();

		for (const el of els) {
			const style = window.getComputedStyle(el);
			if (style.display === 'none' || style.visibility === 'hidden' || el.offsetWidth === 0) continue;

			const sel = getSelector(el);
			if (seen.has(sel)) continue;
			seen.add(sel);

			results.push({ selector: sel, label: getLabel(el) });
		}

		return JSON.stringify(results);
	}`
}
