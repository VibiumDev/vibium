// Standard DOM probe for vibe-inventory walk. Captures per-route state.
// Usage: cat probe.js | vibium eval --stdin
// Returns JSON with everything a single page tells us.

(() => {
  const url = location.href;
  const title = document.title;
  const headings = [...document.querySelectorAll('h1,h2,h3,h4')].slice(0, 12).map(e => e.tagName + ': ' + (e.innerText || '').replace(/\s+/g, ' ').trim());
  const tables = [...document.querySelectorAll('table')].slice(0, 5).map(t => ({
    rows: t.querySelectorAll('tr').length,
    headers: [...t.querySelectorAll('th')].map(th => (th.innerText || '').replace(/\s+/g, ' ').trim()),
  }));
  const grids = [...document.querySelectorAll('.MuiDataGrid-root, [role=grid]')].slice(0, 5).map(g => ({
    rows: g.querySelectorAll('[role=row]').length,
    headers: [...g.querySelectorAll('[role=columnheader]')].map(h => (h.innerText || '').replace(/\s+/g, ' ').trim()),
  }));
  const inputs = [...document.querySelectorAll('input,select,textarea')].slice(0, 25).map(e => ({
    tag: e.tagName.toLowerCase(),
    type: e.type || null,
    name: e.name || null,
    placeholder: e.placeholder || null,
    label: e.getAttribute('aria-label') || null,
  }));
  const buttons = [...document.querySelectorAll('button')].slice(0, 30).map(b => (b.innerText || b.getAttribute('aria-label') || '').replace(/\s+/g, ' ').trim()).filter(Boolean);
  const isErrPage = /sorry,?\s*it.s not you|page not found|something went wrong|forbidden|unauthori[sz]ed|access denied/i.test(document.body.innerText) || /\/(error|404)(\?|$)/.test(location.href);
  const bodyText = (document.body.innerText || '').replace(/\s+/g, ' ').trim();
  return JSON.stringify({ url, title, isErrPage, headings, tables, grids, inputs, buttons, bodyTextLen: bodyText.length, bodyText: bodyText.slice(0, 4000) });
})()
