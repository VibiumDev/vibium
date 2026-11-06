/**
 * JS Library Tests: find by intent. A string is a description, css is
 * explicit, image matches locally, and the element find returns is addressed
 * by what find resolved. No AI provider is configured here.
 */

const { test, describe, before, after } = require("../../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const { browser } = require('../../../../clients/javascript/dist');
const { createTestServer, PATTERN_PNG, makePatternPNG } = require('../../../helpers/test-server');

for (const key of Object.keys(process.env)) {
  if (key.startsWith('VIBIUM_AI_') || /_API_KEY$/.test(key)) delete process.env[key];
}
process.env.VIBIUM_LOAD_AI_ENV = '0';

let server, baseURL, bro, vibe, dir, patternPath;

before(async () => {
  ({ server, baseURL } = await createTestServer());
  dir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-find-'));
  patternPath = path.join(dir, 'pattern.png');
  fs.writeFileSync(patternPath, PATTERN_PNG);
  bro = await browser.start({ headless: true });
  // One page for the whole file: a new tab per test is what stalls on a
  // loaded machine, and every test here navigates first anyway.
  vibe = await bro.page();
});

after(async () => {
  if (bro) await bro.stop();
  if (server) server.close();
  fs.rmSync(dir, { recursive: true, force: true });
});

async function gallery() {
  await vibe.go(`${baseURL}/find-images`);
  return vibe;
}

describe('find: description', () => {
  test('a string is matched against names and roles, then acts natively', async () => {
    const vibe = await gallery();
    const el = await vibe.find('the Add to Cart button');
    assert.strictEqual(el.info.tag, 'button');
    assert.strictEqual(el.info.source, 'tree');
    assert.match(el.info.ref, /^@e\d+$/);
    assert.ok(el.info.selector, 'a DOM-backed match carries its selector');
    await el.click();
    assert.strictEqual(await vibe.evaluate('window.__added'), 1);
  });

  test('a CSS-looking string is refused at once and names the css form', async () => {
    const vibe = await gallery();
    const started = Date.now();
    await assert.rejects(vibe.find('h1'), (e) => /reads like a CSS selector/.test(e.message) && /find\(\{css: "h1"\}\)/.test(e.message));
    assert.ok(Date.now() - started < 5000, 'must not wait out the timeout');
    await assert.rejects(vibe.find({ description: 'h1', timeout: 300 }), /none is configured/);
  });

  test('findAll returns every element of that name', async () => {
    const vibe = await gallery();
    const els = await vibe.findAll('Product thumbnail');
    assert.strictEqual(els.length, 3);
    assert.ok(els.every((e) => e.info.tag === 'img'));
  });

  test('strategy vlm without a provider rejects clearly', async () => {
    const vibe = await gallery();
    await assert.rejects(vibe.find({ description: 'the red button', strategy: 'vlm' }), /configured vision model/);
  });
});

describe('find: explicit locators', () => {
  test('css and tag are exact and still re-resolve by locator', async () => {
    const vibe = await gallery();
    assert.strictEqual(await vibe.find({ css: 'h1' }).text(), 'Gallery');
    assert.strictEqual((await vibe.find({ tag: 'h1' })).info.tag, 'h1');
    assert.strictEqual((await vibe.findAll({ css: 'img' })).length, 4);
    assert.strictEqual((await vibe.find({ selector: '#add' })).info.text, 'Add to Cart');
  });

  test('a model strategy on an exact locator is a validation error', async () => {
    const vibe = await gallery();
    await assert.rejects(vibe.find({ css: 'h1', strategy: 'cv' }), /strategy cv needs an image/);
    await assert.rejects(vibe.find({ role: 'button', strategy: 'vlm' }), /never by a model/);
    await assert.rejects(vibe.find({ description: 'x', css: 'h1' }), /standalone/);
  });

  test('scoped find inside an element takes the same forms', async () => {
    const vibe = await gallery();
    const body = await vibe.find({ css: 'body' });
    assert.strictEqual((await body.find('Add to Cart')).info.tag, 'button');
    assert.strictEqual((await body.findAll({ css: 'img' })).length, 4);
  });
});

describe('find: image', () => {
  test('bytes or a path locate every copy and snap to the element', async () => {
    const vibe = await gallery();
    const els = await vibe.findAll({ image: PATTERN_PNG });
    assert.strictEqual(els.length, 3);
    for (const el of els) {
      assert.strictEqual(el.info.source, 'image');
      assert.strictEqual(el.info.tag, 'img');
      assert.strictEqual(el.info.box.width, 96);
    }
    const one = await vibe.find({ image: patternPath, details: true });
    assert.strictEqual(one.info.details.method, 'sift');
    await one.click();
  });

  test('no match without a provider is empty or a miss, never a configuration error', async () => {
    const vibe = await gallery();
    const none = makePatternPNG(4242);
    assert.deepStrictEqual(await vibe.findAll({ image: none, timeout: 300 }), []);
    await assert.rejects(vibe.find({ image: none, timeout: 300 }), (e) => /no verified match/.test(e.message) && !/VIBIUM_AI_PROVIDER/.test(e.message));
  });

  test('strategy cv is local; vlm needs a provider', async () => {
    const vibe = await gallery();
    assert.strictEqual((await vibe.find({ image: PATTERN_PNG, strategy: 'cv' })).info.tag, 'img');
    await assert.rejects(vibe.find({ image: PATTERN_PNG, strategy: 'vlm' }), /configured vision model/);
  });

  test('a region with no element of its own is driven by coordinates', async () => {
    await vibe.go(`${baseURL}/find-canvas`);
    await vibe.waitForFunction('window.__drawn === true');
    const el = await vibe.find({ image: PATTERN_PNG });
    assert.strictEqual(el.info.selector, undefined);
    assert.match(el.info.ref, /^@e\d+$/);
    await el.click();
    const clicks = await vibe.evaluate('window.__clicks');
    assert.strictEqual(clicks.length, 1);
    assert.ok(Math.abs(clicks[0].x - 348) <= 2 && Math.abs(clicks[0].y - 198) <= 2, JSON.stringify(clicks));
    const box = await el.boundingBox();
    assert.strictEqual(box.width, 96);
    await assert.rejects(el.fill('x'), /visual match with no DOM element/);
    await vibe.evaluate('window.scrollTo(0, 40)');
    await el.click();
    assert.strictEqual((await vibe.evaluate('window.__clicks')).length, 2);
    await vibe.go(`${baseURL}/example`);
    await assert.rejects(el.click(), /stale: the page navigated/);
  });
});
