package api

// getNameJS returns getName(el): the accessible name, the way a person would
// read the control. It is a flat approximation of accname, not the algorithm.
func getNameJS() string {
	return `function getName(el) {
			const clean = (s) => (s || '').replace(/\s+/g, ' ').trim().substring(0, 200);
			const aria = el.getAttribute('aria-label');
			if (aria) return clean(aria);
			const by = el.getAttribute('aria-labelledby');
			if (by) {
				const parts = by.split(/\s+/).map((id) => { const n = document.getElementById(id); return n ? n.textContent : ''; });
				const s = clean(parts.join(' '));
				if (s) return s;
			}
			if (el.id) {
				const lab = document.querySelector('label[for="' + CSS.escape(el.id) + '"]');
				if (lab) { const s = clean(lab.textContent); if (s) return s; }
			}
			const wrap = el.closest ? el.closest('label') : null;
			if (wrap && wrap !== el) { const s = clean(wrap.textContent); if (s) return s; }
			const tag = el.tagName;
			if (tag === 'IMG' || tag === 'AREA') return clean(el.getAttribute('alt'));
			if (tag === 'INPUT') {
				const t = (el.getAttribute('type') || 'text').toLowerCase();
				if (t === 'submit' || t === 'reset' || t === 'button' || t === 'image') return clean(el.value || el.getAttribute('alt'));
				const ph = el.getAttribute('placeholder');
				if (ph) return clean(ph);
			}
			if (tag === 'TEXTAREA') { const ph = el.getAttribute('placeholder'); if (ph) return clean(ph); }
			const text = clean(el.innerText !== undefined ? el.innerText : el.textContent);
			if (text) return text;
			const title = el.getAttribute('title');
			if (title) return clean(title);
			const inner = el.querySelector ? el.querySelector('img[alt], [aria-label]') : null;
			if (inner) return clean(inner.getAttribute('alt') || inner.getAttribute('aria-label'));
			return '';
		}`
}

// describeElJS returns describeEl(el): the record every find surface reports.
func describeElJS() string {
	return GetSelectorJS() + `
		` + GetLabelJS() + `
		` + getNameJS() + `
		function describeEl(el) {
			const rect = el.getBoundingClientRect();
			return {
				selector: getSelector(el),
				label: getLabel(el),
				tag: el.tagName.toLowerCase(),
				text: ((el.innerText !== undefined ? el.innerText : el.textContent) || '').trim().substring(0, 2000),
				role: typeof getImplicitRole === 'function' ? getImplicitRole(el) : (el.getAttribute('role') || ''),
				name: getName(el),
				box: { x: rect.x, y: rect.y, width: rect.width, height: rect.height }
			};
		}`
}

// locateScript finds elements by css or structured locators and returns their
// full records, so one query can both answer a client and mint refs.
// Args: scope, selector, tag, name, role, text, label, placeholder, alt,
// title, testid, xpath, hasText, has, all, limit.
func locateScript() string {
	return `(scope, selector, tag, name, role, text, label, placeholder, alt, title, testid, xpath, hasText, has, all, limit) => {
		` + PierceQueryJS() + `
		const root = scope ? pierceQuery(document, scope) : document;
		if (!root) return '[]';
		` + semanticMatchesHelper() + `
		` + describeElJS() + `
		const semantic = !!(tag || name || role || text || label || placeholder || alt || title || testid || xpath);
		let found;
		if (!semantic) {
			found = all ? Array.from(pierceQueryAll(root, selector)) : [pierceQuery(root, selector)].filter(Boolean);
		} else {
			// <head> holds a <title> whose text can shadow the heading a
			// reader means; nothing there can be interacted with.
			found = collectMatches(root, selector, role, text, label, placeholder, alt, title, testid, xpath)
				.filter((el) => !el.closest('head'));
			if (tag) found = found.filter((el) => el.tagName.toLowerCase() === tag);
			if (name) found = found.filter((el) => getName(el).includes(name));
			if (!all) found = [pickBest(found, text)].filter(Boolean);
		}
		if (hasText) found = found.filter((el) => (el.textContent || '').includes(hasText));
		if (has) found = found.filter((el) => el.querySelector(has) !== null);
		if (found.length === 0) return '[]';
		if (!all) {
			const el = found[0];
			if (el.scrollIntoViewIfNeeded) el.scrollIntoViewIfNeeded(true);
			else el.scrollIntoView({ block: 'center', inline: 'nearest' });
		}
		const n = all && limit > 0 ? Math.min(found.length, limit) : found.length;
		const out = [];
		for (let i = 0; i < n; i++) {
			const d = describeEl(found[i]);
			d.index = i;
			out.push(d);
		}
		return JSON.stringify(out);
	}`
}

// candidatesScript lists the visible elements a description could name:
// controls, headings, images, and labelled fields, with their names and roles.
// Args: scope, limit.
func candidatesScript() string {
	return `(scope, limit) => {
		` + PierceQueryJS() + `
		const root = scope ? pierceQuery(document, scope) : document;
		if (!root) return '[]';
		` + semanticMatchesHelper() + `
		` + describeElJS() + `
		const wanted = 'a[href], button, input, select, textarea, [role], h1, h2, h3, h4, h5, h6, img, summary, [tabindex], [onclick], [contenteditable], label';
		const seen = new Set();
		const out = [];
		const roots = __shadowRootsUnder(root);
		for (let r = 0; r < roots.length && out.length < limit; r++) {
			const els = roots[r].querySelectorAll(wanted);
			for (let i = 0; i < els.length && out.length < limit; i++) {
				const el = els[i];
				const type = (el.getAttribute('type') || '').toLowerCase();
				if (el.tagName === 'INPUT' && type === 'hidden') continue;
				const style = window.getComputedStyle(el);
				if (style.display === 'none' || style.visibility === 'hidden') continue;
				const rect = el.getBoundingClientRect();
				if (rect.width === 0 || rect.height === 0) continue;
				const d = describeEl(el);
				if (seen.has(d.selector)) continue;
				seen.add(d.selector);
				out.push(d);
			}
		}
		return JSON.stringify(out);
	}`
}

// snapScript returns the DOM element behind a viewport box, when one covers
// it, so a visual match can become a real element. Args: x, y, width, height.
func snapScript() string {
	return `(x, y, width, height) => {
		` + PierceQueryJS() + `
		` + semanticMatchesHelper() + `
		` + describeElJS() + `
		const cx = x + width / 2, cy = y + height / 2;
		let el = document.elementFromPoint(cx, cy);
		while (el && el.shadowRoot) {
			const inner = el.shadowRoot.elementFromPoint(cx, cy);
			if (!inner || inner === el) break;
			el = inner;
		}
		if (!el || el === document.documentElement || el === document.body) return null;
		const control = el.closest('a[href], button, input, select, textarea, [role], summary, label, img, h1, h2, h3, h4, h5, h6');
		const area = Math.max(1, width * height);
		const overlap = (r) => {
			const ix = Math.max(0, Math.min(x + width, r.right) - Math.max(x, r.left));
			const iy = Math.max(0, Math.min(y + height, r.bottom) - Math.max(y, r.top));
			return ix * iy;
		};
		if (control) {
			const r = control.getBoundingClientRect();
			if (r.width * r.height <= area * 4 && overlap(r) >= area * 0.5) el = control;
		}
		const rect = el.getBoundingClientRect();
		if (rect.width * rect.height > area * 6) return null;
		return JSON.stringify([describeEl(el)]);
	}`
}

// pageStateScript reads what a coordinate is valid against.
func pageStateScript() string {
	return `() => JSON.stringify({ url: location.href, scrollX: window.scrollX, scrollY: window.scrollY, width: window.innerWidth, height: window.innerHeight })`
}

// highlightBoxScript outlines a viewport box for ms milliseconds.
// Args: x, y, width, height, ms.
func highlightBoxScript() string {
	return `(x, y, width, height, ms) => {
		const box = document.createElement('div');
		box.className = '__vibium_highlight';
		box.style.cssText = 'position:fixed;z-index:2147483647;pointer-events:none;box-sizing:border-box;outline:2px solid #ff2d95;outline-offset:1px;background:rgba(255,45,149,0.08);left:' + x + 'px;top:' + y + 'px;width:' + width + 'px;height:' + height + 'px;';
		document.documentElement.appendChild(box);
		setTimeout(() => box.remove(), ms);
		return 'ok';
	}`
}
