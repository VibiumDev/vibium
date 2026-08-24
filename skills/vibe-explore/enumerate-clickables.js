// Enumerate every interactive element on the current page, classified by safety.
// Usage: cat enumerate-clickables.js | vibium eval --stdin
// Returns JSON: { clickables: [{ ord, label, selector, tag, href, safe, skip_reason }], summary }
//
// To lift the destructive-action filter, replace `INCLUDE_DESTRUCTIVE = false` with
// `INCLUDE_DESTRUCTIVE = true` before piping (the runner does this for `--include-destructive`).

(() => {
  const INCLUDE_DESTRUCTIVE = false;

  const DESTRUCTIVE_RE = /\b(delete|remove|drop|destroy|clear (all|cache|history)|reset|wipe|purge|archive|sign\s*out|log\s*out|disconnect account|revoke|submit|send|pay|charge|subscribe|confirm purchase|place order|checkout|buy|post|publish|tweet|reply( all)?|truncate|migrate|seed)\b/i;

  const cssEscape = (s) => (window.CSS && CSS.escape) ? CSS.escape(s) : String(s).replace(/[^a-zA-Z0-9_-]/g, '\\$&');

  const buildSelector = (el) => {
    if (el.id) return `#${cssEscape(el.id)}`;
    const path = [];
    let cur = el;
    while (cur && cur.nodeType === 1 && cur.tagName !== 'HTML' && path.length < 6) {
      let part = cur.tagName.toLowerCase();
      if (cur.className && typeof cur.className === 'string') {
        const cls = cur.className.split(/\s+/).filter(Boolean).slice(0, 2).map(cssEscape).join('.');
        if (cls) part += '.' + cls;
      }
      const parent = cur.parentNode;
      if (parent) {
        const sibs = [...parent.children].filter(c => c.tagName === cur.tagName);
        if (sibs.length > 1) part += `:nth-of-type(${sibs.indexOf(cur) + 1})`;
      }
      path.unshift(part);
      cur = cur.parentNode;
    }
    return path.join(' > ');
  };

  const isVisible = (el) => {
    const r = el.getBoundingClientRect();
    if (r.width === 0 || r.height === 0) return false;
    const style = getComputedStyle(el);
    if (style.visibility === 'hidden' || style.display === 'none' || parseFloat(style.opacity) === 0) return false;
    return true;
  };

  const labelOf = (el) => {
    const aria = el.getAttribute('aria-label');
    if (aria && aria.trim()) return aria.trim();
    const txt = (el.innerText || el.textContent || '').replace(/\s+/g, ' ').trim();
    if (txt) return txt.slice(0, 80);
    const title = el.getAttribute('title');
    if (title) return title.trim();
    const placeholder = el.getAttribute('placeholder');
    if (placeholder) return placeholder.trim();
    return '(unlabeled)';
  };

  const candidates = new Set();
  const sel = [
    'button',
    '[role="button"]',
    '[role="tab"]',
    '[role="menuitem"]',
    '[role="link"]',
    'a[href]',
    '[onclick]',
    '.MuiButtonBase-root',
  ].join(', ');
  document.querySelectorAll(sel).forEach(el => candidates.add(el));

  const seen = new Set();
  const result = [];
  let ord = 0;

  for (const el of candidates) {
    if (!isVisible(el)) continue;

    const label = labelOf(el);
    const selector = buildSelector(el);
    const tag = el.tagName.toLowerCase();
    const href = el.getAttribute('href');
    const target = el.getAttribute('target');
    const dedupKey = label.toLowerCase() + '|' + selector;

    if (seen.has(dedupKey)) continue;
    seen.add(dedupKey);

    let safe = true;
    let skip_reason = null;

    if (tag === 'a' && target === '_blank') {
      safe = false;
      skip_reason = 'opens new tab';
    }
    if (safe && tag === 'a' && href && /^https?:\/\//i.test(href)) {
      try {
        const u = new URL(href, location.href);
        if (u.origin !== location.origin) {
          safe = false;
          skip_reason = `external origin (${u.origin})`;
        }
      } catch (e) {}
    }
    if (safe && (el.getAttribute('type') === 'submit' || (tag === 'button' && el.closest('form')))) {
      safe = false;
      skip_reason = 'form submit';
    }
    if (safe && !INCLUDE_DESTRUCTIVE && DESTRUCTIVE_RE.test(label)) {
      safe = false;
      skip_reason = 'destructive: matches banned-words';
    }
    if (safe && label === '(unlabeled)') {
      safe = false;
      skip_reason = 'unlabeled — needs human review';
    }

    ord += 1;
    result.push({ ord, label, selector, tag, href: href || null, safe, skip_reason });
  }

  const summary = {
    total: result.length,
    safe: result.filter(r => r.safe).length,
    skipped_destructive: result.filter(r => r.skip_reason && r.skip_reason.startsWith('destructive')).length,
    skipped_external: result.filter(r => r.skip_reason && (r.skip_reason.includes('new tab') || r.skip_reason.includes('external'))).length,
    skipped_form: result.filter(r => r.skip_reason === 'form submit').length,
    skipped_unlabeled: result.filter(r => r.skip_reason && r.skip_reason.startsWith('unlabeled')).length,
  };

  return JSON.stringify({ clickables: result, summary });
})()
