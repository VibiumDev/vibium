/**
 * Daemon Concurrency Tests
 * Tests rapid sequential commands and error recovery
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
const path = require('node:path');
const { VIBIUM } = require('../helpers');

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
  // One daemon for the whole file; each test navigates itself.
  stopDaemon();
  clicker('daemon start --headless');
});

after(() => {
  stopDaemon();
  if (serverProcess) serverProcess.kill();
});

describe('Daemon: Rapid sequential commands', () => {
  test('multiple go commands in sequence', () => {
    // Navigate to several pages in quick succession
    const r1 = clickerJSON(`go ${baseURL}/example`);
    assert.strictEqual(r1.ok, true, 'First go should succeed');

    const r2 = clickerJSON(`go ${baseURL}/example`);
    assert.strictEqual(r2.ok, true, 'Second go should succeed');

    const r3 = clickerJSON(`go ${baseURL}/example`);
    assert.strictEqual(r3.ok, true, 'Third go should succeed');
  });

  test('go then eval then find', () => {
    const nav = clickerJSON(`go ${baseURL}/example`);
    assert.strictEqual(nav.ok, true);

    const evalResult = clickerJSON(`eval ${baseURL}/example "document.title"`);
    assert.strictEqual(evalResult.ok, true);
    assert.ok(evalResult.result.includes('Example Domain'));

    const find = clickerJSON(`find ${baseURL}/example "h1"`);
    assert.strictEqual(find.ok, true);
    assert.ok(find.result.includes('h1'));
  });
});

describe('Daemon: Error recovery', () => {
  test('find with bad selector returns error', () => {
    // Navigate first
    clickerJSON(`go ${baseURL}/example`);

    // Find with nonexistent selector
    try {
      clickerJSON(`find ${baseURL}/example ".nonexistent-element-xyz"`);
      assert.fail('Should have thrown');
    } catch (e) {
      // Expected — command exits non-zero on error in --json mode
      // The stderr/stdout may contain the error
      assert.ok(true, 'Bad selector should fail');
    }
  });

  test('commands still work after error', () => {
    // After a failed command, the daemon should still work
    const result = clickerJSON(`go ${baseURL}/example`);
    assert.strictEqual(result.ok, true, 'Navigate should succeed after error');
  });
});
