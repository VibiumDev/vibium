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

  test('an element found by selector and text acts on that element (#767)', async () => {
    try {
      await vibe.go(`${baseURL}/combined-locator`);
      await (await vibe.find({ selector: 'form button', text: 'Go' })).click();
      assert.strictEqual(await vibe.evaluate("document.getElementById('out').textContent"), 'go-now');
    } finally {
      await vibe.go(`${baseURL}/selectors`);
    }
  });

  test('dragTo reaches a target found by selector and text (#767)', async () => {
    try {
      await vibe.go(`${baseURL}/combined-locator`);
      const target = await vibe.find({ selector: 'form button', text: 'Go' });
      await (await vibe.find({ css: '#src' })).dragTo(target);
      assert.strictEqual(await vibe.evaluate("document.getElementById('out').textContent"), 'go-now');
    } finally {
      await vibe.go(`${baseURL}/selectors`);
    }
  });

  test('element.type() keeps a text-filter locator (#778)', async () => {
    try {
      await vibe.go(`${baseURL}/type-filter`);
      const area = await vibe.find({ text: 'Notes' });
      await area.type(' 2');
      assert.strictEqual(await vibe.evaluate("document.querySelector('#notes').value"), 'Notes 2');
    } finally {
      await vibe.go(`${baseURL}/selectors`);
    }
  });

  test('element.type() still accepts the legacy text payload key', async () => {
    // Old clients send the text to type under "text"; the engine must keep
    // reading it when "value" is absent and not treat it as a filter.
    try {
      await vibe.go(`${baseURL}/type-filter`);
      await vibe['client'].send('vibium:element.type', {
        context: vibe['contextId'],
        selector: '#notes',
        text: ' legacy',
      });
      assert.strictEqual(await vibe.evaluate("document.querySelector('#notes').value"), 'Notes legacy');
    } finally {
      await vibe.go(`${baseURL}/selectors`);
    }
  });

  test('a scoped child re-resolves inside its parent (#779)', async () => {
    try {
      await vibe.go(`${baseURL}/nested-scope`);
      const parent = await vibe.find({ css: '#b' });
      await (await parent.find({ css: '.act' })).click();
      assert.strictEqual(await vibe.evaluate("document.getElementById('out').textContent"), 'B');
    } finally {
      await vibe.go(`${baseURL}/selectors`);
    }
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
      const input = await vibe.find({ css: 'my-card >> #i' });
      await input.focus();
      assert.strictEqual(await vibe.evaluate("document.querySelector('my-card').shadowRoot.activeElement.id"), 'i');
      await input.waitUntil('visible', { timeout: 2000 });
      // findAll elements re-resolve by index.
      const [button] = await vibe.findAll({ css: 'my-card >> button' });
      await button.focus();
      assert.strictEqual(await vibe.evaluate("document.querySelector('my-card').shadowRoot.activeElement.id"), 'b');

      await vibe.go(`${baseURL}/shadow-refs`);
      const [file] = await vibe.findAll({ css: 'x-panel >> input' });
      await file.setFiles([__filename]);
      assert.strictEqual(await vibe.evaluate("document.querySelector('x-panel').shadowRoot.querySelector('input').files[0].name"), require('node:path').basename(__filename));
    } finally {
      await vibe.go(`${baseURL}/selectors`);
    }
  });

  // Semantic matching goes through el.matches, which rejects ">>".
  test('a semantic filter narrows a pierced selector (#748)', async () => {
    await vibe.go(`${baseURL}/shadow-refs`);
    try {
      const el = await vibe.find({ selector: 'x-panel >> button', text: 'B' });
      assert.strictEqual(el.info.text, 'B');
      const all = await vibe.findAll({ selector: 'x-panel >> button', role: 'button' });
      assert.strictEqual(all.length, 6);
    } finally {
      await vibe.go(`${baseURL}/selectors`);
    }
  });
});
