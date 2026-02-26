/**
 * CLI Tests: Process Management
 * Tests that Chrome processes are cleaned up properly
 */

const { test, describe } = require('node:test');
const assert = require('node:assert');
const { execSync, execFileSync, spawn } = require('node:child_process');
const { VIBIUM } = require('../helpers');

/**
 * Get PIDs of Chrome for Testing processes spawned by clicker
 * Returns a Set of PIDs
 */
function getClickerChromePids() {
  try {
    const platform = process.platform;
    let cmd;

    if (platform === 'darwin') {
      // Find Chrome for Testing processes that have --remote-debugging-port (our flag)
      cmd = "pgrep -f 'Chrome for Testing.*--remote-debugging-port' 2>/dev/null || true";
    } else if (platform === 'linux') {
      cmd = "pgrep -f 'chrome.*--remote-debugging-port' 2>/dev/null || true";
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

/**
 * Sleep helper
 */
function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

describe('CLI: Process Cleanup', () => {
  test('go command cleans up Chrome on completion', async () => {
    const pidsBefore = getClickerChromePids();

    execSync(`${VIBIUM} go https://example.com`, {
      encoding: 'utf-8',
      timeout: 30000,
    });

    // Give processes time to exit
    await sleep(2000);

    const pidsAfter = getClickerChromePids();
    const newPids = getNewPids(pidsBefore, pidsAfter);

    assert.strictEqual(
      newPids.length,
      0,
      `Chrome processes should be cleaned up. New PIDs remaining: ${newPids.join(', ')}`
    );
  });

  test('serve command cleans up on SIGTERM', async () => {
    const pidsBefore = getClickerChromePids();

    const server = spawn(VIBIUM, ['serve'], {
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    // Wait for server to start and a browser to potentially be spawned
    await sleep(2000);

    // Shut down the server and its process tree
    if (process.platform === 'win32') {
      try {
        execFileSync('taskkill', ['/T', '/F', '/PID', server.pid.toString()], { stdio: 'ignore' });
      } catch {
        // Process may have already exited
      }
    } else {
      server.kill('SIGTERM');
    }

    // Wait for server to clean up (with timeout)
    await new Promise((resolve) => {
      const timeout = setTimeout(resolve, 5000);
      server.on('exit', () => {
        clearTimeout(timeout);
        resolve();
      });
    });

    // Additional wait for any lingering processes
    await sleep(2000);

    const pidsAfter = getClickerChromePids();
    const newPids = getNewPids(pidsBefore, pidsAfter);

    assert.strictEqual(
      newPids.length,
      0,
      `Chrome processes should be cleaned up after SIGTERM. New PIDs remaining: ${newPids.join(', ')}`
    );
  });
});
