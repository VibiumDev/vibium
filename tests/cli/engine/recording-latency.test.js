/**
 * CLI Tests: Recording latency
 * A navigating action while recording used to burn the whole 5s screenshot
 * timeout, because captureScreenshot is not answered across a navigation and
 * the error was discarded (#289).
 */

const { test, describe, before, after } = require("../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { VIBIUM } = require("../../helpers");

// The bug produced ~5.1s; the fix produces ~0.2s. Anything under this means the
// capture is no longer waiting out its timeout, with room for a slow CI box.
const MAX_MS = 3000;

// A healthy navigating click on the macOS CI runner takes 5.5-8.3s, past the
// ~5.1s the bug produced, so no absolute threshold can separate a stalled
// capture from an ordinarily slow machine there. The Linux legs keep the
// guard.
const DARWIN_SKIP =
  process.platform === 'darwin' &&
  'healthy macOS click latency overlaps the #289 bug signature';

let serverProcess, baseURL, tracePath;

function timed(args) {
  const t0 = Date.now();
  execSync(`${VIBIUM} ${args}`, { encoding: 'utf-8', timeout: 60000 });
  return Date.now() - t0;
}

before(async () => {
  if (DARWIN_SKIP) return;
  serverProcess = spawn('node', [path.join(__dirname, '../../helpers/test-server.js')], {
    stdio: ['pipe', 'pipe', 'pipe'],
  });
  baseURL = await new Promise((resolve) => {
    serverProcess.stdout.once('data', (d) => resolve(d.toString().trim()));
  });
  tracePath = path.join(fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-rec-')), 'trace.zip');
});

after(() => {
  if (DARWIN_SKIP) return;
  try {
    execSync(`${VIBIUM} record stop -o ${tracePath}`, { encoding: 'utf-8', timeout: 30000 });
  } catch {
    // already stopped
  }
  if (tracePath && fs.existsSync(tracePath)) fs.rmSync(path.dirname(tracePath), { recursive: true, force: true });
  if (serverProcess) serverProcess.kill();
});

describe('CLI: recording latency', () => {
  test('a navigating click does not wait out the screenshot timeout (#289)', { skip: DARWIN_SKIP }, () => {
    execSync(`${VIBIUM} go ${baseURL}/form-redirect`, { encoding: 'utf-8', timeout: 30000 });
    execSync(`${VIBIUM} record start --screenshots --name latency`, {
      encoding: 'utf-8',
      timeout: 30000,
    });

    // Submitting this form POSTs and 302-redirects back to the same page, so
    // the click leaves a navigation in flight just as the filmstrip screenshot
    // is taken. Whether the capture loses that race varies, so repeat: before
    // the fix roughly two clicks in three stalled for the full timeout.
    const timings = [];
    for (let i = 0; i < 5; i++) timings.push(timed(`click "#go"`));

    const worst = Math.max(...timings);
    assert.ok(
      worst < MAX_MS,
      `slowest navigating click took ${worst}ms (all: ${timings.join(', ')}ms); ` +
        `over ${MAX_MS}ms means a capture is waiting out its timeout again`
    );

    const result = execSync(`${VIBIUM} record stop -o ${tracePath}`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /saved/i, 'recording should stop cleanly');
    assert.ok(fs.existsSync(tracePath), 'trace should be written');
  });
});
