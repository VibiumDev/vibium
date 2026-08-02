/**
 * CLI Tests: recovery after the browser goes away
 *
 * Closing the browser window (or a crash) leaves the daemon holding a dead
 * handle. Without a liveness check every later command failed with a raw
 * transport error and never recovered (#219).
 *
 * Runs its own daemon so killing Chrome cannot disturb other test files.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
const path = require('path');
const { VIBIUM } = require('../helpers');

let serverProcess, baseURL;

function run(args) {
  try {
    return execSync(`${VIBIUM} ${args} 2>&1`, { encoding: 'utf-8', timeout: 90000 });
  } catch (e) {
    return (e.stdout || '') + (e.stderr || '');
  }
}

function killBrowser() {
  try {
    execSync("pkill -f 'Chrome for Testing' || true", { encoding: 'utf-8' });
  } catch {
    /* nothing to kill */
  }
}

before(async () => {
  serverProcess = spawn('node', [path.join(__dirname, '../helpers/test-server.js')], {
    stdio: ['pipe', 'pipe', 'pipe'],
  });
  baseURL = await new Promise((resolve) => {
    serverProcess.stdout.once('data', (data) => resolve(data.toString().trim()));
  });
  run('daemon stop');
  run('daemon start --headless');
});

after(() => {
  run('daemon stop');
  if (serverProcess) serverProcess.kill();
});

describe('CLI: browser closed externally', () => {
  test('reports the browser is gone, then recovers on the next command', () => {
    run(`go ${baseURL}/example`);
    assert.match(run('title'), /Example Domain/, 'setup: the page should load');

    killBrowser();
    execSync('sleep 2');

    const afterKill = run('title');
    assert.match(
      afterKill,
      /browser is no longer running/,
      `expected an actionable error, got: ${afterKill.slice(0, 200)}`
    );

    // The dead handle must be cleared, so the next command starts fresh rather
    // than failing forever against a connection that is gone.
    run(`go ${baseURL}/example`);
    const recovered = run('title');
    assert.match(
      recovered,
      /Example Domain/,
      `expected recovery on the next command, got: ${recovered.slice(0, 200)}`
    );
  });
});
