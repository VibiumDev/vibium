/**
 * JS Library Tests: Object Model
 * Verifies the Browser → Page → BrowserContext object model works end-to-end.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');

const { browser, Browser, Page, BrowserContext } = require('../../../clients/javascript/dist');
const { createTestServer } = require('../../helpers/test-server');

let server, baseURL;

before(async () => {
  ({ server, baseURL } = await createTestServer());
});

after(() => {
  if (server) server.close();
});

describe('JS Object Model', () => {
  test('browser.launch() returns Browser instance', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      assert.ok(bro instanceof Browser, 'Should return a Browser instance');
    } finally {
      await bro.close();
    }
  });

  test('browser.page() returns Page for default tab', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      assert.ok(vibe instanceof Page, 'Should return a Page instance');
      assert.ok(vibe.id, 'Page should have an id');
    } finally {
      await bro.close();
    }
  });

  test('browser.newPage() creates a new tab', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const page1 = await bro.page();
      const page2 = await bro.newPage();

      assert.ok(page2 instanceof Page, 'Should return a Page instance');
      assert.notStrictEqual(page1.id, page2.id, 'New page should have different context ID');
    } finally {
      await bro.close();
    }
  });

  test('browser.pages() returns all open pages', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      await bro.newPage();
      const pages = await bro.pages();

      // At least 2 pages: initial tab + newly created
      assert.ok(pages.length >= 2, `Should have at least 2 pages, got ${pages.length}`);
      for (const vibe of pages) {
        assert.ok(vibe instanceof Page, 'Each page should be a Page instance');
      }
    } finally {
      await bro.close();
    }
  });

  test('browser.newContext() creates isolated context', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const ctx = await bro.newContext();
      assert.ok(ctx instanceof BrowserContext, 'Should return a BrowserContext instance');
      assert.ok(ctx.id, 'Context should have an id');

      const vibe = await ctx.newPage();
      assert.ok(vibe instanceof Page, 'context.newPage() should return a Page');

      await ctx.close();
    } finally {
      await bro.close();
    }
  });

  test('page.go() + page.url() round-trip', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/');
      const url = await vibe.url();
      assert.ok(url.includes('127.0.0.1'), `URL should contain domain, got: ${url}`);
    } finally {
      await bro.close();
    }
  });

  test('page.title() returns page title', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/');
      const title = await vibe.title();
      assert.match(title, /The Internet/i, 'Should return page title');
    } finally {
      await bro.close();
    }
  });

  test('page.close() closes a tab', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const page2 = await bro.newPage();
      const pagesBefore = await bro.pages();

      await page2.close();

      const pagesAfter = await bro.pages();
      assert.ok(pagesAfter.length < pagesBefore.length, 'Should have fewer pages after closing');
    } finally {
      await bro.close();
    }
  });
});
