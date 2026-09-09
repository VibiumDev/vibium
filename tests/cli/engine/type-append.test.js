/**
 * CLI Tests: type and press act at the end of the value
 *
 * Same regression as the JS suite of the same name (#488), through the CLI,
 * which reaches the separate TypeInto/PressOn implementation rather than
 * the router handlers the language clients use.
 */

const { test, describe, after } = require("../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const { execSync } = require('node:child_process');
const { VIBIUM } = require("../../helpers");

const ALPHABET = 'abcdefghijklmnopqrstuvwxyz';
const PAGE = 'data:text/html,<input id="n" type="text" style="width:100px">';

function cli(args) {
  return execSync(`${VIBIUM} --headless ${args}`, { encoding: 'utf-8', timeout: 30000 });
}

after(() => {
  try { cli('stop'); } catch { /* no session left behind either way */ }
});

describe('CLI: type/press caret moves to the end of the value (#488)', () => {
  test('type appends when the value is wider than the field', () => {
    cli(`go '${PAGE}'`);
    cli(`fill "#n" "${ALPHABET}"`);
    cli(`type "#n" "XX"`);
    assert.strictEqual(cli(`value "#n"`).trim(), ALPHABET + 'XX');
  });

  test('press acts on the last character, whatever the field width', () => {
    cli(`go '${PAGE}'`);
    cli(`fill "#n" "${ALPHABET}"`);
    cli(`press "Backspace" "#n"`);
    assert.strictEqual(cli(`value "#n"`).trim(), ALPHABET.slice(0, -1));
  });
});
