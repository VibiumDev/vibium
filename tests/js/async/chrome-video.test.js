/**
 * JS Library Tests: Chrome video recording behavior
 * Chrome has no BiDi screencast, so requiring video must fail with an
 * actionable error and an ordinary recording must deliver a trace with
 * no video track. The Firefox positives live in firefox.test.js; these
 * run with the Chrome suites, where Chrome is installed.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execFileSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const { browser } = require('../../../clients/javascript/dist');
const { createTestServer } = require('../../helpers/test-server');

// Chrome-only by design (outside the cross-engine roots). Skip loudly rather
// than fail confusingly when someone runs this suite with ENGINE=firefox.
const engine = process.env.VIBIUM_ENGINE || 'chrome';
if (engine !== 'chrome') {
  test(`chrome-video tests require chrome (VIBIUM_ENGINE=${engine})`, { skip: true }, () => {});
  return;
}

let server, baseURL;

// Unzip a recording buffer and return { extractedDir, cleanup }.
function unzipRecording(zipBuffer) {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-chrome-rec-'));
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

describe('Chrome video recording', () => {
  test('required video on chrome fails with an actionable error', async () => {
    const bro = await browser.start({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);
      await assert.rejects(
        () => vibe.context.recording.start({ video: true, path: null }),
        (err) => /not supported by this browser/.test(err.message),
        'Chrome should explain video is unsupported and name the firefox install'
      );
    } finally {
      await bro.stop();
    }
  });

  test('omitted video on chrome records a trace without a video track', async () => {
    const bro = await browser.start({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      await vibe.context.recording.start({ path: null });
      await vibe.find('a[href="/login"]', { timeout: 5000 });
      const result = await vibe.context.recording.stop();
      assert.ok(result.bytes.length > 0, 'Recording should still deliver');
      assert.ok(result.videoUnavailable, 'Result should say why there is no video');

      const { extractedDir, cleanup } = unzipRecording(result.bytes);
      try {
        const files = fs.readdirSync(extractedDir);
        assert.ok(files.some((f) => f.endsWith('.trace')), 'Trace should be present');
        assert.ok(!files.includes('video'), 'No video entries on an engine without support');
      } finally {
        cleanup();
      }
    } finally {
      await bro.stop();
    }
  });
});
