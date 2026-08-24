// Standard DOM probe for vibe-recon. Captures landing-page state without login.
// Usage: cat probe.js | vibium eval --stdin
// Returns JSON.

(() => {
  const url = location.href;
  const title = document.title;
  const headings = [...document.querySelectorAll('h1,h2,h3,h4')].slice(0, 12).map(e => e.tagName + ': ' + (e.innerText || '').replace(/\s+/g, ' ').trim());
  const inputs = [...document.querySelectorAll('input,select,textarea')].slice(0, 25).map(e => ({
    tag: e.tagName.toLowerCase(),
    type: e.type || null,
    name: e.name || null,
    placeholder: e.placeholder || null,
    label: e.getAttribute('aria-label') || null,
  }));
  const buttons = [...document.querySelectorAll('button, [type=submit]')].slice(0, 30).map(b => (b.innerText || b.getAttribute('aria-label') || '').replace(/\s+/g, ' ').trim()).filter(Boolean);
  const linksSample = [...document.querySelectorAll('a[href]')].slice(0, 30).map(a => ({ txt: (a.innerText || '').replace(/\s+/g, ' ').trim().slice(0, 60), href: a.getAttribute('href') }));
  const isErrPage = /sorry|page not found|forbidden|unauthori[sz]ed|access denied/i.test(document.body.innerText) || /\/error(\?|$)/.test(location.href);
  const formCount = document.querySelectorAll('form').length;
  return JSON.stringify({ url, title, headings, inputs, buttons, linksSample, isErrPage, formCount });
})()
