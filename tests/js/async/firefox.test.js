/**
 * JS Library Tests: Firefox
 * Smoke test for launching Firefox instead of the default Chrome.
 *
 * Skips (rather than fails) when Firefox is not installed, so the suite
 * stays green on machines that only have Chrome. Install with:
 *   vibium install --engine firefox
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execFileSync } = require('node:child_process');

const { browser, firefox } = require('../../../clients/javascript/dist');
const { createTestServer } = require('../../helpers/test-server');

let server, baseURL;

function firefoxInstalled() {
  const bin = process.env.VIBIUM_BIN_PATH;
  if (!bin) return false;
  try {
    execFileSync(bin, ['is-installed', '--engine', 'firefox'], { stdio: 'ignore' });
    return true;
  } catch {
    return false;
  }
}

const haveFirefox = firefoxInstalled();

before(async () => {
  ({ server, baseURL } = await createTestServer());
});

after(() => {
  if (server) server.close();
});

describe('JS Firefox', () => {
  test('navigate, click, and screenshot work on Firefox', async () => {
    if (!haveFirefox) {
      console.log('  (skipped: Firefox not installed)');
      return;
    }

    // Named launcher: firefox.start() === browser.start({ engine: 'firefox' })
    const bro = await firefox.start({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);
      const title = await vibe.evaluate('document.title');
      assert.match(title, /The Internet/i, 'Should load the test page');

      const link = await vibe.find('a[href="/login"]', { timeout: 5000 });
      await link.click();
      // find() waits out the navigation; url() right after click races
      // Firefox's transient about:blank
      const form = await vibe.find('#login', { timeout: 10000 });
      assert.ok(form, 'Click should navigate to the login page');
      assert.match(await vibe.url(), /login/, 'Should be on the login page');

      const screenshot = await vibe.screenshot();
      assert.ok(screenshot.length > 1000, 'Should capture a screenshot on Firefox');
    } finally {
      await bro.stop();
    }
  });

  test('screencast records video natively', async () => {
    if (!haveFirefox) {
      console.log('  (skipped: Firefox not installed)');
      return;
    }

    const bro = await browser.start({ engine: 'firefox', headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      try {
        await vibe.screencast.start();
      } catch (err) {
        // Firefox gains BiDi screencast in 154; self-skip on older builds so
        // this activates on its own once the release channel catches up.
        if (/not supported/.test(err.message)) {
          console.log('  (skipped: this Firefox does not support screencast yet)');
          return;
        }
        throw err;
      }

      const link = await vibe.find('a[href="/login"]', { timeout: 5000 });
      await link.click();
      await vibe.find('#login', { timeout: 10000 });

      const video = await vibe.screencast.stop();
      assert.ok(video.length > 1000, 'Video should have real content');
      // WebM/Matroska EBML magic
      assert.deepStrictEqual([...video.subarray(0, 4)], [0x1a, 0x45, 0xdf, 0xa3],
        'Video should be a WebM file');
    } finally {
      await bro.stop();
    }
  });

  test('screencast on chrome fails with an actionable error', async () => {
    const bro = await browser.start({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);
      await assert.rejects(
        () => vibe.screencast.start(),
        (err) => /not supported by this browser/.test(err.message),
        'Chrome should explain screencast is unsupported and suggest firefox'
      );
    } finally {
      await bro.stop();
    }
  });

  test('unknown browser is rejected with a clear error', () => {
    const bin = process.env.VIBIUM_BIN_PATH;
    if (!bin) {
      console.log('  (skipped: VIBIUM_BIN_PATH not set)');
      return;
    }

    assert.throws(
      () => execFileSync(bin, ['--engine', 'netscape', 'version'], { stdio: 'pipe' }),
      (err) => /unsupported engine/.test(err.stderr.toString()),
      'Should reject an unsupported engine name'
    );
  });
});
