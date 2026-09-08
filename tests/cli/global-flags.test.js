/**
 * CLI Tests: global flags on commands that disable cobra flag parsing
 *
 * fill, type, geolocation, and sleep parse their flags inside Run, after the
 * root's PersistentPreRunE has already read them as unset. Regression tests
 * for #482: --session and --channel were accepted and silently ignored, so
 * the command targeted the default session and --channel validation (#314)
 * never ran.
 *
 * All cases must short-circuit on validation before any daemon call, so no
 * browser or daemon is needed.
 */

const { test, describe } = require('node:test');
const assert = require('node:assert');
const { spawnSync } = require('node:child_process');
const { VIBIUM } = require('../helpers');

function run(args) {
  const result = spawnSync(VIBIUM, args, {
    encoding: 'utf-8',
    timeout: 30000,
    env: { ...process.env, VIBIUM_SESSION: `global-flags-test-${process.pid}` },
  });
  assert.strictEqual(result.error, undefined, `spawn failed: ${result.error}`);
  return result;
}

// Arguments that would be valid if the flag under test were honored.
const COMMANDS = {
  fill: ['#a', 'x'],
  type: ['#a', 'x'],
  geolocation: ['37.8', '-122.4'],
  sleep: ['100'],
};

describe('CLI: late-parsed commands honor global flag validation (#482)', () => {
  // --channel validation lives in the global-flag bridge; before the fix it
  // ran against an unset value on these four commands, so a bogus channel
  // was accepted and the setting dropped.
  for (const [cmd, args] of Object.entries(COMMANDS)) {
    test(`${cmd} rejects --channel bogus like every other command`, () => {
      const result = run([cmd, ...args, '--channel', 'bogus']);
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
});
