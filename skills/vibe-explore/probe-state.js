// Capture page state for before/after comparison in explore mode.
// Usage: cat probe-state.js | vibium eval --stdin
// Returns JSON: { url, title, modal_count, body_text_hash, body_text_sample, error_visible }

(() => {
  const url = location.href;
  const title = document.title;
  const modal_count = document.querySelectorAll('[role="dialog"], .modal, [class*="modal" i]:not([class*="modalcontent"])').length;
  const body_text = (document.body.innerText || '').replace(/\s+/g, ' ').trim();
  let h = 5381;
  for (let i = 0; i < body_text.length; i++) {
    h = ((h << 5) + h) ^ body_text.charCodeAt(i);
  }
  const body_text_hash = (h >>> 0).toString(16);
  const body_text_sample = body_text.slice(0, 200);
  const err_re = /sorry,?\s*it.s not you|page not found|something went wrong|forbidden|unauthori[sz]ed|access denied/i;
  const error_visible = err_re.test(body_text) || /\/(error|404)(\?|$|\/)/.test(url);
  return JSON.stringify({ url, title, modal_count, body_text_hash, body_text_sample, error_visible });
})()
