/**
 * JS Library Tests: newPage foregrounds the tab it creates
 *
 * Regression tests for #495: the CLI and MCP activate the tabs they create
 * and the client path did not, so page scripts saw a different
 * visibilityState depending on which surface drove the browser. A
 * background tab is a different execution environment (visibilityState
 * hidden, rAF throttled, no composited frame in headless Chrome), which is
 * the mechanism behind the #491 screenshot hang.
 */

const { test, describe, before, after } = require("../../../helpers/capabilities").suite("core");
const assert = require('node:assert');

const { browser } = require('../../../../clients/javascript/dist');

let bro;

before(async () => {
  bro = await browser.start({ headless: true });
});

after(async () => {
  await bro.stop();
});

describe('Lifecycle: newPage foregrounds the new tab (#495)', () => {
  test('the newest page is visible, like the CLI and MCP', async () => {
    const a = await bro.newPage();
    const b = await bro.newPage();
    try {
      assert.strictEqual(await b.evaluate('document.visibilityState'), 'visible');
      assert.strictEqual(await a.evaluate('document.visibilityState'), 'hidden');
    } finally {
      await a.close();
      await b.close();
    }
  });

  test('bringToFront still moves the foreground back', async () => {
    const a = await bro.newPage();
    const b = await bro.newPage();
    try {
      await a.bringToFront();
      assert.strictEqual(await a.evaluate('document.visibilityState'), 'visible');
      assert.strictEqual(await b.evaluate('document.visibilityState'), 'hidden');
    } finally {
      await a.close();
      await b.close();
    }
  });

  test('a page created inside a user context is visible too', async () => {
    const ctx = await bro.newContext();
    try {
      const p = await ctx.newPage();
      assert.strictEqual(await p.evaluate('document.visibilityState'), 'visible');
    } finally {
      await ctx.close();
    }
  });
});
