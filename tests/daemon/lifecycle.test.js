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

// `daemon status` exits non-zero when nothing is running, which is the contract.
// Use this where the output matters but the exit code is expected to be non-zero.
function clickerAllowExit(args, opts = {}) {
  try {
    return clicker(args, opts);
  } catch (e) {
    return ((e.stdout || '') + (e.stderr || '')).trim();
  }
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
    const result = clickerAllowExit('daemon status');
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
    const status = clickerAllowExit('daemon status');
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


describe('Daemon: status exit code', () => {
  before(() => {
    stopDaemon();
  });

  after(() => {
    stopDaemon();
  });

  test('exits non-zero when the daemon is stopped', () => {
    try {
      execSync(`${VIBIUM} daemon status`, { encoding: 'utf-8', timeout: 10000, stdio: 'pipe' });
      assert.fail('daemon status should exit non-zero when nothing is running');
    } catch (err) {
      assert.match(err.stdout + err.stderr, /not running/i);
    }
  });

  test('--json emits only JSON when the daemon is stopped', () => {
    try {
      execSync(`${VIBIUM} --json daemon status`, { encoding: 'utf-8', timeout: 10000, stdio: 'pipe' });
      assert.fail('daemon status should exit non-zero when nothing is running');
    } catch (err) {
      // The human-readable line used to be printed alongside the JSON, which
      // broke any consumer piping this to a parser.
      assert.deepStrictEqual(JSON.parse(err.stdout.trim()), { running: false });
    }
  });

  test('exits zero when the daemon is running', () => {
    clicker('daemon start --headless');
    const status = clicker('daemon status');
    assert.match(status, /running/i);
  });
});

describe('Daemon: browser mode mismatch', () => {
  before(() => {
    stopDaemon();
  });

  after(() => {
    stopDaemon();
  });

  test('requesting a different mode than the running browser is refused (#194)', () => {
    // Headless daemon, so the request below asks for the opposite.
    clicker('daemon start --headless');
    clicker(`go ${baseURL}/example`);

    try {
      execSync(`${VIBIUM} --headless=false start`, {
        encoding: 'utf-8',
        timeout: 30000,
        stdio: 'pipe',
      });
      assert.fail('should refuse to attach a headed request to a headless browser');
    } catch (err) {
      const out = err.stdout + err.stderr;
      assert.match(out, /already running headless/, `got: ${out.slice(0, 200)}`);
      assert.match(out, /vibium stop/, 'should say how to resolve it');
    }

    // Matching mode still attaches, and so does no flag at all — the flag must
    // only assert a mode when the user actually typed it.
    assert.match(clicker('--headless start'), /already running/i);
    assert.match(clicker('start'), /already running/i);
  });
});
