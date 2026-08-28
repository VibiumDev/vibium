// Aesthetic probe — extracts design tokens from a rendered page.
// Invocation: vibium eval --stdin < probe.js
// Output: JSON describing palette, typography, spacing, composition signals.

(() => {
  function rgb(s) {
    if (!s) return null;
    const m = s.match(/rgba?\(([^)]+)\)/);
    if (!m) return s;
    const parts = m[1].split(",").map((v) => parseFloat(v));
    return parts.length >= 3
      ? { r: parts[0], g: parts[1], b: parts[2], a: parts[3] ?? 1 }
      : s;
  }
  function toHex({ r, g, b }) {
    const h = (n) => Math.round(n).toString(16).padStart(2, "0");
    return `#${h(r)}${h(g)}${h(b)}`;
  }
  function frequency(arr) {
    const m = new Map();
    for (const v of arr) m.set(v, (m.get(v) || 0) + 1);
    return [...m.entries()]
      .sort((a, b) => b[1] - a[1])
      .map(([value, count]) => ({ value, count }));
  }

  const all = Array.from(document.querySelectorAll("*"));
  const visible = all.filter((el) => {
    const r = el.getBoundingClientRect();
    return r.width > 0 && r.height > 0 && el.offsetParent !== null;
  });

  const palette = [];
  const fonts = [];
  const sizes = [];
  const weights = [];
  const lineHeights = [];
  const letterSpacings = [];
  const radii = [];
  const shadows = [];
  const paddings = [];
  const gaps = [];

  for (const el of visible) {
    const s = getComputedStyle(el);
    const txt = (el.textContent || "").trim();
    if (s.color && s.color !== "rgba(0, 0, 0, 0)") palette.push(s.color);
    if (s.backgroundColor && s.backgroundColor !== "rgba(0, 0, 0, 0)")
      palette.push(s.backgroundColor);
    if (
      s.borderTopColor &&
      s.borderTopColor !== "rgba(0, 0, 0, 0)" &&
      parseFloat(s.borderTopWidth) > 0
    )
      palette.push(s.borderTopColor);
    if (s.fontFamily) fonts.push(s.fontFamily.split(",")[0].trim().replace(/['"]/g, ""));
    if (txt.length > 0) {
      sizes.push(s.fontSize);
      weights.push(s.fontWeight);
      lineHeights.push(s.lineHeight);
      letterSpacings.push(s.letterSpacing);
    }
    if (s.borderRadius && s.borderRadius !== "0px") radii.push(s.borderRadius);
    if (s.boxShadow && s.boxShadow !== "none") shadows.push(s.boxShadow);
    if (s.padding && s.padding !== "0px") paddings.push(s.padding);
    if (s.gap && s.gap !== "normal" && s.gap !== "0px") gaps.push(s.gap);
  }

  const paletteHex = palette
    .map(rgb)
    .filter((v) => v && typeof v === "object")
    .filter((v) => (v.a ?? 1) > 0)
    .map((v) => toHex(v));

  const heading = (selector) => {
    const els = Array.from(document.querySelectorAll(selector));
    return els.slice(0, 3).map((el) => {
      const s = getComputedStyle(el);
      return {
        text: (el.textContent || "").trim().slice(0, 80),
        fontFamily: s.fontFamily.split(",")[0].trim().replace(/['"]/g, ""),
        fontSize: s.fontSize,
        fontWeight: s.fontWeight,
        lineHeight: s.lineHeight,
        letterSpacing: s.letterSpacing,
        color: s.color,
      };
    });
  };

  const main = document.querySelector("main, body");
  const mainW = main ? main.getBoundingClientRect().width : window.innerWidth;
  const grain = getComputedStyle(document.body).backgroundImage;

  return JSON.stringify({
    url: location.href,
    title: document.title,
    viewport: {
      w: window.innerWidth,
      h: window.innerHeight,
      dpr: window.devicePixelRatio,
    },
    docSize: {
      scrollHeight: document.documentElement.scrollHeight,
      scrollWidth: document.documentElement.scrollWidth,
    },
    palette: {
      top: frequency(paletteHex).slice(0, 12),
      uniqueCount: new Set(paletteHex).size,
    },
    typography: {
      h1: heading("h1"),
      h2: heading("h2"),
      h3: heading("h3"),
      body: heading("p, li, span:not(:empty)").slice(0, 5),
      cta: heading("a[href], button"),
      fonts: frequency(fonts).slice(0, 6),
      sizes: frequency(sizes).slice(0, 12),
      weights: frequency(weights).slice(0, 6),
      lineHeights: frequency(lineHeights).slice(0, 6),
      letterSpacings: frequency(letterSpacings).slice(0, 6),
    },
    surface: {
      radii: frequency(radii).slice(0, 6),
      shadows: frequency(shadows).slice(0, 4),
      paddingSamples: frequency(paddings).slice(0, 10),
      gaps: frequency(gaps).slice(0, 6),
    },
    layout: {
      mainWidth: mainW,
      viewportWidth: window.innerWidth,
      bodyHasGrain: grain && grain !== "none" ? true : false,
    },
    composition: {
      sections: document.querySelectorAll("section").length,
      headings: {
        h1: document.querySelectorAll("h1").length,
        h2: document.querySelectorAll("h2").length,
        h3: document.querySelectorAll("h3").length,
      },
      ctas: document.querySelectorAll("a[href^='mailto:'], a[href^='#'], button").length,
      images: document.querySelectorAll("img").length,
      videos: document.querySelectorAll("video").length,
      canvases: document.querySelectorAll("canvas").length,
      svgs: document.querySelectorAll("svg").length,
    },
    meta: {
      description:
        document.querySelector("meta[name='description']")?.getAttribute("content") || null,
      ogImage:
        document.querySelector("meta[property='og:image']")?.getAttribute("content") || null,
      ogDescription:
        document.querySelector("meta[property='og:description']")?.getAttribute("content") ||
        null,
      lang: document.documentElement.lang,
    },
  });
})();
