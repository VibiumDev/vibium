/**
 * JS Library Tests: Firefox
 * Smoke test for launching Firefox instead of the default Chrome.
 *
 * Skips (rather than fails) when Firefox is not installed, so the suite
 * stays green on machines that only have Chrome. Install with:
 *   vibium install --engine firefox
 *
 * CI sets VIBIUM_REQUIRE_FIREFOX, which turns every skip into a failure:
 * the green check must prove Firefox and video recording actually ran.
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
// video tests activate on their own once the release channel catches up.
async function startVideoRecordingOrSkip(t, vibe, options) {
  try {
    await vibe.context.recording.start({ video: true, ...options });
    return true;
  } catch (err) {
    if (/not supported/.test(err.message)) {
      skipOrFail(t, 'this Firefox does not support video recording yet');
      return false;
    }
    throw err;
  }
}

// Unzip a recording buffer and return { extractedDir, cleanup }.
function unzipRecording(zipBuffer) {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-firefox-rec-'));
  const zipPath = path.join(tmpDir, 'record.zip');
  fs.writeFileSync(zipPath, zipBuffer);
  execFileSync('unzip', ['-o', zipPath, '-d', path.join(tmpDir, 'extracted')], { stdio: 'pipe' });
  return {
    extractedDir: path.join(tmpDir, 'extracted'),
    cleanup: () => fs.rmSync(tmpDir, { recursive: true, force: true }),
  };
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

  test('recording captures a native video track', async (t) => {
    if (!haveFirefox) return skipOrFail(t, 'Firefox not installed');

    const outDir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-firefox-out-'));
    const zipPath = path.join(outDir, 'run.zip');
    const bro = await browser.start({ engine: 'firefox', headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      if (!(await startVideoRecordingOrSkip(t, vibe, { path: zipPath }))) return;

      const link = await vibe.find('a[href="/login"]', { timeout: 5000 });
      await link.click();
      await vibe.find('#login', { timeout: 10000 });

      const result = await vibe.context.recording.stop();
      assert.strictEqual(result.path, zipPath, 'Result should carry the delivery path');
      assert.ok(fs.existsSync(zipPath), 'Recording should land at the declared path');
      assert.strictEqual(result.videos.length, 1, 'Result should report the video track');
      assert.ok(result.videos[0].durationMs > 0, 'Video should have a duration');
      assert.ok(!result.videos[0].error, 'Video should have no error');

      const { extractedDir, cleanup } = unzipRecording(fs.readFileSync(zipPath));
      try {
        const videoDir = path.join(extractedDir, 'video');
        const videos = fs.readdirSync(videoDir).filter((f) => f.endsWith('.webm'));
        assert.strictEqual(videos.length, 1, 'Zip should contain one video track');
        const video = fs.readFileSync(path.join(videoDir, videos[0]));
        assert.ok(video.length > 1000, 'Video should have real content');
        // WebM/Matroska EBML magic
        assert.deepStrictEqual([...video.subarray(0, 4)], [0x1a, 0x45, 0xdf, 0xa3],
          'Video should be a WebM file');

        const index = JSON.parse(fs.readFileSync(path.join(videoDir, 'index.json'), 'utf-8'));
        assert.strictEqual(index.videos.length, 1, 'Manifest should list the video');
        assert.strictEqual(index.videos[0].file, `video/${videos[0]}`, 'Manifest should name the file');
        assert.strictEqual(index.videos[0].mimeType, 'video/webm');
        assert.ok(index.videos[0].offsetMs >= 0, 'Manifest should carry the start offset');
      } finally {
        cleanup();
      }
    } finally {
      await bro.stop();
      fs.rmSync(outDir, { recursive: true, force: true });
    }
  });

  // #358: recording.start before the first navigation reported 0x0 video
  // dimensions — Firefox refuses the viewport script on its privileged
  // initial page, and the failure was silently swallowed.
  test('video dimensions are reported when recording starts before navigation', async (t) => {
    if (!haveFirefox) return skipOrFail(t, 'Firefox not installed');

    const outDir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-firefox-dims-'));
    const zipPath = path.join(outDir, 'run.zip');
    const bro = await firefox.start({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.setViewport({ width: 1280, height: 720 });

      // Start on the fresh pre-navigation page, like the issue repro.
      if (!(await startVideoRecordingOrSkip(t, vibe, { path: zipPath }))) return;

      await vibe.go(baseURL);
      const result = await vibe.context.recording.stop();

      assert.strictEqual(result.videos.length, 1, 'Result should report the video track');
      const video = result.videos[0];
      assert.ok(video.width > 0, `Video width should be reported, got ${video.width}`);
      assert.ok(video.height > 0, `Video height should be reported, got ${video.height}`);

      const { extractedDir, cleanup } = unzipRecording(fs.readFileSync(zipPath));
      try {
        const index = JSON.parse(fs.readFileSync(path.join(extractedDir, 'video', 'index.json'), 'utf-8'));
        assert.strictEqual(index.videos[0].width, video.width, 'Manifest width should match the stop result');
        assert.strictEqual(index.videos[0].height, video.height, 'Manifest height should match the stop result');
      } finally {
        cleanup();
      }
    } finally {
      await bro.stop();
      fs.rmSync(outDir, { recursive: true, force: true });
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
