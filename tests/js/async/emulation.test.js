/**
 * JS Library Tests: Emulation
 * Tests page.setViewport, page.viewport, page.emulateMedia,
 * page.setContent, page.setGeolocation
 */

const { test, describe, after } = require('node:test');
const assert = require('node:assert');

const { browser } = require('../../../clients/javascript/dist');

describe('JS Emulation', () => {
  let bro;

  test('setup', async () => {
    bro = await browser.launch({ headless: true });
  });

  after(async () => {
    if (bro) await bro.close().catch(() => {});
  });

  // --- setViewport / viewport ---

  test('viewport() returns current size', async () => {
    const vibe = await bro.page();
    const size = await vibe.viewport();
    assert.ok(typeof size.width === 'number' && size.width > 0, `width should be > 0, got ${size.width}`);
    assert.ok(typeof size.height === 'number' && size.height > 0, `height should be > 0, got ${size.height}`);
  });

  test('setViewport() changes viewport size', async () => {
    const vibe = await bro.page();
    await vibe.setViewport({ width: 800, height: 600 });
    const size = await vibe.viewport();
    assert.strictEqual(size.width, 800, `width should be 800, got ${size.width}`);
    assert.strictEqual(size.height, 600, `height should be 600, got ${size.height}`);
  });

  // --- setContent ---

  test('setContent() replaces page HTML', async () => {
    const vibe = await bro.page();
    await vibe.setContent('<html><body><h1>Hello Vibium</h1></body></html>');
    const el = await vibe.find('h1');
    const text = await el.text();
    assert.strictEqual(text, 'Hello Vibium', `h1 text should be "Hello Vibium", got "${text}"`);
  });

  test('setContent() with full document including title', async () => {
    const vibe = await bro.page();
    await vibe.setContent('<!DOCTYPE html><html><head><title>Custom Title</title></head><body><p>content</p></body></html>');
    const title = await vibe.title();
    assert.strictEqual(title, 'Custom Title', `title should be "Custom Title", got "${title}"`);
  });

  // --- emulateMedia ---

  test('emulateMedia({ colorScheme: "dark" })', async () => {
    const vibe = await bro.page();
    await vibe.setContent('<html><body></body></html>');
    await vibe.emulateMedia({ colorScheme: 'dark' });
    const matches = await vibe.eval('window.matchMedia("(prefers-color-scheme: dark)").matches');
    assert.strictEqual(matches, true, 'prefers-color-scheme: dark should match');
  });

  test('emulateMedia({ colorScheme: "light" })', async () => {
    const vibe = await bro.page();
    await vibe.setContent('<html><body></body></html>');
    await vibe.emulateMedia({ colorScheme: 'light' });
    const matches = await vibe.eval('window.matchMedia("(prefers-color-scheme: light)").matches');
    assert.strictEqual(matches, true, 'prefers-color-scheme: light should match');
    const darkMatches = await vibe.eval('window.matchMedia("(prefers-color-scheme: dark)").matches');
    assert.strictEqual(darkMatches, false, 'prefers-color-scheme: dark should NOT match');
  });

  test('emulateMedia({ media: "print" })', async () => {
    const vibe = await bro.page();
    await vibe.setContent('<html><body></body></html>');
    await vibe.emulateMedia({ media: 'print' });
    const matches = await vibe.eval('window.matchMedia("print").matches');
    assert.strictEqual(matches, true, 'print media should match');
  });

  test('emulateMedia({ reducedMotion: "reduce" })', async () => {
    const vibe = await bro.page();
    await vibe.setContent('<html><body></body></html>');
    await vibe.emulateMedia({ reducedMotion: 'reduce' });
    const matches = await vibe.eval('window.matchMedia("(prefers-reduced-motion: reduce)").matches');
    assert.strictEqual(matches, true, 'prefers-reduced-motion: reduce should match');
  });

  test('emulateMedia({ forcedColors: "active" })', async () => {
    const vibe = await bro.page();
    await vibe.setContent('<html><body></body></html>');
    await vibe.emulateMedia({ forcedColors: 'active' });
    const matches = await vibe.eval('window.matchMedia("(forced-colors: active)").matches');
    assert.strictEqual(matches, true, 'forced-colors: active should match');
  });

  test('emulateMedia({ contrast: "more" })', async () => {
    const vibe = await bro.page();
    await vibe.setContent('<html><body></body></html>');
    await vibe.emulateMedia({ contrast: 'more' });
    const matches = await vibe.eval('window.matchMedia("(prefers-contrast: more)").matches');
    assert.strictEqual(matches, true, 'prefers-contrast: more should match');
  });

  test('emulateMedia(null) resets overrides', async () => {
    const vibe = await bro.page();
    await vibe.setContent('<html><body></body></html>');

    // Set override
    await vibe.emulateMedia({ colorScheme: 'dark' });
    let matches = await vibe.eval('window.matchMedia("(prefers-color-scheme: dark)").matches');
    assert.strictEqual(matches, true, 'dark should match after setting');

    // Reset
    await vibe.emulateMedia({ colorScheme: null });
    // After reset, the query should use browser default (which may or may not be dark)
    // The key test is that the override was removed — query passthrough to native matchMedia
    const result = await vibe.eval('typeof window.__vibiumMediaOverrides.colorScheme');
    assert.strictEqual(result, 'undefined', 'colorScheme override should be removed');
  });

  // --- setWindow / window ---

  test('window() returns current state and dimensions', async () => {
    const vibe = await bro.page();
    const win = await vibe.window();
    assert.ok(typeof win.state === 'string', `state should be a string, got ${typeof win.state}`);
    assert.ok(typeof win.width === 'number' && win.width > 0, `width should be > 0, got ${win.width}`);
    assert.ok(typeof win.height === 'number' && win.height > 0, `height should be > 0, got ${win.height}`);
    assert.ok(typeof win.x === 'number', `x should be a number, got ${typeof win.x}`);
    assert.ok(typeof win.y === 'number', `y should be a number, got ${typeof win.y}`);
  });

  test('setWindow() resizes the window', async () => {
    const vibe = await bro.page();
    await vibe.setWindow({ width: 900, height: 700 });
    const win = await vibe.window();
    assert.strictEqual(win.width, 900, `width should be 900, got ${win.width}`);
    assert.strictEqual(win.height, 700, `height should be 700, got ${win.height}`);
  });

  test('setWindow() moves the window', async () => {
    const vibe = await bro.page();
    await vibe.setWindow({ x: 50, y: 50, width: 800, height: 600 });
    const win = await vibe.window();
    assert.strictEqual(win.x, 50, `x should be 50, got ${win.x}`);
    assert.strictEqual(win.y, 50, `y should be 50, got ${win.y}`);
  });

  test('setWindow({ state: "maximized" }) maximizes', async () => {
    const vibe = await bro.page();
    await vibe.setWindow({ state: 'maximized' });
    const win = await vibe.window();
    assert.strictEqual(win.state, 'maximized', `state should be "maximized", got "${win.state}"`);
  });

  // --- setGeolocation ---

  test('setGeolocation() overrides position', async () => {
    const vibe = await bro.page();
    await vibe.setContent('<html><body></body></html>');
    await vibe.setGeolocation({ latitude: 51.5074, longitude: -0.1278 });

    const coords = await vibe.eval(`
      new Promise((resolve, reject) => {
        navigator.geolocation.getCurrentPosition(
          pos => resolve({ lat: pos.coords.latitude, lng: pos.coords.longitude }),
          err => reject(err),
          { timeout: 5000 }
        );
      })
    `);

    assert.ok(Math.abs(coords.lat - 51.5074) < 0.001, `latitude should be ~51.5074, got ${coords.lat}`);
    assert.ok(Math.abs(coords.lng - (-0.1278)) < 0.001, `longitude should be ~-0.1278, got ${coords.lng}`);
  });
});
