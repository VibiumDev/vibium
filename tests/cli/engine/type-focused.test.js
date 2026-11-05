/**
 * CLI Tests: type with only text goes to whatever has focus
 *
 * `vibium type <text>` with no selector sends the keys to the focused
 * element, as `press` does, so a device with no DOM (a phone through
 * Roadie) can be typed into.
 */

const { test, describe, after } = require("../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const { execSync } = require('node:child_process');
const { VIBIUM } = require("../../helpers");

const PAGE = '<input id="a" type="text"><input id="b" type="text">';

function cli(args) {
  return execSync(`${VIBIUM} --headless ${args}`, { encoding: 'utf-8', timeout: 30000 });
}

after(() => {
  try { cli('stop'); } catch { /* no session left behind either way */ }
});

describe('CLI: type with only text types into the focused element', () => {
  test('the text lands in the focused field and nowhere else', () => {
    cli(`content '${PAGE}'`);
    cli(`focus "#b"`);
    cli(`type "calc"`);
    assert.strictEqual(cli(`value "#b"`).trim(), 'calc');
    assert.strictEqual(cli(`value "#a"`).trim(), '');
  });

  test('with a selector the element is focused first, as before', () => {
    cli(`content '${PAGE}'`);
    cli(`focus "#b"`);
    cli(`type "#a" "x"`);
    assert.strictEqual(cli(`value "#a"`).trim(), 'x');
    assert.strictEqual(cli(`value "#b"`).trim(), '');
  });
});
