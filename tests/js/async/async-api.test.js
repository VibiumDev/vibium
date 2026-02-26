/**
 * JS Library Tests: Async API
 * Tests browser.launch() and Browser → Page methods.
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

const HOME_HTML = `<html><head><title>Test App</title></head><body>
  <h1 class="heading">Welcome to test-app</h1>
  <a href="/subpage">Go to subpage</a>
  <a href="/inputs">Inputs</a>
</body></html>`;

const SUBPAGE_HTML = `<html><head><title>Subpage</title></head><body>
  <h3>Add/Remove Elements</h3>
</body></html>`;

const INPUTS_HTML = `<html><head><title>Inputs</title></head><body>
  <input type="text" />
</body></html>`;

before(async () => {
  server = http.createServer((req, res) => {
    res.writeHead(200, { 'Content-Type': 'text/html' });
    if (req.url === '/subpage') {
      res.end(SUBPAGE_HTML);
    } else if (req.url === '/inputs') {
      res.end(INPUTS_HTML);
    } else {
      res.end(HOME_HTML);
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

// --- Tests ---

describe('JS Async API', () => {
  let bro;
  before(async () => {
    bro = await browser.launch({ headless: true });
  });
  after(async () => {
    await bro.close();
  });

  test('browser.launch() returns a Browser instance', async () => {
    assert.ok(bro, 'Should return a Browser instance');
  });

  test('page.go() navigates to URL', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL);
    assert.ok(true);
  });

  test('page.screenshot() returns PNG buffer', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL);
    const screenshot = await vibe.screenshot();

    assert.ok(Buffer.isBuffer(screenshot), 'Should return a Buffer');
    assert.ok(screenshot.length > 100, 'Screenshot should have reasonable size');

    // Check PNG magic bytes
    assert.strictEqual(screenshot[0], 0x89, 'Should be valid PNG');
    assert.strictEqual(screenshot[1], 0x50, 'Should be valid PNG');
    assert.strictEqual(screenshot[2], 0x4E, 'Should be valid PNG');
    assert.strictEqual(screenshot[3], 0x47, 'Should be valid PNG');
  });

  test('page.evaluate() executes JavaScript', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL);
    const title = await vibe.evaluate('return document.title');
    assert.match(title, /Test App/i, 'Should return page title');
  });

  test('page.find() locates element', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL);
    const heading = await vibe.find('h1.heading');

    assert.ok(heading, 'Should return an Element');
    assert.ok(heading.info, 'Element should have info');
    assert.match(heading.info.tag, /^h1$/i, 'Should be an h1 tag');
    assert.match(heading.info.text, /Welcome to test-app/i, 'Should have heading text');
  });

  test('element.click() works', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL);
    const link = await vibe.find('a[href="/subpage"]');
    await link.click();

    const heading = await vibe.find('h3');
    assert.match(heading.info.text, /Add\/Remove Elements/i, 'Should have navigated to new page');
  });

  test('element.type() enters text', async () => {
    const vibe = await bro.page();
    await vibe.go(`${baseURL}/inputs`);
    const input = await vibe.find('input');
    await input.type('12345');

    const value = await vibe.evaluate(`
      return document.querySelector('input').value;
    `);
    assert.strictEqual(value, '12345', 'Input should have typed value');
  });

  test('element.text() returns element text', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL);
    const heading = await vibe.find('h1.heading');
    const text = await heading.text();
    assert.match(text, /Welcome to test-app/i, 'Should return heading text');
  });
});
