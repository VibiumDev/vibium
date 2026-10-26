package api

// PierceQueryJS defines pierceQuery/pierceQueryAll, the selector resolvers used
// everywhere vibium turns a CSS selector into elements.
//
// querySelector cannot cross a shadow boundary, so components built with shadow
// DOM — native controls, Lit/FAST/Shoelace, most design systems — were reachable
// only through page.evaluate and manual shadowRoot walking (#118).
//
//	"my-card >> button"          one hop: button inside my-card's shadow root
//	"outer >>> button"           deep: any button below outer, through nesting
//
// The combinators match Playwright (>>) and Selenium 4 (>>>). A selector with
// neither is passed straight to querySelectorAll, so this costs nothing for the
// common case and cannot change its behavior.
func PierceQueryJS() string {
	return `
		// Splits on >> and >>> outside quotes, brackets and escapes,
		// so [aria-label=">>"] stays one selector. >>> wins over >> at one
		// index. Parts are trimmed of whitespace (any \s, as the old regex
		// split did) except an escaped space, which ends a ref for an id
		// like "a ".
		function __pierceSplit(selector) {
			const out = [];
			const skipSpace = (j) => { while (j < selector.length && /\s/.test(selector[j])) j++; return j; };
			let start = skipSpace(0), end = start, quote = '', depth = 0;
			for (let i = start; i < selector.length; i++) {
				const c = selector[i];
				const prevEnd = end;
				if (c === '\\') { i++; end = i + 1; continue; }
				if (!/\s/.test(c)) end = i + 1;
				if (quote) { if (c === quote) quote = ''; continue; }
				if (c === '"' || c === "'") { quote = c; continue; }
				if (c === '[' || c === '(') { depth++; continue; }
				if (c === ']' || c === ')') { if (depth) depth--; continue; }
				if (depth || c !== '>' || selector[i + 1] !== '>') continue;
				const comb = selector[i + 2] === '>' ? '>>>' : '>>';
				out.push(selector.slice(start, Math.max(start, prevEnd)), comb);
				start = end = skipSpace(i + comb.length);
				i = start - 1;
			}
			out.push(selector.slice(start, Math.max(start, end)));
			return out;
		}

		function __shadowRootsUnder(root) {
			const roots = [];
			// A host passed in directly owns a shadow root that is not reachable
			// by querying its light-DOM children, so seed with it explicitly.
			const stack = [root];
			if (root.shadowRoot) stack.push(root.shadowRoot);
			while (stack.length) {
				const r = stack.pop();
				if (!r) continue;
				roots.push(r);
				const hosts = r.querySelectorAll('*');
				for (let i = 0; i < hosts.length; i++) {
					if (hosts[i].shadowRoot) stack.push(hosts[i].shadowRoot);
				}
			}
			return roots;
		}

		function pierceQueryAll(root, selector) {
			if (!root || !selector) return [];
			if (selector.indexOf('>>') === -1) {
				return Array.prototype.slice.call(root.querySelectorAll(selector));
			}

			const parts = __pierceSplit(selector);
			let current = Array.prototype.slice.call(root.querySelectorAll(parts[0]));

			for (let i = 1; i < parts.length; i += 2) {
				const deep = parts[i] === '>>>';
				const next = parts[i + 1];
				const out = [];
				for (let h = 0; h < current.length; h++) {
					const host = current[h];
					if (deep) {
						// Search the host's own subtree and every shadow root beneath it.
						const roots = __shadowRootsUnder(host.shadowRoot || host);
						for (let r = 0; r < roots.length; r++) {
							const found = roots[r].querySelectorAll(next);
							for (let f = 0; f < found.length; f++) out.push(found[f]);
						}
					} else if (host.shadowRoot) {
						const found = host.shadowRoot.querySelectorAll(next);
						for (let f = 0; f < found.length; f++) out.push(found[f]);
					}
				}
				current = out;
			}
			return current;
		}

		function pierceQuery(root, selector) {
			if (!root || !selector) return null;
			if (selector.indexOf('>>') === -1) return root.querySelector(selector);
			const all = pierceQueryAll(root, selector);
			return all.length ? all[0] : null;
		}
	`
}
