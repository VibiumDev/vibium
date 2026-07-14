/**
 * JS Library Tests: Lifecycle
 * Tests browser.page(), newPage(), newContext(), pages(), stop(),
 * context.newPage(), context.close(), page.activate(), page.close()
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

// Close every page except the default (first) one to keep tests isolated
// without paying a fresh-browser cost per test.
async function resetPages(bro) {
  const pages = await bro.pages();
  for (let i = pages.length - 1; i >= 1; i--) {
    try { await pages[i].close(); } catch {}
  }
}

describe('JS Lifecycle', () => {
  let bro;

  before(async () => {
    bro = await browser.start({ headless: true });
  });

  after(async () => {
    if (bro) await bro.stop();
  });

  test('browser.page() returns default page', async () => {
    const vibe = await bro.page();
    assert.ok(vibe, 'Should return a page');
    assert.ok(vibe.id, 'Page should have an id');
  });

  test('browser.newPage() creates new tab with unique ID', async () => {
    await resetPages(bro);
    const page1 = await bro.page();
    const page2 = await bro.newPage();
    try {
      assert.notStrictEqual(page1.id, page2.id, 'Pages should have different IDs');
    } finally {
      await page2.close();
    }
  });

  test('browser.pages() lists all tabs', async () => {
    await resetPages(bro);
    const pagesBefore = await bro.pages();
    const p1 = await bro.newPage();
    const p2 = await bro.newPage();
    try {
      const pagesAfter = await bro.pages();
      assert.ok(
        pagesAfter.length >= pagesBefore.length + 2,
        `Should have at least 2 more pages. Before: ${pagesBefore.length}, After: ${pagesAfter.length}`
      );
    } finally {
      await p1.close();
      await p2.close();
    }
  });

  test('page.close() removes a tab', async () => {
    await resetPages(bro);
    const newPage = await bro.newPage();
    const pagesBefore = await bro.pages();

    await newPage.close();

    const pagesAfter = await bro.pages();
    assert.strictEqual(
      pagesAfter.length,
      pagesBefore.length - 1,
      'Should have one fewer page'
    );
  });

  test('page.bringToFront() activates a tab', async () => {
    await resetPages(bro);
    const page1 = await bro.page();
    const page2 = await bro.newPage();
    try {
      // Activate page1 (should not throw)
      await page1.bringToFront();
      assert.ok(true, 'bringToFront should succeed');
    } finally {
      await page2.close();
    }
  });

  test('browser.newContext() creates isolated context', async () => {
    await resetPages(bro);
    const ctx = await bro.newContext();
    try {
      assert.ok(ctx.id, 'Context should have an id');

      const vibe = await ctx.newPage();
      assert.ok(vibe.id, 'Page in new context should have an id');

      // Navigate in the new context
      await vibe.go(baseURL + '/');
      const title = await vibe.title();
      assert.match(title, /The Internet/i, 'Should navigate in new context');
    } finally {
      await ctx.close();
    }
  });

  test('context.close() removes all pages in context', async () => {
    await resetPages(bro);
    const ctx = await bro.newContext();
    await ctx.newPage();
    await ctx.newPage();

    const pagesBefore = await bro.pages();
    await ctx.close();
    const pagesAfter = await bro.pages();

    assert.ok(
      pagesAfter.length < pagesBefore.length,
      'Closing context should remove its pages'
    );
  });

  test('multiple pages can navigate independently', async () => {
    await resetPages(bro);
    const page1 = await bro.page();
    const page2 = await bro.newPage();
    try {
      await page1.go(baseURL + '/');
      await page2.go(baseURL + '/login');

      const url1 = await page1.url();
      const url2 = await page2.url();

      assert.ok(!url1.includes('/login'), 'Page 1 should not be on login');
      assert.ok(url2.includes('/login'), 'Page 2 should be on login');
    } finally {
      await page2.close();
    }
  });

  test('browser.onPage() fires for new tabs', async () => {
    await resetPages(bro);
    bro.removeAllListeners('page');
    // Flush any pending contextCreated events
    await bro.page();
    await new Promise(r => setTimeout(r, 200));

    const pages = [];
    bro.onPage((p) => pages.push(p));
    try {
      const p = await bro.newPage();
      await new Promise(r => setTimeout(r, 200));
      assert.strictEqual(pages.length, 1);
      assert.ok(pages[0].id);
      await p.close();
    } finally {
      bro.removeAllListeners('page');
    }
  });

  test('browser.onPopup() fires for window.open', async () => {
    await resetPages(bro);
    bro.removeAllListeners('popup');
    const popups = [];
    bro.onPopup((p) => popups.push(p));
    try {
      const page = await bro.page();
      await page.evaluate("window.open('about:blank')");
      await new Promise(r => setTimeout(r, 200));
      assert.strictEqual(popups.length, 1);
      assert.ok(popups[0].id);
    } finally {
      bro.removeAllListeners('popup');
    }
  });

  test('browser.removeAllListeners() stops callbacks', async () => {
    await resetPages(bro);
    bro.removeAllListeners('page');
    // Flush any pending contextCreated events
    await bro.page();
    await new Promise(r => setTimeout(r, 200));

    const pages = [];
    bro.onPage((p) => pages.push(p));
    const p1 = await bro.newPage();
    await new Promise(r => setTimeout(r, 200));
    assert.strictEqual(pages.length, 1);

    bro.removeAllListeners('page');
    const p2 = await bro.newPage();
    await new Promise(r => setTimeout(r, 200));
    assert.strictEqual(pages.length, 1, 'Should still be 1 after removing listener');
    await p1.close();
    await p2.close();
  });

  // browser.stop() is exercised by the shared `after()` hook above.
  // Run it once with its own browser to verify it shuts down cleanly without
  // tearing down the shared instance used by other tests.
  test('browser.stop() shuts down cleanly', async () => {
    const bro2 = await browser.start({ headless: true });
    const vibe = await bro2.page();
    await vibe.go(baseURL + '/');

    // close() should not throw
    await bro2.stop();
    assert.ok(true, 'browser.stop() should complete without error');
  });
});
