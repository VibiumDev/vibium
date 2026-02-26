/**
 * JS Library Tests: Element State + Waiting
 * Tests el.text, innerText, html, value, attr, bounds, isVisible, isHidden,
 * isEnabled, isChecked, isEditable, eval, screenshot, waitUntil.
 * Also tests page.wait, page.waitUntil.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');

const { browser } = require('../../../clients/javascript/dist');
const { createTestServer } = require('../../helpers/test-server');

let server, baseURL;

before(async () => {
  ({ server, baseURL } = await createTestServer());
});

after(() => {
  if (server) server.close();
});

// --- Element State ---

describe('Element State: text and content', () => {
  test('text() returns textContent', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const h1 = await vibe.find('h1');
      const text = await h1.text();
      assert.strictEqual(text, 'Example Domain');
    } finally {
      await bro.close();
    }
  });

  test('innerText() returns rendered text', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const h1 = await vibe.find('h1');
      const text = await h1.innerText();
      assert.strictEqual(text, 'Example Domain');
    } finally {
      await bro.close();
    }
  });

  test('html() returns innerHTML', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const h1 = await vibe.find('h1');
      const html = await h1.html();
      assert.strictEqual(html, 'Example Domain');
    } finally {
      await bro.close();
    }
  });

  test('value() returns input value', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(`${baseURL}/login`);

      const input = await vibe.find('#username');
      await input.fill('testuser');
      const val = await input.value();
      assert.strictEqual(val, 'testuser');
    } finally {
      await bro.close();
    }
  });
});

describe('Element State: attributes', () => {
  test('attr() returns attribute value', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const link = await vibe.find('a');
      const href = await link.attr('href');
      assert.ok(href.includes('iana.org'), `href should contain iana.org, got: ${href}`);
    } finally {
      await bro.close();
    }
  });

  test('attr() returns null for missing attribute', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const h1 = await vibe.find('h1');
      const val = await h1.attr('data-nonexistent');
      assert.strictEqual(val, null);
    } finally {
      await bro.close();
    }
  });

  test('getAttribute() is an alias for attr()', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const link = await vibe.find('a');
      const attr = await link.attr('href');
      const getAttribute = await link.getAttribute('href');
      assert.strictEqual(attr, getAttribute);
    } finally {
      await bro.close();
    }
  });
});

describe('Element State: bounds', () => {
  test('bounds() returns bounding box', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const h1 = await vibe.find('h1');
      const box = await h1.bounds();
      assert.ok(typeof box.x === 'number', 'x should be a number');
      assert.ok(typeof box.y === 'number', 'y should be a number');
      assert.ok(box.width > 0, 'width should be > 0');
      assert.ok(box.height > 0, 'height should be > 0');
    } finally {
      await bro.close();
    }
  });

  test('boundingBox() is an alias for bounds()', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const h1 = await vibe.find('h1');
      const bounds = await h1.bounds();
      const boundingBox = await h1.boundingBox();
      assert.deepStrictEqual(bounds, boundingBox);
    } finally {
      await bro.close();
    }
  });
});

describe('Element State: visibility', () => {
  test('isVisible() returns true for visible element', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const h1 = await vibe.find('h1');
      const visible = await h1.isVisible();
      assert.strictEqual(visible, true);
    } finally {
      await bro.close();
    }
  });

  test('isHidden() returns false for visible element', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const h1 = await vibe.find('h1');
      const hidden = await h1.isHidden();
      assert.strictEqual(hidden, false);
    } finally {
      await bro.close();
    }
  });
});

describe('Element State: enabled/checked/editable', () => {
  test('isEnabled() returns true for enabled input', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(`${baseURL}/login`);

      const input = await vibe.find('#username');
      const enabled = await input.isEnabled();
      assert.strictEqual(enabled, true);
    } finally {
      await bro.close();
    }
  });

  test('isChecked() returns state of checkbox', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(`${baseURL}/checkboxes`);

      const checkboxes = await vibe.findAll('input[type="checkbox"]');
      // First checkbox is unchecked, second is checked
      const firstChecked = await checkboxes.first().isChecked();
      const secondChecked = await checkboxes.nth(1).isChecked();
      assert.strictEqual(firstChecked, false, 'First checkbox should be unchecked');
      assert.strictEqual(secondChecked, true, 'Second checkbox should be checked');
    } finally {
      await bro.close();
    }
  });

  test('isEditable() returns true for editable input', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(`${baseURL}/login`);

      const input = await vibe.find('#username');
      const editable = await input.isEditable();
      assert.strictEqual(editable, true);
    } finally {
      await bro.close();
    }
  });
});

describe('Element State: eval', () => {
  test('eval() runs function with element', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const h1 = await vibe.find('h1');
      const tagName = await h1.eval('return el.tagName.toLowerCase()');
      assert.strictEqual(tagName, 'h1');
    } finally {
      await bro.close();
    }
  });
});

describe('Element State: screenshot', () => {
  test('screenshot() returns a PNG buffer', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const link = await vibe.find('a');
      const buf = await link.screenshot();
      assert.ok(buf.length > 0, 'Screenshot buffer should not be empty');
      // PNG magic bytes
      assert.ok(buf[0] === 0x89 && buf[1] === 0x50 && buf[2] === 0x4e && buf[3] === 0x47,
        'Screenshot should be a PNG');
    } finally {
      await bro.close();
    }
  });
});

// --- Page-level Waiting ---

describe('Page Waiting', () => {
  test('find(selector) auto-waits for element', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const h1 = await vibe.find('h1');
      assert.ok(h1, 'find should return an element (auto-waits)');
      const text = await h1.text();
      assert.strictEqual(text, 'Example Domain');
    } finally {
      await bro.close();
    }
  });

  test('wait(ms) delays execution', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const start = Date.now();
      await vibe.wait(200);
      const elapsed = Date.now() - start;
      assert.ok(elapsed >= 150, `Should wait at least 150ms, waited ${elapsed}ms`);
    } finally {
      await bro.close();
    }
  });

  test('waitUntil(fn) resolves when function returns truthy', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const result = await vibe.waitUntil('() => document.querySelector("h1") !== null');
      assert.ok(result, 'waitUntil should return truthy value');
    } finally {
      await bro.close();
    }
  });
});

// --- Fluent Chaining with State ---

describe('Fluent chaining: state methods', () => {
  test('find().text() chains fluently', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const text = await vibe.find('h1').text();
      assert.strictEqual(text, 'Example Domain');
    } finally {
      await bro.close();
    }
  });

  test('find().attr() chains fluently', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const href = await vibe.find('a').attr('href');
      assert.ok(href.includes('iana.org'));
    } finally {
      await bro.close();
    }
  });

  test('find().isVisible() chains fluently', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const visible = await vibe.find('h1').isVisible();
      assert.strictEqual(visible, true);
    } finally {
      await bro.close();
    }
  });
});

// --- Checkpoint ---

describe('Element State Checkpoint', () => {
  test('full checkpoint: state queries + waiting + screenshot', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.newPage();
      await vibe.go('https://example.com');

      const link = await vibe.find('a');
      const linkText = await link.text();
      console.log('text:', linkText);
      assert.ok(linkText.length > 0, 'link text should not be empty');

      console.log('attr:', await link.attr('href'));
      assert.ok((await link.attr('href')).includes('iana.org'));

      console.log('isVisible:', await link.isVisible());
      assert.strictEqual(await link.isVisible(), true);

      console.log('isHidden:', await link.isHidden());
      assert.strictEqual(await link.isHidden(), false);

      console.log('bounds:', await link.bounds());
      const box = await link.bounds();
      assert.ok(box.width > 0 && box.height > 0);

      const h1 = await vibe.find('h1');
      console.log('html:', await h1.html());
      assert.strictEqual(await h1.html(), 'Example Domain');

      // Wait for element (find auto-waits)
      await vibe.find('h1');

      // Element screenshot
      const buf = await link.screenshot();
      console.log('screenshot length:', buf.length);
      assert.ok(buf.length > 0);

      await bro.close();
      console.log('Element state checkpoint passed');
    } catch (e) {
      await bro.close();
      throw e;
    }
  });
});
