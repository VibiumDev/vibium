/**
 * JS Library Tests: Selector Strategies
 * Tests xpath, testid, placeholder, alt, title, and label selectors.
 */

const { test, describe, before, after } = require("../../../helpers/capabilities").suite("core");
const assert = require('node:assert');

const { browser } = require('../../../../clients/javascript/dist');
const { createTestServer } = require('../../../helpers/test-server');

let server, baseURL, bro, vibe;

before(async () => {
  ({ server, baseURL } = await createTestServer());
  bro = await browser.start({ headless: true });
  vibe = await bro.page();
  await vibe.go(`${baseURL}/selectors`);
});

after(async () => {
  if (bro) await bro.stop();
  if (server) server.close();
});

describe('Selector Strategies', () => {
  test('find by xpath', async () => {
    const el = await vibe.find({ xpath: '//button' });
    assert.ok(el, 'Should find element by xpath');
    assert.strictEqual(el.info.tag, 'button');
    assert.ok(el.info.text.includes('Submit'));
  });

  test('find by testid', async () => {
    const el = await vibe.find({ testid: 'search-input' });
    assert.ok(el, 'Should find element by data-testid');
    assert.strictEqual(el.info.tag, 'input');
  });

  test('find by placeholder', async () => {
    const el = await vibe.find({ placeholder: 'Search...' });
    assert.ok(el, 'Should find element by placeholder');
    assert.strictEqual(el.info.tag, 'input');
  });

  test('find by alt', async () => {
    const el = await vibe.find({ alt: 'Logo image' });
    assert.ok(el, 'Should find element by alt text');
    assert.strictEqual(el.info.tag, 'img');
  });

  test('find by title', async () => {
    const el = await vibe.find({ title: 'Search field' });
    assert.ok(el, 'Should find element by title attribute');
    assert.strictEqual(el.info.tag, 'input');
  });

  test('find by label', async () => {
    const el = await vibe.find({ label: 'Username' });
    assert.ok(el, 'Should find input by associated label');
    assert.strictEqual(el.info.tag, 'input');
  });

  test('xpath finds nested element', async () => {
    const el = await vibe.find({ xpath: '//div[@class="container"]/span' });
    assert.ok(el, 'Should find nested element by xpath');
    assert.strictEqual(el.info.tag, 'span');
    assert.ok(el.info.text.includes('Hello from span'));
  });

  // Element methods re-resolve their selector; focus and waitUntil did it with
  // plain querySelector, which threw or timed out on ">>".
  test('element methods resolve a pierced selector (#748)', async () => {
    await vibe.go(`${baseURL}/shadow`);
    try {
      const input = await vibe.find('my-card >> #i');
      await input.focus();
      assert.strictEqual(await vibe.evaluate("document.querySelector('my-card').shadowRoot.activeElement.id"), 'i');
      await input.waitUntil('visible', { timeout: 2000 });
      // findAll elements re-resolve by index.
      const [button] = await vibe.findAll('my-card >> button');
      await button.focus();
      assert.strictEqual(await vibe.evaluate("document.querySelector('my-card').shadowRoot.activeElement.id"), 'b');

      await vibe.go(`${baseURL}/shadow-refs`);
      const [file] = await vibe.findAll('x-panel >> input');
      await file.setFiles([__filename]);
      assert.strictEqual(await vibe.evaluate("document.querySelector('x-panel').shadowRoot.querySelector('input').files[0].name"), require('node:path').basename(__filename));
    } finally {
      await vibe.go(`${baseURL}/selectors`);
    }
  });
});
