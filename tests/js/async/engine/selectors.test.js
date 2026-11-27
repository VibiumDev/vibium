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

describe('Find filters (#709)', () => {
  before(async () => {
    await vibe.go(`${baseURL}/filter-list`);
  });

  after(async () => {
    await vibe.go(`${baseURL}/selectors`);
  });

  const logged = () => vibe.evaluate("document.getElementById('log').textContent");

  test('find picks the nth match, negative from the end', async () => {
    const third = await vibe.find({ css: '.row', nth: 2 });
    assert.ok(third.info.text.includes('Cherries'), `nth 2 found: ${third.info.text}`);
    const last = await vibe.find({ css: '.row', nth: -1 });
    assert.ok(last.info.text.includes('Dates'), `nth -1 found: ${last.info.text}`);
  });

  test('find filters by hasText and has before it picks', async () => {
    const bananas = await vibe.find({ css: '.row', hasText: 'Banan' });
    assert.ok(bananas.info.text.includes('Bananas'), `hasText found: ${bananas.info.text}`);
    const sale = await vibe.find({ css: '.row', has: 'em' });
    assert.ok(sale.info.text.includes('Cherries'), `has found: ${sale.info.text}`);
    const semantic = await vibe.find({ role: 'listitem', hasText: 'Dates' });
    assert.ok(semantic.info.text.includes('Dates'), `role with hasText found: ${semantic.info.text}`);
    const second = await vibe.find({ css: '.row', hasText: 'es', nth: 1 });
    assert.ok(second.info.text.includes('Cherries'), `nth after hasText found: ${second.info.text}`);
  });

  test('a filtered find acts on the element it matched', async () => {
    const cherries = await vibe.find({ css: '.row', hasText: 'Cherries' });
    await cherries.click();
    assert.strictEqual(await logged(), 'Cherries');
    const last = await vibe.find({ css: '.row', nth: -1 });
    await last.click();
    assert.strictEqual(await logged(), 'Dates');
    const scoped = await vibe.find({ css: '#list' }).find({ css: '.row', nth: 1 });
    await scoped.click();
    assert.strictEqual(await logged(), 'Bananas');
  });

  test('a filtered findAll element acts on its own match', async () => {
    const rows = await vibe.findAll({ css: '.row', hasText: 'es' });
    assert.strictEqual(rows.length, 3, 'Apples, Cherries and Dates contain "es"');
    await rows[1].click();
    assert.strictEqual(await logged(), 'Cherries');
  });

  test('findAll refuses nth', async () => {
    await assert.rejects(vibe.findAll({ css: '.row', nth: 1 }), /nth picks one element/);
  });

  test('a filtered find waits for its match and names the filter on a miss', async () => {
    await vibe.evaluate('setTimeout(appendRow, 300)');
    const added = await vibe.find({ css: '.row', hasText: 'Elderberries' }, { timeout: 5000 });
    assert.ok(added.info.text.includes('Elderberries'));
    await assert.rejects(
      vibe.find({ css: '.row', hasText: 'Grapes' }, { timeout: 500 }),
      /hasText=Grapes.*element not found/,
    );
    await assert.rejects(
      vibe.find({ css: '.row', nth: 99 }, { timeout: 500 }),
      /nth=99.*element not found/,
    );
  });
});
