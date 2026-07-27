/**
 * CLI Tests: Process Management
 * Tests that Chrome processes are cleaned up properly
 */

const { test, describe } = require('node:test');
const assert = require('node:assert');
const { execSync, execFileSync, spawn, spawnSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { VIBIUM } = require('../helpers');

/**
 * Get PIDs of Chrome for Testing processes spawned by clicker
 * Returns a Set of PIDs
 */
function getClickerChromePids() {
  try {
    const platform = process.platform;
    let cmd, opts;

    if (platform === 'darwin') {
      // Find Chrome for Testing processes that have --remote-debugging-port
      // (our flag). The [-] class stops Linux pgrep from matching the sh -c
      // wrapper, whose command line contains this pattern.
      cmd = "pgrep -f 'Chrome for Testing.*--remote-debugging[-]port' 2>/dev/null || true";
      opts = { encoding: 'utf-8', stdio: ['pipe', 'pipe', 'pipe'] };
    } else if (platform === 'linux') {
      cmd = "pgrep -f 'chrome.*--remote-debugging[-]port' 2>/dev/null || true";
      opts = { encoding: 'utf-8', stdio: ['pipe', 'pipe', 'pipe'] };
    } else if (platform === 'win32') {
      cmd = `powershell -NoProfile -Command "Get-CimInstance Win32_Process -Filter \\"name='chrome.exe' and commandline like '%--remote-debugging-port%'\\" | Select-Object -ExpandProperty ProcessId"`;
      opts = { encoding: 'utf-8', stdio: ['pipe', 'pipe', 'pipe'], shell: true };
    } else {
      return new Set();
    }

    const result = execSync(cmd, opts);
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

/**
 * Poll until predicate returns true, or timeout.
 */
async function waitUntil(fn, description, { timeout = 15000, interval = 500 } = {}) {
  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    if (fn()) return;
    await sleep(interval);
  }
  throw new Error(`waitUntil timed out after ${timeout}ms: ${description}`);
}

describe('CLI: Process Cleanup', () => {
  test('daemon stop cleans up Chrome', async () => {
    // Ensure clean state: daemon stop waits for process exit
    try { execSync(`${VIBIUM} daemon stop`, { encoding: 'utf-8', timeout: 10000 }); } catch {}

    // Capture PIDs BEFORE starting the daemon so we only track new ones
    const pidsBefore = getClickerChromePids();

    // Start a fresh daemon (daemon start polls for socket availability)
    execSync(`${VIBIUM} daemon start --headless`, { encoding: 'utf-8', timeout: 30000 });

    // Navigate to launch the browser
    execSync(`${VIBIUM} go https://example.com`, {
      encoding: 'utf-8',
      timeout: 30000,
    });

    // Verify Chrome was actually spawned
    const newPids = getNewPids(pidsBefore, getClickerChromePids());
    assert.ok(newPids.length > 0, 'Chrome should have been spawned');

    // Stop daemon — should clean up Chrome
    // Windows daemon stop can take 10-15s waiting for Chrome to fully exit.
    execSync(`${VIBIUM} daemon stop`, { encoding: 'utf-8', timeout: 30000 });

    // Poll until the new Chrome processes are gone (daemon cleanup is async)
    await waitUntil(() => {
      const remaining = newPids.filter(pid => getClickerChromePids().has(pid));
      return remaining.length === 0;
    }, 'Chrome PIDs cleaned up after daemon stop');

    const pidsAfter = getClickerChromePids();
    const remainingNewPids = newPids.filter(pid => pidsAfter.has(pid));
    assert.strictEqual(
      remainingNewPids.length,
      0,
      `Chrome processes should be cleaned up after daemon stop. Remaining PIDs: ${remainingNewPids.join(', ')}`
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

    // Wait for any Chrome processes spawned by this test to be cleaned up
    await waitUntil(() => {
      const newPids = getNewPids(pidsBefore, getClickerChromePids());
      return newPids.length === 0;
    }, 'Chrome PIDs cleaned up after SIGTERM');

    const pidsAfter = getClickerChromePids();
    const newPids = getNewPids(pidsBefore, pidsAfter);

    assert.strictEqual(
      newPids.length,
      0,
      `Chrome processes should be cleaned up after SIGTERM. New PIDs remaining: ${newPids.join(', ')}`
    );
  });

  test('shutdown cleanup ignores other tools\' chromedriver processes', async (t) => {
    // Cleanup on Windows kills by executable name, not command line
    if (process.platform === 'win32') {
      t.skip('POSIX-only cleanup path');
      return;
    }

    // Stand in for another tool's chromedriver (e.g. Selenium's): a script
    // whose command line says "chromedriver" but does not run from vibium's
    // cache dir. Orphaned so cleanup would consider it a leftover.
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'other-tool-'));
    const decoy = path.join(dir, 'chromedriver');
    fs.writeFileSync(decoy, '#!/bin/sh\nsleep 120\n');
    fs.chmodSync(decoy, 0o755);
    // The shell backgrounds the decoy, prints its PID, and exits, which
    // orphans the decoy. detached puts the shell (and so the decoy) in its
    // own process group, so a group kill cannot take down this test runner.
    const pid = Number(
      spawnSync('sh', ['-c', `"${decoy}" 120 > /dev/null 2>&1 & echo $!`], {
        encoding: 'utf-8',
        detached: true,
      }).stdout.trim()
    );
    assert.ok(Number.isInteger(pid) && pid > 0, 'decoy failed to start');

    try {
      // Trigger vibium's shutdown cleanup, same as the SIGTERM test above
      const server = spawn(VIBIUM, ['serve'], { stdio: ['pipe', 'pipe', 'pipe'] });
      await sleep(2000);
      server.kill('SIGTERM');
      await new Promise((resolve) => {
        const timeout = setTimeout(resolve, 5000);
        server.on('exit', () => {
          clearTimeout(timeout);
          resolve();
        });
      });

      let alive = true;
      try {
        process.kill(pid, 0);
      } catch {
        alive = false;
      }
      assert.ok(alive, 'vibium cleanup killed a chromedriver process it does not own');
    } finally {
      try { process.kill(pid, 'SIGKILL'); } catch {}
      fs.rmSync(dir, { recursive: true, force: true });
    }
  });
});
