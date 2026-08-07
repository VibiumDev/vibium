/**
 * JS Library Tests: Firefox
 * Smoke test for launching Firefox instead of the default Chrome.
 *
 * Skips (rather than fails) when Firefox is not installed, so the suite
 * stays green on machines that only have Chrome. Install with:
 *   vibium install --engine firefox
 *
 * CI sets VIBIUM_REQUIRE_FIREFOX, which turns every skip into a failure:
 * the green check must prove Firefox and screencast actually ran.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execFileSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

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

function skipOrFail(t, reason) {
  if (process.env.VIBIUM_REQUIRE_FIREFOX) {
    assert.fail(`${reason}, and VIBIUM_REQUIRE_FIREFOX is set`);
  }
  t.skip(reason);
}

// Firefox gains BiDi screencast in 154; self-skip on older builds so the
// recording tests activate on their own once the release channel catches up.
async function startScreencastOrSkip(t, vibe) {
  try {
    await vibe.screencast.start();
    return true;
  } catch (err) {
    if (/not supported/.test(err.message)) {
      skipOrFail(t, 'this Firefox does not support screencast yet');
      return false;
    }
    throw err;
  }
}

before(async () => {
  ({ server, baseURL } = await createTestServer());
});

after(() => {
  if (server) server.close();
});

describe('JS Firefox', () => {
  test('navigate, click, and screenshot work on Firefox', async (t) => {
    if (!haveFirefox) return skipOrFail(t, 'Firefox not installed');

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

  test('screencast records video natively', async (t) => {
    if (!haveFirefox) return skipOrFail(t, 'Firefox not installed');

    const bro = await browser.start({ engine: 'firefox', headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      if (!(await startScreencastOrSkip(t, vibe))) return;

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

  test('screencast stop can be retried after a failed delivery', async (t) => {
    if (!haveFirefox) return skipOrFail(t, 'Firefox not installed');

    // A destination under a regular file is unwritable on every platform,
    // without needing permission tricks.
    const blocker = path.join(os.tmpdir(), `vibium-test-blocker-${process.pid}`);
    fs.writeFileSync(blocker, 'not a directory');
    const badDest = path.join(blocker, 'sub', 'video.webm');

    const bro = await browser.start({ engine: 'firefox', headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      if (!(await startScreencastOrSkip(t, vibe))) return;

      await vibe.find('a[href="/login"]', { timeout: 5000 });

      await assert.rejects(
        () => vibe.screencast.stop({ path: badDest }),
        (err) => /failed to save screencast/.test(err.message),
        'Delivering to an unwritable path should fail'
      );

      // The recording must survive the failed delivery: a retried stop()
      // without a path returns the video inline.
      const video = await vibe.screencast.stop();
      assert.deepStrictEqual([...video.subarray(0, 4)], [0x1a, 0x45, 0xdf, 0xa3],
        'Retried stop should still deliver the WebM');
    } finally {
      await bro.stop();
      fs.rmSync(blocker, { force: true });
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

  test('unknown browser is rejected with a clear error', (t) => {
    const bin = process.env.VIBIUM_BIN_PATH;
    if (!bin) return skipOrFail(t, 'VIBIUM_BIN_PATH not set');

    assert.throws(
      () => execFileSync(bin, ['--engine', 'netscape', 'version'], { stdio: 'pipe' }),
      (err) => /unsupported engine/.test(err.stderr.toString()),
      'Should reject an unsupported engine name'
    );
  });

  test('--firefox-channel selects a separate install', (t) => {
    const bin = process.env.VIBIUM_BIN_PATH;
    if (!bin) return skipOrFail(t, 'VIBIUM_BIN_PATH not set');

    // Channels live in separate cache directories, so a channel that was
    // never installed must report not-installed (exit 1) — even on machines
    // where another channel is present. Purely a local directory check.
    assert.throws(
      () => execFileSync(bin,
        ['is-installed', '--engine', 'firefox', '--firefox-channel', 'no-such-channel'],
        { stdio: 'ignore' }),
      (err) => err.status === 1,
      'A never-installed channel should report not-installed'
    );
  });
});
