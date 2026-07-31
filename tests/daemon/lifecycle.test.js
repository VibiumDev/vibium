/**
 * Daemon Lifecycle Tests
 * Tests daemon start/stop, navigate+find across commands, auto-start
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
const path = require('node:path');
const { VIBIUM } = require('../helpers');

// Helper to run clicker with --json and parse output
function clicker(args, opts = {}) {
  const result = execSync(`${VIBIUM} ${args}`, {
    encoding: 'utf-8',
    timeout: opts.timeout || 60000,
    env: { ...process.env, ...opts.env },
  });
  return result.trim();
}

function clickerJSON(args, opts = {}) {
  const result = clicker(`--json ${args}`, opts);
  return JSON.parse(result);
}

// Helper to stop daemon (ignore errors if not running)
function stopDaemon() {
  try {
    execSync(`${VIBIUM} daemon stop`, { encoding: 'utf-8', timeout: 10000 });
  } catch (e) {
    // Daemon may not be running
  }
}

let serverProcess, baseURL;

before(async () => {
  serverProcess = spawn('node', [path.join(__dirname, '../helpers/test-server.js')], {
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

describe('Daemon: Lifecycle', () => {
  before(() => {
    // Ensure no daemon is running before tests
    stopDaemon();
  });

  after(() => {
    // Clean up
    stopDaemon();
  });

  test('daemon status reports not running when stopped', () => {
    const result = clicker('daemon status');
    assert.match(result, /not running/i, 'Should report not running');
  });

  test('daemon start starts background daemon', () => {
    const result = clicker('daemon start --headless');
    assert.match(result, /started|pid/i, 'Should confirm daemon started');

    // Verify status
    const status = clicker('daemon status');
    assert.match(status, /running/i, 'Should report running');
  });

  test('daemon stop shuts down cleanly', () => {
    // Should be running from previous test
    const result = clicker('daemon stop');
    assert.match(result, /stopped/i, 'Should confirm daemon stopped');

    // Verify not running
    const status = clicker('daemon status');
    assert.match(status, /not running/i, 'Should report not running');
  });
});

describe('Daemon: Multi-step workflow', () => {
  before(() => {
    stopDaemon();
    // Start daemon explicitly for this test suite
    clicker('daemon start --headless');
  });

  after(() => {
    stopDaemon();
  });

  test('go then find reuses browser session', () => {
    // Navigate
    const navResult = clickerJSON(`go ${baseURL}/example`);
    assert.strictEqual(navResult.ok, true, 'Navigate should succeed');
    assert.ok(
      navResult.result.includes('/example'),
      'Should confirm navigation'
    );

    // Find element on same page (no URL needed — session persists)
    const findResult = clickerJSON(`find ${baseURL}/example "h1"`);
    assert.strictEqual(findResult.ok, true, 'Find should succeed');
    assert.ok(
      findResult.result.includes('h1'),
      'Should find h1 element'
    );
  });

  test('eval on current page works', () => {
    const result = clickerJSON(`eval ${baseURL}/example "document.title"`);
    assert.strictEqual(result.ok, true, 'Eval should succeed');
    assert.ok(
      result.result.includes('Example Domain'),
      'Should return page title'
    );
  });
});

describe('Daemon: Auto-start', () => {
  before(() => {
    stopDaemon();
  });

  after(() => {
    stopDaemon();
  });

  test('CLI command auto-starts daemon when not running', () => {
    // No daemon running — this should auto-start one
    const result = clickerJSON(`go ${baseURL}/example --headless`);
    assert.strictEqual(result.ok, true, 'Navigate should succeed via auto-start');

    // Verify daemon is now running
    const status = clicker('daemon status');
    assert.match(status, /running/i, 'Daemon should be running after auto-start');
  });
});

