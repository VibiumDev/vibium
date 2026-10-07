/**
 * CLI Tests: global flags on commands that disable cobra flag parsing
 *
 * fill, type, geolocation, sleep, mouse click, mouse move, and window parse
 * their flags inside Run, after the root's PersistentPreRunE has already
 * read them as unset. Regression tests for #482: --session and --channel
 * were accepted and silently ignored, so the command targeted the default
 * session and --channel validation (#314) never ran.
 *
 * All cases must short-circuit on validation before any daemon call, so no
 * browser or daemon is needed.
 */

const { test, describe } = require('node:test');
const assert = require('node:assert');
const { spawnSync } = require('node:child_process');
const { VIBIUM } = require('../helpers');

function run(args, env = {}) {
  const result = spawnSync(VIBIUM, args, {
    encoding: 'utf-8',
    timeout: 30000,
    env: { ...process.env, VIBIUM_SESSION: `global-flags-test-${process.pid}`, ...env },
  });
  assert.strictEqual(result.error, undefined, `spawn failed: ${result.error}`);
  return result;
}

// Full invocations that would be valid if the flag under test were honored.
const COMMANDS = {
  fill: ['fill', '#a', 'x'],
  type: ['type', '#a', 'x'],
  geolocation: ['geolocation', '37.8', '-122.4'],
  sleep: ['sleep', '100'],
  'mouse click': ['mouse', 'click', '100', '200'],
  'mouse move': ['mouse', 'move', '100', '200'],
  window: ['window', '1280', '720'],
};

describe('CLI: late-parsed commands honor global flag validation (#482)', () => {
  // --channel validation lives in the global-flag bridge; before the fix it
  // ran against an unset value on these four commands, so a bogus channel
  // was accepted and the setting dropped.
  for (const [cmd, argv] of Object.entries(COMMANDS)) {
    test(`${cmd} rejects --channel bogus like every other command`, () => {
      const result = run([...argv, '--channel', 'bogus']);
      assert.strictEqual(result.status, 1, `expected exit 1, got ${result.status}\nstdout: ${result.stdout}`);
      assert.match(result.stderr, /unsupported channel "bogus"/);
    });
  }

  // --session goes through the same bridge; an invalid name proves the value
  // now lands there instead of being read as empty.
  test('sleep rejects an invalid --session name', () => {
    const result = run(['sleep', '100', '--session', 'bad/name']);
    assert.strictEqual(result.status, 1, `expected exit 1, got ${result.status}\nstdout: ${result.stdout}`);
    assert.match(result.stderr, /invalid session name/);
  });

  test('fill rejects an invalid --session name given before the command', () => {
    const result = run(['--session', 'bad/name', 'fill', '#a', 'x']);
    assert.strictEqual(result.status, 1, `expected exit 1, got ${result.status}\nstdout: ${result.stdout}`);
    assert.match(result.stderr, /invalid session name/);
  });

  // Guardrail: a validation failure must not report the arity error instead,
  // which would mean the parse itself broke.
  test('geolocation with a negative positional still reaches channel validation', () => {
    const result = run(['geolocation', '37.8', '-122.4', '--channel', 'bogus']);
    assert.doesNotMatch(result.stderr, /accepts 2 arg\(s\)/);
    assert.match(result.stderr, /unsupported channel "bogus"/);
  });

  test('window with a negative position still reaches channel validation', () => {
    const result = run(['window', '1920', '1080', '-1920', '25', '--channel', 'bogus']);
    assert.doesNotMatch(result.stderr, /provide both width and height|unknown (shorthand )?flag/);
    assert.match(result.stderr, /unsupported channel "bogus"/);
  });

  // #620: the first applyGlobalFlags pass ran before the late parse, so the
  // VIBIUM_ENGINE_PATH check saw the default engine and rejected a command
  // that passed --engine firefox. --channel bogus keeps the case
  // short-circuiting before any daemon call.
  test('fill honors --engine firefox with VIBIUM_ENGINE_PATH set', () => {
    const result = run(['fill', '#a', 'x', '--engine', 'firefox', '--channel', 'bogus'],
      { VIBIUM_ENGINE_PATH: '/tmp/fakefox' });
    assert.doesNotMatch(result.stderr, /VIBIUM_ENGINE_PATH is not supported/);
    assert.match(result.stderr, /unsupported channel "bogus" for firefox/);
  });

  // The engine-path rejection itself must survive the fix.
  test('fill still rejects VIBIUM_ENGINE_PATH for chrome', () => {
    const result = run(['fill', '#a', 'x', '--engine', 'chrome'],
      { VIBIUM_ENGINE_PATH: '/tmp/fakefox' });
    assert.strictEqual(result.status, 1, `expected exit 1, got ${result.status}\nstdout: ${result.stdout}`);
    assert.match(result.stderr, /VIBIUM_ENGINE_PATH is not supported for engine "chrome"/);
  });

  // Engine validation now runs only on the late pass; it must still run.
  test('sleep rejects an unsupported --engine', () => {
    const result = run(['sleep', '100', '--engine', 'bogus']);
    assert.strictEqual(result.status, 1, `expected exit 1, got ${result.status}\nstdout: ${result.stdout}`);
    assert.match(result.stderr, /unsupported engine "bogus"/);
  });
});
