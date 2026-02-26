/**
 * JS Library Tests: Navigation
 * Tests page.go(), back(), forward(), reload(), url(), title(), content(),
 * waitUntil.url(), waitUntil.loaded()
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

describe('JS Navigation', () => {
  test('page.go() navigates to URL', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/');
      const url = await vibe.url();
      assert.ok(url.includes('127.0.0.1'), 'Should have navigated');
    } finally {
      await bro.close();
    }
  });

  test('page.back() and page.forward()', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/');
      await vibe.go(baseURL + '/login');

      const urlAfterNav = await vibe.url();
      assert.ok(urlAfterNav.includes('/login'), 'Should be on login page');

      await vibe.back();
      const urlAfterBack = await vibe.url();
      assert.ok(!urlAfterBack.includes('/login'), 'Should have gone back');

      await vibe.forward();
      const urlAfterForward = await vibe.url();
      assert.ok(urlAfterForward.includes('/login'), 'Should have gone forward');
    } finally {
      await bro.close();
    }
  });

  test('page.reload() reloads the page', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/');
      await vibe.reload();
      const url = await vibe.url();
      assert.ok(url.includes('127.0.0.1'), 'Should still be on same page after reload');
    } finally {
      await bro.close();
    }
  });

  test('page.url() returns current URL', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/login');
      const url = await vibe.url();
      assert.ok(url.includes('/login'), 'URL should contain /login');
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

  test('page.content() returns full HTML', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/');
      const content = await vibe.content();
      assert.ok(content.includes('<html'), 'Should contain <html tag');
      assert.ok(content.includes('Welcome to the-internet'), 'Should contain page content');
    } finally {
      await bro.close();
    }
  });

  test('page.waitUntil.url() waits for matching URL', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/login');

      // URL should already match — waitUntil.url should return immediately
      await vibe.waitUntil.url('/login', { timeout: 5000 });

      const url = await vibe.url();
      assert.ok(url.includes('/login'), 'Should have matched login URL');
    } finally {
      await bro.close();
    }
  });

  test('page.waitUntil.loaded() waits for load state', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/');
      await vibe.waitUntil.loaded('complete', { timeout: 10000 });
      // If we get here, it passed
      assert.ok(true);
    } finally {
      await bro.close();
    }
  });

  test('page.waitUntil.url() times out on mismatch', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/');

      await assert.rejects(
        () => vibe.waitUntil.url('**/nonexistent-page-xyz', { timeout: 1000 }),
        /timeout/i,
        'Should timeout when URL does not match'
      );
    } finally {
      await bro.close();
    }
  });
});
