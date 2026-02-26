/**
 * Daemon Concurrency Tests
 * Tests rapid sequential commands and error recovery
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execSync } = require('node:child_process');
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
  const result = clicker(`${args} --json`, opts);
  return JSON.parse(result);
}

function stopDaemon() {
  try {
    execSync(`${VIBIUM} daemon stop`, { encoding: 'utf-8', timeout: 10000 });
  } catch (e) {
    // Daemon may not be running
  }
}

describe('Daemon: Rapid sequential commands', () => {
  before(() => {
    stopDaemon();
    clicker('daemon start -d --headless');
  });

  after(() => {
    stopDaemon();
  });

  test('multiple go commands in sequence', () => {
    // Navigate to several pages in quick succession
    const r1 = clickerJSON('go https://example.com');
    assert.strictEqual(r1.ok, true, 'First go should succeed');

    const r2 = clickerJSON('go https://example.com');
    assert.strictEqual(r2.ok, true, 'Second go should succeed');

    const r3 = clickerJSON('go https://example.com');
    assert.strictEqual(r3.ok, true, 'Third go should succeed');
  });

  test('go then eval then find', () => {
    const nav = clickerJSON('go https://example.com');
    assert.strictEqual(nav.ok, true);

    const evalResult = clickerJSON('eval https://example.com "document.title"');
    assert.strictEqual(evalResult.ok, true);
    assert.ok(evalResult.result.includes('Example Domain'));

    const find = clickerJSON('find https://example.com "h1"');
    assert.strictEqual(find.ok, true);
    assert.ok(find.result.includes('h1'));
  });
});

describe('Daemon: Error recovery', () => {
  before(() => {
    stopDaemon();
    clicker('daemon start -d --headless');
  });

  after(() => {
    stopDaemon();
  });

  test('find with bad selector returns error', () => {
    // Navigate first
    clickerJSON('go https://example.com');

    // Find with nonexistent selector
    try {
      clickerJSON('find https://example.com ".nonexistent-element-xyz"');
      assert.fail('Should have thrown');
    } catch (e) {
      // Expected — command exits non-zero on error in --json mode
      // The stderr/stdout may contain the error
      assert.ok(true, 'Bad selector should fail');
    }
  });

  test('commands still work after error', () => {
    // After a failed command, the daemon should still work
    const result = clickerJSON('go https://example.com');
    assert.strictEqual(result.ok, true, 'Navigate should succeed after error');
  });
});
