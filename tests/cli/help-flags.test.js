/**
 * CLI Tests: -h/--help on commands that disable cobra flag parsing
 *
 * fill, type, geolocation, and sleep set DisableFlagParsing to support
 * negative-number positionals, which also bypasses cobra's built-in help
 * interception. Regression tests for #422 (bare --help returned an arity
 * error) and #423 (--help alongside positionals executed the command).
 *
 * All cases must short-circuit before any daemon call, so no browser or
 * daemon is needed.
 */

const { test, describe } = require('node:test');
const assert = require('node:assert');
const { spawnSync } = require('node:child_process');
const { VIBIUM } = require('../helpers');

function run(args) {
  const result = spawnSync(VIBIUM, args, {
    encoding: 'utf-8',
    timeout: 30000,
    env: { ...process.env, VIBIUM_SESSION: `help-flags-test-${process.pid}` },
  });
  assert.strictEqual(result.error, undefined, `spawn failed: ${result.error}`);
  return result;
}

const COMMANDS = {
  fill: { short: 'Clear an input field', executed: /Filled/ },
  type: { short: 'Type text into an element', executed: /Typed/ },
  geolocation: { short: 'Override the browser geolocation', executed: /Geolocation set/ },
  sleep: { short: 'Pause execution', executed: /Slept/ },
};

function assertHelp(result, cmd, invocation) {
  const { short, executed } = COMMANDS[cmd];
  assert.strictEqual(result.status, 0, `${invocation}: expected exit 0, got ${result.status}\nstderr: ${result.stderr}`);
  assert.match(result.stdout, /Usage:/, `${invocation}: help text should include usage`);
  assert.ok(result.stdout.includes(short), `${invocation}: help text should include the command description`);
  assert.doesNotMatch(result.stdout, executed, `${invocation}: command must not execute on a help request`);
  assert.doesNotMatch(result.stderr, /Error:/, `${invocation}: help request should not error`);
}

describe('CLI: help flags on DisableFlagParsing commands', () => {
  // #422: bare -h/--help must print help, not an arg-count error
  for (const cmd of Object.keys(COMMANDS)) {
    for (const flag of ['-h', '--help']) {
      test(`${cmd} ${flag} prints help`, () => {
        assertHelp(run([cmd, flag]), cmd, `${cmd} ${flag}`);
      });
    }
  }

  // #423: --help alongside positionals must print help, not execute
  test('sleep 5000 --help prints help without sleeping', () => {
    assertHelp(run(['sleep', '5000', '--help']), 'sleep', 'sleep 5000 --help');
  });

  test('fill <selector> <text> --help prints help without filling', () => {
    assertHelp(run(['fill', '#a', 'x', '--help']), 'fill', 'fill #a x --help');
  });

  test('fill --help <selector> <text> prints help (flag-first form)', () => {
    assertHelp(run(['fill', '--help', '#a', 'x']), 'fill', 'fill --help #a x');
  });

  test('type <selector> <text> -h prints help without typing', () => {
    assertHelp(run(['type', '#a', 'x', '-h']), 'type', 'type #a x -h');
  });

  test('geolocation 37 -122 --help prints help alongside a negative positional', () => {
    assertHelp(run(['geolocation', '37', '-122', '--help']), 'geolocation', 'geolocation 37 -122 --help');
  });

  // Guardrails: the behaviors DisableFlagParsing exists for must survive the fix
  test('negative numbers still parse as positionals, not flags', () => {
    const result = run(['geolocation', '-122']);
    assert.strictEqual(result.status, 1);
    assert.match(result.stderr, /accepts 2 arg\(s\), received 1/);
    assert.doesNotMatch(result.stderr, /unknown flag/);
  });

  test('unknown flags are still rejected', () => {
    const result = run(['fill', '--bogusflag']);
    assert.strictEqual(result.status, 1);
    assert.match(result.stderr, /unknown flag: --bogusflag/);
  });
});

describe('CLI: mcp inherits the root --headless flag (#452)', () => {
  // A section is everything from its header to the next blank line, so the
  // assertions cannot pass on a flag rendered in the other section.
  function flagSection(helpText, flag) {
    for (const header of ['Flags:', 'Global Flags:']) {
      const start = helpText.indexOf(header);
      if (start === -1) continue;
      const body = helpText.slice(start + header.length).split('\n\n')[0];
      if (body.includes(flag)) return header;
    }
    return null;
  }

  test('--headless renders under Global Flags with the shared description', () => {
    const result = spawnSync(VIBIUM, ['mcp', '--help'], { encoding: 'utf-8', timeout: 30000 });
    assert.strictEqual(result.status, 0);
    assert.strictEqual(
      flagSection(result.stdout, '--headless'),
      'Global Flags:',
      'mcp used to redeclare --headless locally, shadowing the root flag'
    );
    assert.match(result.stdout, /Hide browser window \(visible by default\)/, 'the root description should render');
    assert.doesNotMatch(result.stdout, /Launch browsers in headless mode/, 'the divergent local description must be gone');
  });

  test('--headless renders where every other command puts it', () => {
    const click = spawnSync(VIBIUM, ['click', '--help'], { encoding: 'utf-8', timeout: 30000 });
    const mcp = spawnSync(VIBIUM, ['mcp', '--help'], { encoding: 'utf-8', timeout: 30000 });
    assert.strictEqual(flagSection(click.stdout, '--headless'), 'Global Flags:');
    assert.strictEqual(flagSection(mcp.stdout, '--headless'), flagSection(click.stdout, '--headless'));
  });

  test('mcp --headless still parses', () => {
    // Guards the opposite regression: deleting the local flag must not drop
    // the option. Bad flags fail fast with a usage error before any server
    // starts; a flag cobra accepts leaves the server waiting on stdin, so
    // closed stdin plus a parse check keeps this browser-free.
    const result = spawnSync(VIBIUM, ['mcp', '--headless'], {
      encoding: 'utf-8',
      timeout: 30000,
      input: '',
    });
    assert.doesNotMatch(result.stderr || '', /unknown flag/, '--headless must still be accepted');
  });
});

describe('CLI: scroll help names its full surface (#445)', () => {
  test('all four directions and the real --selector meaning are in the help', () => {
    const result = spawnSync(VIBIUM, ['scroll', '--help'], { encoding: 'utf-8', timeout: 30000 });
    assert.strictEqual(result.status, 0);
    assert.match(result.stdout, /\[up\|down\|left\|right\]/, 'usage line should name the direction set');
    assert.match(result.stdout, /scroll within/, '--selector should say it scrolls within the element');
    assert.doesNotMatch(result.stdout, /element to scroll to\b/, 'the old scroll-to wording pointed at into-view');
  });
});
