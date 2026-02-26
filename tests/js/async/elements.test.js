/**
 * JS Library Tests: Element Finding
 * Tests findAll, scoped find, semantic selectors, and locator chaining.
 */

const { test, describe } = require('node:test');
const assert = require('node:assert');

const { browser } = require('../../../clients/javascript/dist');

describe('Element Finding', () => {
  // --- findAll with CSS ---

  test('findAll returns multiple elements', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const paragraphs = await vibe.findAll('p');
      assert.ok(paragraphs.count() > 0, 'Should find at least one paragraph');
    } finally {
      await bro.close();
    }
  });

  test('findAll().first() returns first element', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const paragraphs = await vibe.findAll('p');
      const first = paragraphs.first();
      assert.ok(first, 'Should return first element');
      assert.ok(first.info.tag === 'p', 'First element should be a <p>');
    } finally {
      await bro.close();
    }
  });

  test('findAll().last() returns last element', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const paragraphs = await vibe.findAll('p');
      const last = paragraphs.last();
      assert.ok(last, 'Should return last element');
      assert.ok(last.info.tag === 'p', 'Last element should be a <p>');
    } finally {
      await bro.close();
    }
  });

  test('findAll().nth(0) returns element at index', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const paragraphs = await vibe.findAll('p');
      const zeroth = paragraphs.nth(0);
      assert.ok(zeroth, 'Should return element at index 0');
      assert.ok(zeroth.info.tag === 'p', 'Element at index 0 should be a <p>');
    } finally {
      await bro.close();
    }
  });

  test('findAll().count() returns number', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const paragraphs = await vibe.findAll('p');
      const count = paragraphs.count();
      assert.ok(typeof count === 'number', 'count() should return a number');
      assert.ok(count > 0, 'count() should be > 0');
    } finally {
      await bro.close();
    }
  });

  // --- Scoped find ---

  test('element.find() scoped to parent', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const body = await vibe.find('body');
      assert.ok(body, 'Should find body');

      const nested = await body.find('a');
      assert.ok(nested, 'Should find nested <a> inside body');
      assert.ok(nested.info.tag === 'a', 'Nested element should be an <a>');
    } finally {
      await bro.close();
    }
  });

  // --- Semantic selectors ---

  test('find({ role: "link" }) finds a link', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const link = await vibe.find({ role: 'link' });
      assert.ok(link, 'Should find element with role=link');
      assert.ok(link.info.tag === 'a', 'Element with role=link should be an <a>');
    } finally {
      await bro.close();
    }
  });

  test('find({ text: "Learn more" }) finds element by text', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const el = await vibe.find({ text: 'Learn more' });
      assert.ok(el, 'Should find element containing text');
      assert.ok(el.info.text.includes('Learn more'), 'Element text should contain "Learn more"');
    } finally {
      await bro.close();
    }
  });

  test('find({ role: "link", text: "Learn" }) combo selector', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const link = await vibe.find({ role: 'link', text: 'Learn' });
      assert.ok(link, 'Should find link with matching text');
      assert.ok(link.info.tag === 'a', 'Element should be an <a>');
      assert.ok(link.info.text.includes('Learn'), 'Element text should include "Learn"');
    } finally {
      await bro.close();
    }
  });

  // --- Iterator ---

  test('ElementList is iterable', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const paragraphs = await vibe.findAll('p');
      let count = 0;
      for (const el of paragraphs) {
        assert.ok(el.info.tag === 'p', 'Each iterated element should be a <p>');
        count++;
      }
      assert.ok(count > 0, 'Should iterate over at least one element');
      assert.strictEqual(count, paragraphs.count(), 'Iterator count should match count()');
    } finally {
      await bro.close();
    }
  });
});
