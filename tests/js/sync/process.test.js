/**
 * JS Library Tests: Sync Process Management
 * Tests that browser processes are cleaned up properly (sync API)
 *
 * Uses a subprocess test server because the sync API blocks the event loop.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
const path = require('path');

const { browser: browserSync } = require('../../../clients/javascript/dist/sync');

let serverProcess, baseURL;

before(async () => {
  serverProcess = spawn('node', [path.join(__dirname, '../../helpers/test-server.js')], {
    stdio: ['pipe', 'pipe', 'pipe'],
  });
  baseURL = await new Promise((resolve) => {
    serverProcess.stdout.once('data', (data) => {
      resolve(data.toString().trim());
    });
  });
});

after(() => {
  if (serverProcess) serverProcess.kill();
});

/**
 * Get PIDs of Chrome for Testing processes spawned by clicker
 * Returns a Set of PIDs
 */
function getClickerChromePids() {
  try {
    const platform = process.platform;
    let cmd;

    if (platform === 'darwin') {
      // The [-] class stops Linux pgrep from matching the sh -c wrapper,
      // whose command line contains this pattern.
      cmd = "pgrep -f 'Chrome for Testing.*--remote-debugging[-]port' 2>/dev/null || true";
    } else if (platform === 'linux') {
      cmd = "pgrep -f 'chrome.*--remote-debugging[-]port' 2>/dev/null || true";
    } else {
      return new Set();
    }

    const result = execSync(cmd, { encoding: 'utf-8', stdio: ['pipe', 'pipe', 'pipe'] });
    const pids = result.trim().split('\n').filter(Boolean).map(Number);
    return new Set(pids);
  } catch {
    return new Set();
  }
}

/**
 * Get new PIDs that appeared between two sets
 */
function getNewPids(before, after) {
  return [...after].filter(pid => !before.has(pid));
}

function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function waitUntil(fn, description, { timeout = 15000, interval = 500 } = {}) {
  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    if (fn()) return;
    await sleep(interval);
  }
  throw new Error(`waitUntil timed out after ${timeout}ms: ${description}`);
}

describe('JS Sync Process Cleanup', () => {
  // 60s: one browser.start() on macOS is ~16s; remainder is for the
  // post-stop waitUntil() polling Chrome PIDs to disappear.
  test('sync API cleans up Chrome on stop()', { timeout: 60000 }, async () => {
    const pidsBefore = getClickerChromePids();

    const bro = browserSync.start({ headless: true });
    const vibe = bro.page();
    vibe.go(baseURL);
    bro.stop();

    await waitUntil(() => {
      const newPids = getNewPids(pidsBefore, getClickerChromePids());
      return newPids.length === 0;
    }, 'Chrome PIDs cleaned up after sync stop()');

    const pidsAfter = getClickerChromePids();
    const newPids = getNewPids(pidsBefore, pidsAfter);

    assert.strictEqual(
      newPids.length,
      0,
      `Chrome processes should be cleaned up. New PIDs remaining: ${newPids.join(', ')}`
    );
  });
});
