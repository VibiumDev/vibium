/**
 * JS Library Tests: Navigation
 * Tests page.go(), back(), forward(), reload(), url(), title(), content(),
 * waitUntil.url(), waitUntil.loaded()
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');

const { browser } = require('../../../clients/javascript/dist');
const { createTestServer } = require('../../helpers/test-server');

let server, baseURL, bro;

before(async () => {
  ({ server, baseURL } = await createTestServer());
  bro = await browser.start({ headless: true });
});

after(async () => {
  await bro.stop();
  if (server) server.close();
});

describe('JS Navigation', () => {
  test('page.go() navigates to URL', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL + '/');
    const url = await vibe.url();
    assert.ok(url.includes('127.0.0.1'), 'Should have navigated');
  });

  test('page.back() and page.forward()', async () => {
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
  });

  test('page.reload() reloads the page', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL + '/');
    await vibe.reload();
    const url = await vibe.url();
    assert.ok(url.includes('127.0.0.1'), 'Should still be on same page after reload');
  });

  test('page.url() returns current URL', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL + '/login');
    const url = await vibe.url();
    assert.ok(url.includes('/login'), 'URL should contain /login');
  });

  test('page.title() returns page title', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL + '/');
    const title = await vibe.title();
    assert.match(title, /The Internet/i, 'Should return page title');
  });

  test('page.content() returns full HTML', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL + '/');
    const content = await vibe.content();
    assert.ok(content.includes('<html'), 'Should contain <html tag');
    assert.ok(content.includes('Welcome to the-internet'), 'Should contain page content');
  });

  test('page.waitUntil.url() waits for matching URL', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL + '/login');

    // URL should already match — waitUntil.url should return immediately
    await vibe.waitUntil.url('/login', { timeout: 5000 });

    const url = await vibe.url();
    assert.ok(url.includes('/login'), 'Should have matched login URL');
  });

  test('page.waitUntil.loaded() waits for load state', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL + '/');
    await vibe.waitUntil.loaded('complete', { timeout: 10000 });
    // If we get here, it passed
    assert.ok(true);
  });

  test('page.waitUntil.url() times out on mismatch', async () => {
    const vibe = await bro.page();
    await vibe.go(baseURL + '/');

    await assert.rejects(
      () => vibe.waitUntil.url('**/nonexistent-page-xyz', { timeout: 1000 }),
      /timeout/i,
      'Should timeout when URL does not match'
    );
  });

  test('a non-timeout error is not mislabeled as a timeout (#64)', async () => {
    const vibe = await bro.page();
    let message = '';
    try {
      await vibe.go('http://'); // invalid URL → BiDi "invalid argument"
      assert.fail('expected navigation to an invalid URL to throw');
    } catch (e) {
      message = e.message || String(e);
    }
    assert.match(message, /invalid argument|invalid url/i, `expected the real error, got: ${message}`);
    assert.doesNotMatch(message, /^timeout:/i, `validation errors must not be tagged as timeouts: ${message}`);
  });
});
