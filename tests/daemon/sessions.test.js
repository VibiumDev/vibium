/**
 * Daemon Session Tests
 * Tests VIBIUM_SESSION / --session isolation: separate daemons, sockets, PIDs
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execSync } = require('node:child_process');
const { VIBIUM } = require('../helpers');

// Build a child env with any ambient session stripped, so bare commands
// deterministically target the default session; tests that need a session
// set it via overrides or --session.
function cleanEnv(overrides) {
  const env = { ...process.env };
  delete env.VIBIUM_SESSION;
  return Object.assign(env, overrides);
}

function clicker(args, opts = {}) {
  const result = execSync(`${VIBIUM} ${args}`, {
    encoding: 'utf-8',
    timeout: opts.timeout || 60000,
    env: cleanEnv(opts.env),
  });
  return result.trim();
}

function clickerJSON(args, opts = {}) {
  return JSON.parse(clicker(`${args} --json`, opts));
}

// Helper to stop a session's daemon (ignore errors if not running)
function stopDaemon(session) {
  try {
    const flag = session ? `--session ${session} ` : '';
    execSync(`${VIBIUM} ${flag}daemon stop`, {
      encoding: 'utf-8',
      timeout: 10000,
      env: cleanEnv(),
    });
  } catch (e) {
    // Daemon may not be running
  }
}

function stopAll() {
  stopDaemon();
  stopDaemon('s1');
  stopDaemon('s2');
}

describe('Daemon: Named sessions', () => {
  before(stopAll);
  after(stopAll);

  test('two sessions run separate daemons concurrently', () => {
    clicker('--session s1 daemon start --headless');
    clicker('--session s2 daemon start --headless');

    const s1 = clickerJSON('--session s1 daemon status');
    const s2 = clickerJSON('--session s2 daemon status');

    assert.strictEqual(s1.running, true, 's1 should be running');
    assert.strictEqual(s2.running, true, 's2 should be running');
    assert.strictEqual(s1.session, 's1', 'status should name session s1');
    assert.strictEqual(s2.session, 's2', 'status should name session s2');
    assert.notStrictEqual(s1.pid, s2.pid, 'sessions should be separate processes');
    assert.notStrictEqual(s1.socket, s2.socket, 'sessions should use separate sockets');

    const def = clicker('daemon status');
    assert.match(def, /not running/i, 'default session should be unaffected');
  });

  test('stopping one session leaves the other running', () => {
    clicker('--session s1 daemon stop');

    const s1 = clicker('--session s1 daemon status');
    assert.match(s1, /not running/i, 's1 should be stopped');

    const s2 = clickerJSON('--session s2 daemon status');
    assert.strictEqual(s2.running, true, 's2 should still be running');
  });

  test('VIBIUM_SESSION env var selects the session', () => {
    const s2 = clickerJSON('daemon status', { env: { VIBIUM_SESSION: 's2' } });
    assert.strictEqual(s2.running, true, 'env var should route to the s2 daemon');
    assert.strictEqual(s2.session, 's2');
  });

  test('auto-start uses the named session', () => {
    const nav = clickerJSON('go https://example.com --headless', {
      env: { VIBIUM_SESSION: 's1' },
    });
    assert.strictEqual(nav.ok, true, 'navigate should auto-start the s1 daemon');

    const s1 = clickerJSON('--session s1 daemon status');
    assert.strictEqual(s1.running, true, 's1 should be running after auto-start');

    const def = clicker('daemon status');
    assert.match(def, /not running/i, 'default session should be unaffected');
  });

  test('invalid session name is rejected', () => {
    assert.throws(
      () => clicker('--session bad/name daemon status'),
      /invalid session name/i,
      'should reject unsafe session names'
    );
  });
});
