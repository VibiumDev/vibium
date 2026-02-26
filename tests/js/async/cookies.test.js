/**
 * JS Library Tests: Cookies & Storage
 * Tests context.cookies, context.setCookies, context.clearCookies,
 * context.storageState, and context.addInitScript.
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

before(async () => {
  server = http.createServer((req, res) => {
    if (req.url === '/set-cookie') {
      res.writeHead(200, {
        'Content-Type': 'text/html',
        'Set-Cookie': [
          'server_cookie=hello; Path=/',
          'another_cookie=world; Path=/sub',
        ],
      });
      res.end('<html><body>Cookies set</body></html>');
    } else if (req.url === '/sub/page') {
      res.writeHead(200, { 'Content-Type': 'text/html' });
      res.end('<html><body>Sub page</body></html>');
    } else {
      res.writeHead(200, { 'Content-Type': 'text/html' });
      res.end('<html><head><title>Cookie Test</title></head><body>Hello</body></html>');
    }
  });

  await new Promise((resolve) => {
    server.listen(0, '127.0.0.1', () => {
      const { port } = server.address();
      baseURL = `http://127.0.0.1:${port}`;
      resolve();
    });
  });
});

after(() => {
  if (server) server.close();
});

// --- context.cookies() ---

describe('Cookies: context.cookies()', () => {
  test('cookies() returns server-set cookies', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const ctx = await bro.newContext();
      const vibe = await ctx.newPage();
      await vibe.go(`${baseURL}/set-cookie`);

      const cookies = await ctx.cookies();
      assert.ok(Array.isArray(cookies), 'cookies() should return an array');
      assert.ok(cookies.length >= 1, `Should have at least 1 cookie, got ${cookies.length}`);

      const serverCookie = cookies.find(c => c.name === 'server_cookie');
      assert.ok(serverCookie, 'Should find server_cookie');
      assert.strictEqual(serverCookie.value, 'hello');
      assert.strictEqual(serverCookie.path, '/');

      await ctx.close();
    } finally {
      await bro.close();
    }
  });

  test('cookies(urls) filters by URL', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const ctx = await bro.newContext();
      const vibe = await ctx.newPage();
      await vibe.go(`${baseURL}/set-cookie`);

      // Filter to /sub path — should include another_cookie but not server_cookie (path /)
      // Actually both match because server_cookie has path / which matches everything
      const subCookies = await ctx.cookies([`${baseURL}/sub/page`]);
      assert.ok(Array.isArray(subCookies), 'cookies(urls) should return an array');

      // server_cookie with path "/" should match /sub/page
      const serverCookie = subCookies.find(c => c.name === 'server_cookie');
      assert.ok(serverCookie, 'server_cookie with path / should match /sub/page');

      // another_cookie with path "/sub" should match /sub/page
      const anotherCookie = subCookies.find(c => c.name === 'another_cookie');
      assert.ok(anotherCookie, 'another_cookie with path /sub should match /sub/page');

      await ctx.close();
    } finally {
      await bro.close();
    }
  });
});

// --- context.setCookies() ---

describe('Cookies: context.setCookies()', () => {
  test('setCookies() creates readable cookies', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const ctx = await bro.newContext();
      const vibe = await ctx.newPage();
      await vibe.go(baseURL);

      await ctx.setCookies([
        { name: 'test', value: 'val', domain: '127.0.0.1' },
      ]);

      const cookies = await ctx.cookies();
      const testCookie = cookies.find(c => c.name === 'test');
      assert.ok(testCookie, 'Should find the test cookie');
      assert.strictEqual(testCookie.value, 'val');

      await ctx.close();
    } finally {
      await bro.close();
    }
  });

  test('setCookies() with url (no domain) works', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const ctx = await bro.newContext();
      const vibe = await ctx.newPage();
      await vibe.go(baseURL);

      await ctx.setCookies([
        { name: 'url_cookie', value: 'from_url', url: baseURL },
      ]);

      const cookies = await ctx.cookies();
      const urlCookie = cookies.find(c => c.name === 'url_cookie');
      assert.ok(urlCookie, 'Should find the url_cookie');
      assert.strictEqual(urlCookie.value, 'from_url');

      await ctx.close();
    } finally {
      await bro.close();
    }
  });
});

// --- context.clearCookies() ---

describe('Cookies: context.clearCookies()', () => {
  test('clearCookies() removes all cookies', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const ctx = await bro.newContext();
      const vibe = await ctx.newPage();
      await vibe.go(`${baseURL}/set-cookie`);

      // Verify cookies exist
      let cookies = await ctx.cookies();
      assert.ok(cookies.length > 0, 'Should have cookies before clearing');

      // Clear
      await ctx.clearCookies();

      // Verify cleared
      cookies = await ctx.cookies();
      assert.strictEqual(cookies.length, 0, `Should have no cookies after clearing, got ${cookies.length}`);

      await ctx.close();
    } finally {
      await bro.close();
    }
  });
});

// --- context.storageState() ---

describe('Storage: context.storageState()', () => {
  test('storageState() returns cookies + localStorage', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const ctx = await bro.newContext();
      const vibe = await ctx.newPage();
      await vibe.go(baseURL);

      // Set a cookie and localStorage item
      await ctx.setCookies([
        { name: 'state_cookie', value: 'state_val', domain: '127.0.0.1' },
      ]);
      await vibe.eval('localStorage.setItem("key1", "value1")');

      const state = await ctx.storageState();

      // Check cookies
      assert.ok(Array.isArray(state.cookies), 'storageState should have cookies array');
      const stateCookie = state.cookies.find(c => c.name === 'state_cookie');
      assert.ok(stateCookie, 'Should find state_cookie in storageState');

      // Check origins
      assert.ok(Array.isArray(state.origins), 'storageState should have origins array');
      assert.ok(state.origins.length > 0, 'Should have at least one origin');

      const origin = state.origins[0];
      assert.ok(origin.origin, 'Origin should have an origin field');
      assert.ok(Array.isArray(origin.localStorage), 'Origin should have localStorage array');

      const lsItem = origin.localStorage.find(item => item.name === 'key1');
      assert.ok(lsItem, 'Should find key1 in localStorage');
      assert.strictEqual(lsItem.value, 'value1');

      await ctx.close();
    } finally {
      await bro.close();
    }
  });
});

// --- context.addInitScript() ---

describe('Init Scripts: context.addInitScript()', () => {
  test('addInitScript() runs before page scripts', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const ctx = await bro.newContext();

      await ctx.addInitScript('window.__injected = "hello from init"');

      const vibe = await ctx.newPage();
      await vibe.go(baseURL);

      const value = await vibe.eval('window.__injected');
      assert.strictEqual(value, 'hello from init');

      await ctx.close();
    } finally {
      await bro.close();
    }
  });

  test('addInitScript() persists across navigations', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const ctx = await bro.newContext();

      await ctx.addInitScript('window.__persistent = 42');

      const vibe = await ctx.newPage();
      await vibe.go(baseURL);

      let value = await vibe.eval('window.__persistent');
      assert.strictEqual(value, 42);

      // Navigate again
      await vibe.go(`${baseURL}/sub/page`);

      value = await vibe.eval('window.__persistent');
      assert.strictEqual(value, 42);

      await ctx.close();
    } finally {
      await bro.close();
    }
  });
});

// --- Checkpoint ---

describe('Cookies & Storage Checkpoint', () => {
  test('setCookies + cookies + clearCookies round-trip', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const ctx = await bro.newContext();
      const vibe = await ctx.newPage();
      await vibe.go(baseURL);

      await ctx.setCookies([{ name: 'test', value: 'val', url: baseURL }]);
      const cookies = await ctx.cookies();
      assert.ok(cookies.some(c => c.name === 'test'), 'Should find test cookie');

      await ctx.clearCookies();
      const after = await ctx.cookies();
      assert.ok(!after.some(c => c.name === 'test'), 'test cookie should be cleared');

      await ctx.close();
    } finally {
      await bro.close();
    }
  });
});
