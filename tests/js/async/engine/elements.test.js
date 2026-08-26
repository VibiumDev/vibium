/**
 * JS Library Tests: Element Finding
 * Tests findAll, scoped find, semantic selectors, and locator chaining.
 *
 * Uses a local HTTP server — no external network dependencies.
 */

const { test, describe, before, after } = require("../../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const http = require('http');

const { browser } = require('../../../../clients/javascript/dist');

// --- Local test server ---

let server, baseURL, bro;

const PAGE_HTML = `<html><head><title>Elements Test</title></head><body>
  <p>First paragraph with some text.</p>
  <p>Second paragraph with more content.</p>
  <p><a href="/other">Learn more about testing</a></p>
  <input type="submit" value="Login" />
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

  test('findAll returns an empty array when nothing matches (#411)', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL);

    // timeout 0 means a single immediate check, so "none" comes back fast
    // instead of after the default 30s wait.
    const start = Date.now();
    const none = await vibe.findAll('#definitely-not-on-this-page', { timeout: 0 });
    assert.deepStrictEqual(none, [], 'Should resolve to an empty array, not throw');
    assert.ok(Date.now() - start < 5000, 'timeout 0 should not wait out the default');
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

  test('find({ role: "button", label: "Login" }) matches a submit input by value (#204)', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL);

    // <input type="submit"> has no textContent; per HTML-AAM its accessible
    // name comes from the value attribute.
    const el = await vibe.find({ role: 'button', label: 'Login' });
    assert.ok(el, 'Should find the submit input by its accessible name');
    assert.strictEqual(el.info.tag, 'input');
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

describe('Element highlight', () => {
  test('highlight() outlines the element (#435 drift find)', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL);

    const el = await vibe.find('p');
    await el.highlight();

    const outline = await vibe.evaluate('document.querySelector("p").style.outline');
    assert.match(outline, /solid/, `highlight should set an outline, got "${outline}"`);
  });
});

describe('Stale findAll handles (#338)', () => {
  test('the info snapshot reads all matches with no extra round trips', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL);

    const paragraphs = await vibe.findAll('p');
    const texts = paragraphs.map((el) => el.info.text);
    assert.strictEqual(texts.length, 3);
    assert.match(texts[0], /First paragraph/);
    // The snapshot stays readable even after the elements are gone
    await vibe.evaluate('document.querySelectorAll("p").forEach(p => p.remove())');
    assert.match(paragraphs[1].info.text, /Second paragraph/);
  });

  test('a live read on a vanished handle names findAll, a bad selector does not', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL);

    const paragraphs = await vibe.findAll('p');
    await vibe.evaluate('document.querySelectorAll("p").forEach(p => p.remove())');

    await assert.rejects(
      () => paragraphs[2].text(),
      (err) => {
        assert.match(err.message, /findAll/, `stale-handle error should name findAll, got: ${err.message}`);
        assert.match(err.message, /element 2/, `should name which element, got: ${err.message}`);
        return true;
      }
    );

    // A plain bad selector must not get the findAll explanation
    await assert.rejects(
      () => vibe.find('#never-existed', { timeout: 500 }).then((el) => el.text()),
      (err) => {
        assert.doesNotMatch(err.message, /findAll/, `bad-selector error must stay plain, got: ${err.message}`);
        return true;
      }
    );
  });
});
