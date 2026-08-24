// Dismiss the most common cookie/consent banners so a click loop can run.
// Strategy ladder: prefer clicking the visible button (works on most vendors).
// Fall through to vendor-specific APIs only when nothing visible matches.
// Force-hide CSS as last resort if a banner persists.
// Returns JSON: { dismissed: bool, vendor: string|null, attempts: [...] }

(() => {
  const attempts = [];

  const isVisible = (el) => {
    if (!el) return false;
    const r = el.getBoundingClientRect();
    if (r.width === 0 || r.height === 0) return false;
    const s = getComputedStyle(el);
    if (s.visibility === 'hidden' || s.display === 'none' || parseFloat(s.opacity) === 0) return false;
    return true;
  };

  const bannerStillUp = () => {
    const re = /\b(cookie|consent|privacy|tracking|gdpr)\b/i;
    return [...document.querySelectorAll('*')].some(el => {
      if (!isVisible(el)) return false;
      const s = getComputedStyle(el);
      const z = parseInt(s.zIndex, 10);
      if (!(s.position === 'fixed' || s.position === 'sticky') || isNaN(z) || z < 100) return false;
      if (el.offsetHeight < 100 || el.offsetWidth < 200) return false;
      const txt = (el.innerText || '').slice(0, 1000);
      return re.test(txt);
    });
  };

  // Strategy A: visible button text-match (most reliable across vendors)
  const TEXT_RE = /^(accept(\s| )+all(\s| )*(cookies?)?|accept(\s| )+cookies?|allow(\s| )+all|i(\s| )+agree|got(\s| )+it|consent|i(\s| )+understand|continue|ok(ay)?)$/i;
  const allClickable = [...document.querySelectorAll('button, a[role=button], [role=button], [data-testid*=accept i], [aria-label*=accept i]')];
  const visibleAccept = allClickable.find(el => {
    if (!isVisible(el)) return false;
    const t = (el.innerText || el.getAttribute('aria-label') || '').replace(/\s+/g, ' ').trim();
    return TEXT_RE.test(t);
  });
  if (visibleAccept) {
    visibleAccept.click();
    attempts.push({ strategy: 'visible-text', text: visibleAccept.innerText.slice(0, 60), clicked: true });
    return JSON.stringify({ dismissed: true, vendor: 'visible-text', attempts });
  }
  attempts.push({ strategy: 'visible-text', clicked: false });

  // Strategy B: vendor-specific selectors
  const ot = document.querySelector('#onetrust-accept-btn-handler');
  if (ot && isVisible(ot)) {
    ot.click();
    attempts.push({ strategy: 'onetrust-button', clicked: true });
    return JSON.stringify({ dismissed: true, vendor: 'onetrust', attempts });
  }
  const cb = document.querySelector('#CybotCookiebotDialogBodyButtonAccept, #CybotCookiebotDialogBodyLevelButtonAccept');
  if (cb && isVisible(cb)) {
    cb.click();
    attempts.push({ strategy: 'cookiebot-button', clicked: true });
    return JSON.stringify({ dismissed: true, vendor: 'cookiebot', attempts });
  }
  const ucHost = document.querySelector('uc-consent-banner, uc-consent-modal, #usercentrics-root, [id*=usercentrics i]');
  if (ucHost && ucHost.shadowRoot) {
    const ucBtn = ucHost.shadowRoot.querySelector('[data-testid*=accept-all i], button[aria-label*=accept i]');
    if (ucBtn) {
      ucBtn.click();
      attempts.push({ strategy: 'usercentrics-shadow', clicked: true });
      return JSON.stringify({ dismissed: true, vendor: 'usercentrics-shadow', attempts });
    }
  }
  try {
    if (window.UC_UI && typeof window.UC_UI.acceptAllConsents === 'function') {
      window.UC_UI.acceptAllConsents();
      attempts.push({ strategy: 'usercentrics-api', called: true });
      if (!bannerStillUp()) {
        return JSON.stringify({ dismissed: true, vendor: 'usercentrics-api', attempts });
      }
      attempts.push({ strategy: 'usercentrics-api', note: 'API set consent but banner still visible' });
    }
  } catch (e) {
    attempts.push({ strategy: 'usercentrics-api', error: String(e).slice(0, 80) });
  }

  // Strategy C: walk every shadow root looking for an accept-button
  const allElements = document.querySelectorAll('*');
  for (const el of allElements) {
    if (!el.shadowRoot) continue;
    const btn = el.shadowRoot.querySelector('button[aria-label*=accept i], button[id*=accept i], [data-testid*=accept i]');
    if (btn && isVisible(btn)) {
      btn.click();
      attempts.push({ strategy: 'shadow-walk', host: el.tagName, clicked: true });
      return JSON.stringify({ dismissed: true, vendor: 'shadow-walk', attempts });
    }
  }
  attempts.push({ strategy: 'shadow-walk', clicked: false });

  // Strategy D: force-hide CSS as last resort. Does not record consent;
  // just unblocks the UI so the click loop can proceed.
  if (bannerStillUp()) {
    const overlays = [...document.querySelectorAll('*')].filter(el => {
      if (!isVisible(el)) return false;
      const s = getComputedStyle(el);
      const z = parseInt(s.zIndex, 10);
      if (!(s.position === 'fixed' || s.position === 'sticky') || isNaN(z) || z < 100) return false;
      if (el.offsetHeight < 100 || el.offsetWidth < 200) return false;
      const txt = (el.innerText || '').slice(0, 1000);
      return /\b(cookie|consent|privacy|tracking|gdpr)\b/i.test(txt);
    });
    overlays.forEach(el => { el.style.setProperty('display', 'none', 'important'); });
    if (overlays.length > 0) {
      attempts.push({ strategy: 'force-hide-css', count: overlays.length });
      return JSON.stringify({ dismissed: true, vendor: 'force-hide-css', attempts });
    }
  }

  return JSON.stringify({ dismissed: false, vendor: null, attempts });
})()
