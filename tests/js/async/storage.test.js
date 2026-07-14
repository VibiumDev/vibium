/**
 * JS Library Tests: Storage
 * Tests context.storage, context.setStorage, context.clearStorage.
 *
 * Uses a local HTTP server — no external network dependencies.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const http = require('http');

const { browser } = require('../../../clients/javascript/dist');

// --- Local test server ---

let server;
let baseURL;
let bro;

before(async () => {
  server = http.createServer((req, res) => {
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end('<html><head><title>Storage Test</title></head><body>Hello</body></html>');
  });

  await new Promise((resolve) => {
    server.listen(0, '127.0.0.1', () => {
      const { port } = server.address();
      baseURL = `http://127.0.0.1:${port}`;
      resolve();
    });
  });

  bro = await browser.start({ headless: true });
});

after(async () => {
  if (bro) await bro.stop();
  if (server) server.close();
});

// --- context.storage() ---

describe('Storage: context.storage()', () => {
  test('storage() returns cookies + localStorage', async () => {
    // Share top-level browser; tests use isolated contexts.
    try {
      const ctx = await bro.newContext();
      const vibe = await ctx.newPage();
      await vibe.go(baseURL);

      // Set a cookie and localStorage item
      await ctx.setCookies([
        { name: 'state_cookie', value: 'state_val', domain: '127.0.0.1' },
      ]);
      await vibe.evaluate('localStorage.setItem("key1", "value1")');

      const state = await ctx.storage();

      // Check cookies
      assert.ok(Array.isArray(state.cookies), 'storage should have cookies array');
      const stateCookie = state.cookies.find(c => c.name === 'state_cookie');
      assert.ok(stateCookie, 'Should find state_cookie in storage');

      // Check origins
      assert.ok(Array.isArray(state.origins), 'storage should have origins array');
      assert.ok(state.origins.length > 0, 'Should have at least one origin');

      const origin = state.origins[0];
      assert.ok(origin.origin, 'Origin should have an origin field');
      assert.ok(Array.isArray(origin.localStorage), 'Origin should have localStorage array');

      const lsItem = origin.localStorage.find(item => item.name === 'key1');
      assert.ok(lsItem, 'Should find key1 in localStorage');
      assert.strictEqual(lsItem.value, 'value1');

      await ctx.close();
    } finally {

    }
  });
});

// --- context.setStorage() ---

describe('Storage: context.setStorage()', () => {
  test('setStorage() sets cookies and localStorage from state object', async () => {
    // Share top-level browser; tests use isolated contexts.
    try {
      const ctx = await bro.newContext();
      const vibe = await ctx.newPage();
      await vibe.go(baseURL);

      // Set storage from a manually constructed state object
      await ctx.setStorage({
        cookies: [
          { name: 'set_cookie', value: 'set_val', domain: '127.0.0.1', path: '/' },
        ],
        origins: [
          {
            origin: baseURL,
            localStorage: [{ name: 'set_key', value: 'set_value' }],
            sessionStorage: [],
          },
        ],
      });

      // Verify cookies were set
      const cookies = await ctx.cookies();
      const setCookie = cookies.find(c => c.name === 'set_cookie');
      assert.ok(setCookie, 'Should find set_cookie');
      assert.strictEqual(setCookie.value, 'set_val');

      // Verify localStorage was set
      const lsVal = await vibe.evaluate('localStorage.getItem("set_key")');
      assert.strictEqual(lsVal, 'set_value');

      await ctx.close();
    } finally {

    }
  });

  test('setStorage() round-trip: capture → clear → restore → verify', async () => {
    // Share top-level browser; tests use isolated contexts.
    try {
      const ctx = await bro.newContext();
      const vibe = await ctx.newPage();
      await vibe.go(baseURL);

      // Set up initial state
      await ctx.setCookies([
        { name: 'rt_cookie', value: 'rt_val', domain: '127.0.0.1' },
      ]);
      await vibe.evaluate('localStorage.setItem("rt_key", "rt_value")');

      // Capture state
      const state = await ctx.storage();
      assert.ok(state.cookies.length > 0, 'Should have cookies');

      // Clear everything
      await ctx.clearStorage();
      const cleared = await ctx.storage();
      assert.strictEqual(cleared.cookies.length, 0, 'Cookies should be cleared');

      // Strip read-only fields from cookies before restoring
      const cleanState = {
        cookies: state.cookies.map(c => ({
          name: c.name,
          value: c.value,
          domain: c.domain,
          path: c.path,
        })),
        origins: state.origins,
      };

      // Restore state
      await ctx.setStorage(cleanState);

      // Verify restored cookies
      const restored = await ctx.cookies();
      const restoredCookie = restored.find(c => c.name === 'rt_cookie');
      assert.ok(restoredCookie, `Should find restored cookie, got: ${restored.map(c => c.name).join(', ')}`);
      assert.strictEqual(restoredCookie.value, 'rt_val');

      // Verify restored localStorage
      const lsVal = await vibe.evaluate('localStorage.getItem("rt_key")');
      assert.strictEqual(lsVal, 'rt_value');

      await ctx.close();
    } finally {

    }
  });
});

// --- context.clearStorage() ---

describe('Storage: context.clearStorage()', () => {
  test('clearStorage() clears cookies + localStorage + sessionStorage', async () => {
    // Share top-level browser; tests use isolated contexts.
    try {
      const ctx = await bro.newContext();
      const vibe = await ctx.newPage();
      await vibe.go(baseURL);

      // Set up state
      await ctx.setCookies([
        { name: 'clear_cookie', value: 'val', domain: '127.0.0.1' },
      ]);
      await vibe.evaluate('localStorage.setItem("clear_key", "val")');
      await vibe.evaluate('sessionStorage.setItem("clear_ss", "val")');

      // Verify state exists
      const before = await ctx.storage();
      assert.ok(before.cookies.length > 0, 'Should have cookies before clear');

      // Clear all storage
      await ctx.clearStorage();

      // Verify everything is cleared
      const after = await ctx.storage();
      assert.strictEqual(after.cookies.length, 0, 'Cookies should be cleared');
      if (after.origins.length > 0) {
        assert.strictEqual(after.origins[0].localStorage.length, 0, 'localStorage should be cleared');
        assert.strictEqual(after.origins[0].sessionStorage.length, 0, 'sessionStorage should be cleared');
      }

      await ctx.close();
    } finally {

    }
  });
});
