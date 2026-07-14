/**
 * JS Library Tests: Element Finding
 * Tests findAll, scoped find, semantic selectors, and locator chaining.
 *
 * Uses a local HTTP server — no external network dependencies.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const http = require('http');

const { browser } = require('../../../clients/javascript/dist');

// --- Local test server ---

let server, baseURL, bro;

const PAGE_HTML = `<html><head><title>Elements Test</title></head><body>
  <p>First paragraph with some text.</p>
  <p>Second paragraph with more content.</p>
  <p><a href="/other">Learn more about testing</a></p>
</body></html>`;

before(async () => {
  server = http.createServer((req, res) => {
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end(PAGE_HTML);
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

describe('Element Finding', () => {
  // --- findAll with CSS ---

  test('findAll returns multiple elements', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL);

    const paragraphs = await vibe.findAll('p');
    assert.ok(paragraphs.length > 0, 'Should find at least one paragraph');
  });

  test('findAll()[0] returns first element', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL);

    const paragraphs = await vibe.findAll('p');
    const first = paragraphs[0];
    assert.ok(first, 'Should return first element');
    assert.ok(first.info.tag === 'p', 'First element should be a <p>');
  });

  test('findAll().at(-1) returns last element', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL);

    const paragraphs = await vibe.findAll('p');
    const last = paragraphs.at(-1);
    assert.ok(last, 'Should return last element');
    assert.ok(last.info.tag === 'p', 'Last element should be a <p>');
  });

  test('findAll()[0] returns element at index', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL);

    const paragraphs = await vibe.findAll('p');
    const zeroth = paragraphs[0];
    assert.ok(zeroth, 'Should return element at index 0');
    assert.ok(zeroth.info.tag === 'p', 'Element at index 0 should be a <p>');
  });

  test('findAll().length returns number', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL);

    const paragraphs = await vibe.findAll('p');
    const count = paragraphs.length;
    assert.ok(typeof count === 'number', 'length should return a number');
    assert.ok(count > 0, 'length should be > 0');
  });

  // --- Scoped find ---

  test('element.find() scoped to parent', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL);

    const body = await vibe.find('body');
    assert.ok(body, 'Should find body');

    const nested = await body.find('a');
    assert.ok(nested, 'Should find nested <a> inside body');
    assert.ok(nested.info.tag === 'a', 'Nested element should be an <a>');
  });

  // --- Semantic selectors ---

  test('find({ role: "link" }) finds a link', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL);

    const link = await vibe.find({ role: 'link' });
    assert.ok(link, 'Should find element with role=link');
    assert.ok(link.info.tag === 'a', 'Element with role=link should be an <a>');
  });

  test('find({ text: "Learn more" }) finds element by text', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL);

    const el = await vibe.find({ text: 'Learn more' });
    assert.ok(el, 'Should find element containing text');
    assert.ok(el.info.text.includes('Learn more'), 'Element text should contain "Learn more"');
  });

  test('find({ role: "link", text: "Learn" }) combo selector', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL);

    const link = await vibe.find({ role: 'link', text: 'Learn' });
    assert.ok(link, 'Should find link with matching text');
    assert.ok(link.info.tag === 'a', 'Element should be an <a>');
    assert.ok(link.info.text.includes('Learn'), 'Element text should include "Learn"');
  });

  // --- Iterator ---

  test('findAll result is iterable', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL);

    const paragraphs = await vibe.findAll('p');
    let count = 0;
    for (const el of paragraphs) {
      assert.ok(el.info.tag === 'p', 'Each iterated element should be a <p>');
      count++;
    }
    assert.ok(count > 0, 'Should iterate over at least one element');
    assert.strictEqual(count, paragraphs.length, 'Iterator count should match length');
  });
});
